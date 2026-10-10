// The pool: up to `size` connections shared by every task of the program,
// opened when they are needed and kept for the next call. A statement takes
// a connection for as long as it runs and gives it back; a transaction
// keeps one from `begin` to its end.
use otel
use time

/// Where the pool's shared state lives: handles that every copy of the `Pool`
/// (and the task that looks after it) sees alike.
struct PoolState {
  // connections waiting for a statement, with when each was put back
  idle: Mutex<MutableList<Idle>>
  // one permit per connection in use: the pool's `size`
  slots:  Semaphore
  closed: Atomic<bool>
}

struct Idle {
  conn:  Conn
  since: i64
}

fun newIdleList(): Mutex<MutableList<Idle>> {
  val empty: MutableList<Idle> = []
  Mutex(value: empty)
}

/// A pool of connections to one database, made by `db.open`. It holds a task
/// (its connections are checked in the background, D111), so it is
/// `with`-bound or returned; closing it closes every idle connection, and
/// the ones in use are closed as they come back.
///
/// ```veles
/// with pool = try db.open(url: Secret.of(env.databaseUrl), size: 10)
/// val users = try pool.query<User>(sql"select id, name from users where age > ${18}")
/// val inserted = try pool.exec(sql"insert into users (name) values (${name})")
/// ```
public struct Pool {
  target: Target
  size:   i64
  /// How long a statement may run before it is abandoned and its connection
  /// dropped; each call can set its own.
  public timeout: Duration
  connectTimeout: Duration
  state:          PoolState
  task:           Task<()>

  /// One statement, its rows read as `T` (a struct by column name, or a scalar
  /// from the first column). `timeout` replaces the pool's for this call.
  public fun query<T: Decodable>(statement: Sql, timeout: Duration? = null): List<T> suspends throws DbError | DecodeError {
    with lease = try this.acquire()
    val rows = try runTraced(lease.conn, statement, timeout ?: this.timeout)
    try decodeRows<T>(rows)
  }

  /// The first row of a statement as `T`, or `null` when there is none.
  public fun queryOne<T: Decodable>(statement: Sql, timeout: Duration? = null): T? suspends throws DbError | DecodeError {
    val all = try this.query<T>(statement, timeout)
    all.at(0)
  }

  /// A statement whose rows are not wanted — insert, update, delete, DDL —
  /// and how many rows it affected.
  public fun exec(statement: Sql, timeout: Duration? = null): i64 suspends throws DbError {
    with lease = try this.acquire()
    val rows = try runTraced(lease.conn, statement, timeout ?: this.timeout)
    rows.affected()
  }

  /// Checks that the database answers.
  public fun ping() suspends throws DbError {
    val _ = try this.exec(Sql.dangerouslyRaw("select 1"), Duration.seconds(5))
  }

  /// Starts a transaction on a connection of its own. It ends with
  /// `commit()`; leaving the block any other way — an error, a panic, a
  /// cancelled task, or forgetting — abandons it, and the database rolls
  /// back what it did.
  ///
  /// ```veles
  /// with tx = try pool.begin()
  /// try tx.exec(sql"update accounts set balance = balance - ${n} where id = ${from}")
  /// try tx.exec(sql"update accounts set balance = balance + ${n} where id = ${to}")
  /// try tx.commit()
  /// ```
  public fun begin(timeout: Duration? = null): Tx suspends throws DbError {
    val lease = try this.acquire()
    val tx = Tx(lease, timeout: timeout ?: this.timeout, finished: Atomic(value: false))
    try tx.start() catch (e) {
      lease.conn.close()
      lease.close()
      throw e
    }
    tx
  }

