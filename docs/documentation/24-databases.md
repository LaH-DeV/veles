# Databases

`std/db` talks to PostgreSQL. A query is written with `sql"…"`, whose values travel to the server apart from its text, so nothing a value holds can change what the query means; rows come back as structs; a pool of connections is shared by every task.

```veles
// fragment
use db
use io

struct User {
  id: i64
  fullName: string
  email: string?
  implement Decodable
}

fun main() suspends throws db.DbError | DecodeError {
  with pool = try db.open(url: Secret.of("postgres://app:s3cret@db.internal/shop"), size: 10)
  val minAge = 18
  val users = try pool.query<User>(sql"select id, full_name, email from users where age > ${minAge}")
  loop (u in users) {
    io.println("${u.id} ${u.fullName}")
  }
}
```

## The pool

`db.open(url:, size: 10)` connects once, so a wrong host or password fails here rather than at the first query, and returns a `Pool` of up to `size` connections. Connections are opened when they are needed and kept for the next call; a call that finds all `size` in use waits for one. The pool holds a task that checks the idle connections in the background (it closes ones idle for five minutes and finds ones the network dropped), so — like every value that holds a task — it is received with `with`, or returned.

The URL is a `Secret`, so it never shows in a log. It is `postgres://user:password@host:5432/database`; `@`, `:` and `/` inside the user or the password are written `%40`, `%3A`, `%2F`. Three query parameters are read: `sslmode`, `sslrootcert` and `application_name`; any other is an error, so a misspelling cannot silently weaken a connection.

**TLS is on, and verified, for every host but the machine itself.** `sslmode=verify-full` (the default for a remote host) checks the certificate against the system's trusted roots — or against the PEM file in `sslrootcert`, for a private authority — and checks the host name. `sslmode=disable` is the default for `localhost`, `127.0.0.1` and `::1`. `sslmode=insecure` is TLS without the check: the traffic is encrypted but anyone can be the server. There is no `prefer`: a connection that quietly falls back to plain text is the one an attacker asks for. A server that asks for the password in clear text is refused unless the connection is encrypted or local; MD5 passwords are refused (`scram-sha-256` is the server's modern default); SCRAM-SHA-256 is spoken, and the server is made to prove it knows the password too.

## A query is a template

`sql"…"` is a template literal ([chapter 2](02-values-and-strings.md)): the text between the `${…}` stays text, and every value becomes a parameter of the statement.

```veles
// fragment
val name = "O'Brien'); drop table users;--"
val q = sql"select id from users where name = ${name} and age > ${18}"
// the server receives:  select id from users where name = $1 and age > $2
// and, separately, the values 'O''Brien…' and 18
```

A `Sql` can only be made by `sql"…"` — never from a `string` that was computed — and every query method takes a `Sql`, so there is no way to build a query by pasting a user's text into it. What can be a value: the numbers, `bool`, `string`, `List<u8>` (bytea), `Timestamp`, `Duration` (an interval), `Uuid`, a `T?` of any of those (`null` is SQL NULL), a `Secret` (sent like any other value, never printed or traced) — and your own types, by saying which of those they are sent as:

```veles
// fragment
struct UserId {
  value: i64
}

implement db.Param for UserId {
  fun toArg(): db.Arg => this.value.toArg()
}
```

A `string` is sent without a type, so the server reads it as whatever the column is — a `uuid`, a `timestamptz`, an enum — and a number is sent as a number.

A `Sql` among the values is spliced in as query text, with its own values renumbered. That is how a query is assembled in pieces:

```veles
// fragment
val adult = sql"age >= ${18}"
val q = sql"select id from users where active = ${true} and ${adult}"
```

What a value cannot be is a table or a column name. For those there is `db.ident(name)`, which quotes a name made of letters, digits, `_` and dots and is `null` for anything else — so a name that came from a user is checked before it reaches a query:

```veles
// fragment
val order = db.ident(column) ?: throw http.badRequest("unknown column")
val q = sql"select id from users order by ${order} desc"
```

`Sql.dangerouslyRaw(text)` exists for what is not data — a migration read from a file — and is named for what it risks: whatever is in `text` is the query.

## Rows

