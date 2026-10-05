// The pool and its transactions against a real PostgreSQL (see conn.test.vs: these
// pass without doing anything when VELES_TEST_PG_URL is not set). Each test makes
// its own table, named by a random suffix, so tests may run side by side.
use crypto, io, time

struct Account {
  id:       i64
  owner:    string
  balance:  i64
  note:     string?
  openedAt: time.Timestamp
  implement Decodable
}

struct Count {
  n: i64
  implement Decodable
}

// a table name no other test uses
test fun tableName(): string = "veles_t_" + crypto.randomBytes(6).map(b => (if (b < 16) "0" else "") + b.toI64().toString(radix: 16)).join("")

test fun createAccounts(pool: Pool, table: Sql) suspends throws DbError {
  val _ = try pool.exec(sql"create table ${table} (id bigserial primary key, owner text not null unique, balance bigint not null default 0, note text, opened_at timestamptz not null default now())")
}

test fun dropTable(pool: Pool, table: Sql) suspends {
  val _ = pool.exec(sql"drop table if exists ${table}")
}

test "rows read into structs; values travel apart from the text" {
  val url = testUrl() ?: return
  with pool = try open(url)
  val name = tableName()
  val table = ident(name) ?: fail("a plain name")
  try createAccounts(pool, table)
  val noNote: string? = null
  val evil = "Robert'); drop table ${name};--"
  val inserted = try pool.exec(sql"insert into ${table} (owner, balance, note) values (${evil}, ${100}, ${"noted"}), (${"Ada"}, ${250}, ${noNote})")
  expect(inserted == 2)
  val all = try pool.query<Account>(sql"select id, owner, balance, note, opened_at from ${table} order by balance")
  expect(all.len() == 2)
  expect(all.at(0)?.owner == evil)
  expect(all.at(0)?.note == "noted")
  expect(all.at(1)?.owner == "Ada")
  expect(all.at(1)?.note == null)
  expect(time.now().toMicros() - (all.at(1)?.openedAt?.toMicros() ?: 0) < 60000000)
  val one = try pool.queryOne<Account>(sql"select id, owner, balance, note, opened_at from ${table} where owner = ${"Ada"}")
  expect(one?.balance == 250)
  val none = try pool.queryOne<Account>(sql"select id, owner, balance, note, opened_at from ${table} where owner = ${"nobody"}")
  expect(none == null)
  val total = try pool.query<i64>(sql"select sum(balance)::bigint from ${table}")
  expect(total == [350])
  val updated = try pool.exec(sql"update ${table} set balance = balance + ${1} where balance > ${0}")
  expect(updated == 2)
  val removed = try pool.exec(sql"delete from ${table}")
  expect(removed == 2)
  dropTable(pool, table)
}

struct Echo {
  a: i64
  b: f64
  c: bool
  d: string
  e: List<u8>
  f: string
  g: time.Timestamp
  h: string
  implement Decodable
}

test "every standard parameter type arrives as itself" {
  val url = testUrl() ?: return
  with pool = try open(url)
  val bytes: List<u8> = [0, 1, 254, 255]
  val id = crypto.uuidV4()
  val at = time.parseRfc3339("2026-10-05T10:34:56.789012Z") ?: fail("a timestamp")
  val rows = try pool.query<Echo>(sql"select ${7}::bigint as a, ${2.5}::float8 as b, ${true}::bool as c, ${"héllo wörld ☃"}::text as d, ${bytes}::bytea as e, ${id}::text as f, ${at}::timestamptz as g, ${Duration.millis(1500)}::interval::text as h")
  val r = rows.at(0) ?: fail("a row")
  expect(r.a == 7)
  expect(r.b == 2.5)
  expect(r.c)
  expect(r.d == "héllo wörld ☃")
  expect(r.e == bytes)
  expect(r.f == "$id")
  // whatever zone the session is in, it is the same instant
  expect(r.g.toMicros() == at.toMicros())
  expect(r.h == "00:00:01.5")
}

test "a constraint violation names its code and leaves the pool usable" {
  val url = testUrl() ?: return
  with pool = try open(url, size: 1)
  val name = tableName()
  val table = ident(name) ?: fail("a plain name")
  try createAccounts(pool, table)
  val _ = try pool.exec(sql"insert into ${table} (owner) values (${"a"})")
  val again = pool.exec(sql"insert into ${table} (owner) values (${"a"})")
  when (again) {
    is Ok(_)  => fail("a duplicate owner must be refused")
    is Err(e) => {
      expect(e.kind == ErrorKind.Server)
      expect(e.code == "23505")
      expect(e.isUniqueViolation())
      expect(e.isConstraintViolation())
      expect(!e.isRetryable())
    }
  }
  // the one connection of the pool is still in order
  expect(try pool.query<i64>(sql"select count(*)::bigint from ${table}") == [1])
  val syntax = pool.exec(Sql.dangerouslyRaw("selct 1"))
  expect((syntax catch (e) {
    if (e.code == "42601") 1 else 0
  }) == 1)
  expect(try pool.query<i64>(sql"select 1") == [1])
  dropTable(pool, table)
}

