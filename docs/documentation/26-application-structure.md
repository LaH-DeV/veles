# 26. Application structure

A program of any size has values every part of it uses: a table of
constants, a counter, a database, a clock. Veles has a place for each
kind, and the place says what the value is allowed to do. This chapter
is about the last two kinds — the ones that open something, can fail and
must be closed, and the ones a test replaces — and how a program hands
them from `main` to the code that needs them (D211).

## Three kinds of shared value

| What it is                                                  | Where it lives                                                         | Chapter |
| ----------------------------------------------------------- | ---------------------------------------------------------------------- | ------- |
| data that never changes: a table, a set of keywords, limits | `const`, a module `val`, a `lazy val` built on first read              | 2, 12   |
| state every task changes: a counter, a cache                | a module `val` of `Atomic`, `Mutex`, `RwLock` …; an executor's state   | 12      |
| a resource: a database pool, a client, a log sink, a file   | made in `main` with `with`, handed down as a parameter or a task-local | here    |

The first two are module values: built before `main` (or on first read,
for a `lazy val`) from code that cannot fail or wait. A resource is
different on every count. Opening it can fail (the database is down),
it can wait (the network), it must be closed, and a test wants a
different one. So it is never a module value:

```veles
// fragment
val settings = try fs.readFile("settings.json")  // error: a global initializer cannot fail: there is no caller to pass the error to
val db = try Db.connect(url)                       // the same: a resource is made in `main`
```

There is no caller to give a module value's error to, and nothing would
close it. The deeper reason is testing: a value made by I/O when the
program starts cannot be replaced in a test, and every `veles test` run
would open it. Go's `init()` functions taught that lesson ("don't do I/O
in init"); Veles makes it a rule.

## Resources are made in `main`

`main` opens what the program uses, with `with`, so each is closed when
`main` ends, in the reverse order — whether it returns, throws or is
interrupted:

```veles
// fragment
fun main() throws config.Error | db.DbError | IoError {
  val settings = try config.load<Settings>(files: [".env"])
  with pool = try db.open(settings.databaseUrl)
  with listener = try net.listen(port: settings.port)
  http.serve(listener, routes(App(store: SqlStore(pool), name: settings.name)))
}
```

Everything the program depends on is on those few lines: what is opened,
in what order, with which settings, and what can stop it. Nothing else
in the program opens a connection behind your back, and reading `main`
tells you what the program touches.

## Handing it down

What `main` makes, the rest of the program receives. The plainest way is
a parameter, and the tidiest form of it is one struct holding everything
the handlers need. A function's signature then says what it uses, and a
test builds its own:

```veles
use io
use time

/// Where notes are kept: a file or a database in production, a list in a test.
trait Store : Sendable {
  fun add(text: string): i64
  fun count(): i64
}

/// Notes kept in memory, shared by every task that holds the store.
struct MemoryStore {
  notes: Mutex<MutableList<string>> = Mutex(value: [])

  implement Store {
    fun add(text: string): i64 => notes.withLock(list => {
      list.push(text)
      list.len()
    })
    fun count(): i64 => notes.withLock(list => list.len())
  }
}

/// What the handlers need, made once in `main` and handed down.
struct App {
  store: Store
  name:  string
}

/// The clock: the system's unless a caller binds another (D72).
trait Clock : Sendable {
  fun now(): time.Timestamp
}

struct SystemClock {
  implement Clock {
    fun now(): time.Timestamp => time.now()
  }
}

struct FixedClock {
  at: time.Timestamp
  implement Clock {
    fun now(): time.Timestamp => at
  }
}

val clock = TaskLocal<Clock>(fallback: SystemClock())

fun stamp(text: string): string => "[${time.formatRfc3339(clock.get().now())}] $text"

fun handle(app: App, request: string): string {
  val number = app.store.add(stamp(request))
  "${app.name}: note $number of ${app.store.count()}"
}

fun main() {
  val app = App(store: MemoryStore(), name: "notes")
  with fixed = clock.bind(FixedClock(at: time.Timestamp.ofSeconds(0)))
  io.println(handle(app, "buy milk"))
  io.println(handle(app, "call home"))
  io.println(stamp("done"))
}

test "a handler stores the note it is given" {
  val app = App(store: MemoryStore(), name: "test")
  expect(handle(app, "x") == "test: note 1 of 1")
}

test "stamps come from the clock in force" {
  with fixed = clock.bind(FixedClock(at: time.Timestamp.ofSeconds(60)))
  expect(stamp("hi") == "[1970-01-01T00:01:00Z] hi")
}
```

Output:
```text
notes: note 1 of 1
notes: note 2 of 2
[1970-01-01T00:00:00Z] done
```

Two ways of handing a value down are at work here, and each fits a kind
of dependency:

- **A parameter** for what the code is about: `handle` stores notes, so
  it takes the `App` with the store in it. The dependency is in the
  signature, the compiler checks it is passed, and nothing is hidden. A
  trait type (`Store`) is what lets a test pass a different one; the
  struct that holds it is `Sendable` because the trait requires it, so the
  `App` can go to every task that serves a request.
- **A task-local** (D72) for what the code only consults on the way: the
  clock, the request's id, the current trace span, the log fields. Passing
  those through every function on the path would add a parameter to each
  one for the sake of the few that read it. A `TaskLocal` has a fallback —
  the production value — and `with b = clock.bind(other)` replaces it for
  the rest of the block and every task started there. The binding ends
  with the block, so a test cannot leak its fake into the next one.

A rule of thumb: if a reader of the function should know it uses the
value, make it a parameter; if knowing would only be noise, a task-local.

## Fakes in tests

No mocking framework is needed. A test builds what the code under test
receives: an `App` with a `MemoryStore` instead of a database, a
`FixedClock` bound for the test's block. Because the real resources are
made only in `main`, a test never opens one by accident, and `veles test`
runs without a database, a network or a clock that moves.

The fakes are ordinary types implementing the same trait as the real
thing. Keep them next to the trait when other packages will want them
(a `MemoryStore` in the module that declares `Store`), or in a test file
when only your own tests use them. A fake that records what it was asked
— a list of the notes added, behind a `Mutex` — is how a test checks what
the code did, not only what it returned.

## What lives where, at a glance

- A value computed once and never changed: `const` if the compiler can
  compute it, otherwise a module `val`, or a `lazy val` if building it is
  expensive and the program may not need it.
- State that changes: a module `val` holding `Atomic`, `Mutex` or
  another synchronized type, or the state of an executor (chapter 12).
- Something opened: in `main`, with `with`, closed when `main` ends.
- Reaching the code that uses it: a parameter (often one struct of what
  the handlers need) for what the code is about; a task-local for what it
  consults on the way.
- A test: builds its own values and binds its own task-locals.

A long-running service follows the same shape, with the server's loop at
the end of `main`: chapter 17 shows how a server stops gracefully, so
the resources `main` opened are closed after the last request is served.