`pool.query<T>(sql)` runs the statement and reads each row as a `T`. A struct is the row: its fields are the columns, with names in `snake_case` (`fullName` is `full_name`), and `@key(db: "user_id")` says a column outright. A scalar target reads the first column.

```veles
// fragment
val ids = try pool.query<i64>(sql"select id from users")              // List<i64>
val one = try pool.queryOne<User>(sql"select * from users where id = ${id}")   // User?: null when there is no row
val inserted = try pool.exec(sql"insert into users (email) values (${email})")  // rows affected
```

The server sends every value as text and the decoder parses it, so a column and a field of the wrong kind is a problem at that column — all of a row's problems are reported together, as for JSON — and the row is not built. What is read: `i8`…`i64`, `u8`…`u64`, `f32`, `f64`, `bool`, `string`, `Timestamp` (`timestamptz` and `timestamp`, read as UTC), `Uuid`, a `T?` (NULL), enums by name as in [chapter 18](18-codable-and-json.md), a `List<u8>` from `bytea`, and a `List<T>` from a one-dimensional array of scalars. Not yet: `interval` and `numeric` with a fraction into a number type, `json`/`jsonb` into a struct (read it as `string` and `json.decode` it), and arrays of arrays.

## Errors

Every failure is a `DbError`. Its `kind` says where the failure is — `Connection`, `Closed`, `Timeout`, `Auth`, `Server`, `Protocol`, `Config` — and for a `Server` failure `code` is PostgreSQL's SQLSTATE, with `detail` and `hint`. The common questions have methods:

```veles
// fragment
val inserted = pool.exec(sql"insert into users (email) values (${email})") catch (e) {
  if (e.isUniqueViolation()) return Response.text("that address is taken", status: Status.conflict)
  throw e
}
```

`isUniqueViolation()`, `isForeignKeyViolation()`, `isConstraintViolation()` (any constraint), `isRetryable()` (a deadlock or a serialization failure: run the transaction again) and `isCancelled()`. A statement the server refused leaves the connection in order and in the pool; one that ran out of time, was cancelled, or lost its connection does not — it is dropped, and the next call opens a new one.

## Transactions

```veles
// fragment
with tx = try pool.begin()
try tx.exec(sql"update accounts set balance = balance - ${n} where id = ${from}")
try tx.exec(sql"update accounts set balance = balance + ${n} where id = ${to}")
try tx.commit()
```

A transaction keeps one connection from `begin` to its end. It ends with `commit()` or `rollback()`. **Leaving the block any other way — an error, a panic, a cancelled task, or forgetting `commit` — rolls the transaction back:** the `with`'s close sends `ROLLBACK` and waits for it (a `close()` may suspend, D147), within the transaction's timeout, and the connection goes back to the pool; if the rollback fails, the connection is dropped and the database rolls back on its own. Call `tx.rollback()` to see its error. After a failed statement the transaction can only be rolled back (PostgreSQL says so itself).

## Time

Every call has a statement timeout — the pool's, 30 seconds by default, or `timeout:` for that call. Past it the statement is abandoned, the connection is dropped (which cancels the statement on the server) and the call throws `DbError(kind: Timeout)`. A task that is cancelled in the middle of a statement — `withTimeout`, a stopping server — leaves the same way, and the pool carries on.

## Observability

While `otel` runs ([chapter 23](23-observability.md)), each statement is a client span named for its verb (`SELECT`, `UPDATE`) with `db.query.text` — the text with its `$1`, `$2`, never the values — and a failure's SQLSTATE.

## Underneath

The driver is written in Veles: the PostgreSQL wire protocol (version 3, the extended query protocol), SCRAM-SHA-256 on `std/crypto`, and TLS on `std/tls`. There is no C library to install, and every wait is a task suspension like any socket's. A connection runs one statement at a time; each statement is sent as its own parse, bind and execute, with the values in the bind message.

Not in the module yet: prepared-statement caching, streaming a large result row by row (a result is read whole), `COPY`, `LISTEN`/`NOTIFY`, other databases, `interval` and `json` columns into typed fields, and SASLprep for passwords with non-ASCII characters (the password is used as written).

Next: back to the [index](index.md).