test fun transfer(pool: Pool, table: Sql) suspends throws DbError | DecodeError {
  with tx = try pool.begin()
  val _ = try tx.exec(sql"update ${table} set balance = balance - ${30} where owner = ${"a"}")
  val _ = try tx.exec(sql"update ${table} set balance = balance + ${30} where owner = ${"b"}")
  try tx.commit()
}

test fun abandon(pool: Pool, table: Sql) suspends throws DbError | DecodeError {
  with tx = try pool.begin()
  val _ = try tx.exec(sql"update ${table} set balance = 0")
  expect(try tx.query<i64>(sql"select sum(balance)::bigint from ${table}") == [0])
}

test fun rollbackByName(pool: Pool, table: Sql) suspends throws DbError {
  with tx = try pool.begin()
  val _ = try tx.exec(sql"delete from ${table}")
  try tx.rollback()
}

test fun poisoned(pool: Pool, table: Sql) suspends throws DbError {
  with tx = try pool.begin()
  expect(tx.exec(sql"insert into ${table} (owner) values (${"a"})") is Err)
  expect(tx.query<i64>(sql"select 1") is Err)
}

test fun endedTransaction(pool: Pool) suspends throws DbError {
  with tx = try pool.begin()
  try tx.commit()
  expect(tx.exec(sql"select 1") is Err)
}

test "a transaction commits, rolls back, and abandons what it did not commit" {
  val url = testUrl() ?: return
  with pool = try open(url)
  val name = tableName()
  val table = ident(name) ?: fail("a plain name")
  try createAccounts(pool, table)
  val _ = try pool.exec(sql"insert into ${table} (owner, balance) values (${"a"}, ${100}), (${"b"}, ${0})")
  try transfer(pool, table)
  expect(try pool.query<i64>(sql"select balance from ${table} order by owner") == [70, 30])
  // left without a commit: the database rolls it back
  try abandon(pool, table)
  expect(try pool.query<i64>(sql"select balance from ${table} order by owner") == [70, 30])
  // rolled back by name; the connection returns to the pool
  try rollbackByName(pool, table)
  expect(try pool.query<i64>(sql"select count(*)::bigint from ${table}") == [2])
  // a failed statement poisons the transaction; the pool is fine afterwards
  try poisoned(pool, table)
  expect(try pool.query<i64>(sql"select count(*)::bigint from ${table}") == [2])
  // after the end, the transaction refuses more
  try endedTransaction(pool)
  dropTable(pool, table)
}

test "a pool of four serves fifty tasks on at most four connections" {
  val url = testUrl() ?: return
  with pool = try open(url, size: 4)
  val set: MutableSet<i64> = []
  val seen = Mutex(value: set)
  scope {
    loop (i in 0..<50) {
      async worker(pool, i, seen)
    }
  }
  // backend process ids: how many different connections did the work?
  expect(seen.withLock(s => s.len()) <= 4)
  expect(seen.withLock(s => s.len()) >= 1)
}

test fun worker(pool: Pool, i: i64, seen: Mutex<MutableSet<i64>>) suspends {
  val pid = pool.query<i64>(sql"select pg_backend_pid()::bigint from pg_sleep(0.02)") catch (e) {
    fail("worker $i: ${e.message()}")
  }
  seen.withLock(s => {
    s.add(pid.at(0) ?: 0)
  })
}

test "a statement past its time is abandoned, and the pool carries on" {
  val url = testUrl() ?: return
  with pool = try open(url, size: 1)
  val slow = pool.exec(sql"select pg_sleep(5)", timeout: Duration.millis(200))
  when (slow) {
    is Ok(_)  => fail("the statement should have timed out")
    is Err(e) => expect(e.kind == ErrorKind.Timeout)
  }
  expect(try pool.query<i64>(sql"select 2") == [2])
}

test "a task cancelled mid-statement does not poison the connection" {
  val url = testUrl() ?: return
  with pool = try open(url, size: 1)
  val r = withTimeout(Duration.millis(150), () => try pool.exec(sql"select pg_sleep(5)"))
  expect(r is Err)
  expect(try pool.query<i64>(sql"select 3") == [3])
}

test fun shutDown(pool: Pool) {
  pool.close()
}

test "ping, and a closed pool refuses" {
  val url = testUrl() ?: return
  with pool = try open(url)
  try pool.ping()
  shutDown(pool)
  val after = pool.ping()
  when (after) {
    is Ok(_)  => fail("a closed pool must refuse")
    is Err(e) => expect(e.kind == ErrorKind.Closed)
  }
}

test fun openBad() suspends throws DbError {
  with pool = try open(Secret.of("postgres://nobody@127.0.0.1:1/none"))
  try pool.ping()
}

test "a connection that fails to open is an error at open, not at the first query" {
  when (openBad()) {
    is Ok(_)  => fail("nothing listens on port 1")
    is Err(e) => expect(e.kind == ErrorKind.Connection)
  }
}