  // a connection for the caller, waiting for a free place if the pool is at its size
  fun acquire(): Lease suspends throws DbError {
    if (this.state.closed.load()) throw DbError(kind: ErrorKind.Closed, text: "the pool is closed")
    val permit = this.state.slots.acquire()
    if (this.state.closed.load()) {
      permit.close()
      throw DbError(kind: ErrorKind.Closed, text: "the pool is closed")
    }
    var conn = this.takeIdle()
    if (conn == null) {
      conn = Conn.open(this.target, this.connectTimeout) catch (e) {
        permit.close()
        throw e
      }
    }
    Lease(conn, permit, state: this.state, size: this.size)
  }

  // the most recently used connection that still works
  fun takeIdle(): Conn? {
    loop {
      val entry = this.state.idle.withLock(list => list.pop())
      if (entry == null) return null
      if (entry.conn.isUsable()) return entry.conn
      entry.conn.close()
    }
  }

  implement Closeable {
    fun close() {
      this.state.closed.store(true)
      val all = this.state.idle.withLock(list => {
        val taken = list.toList()
        list.clear()
        taken
      })
      loop (entry in all) {
        entry.conn.close()
      }
    }
  }
}

/// A connection on loan to one call or transaction; closing it gives the
/// connection back (or drops it if it cannot be used again) and the place in
/// the pool with it.
struct Lease {
  conn:   Conn
  permit: Permit
  state:  PoolState
  size:   i64

  implement Closeable {
    fun close() {
      val keep = this.conn.isUsable() && !this.conn.inTransaction.load() && !this.state.closed.load()
      if (keep) {
        val full = this.state.idle.withLock(list => {
          if (list.len() >= this.size) {
            true
          } else {
            list.push(Idle(conn: this.conn, since: time.monotonicNanos()))
            false
          }
        })
        if (full) this.conn.close()
      } else {
        this.conn.close()
      }
      this.permit.close()
    }
  }
}

/// A transaction: statements that succeed or fail together. See `Pool.begin`.
public struct Tx {
  lease:    Lease
  timeout:  Duration
  finished: Atomic<bool>

  fun start() suspends throws DbError {
    val _ = try this.lease.conn.run(Sql.dangerouslyRaw("begin"), this.timeout)
  }

  /// One statement in the transaction, as `Pool.query`.
  public fun query<T: Decodable>(statement: Sql, timeout: Duration? = null): List<T> suspends throws DbError | DecodeError {
    try this.check()
    val rows = try runTraced(this.lease.conn, statement, timeout ?: this.timeout)
    try decodeRows<T>(rows)
  }

  /// The first row as `T`, or `null`.
  public fun queryOne<T: Decodable>(statement: Sql, timeout: Duration? = null): T? suspends throws DbError | DecodeError {
    val all = try this.query<T>(statement, timeout)
    all.at(0)
  }

  /// A statement whose rows are not wanted; the number of rows it affected.
  public fun exec(statement: Sql, timeout: Duration? = null): i64 suspends throws DbError {
    try this.check()
    val rows = try runTraced(this.lease.conn, statement, timeout ?: this.timeout)
    rows.affected()
  }

  /// Makes the transaction's changes permanent. A transaction that failed a
  /// statement can only be rolled back, and the database says so here.
  public fun commit() suspends throws DbError {
    try this.check()
    this.finished.store(true)
    val _ = try this.lease.conn.run(Sql.dangerouslyRaw("commit"), this.timeout)
  }

  /// Undoes the transaction's changes. Leaving the block without `commit()`
  /// does the same: the close sends `ROLLBACK` and waits for it (D147), and
  /// the connection goes back to the pool; call `rollback()` to see its error.
  public fun rollback() suspends throws DbError {
    if (this.finished.swap(true)) return
    val _ = try this.lease.conn.run(Sql.dangerouslyRaw("rollback"), this.timeout)
  }

  fun check() throws DbError {
    if (this.finished.load()) throw DbError(kind: ErrorKind.Closed, text: "the transaction has ended")
  }

  implement Closeable {
    fun close() suspends {
      // a transaction neither committed nor rolled back is rolled back here, within
      // the transaction's timeout; if that fails the connection is dropped, and the
      // server rolls the transaction back as it notices
      if (!this.finished.swap(true)) {
        val undone = this.lease.conn.run(Sql.dangerouslyRaw("rollback"), this.timeout)
        if (undone is Err) this.lease.conn.close()
      }
      this.lease.close()
    }
  }
}

// the statement on a connection, as a client span while `otel` runs: the text
// with its placeholders, never the values
fun runTraced(conn: Conn, statement: Sql, timeout: Duration): Rows suspends throws DbError {
  if (!otel.active()) return try conn.run(statement, timeout)
  val text = statement.text()
  val verb = (text.trim().split(" ").at(0) ?: "").toUpper()
  with span = otel.span(if (verb.isEmpty()) "db" else verb, attrs: [otel.attr("db.system.name", "postgresql"), otel.attr("db.query.text", text), otel.attr("server.address", conn.target)], kind: otel.SpanKind.Client)
  when (conn.run(statement, timeout)) {
    is Ok(rows) => {
      span.end()
      rows
    }
    is Err(e)   => {
      span.set(otel.attr("error.type", "${e.kind}"))
      if (!e.code.isEmpty()) span.set(otel.attr("db.response.status_code", e.code))
      span.fail(e)
      throw e
    }
  }
}

/// Connects to the database at `url` and returns a pool of up to `size`
/// connections; one is opened now, so a wrong host or password fails here
/// rather than at the first query.
///
/// `url` is `postgres://user:password@host:5432/database`, held as a `Secret`
/// so it is never printed; the user and password are %-escaped where they hold
/// `@`, `:` or `/`. Query parameters: `sslmode` — `verify-full` (TLS, the
/// certificate and host name checked: the default for any host but the machine
/// itself), `disable` (the default for localhost) or `insecure` (TLS without
/// checking the certificate) — `sslrootcert` (a PEM file of the authority to
/// trust instead of the system's) and `application_name`.
///
/// ```veles
/// with pool = try db.open(url: Secret.of("postgres://app:s3cret@db.internal/shop"), size: 10)
/// ```
public fun open(
  url: Secret<string>,
  size: i64 = 10,
  timeout: Duration = Duration.seconds(30),
  connectTimeout: Duration = Duration.seconds(10),
  application: string = "veles",
  checkEvery: Duration = Duration.seconds(30),
  idleTimeout: Duration = Duration.minutes(5),
): Pool suspends throws DbError {
  if (size < 1) panic("db.open: size must be at least 1, got $size")
  val target = try parseTarget(url, application)
  val first = try Conn.open(target, connectTimeout)
  val state = PoolState(idle: newIdleList(), slots: Semaphore(permits: size), closed: Atomic(value: false))
  state.idle.withLock(list => {
    list.push(Idle(conn: first, since: time.monotonicNanos()))
  })
  Pool(target, size, timeout, connectTimeout, state, task: async maintain(state, checkEvery, idleTimeout))
}

// Looks after the idle connections until the pool closes: one that has sat
// unused past `idleTimeout` is closed, and the others are asked that they
// are alive, so a restart of the database or a firewall that dropped them
// is found here and not by the next statement.
fun maintain(state: PoolState, every: Duration, idleTimeout: Duration) suspends {
  loop {
    await sleep(every)
    if (state.closed.load()) return
    val now = time.monotonicNanos()
    val entries = state.idle.withLock(list => {
      val taken = list.toList()
      list.clear()
      taken
    })
    loop (entry in entries) {
      val tooOld = now - entry.since > idleTimeout.toNanos()
      if (tooOld || !entry.conn.isUsable()) {
        entry.conn.close()
        continue
      }
      val alive = entry.conn.run(Sql.dangerouslyRaw("select 1"), Duration.seconds(5))
      when (alive) {
        is Ok(_)  => state.idle.withLock(list => {
          list.push(Idle(conn: entry.conn, since: entry.since))
        })
        is Err(_) => entry.conn.close()
      }
    }
  }
}
