# Patterns in Veles: singletons and the rest (menu, 2026-10-26)

**Status 2026-10-26:** P1 → **D210** (`lazy val`, built 2026-10-27), P3 → **D211** (resources
made in `main`), both decided by the user as recommended. P2 → Q59 and P4 →
Q60 are open (the user: "not to be decided today", "probably keyword but do
not decide now"). P5 and R1–R8 have not been asked yet. F1 is in checklist
§11 and plan B29.

A menu for the user. Apart from the status line above, nothing here is
decided. It lists the common design
patterns, says what Veles already spells for each (checked by probe programs
against the compiler on 2026-10-26), and proposes language or std changes
where a gap is real. The decisions go through `veles-decide`. Taken items
become `D` entries in the spec, and this file is then archived.

The test for every proposition is the project's own (veles-plan.md, Principles):
the high-level spelling has to be the shortest correct program, the low-level
escape has to stay visible, it has to cost nothing when unused, and it should
help write the compiler in Veles.

A rule of thumb runs through the whole file: **most "patterns" exist to work
around something a language lacks.** Veles already has named and default
arguments, sealed traits with exhaustive `when`, trait default bodies,
function types, `copy`, `with`/`Closeable`, task-locals and executors, so most
of the classic Gang-of-Four list needs no new feature (Part 2). The real gaps
are around **global state**: building it lazily, changing it without a lock,
replacing it in tests, and giving it a lifecycle (Part 1). Two more come from
programs that grow large: registries across modules, and delegation (Part 3).

---

## Part 0 — What a singleton is in Veles today

Probed today (`scratchpad/p1`, `p2`, `p3`):

```veles
use io

struct Config {
  port: i64
  name: string
  static val default = Config(port: 8080, name: "app")   // a type's shared value, built before main
}

val config = Lazy(init: () => load())                   // built once, on the first get(), by whichever task gets there first
val hits = Atomic(value: 0)                             // a mutable counter, lock-free
val registry = Mutex(value: MutableMap<string, i64>())  // mutable shared state behind a lock

executor gl {                                           // mutable state with no lock: one thread runs all its code (D150)
  var textures: MutableMap<i64, Texture> = [:]
  public fun texture(id: i64): Texture { … }
}

val requestId = TaskLocal(fallback: "-")                // a value that is replaceable per task tree (D72)
```

- A module is already a singleton namespace, as in Go: its `val`s are
  initialized once, before `main`, in dependency order (a cycle is an error,
  D113).
- `var` at module level is refused with a fix (D66): *"module-level 'var
  counter' is shared by every task … keep the state behind a lock: 'val
  counter = Atomic(value: ...)'"*. A module `var` of a synchronized type is refused too (D66 addendum, v0.114).
- The only mutable state without a lock is an executor's `var`, and that
  costs a dedicated OS thread (D150).
- A type controls who constructs it by its field visibility: a private field
  blocks the implicit constructor outside the module. So "exactly one
  instance" is a module `val` of a type with a private field. That works
  today.

### F1 — Defect found by this pass: a module `val` of a mutable collection races

```veles
val cache = MutableMap<i64, i64>()     // compiles
fun fill(from: i64) {
  loop (i in from..from + 100000) cache.set(i, i)
}
fun main() {
  scope {
    async fill(0)
    async fill(1000000)
    async fill(2000000)
    async fill(3000000)
  }
}
```

This compiles. Three runs with `VELES_THREADS=8` gave: a hang, a segfault,
and a hang. The module `var` check (`sema/check_fn.go:486`) covers `var` only.
A `val` of a reference type such as `MutableMap`, `MutableList` or
`StringBuilder` can still be changed through methods, because the
global-`val` mutation check in `sema/receivers.go:185` covers value receivers
only (`RecvPlace`), not handles. D66 and D150 both say the rule ("a `val` of a
`Sendable` type at module level"), so the fix is decision-free: a module
`val` (and a `static val`) has to have a `Sendable` type, and the error
suggests `Mutex(value: …)`, a `List` instead of a `MutableList`, or an
executor. Recorded in checklist §11.

---

## Part 1 — The singleton, by what it is used for

"Singleton" covers six different needs. Veles answers some of them well and
some badly:

| # | Need | Example | Today | Gap |
|---|---|---|---|---|
| 1 | an immutable shared value | default config, a keyword table, a CRC table | module `val`, `static val`, `const` (D113) | none |
| 2 | an expensive value, built on first use | a regex set, a Unicode table, a parsed embedded file | `Lazy(init:)` + `.get()` | ceremony on every read; easy to forget and pay at startup instead (**P1**) |
| 3 | mutable shared state | metrics, an id counter, a cache, an interner | `Atomic`, `Mutex`, `RwLock`, an executor | a lock per access, or a whole OS thread for lock-free state (**P2**) |
| 4 | a resource with a lifecycle | a DB pool, a logger sink, an HTTP client | made in `main`, passed down | cannot be a global: init cannot throw or suspend, and nothing closes it (**P3**) |
| 5 | a replaceable singleton | a clock, a random source, an HTTP transport faked in tests | `TaskLocal(fallback:)` | works, but the pattern is not known and reads are noisy (**P4**) |
| 6 | a stateless instance of a trait | `SystemClock` as a `Clock` | `struct SystemClock {}` + `SystemClock()` | none; an `object` keyword would add little (P5) |

### P1 — A lazy global without `.get()`

```veles
// today
val keywords = Lazy(init: () => buildKeywordTable())
fun isKeyword(s: string): bool => keywords.get().contains(s)
```

- **(a) Leave it.** `Lazy` is explicit, and the cost (an acquire load and a
  branch) is visible at the call. Rust spells it the same way
  (`LazyLock`, `*KEYWORDS`).
- **(b) A `lazy` modifier on module and static `val`s** *(recommended)*:

  ```veles
  lazy val keywords = buildKeywordTable()          // built on the first read, once, thread-safe
  fun isKeyword(s: string): bool => keywords.contains(s)

  struct Tz {
    static lazy val utc = Tz.load("UTC")
  }
  ```

  It lowers to `Lazy`'s once-cell with a fast path: one acquire load and a
  predictable branch, and after that a plain read. Inside a loop the check is
  hoisted, because the value cannot change once built. The rules are
  `Lazy`'s: the initializer cannot suspend and runs as `sendable`, a panic in
  it is raised again on every read, the type has to be `Sendable`, and
  compile-time cycle detection still applies. A `lazy val` cannot be read in
  a `const` (the error names the reason). Hover says "lazy: built on first
  read". `Lazy<T>` stays as the low-level form, for a value held in a field
  or passed around. This also answers D187's "a `lazy val` field … stays a
  possible later sugar", but **only for globals**: a `lazy` field would put a
  flag and a branch in every value of the type.
- **(c) Every module `val` with a non-constant initializer is lazy** (Swift's
  rule). No keyword is needed, but: initialization order and side effects
  become invisible (a `val` that logs or registers runs at an unknown time,
  or never); every read of every global pays a check unless the compiler can
  prove it was already built; and startup is no longer deterministic. Go is
  eager and Kotlin is eager per file; only Swift does this, and Swift
  developers trip on it ("why didn't my global run?").

Decides it: (b) makes the cheap thing short and keeps eager the default,
which is what a reader expects. Cost: parser (contextual `lazy` before `val`
at module and static level), sema, codegen (the once-cell and the
fast-path read), formatter, LSP hover, docs chapter on globals. Self-hosting:
the compiler's keyword, operator and builtin tables, and the LSP's std
cache, all want it.

### P2 — Mutable singleton state without a lock and without an OS thread

```veles
// today, option 1: a lock around every access
val counts = Mutex(value: MutableMap<string, i64>())
fun hit(route: string) => counts.withLock(m => m.set(route, (m.get(route) ?: 0) + 1))

// today, option 2: a dedicated OS thread (D150)
executor metrics {
  var counts: MutableMap<string, i64> = [:]
  public fun hit(route: string) { counts.set(route, (counts.get(route) ?: 0) + 1) }
}
```

- **(a) Leave it.** Use `Mutex` for small state and an executor when a thread
  is wanted anyway (GL, GUI, a C library).
- **(b) A serial executor** *(recommended)*: the state and the rules of
  D150's one-thread executor, but run on the shared pool, one task at a time.
  This is Swift's `actor`, and Java's or Tokio's "strand":

  ```veles
  executor metrics(serial: true) {
    var counts: MutableMap<string, i64> = [:]      // no lock: one task runs metrics' code at a time
    public fun hit(route: string) { counts.set(route, (counts.get(route) ?: 0) + 1) }
    public fun snapshot(): Map<string, i64> => counts.toMap()
  }

  await metrics.hit(req.path)       // from outside: a hop, as for gl
  ```

  When the executor is idle, a hop runs the call on the caller's worker
  thread with no context switch; when it is busy, the caller waits in its
  queue. That makes it cheaper than a `Mutex` under contention (a waiting
  task parks once instead of spinning) and cheaper than a thread always. The
  edge cases are D150's: a plain, non-suspending function cannot call it
  (the error names `metrics.runBlocking`); arguments and results have to be
  `Sendable`; state is reachable only through its functions. One new edge:
  work between suspension points is not reentrant, so code that suspends
  inside metrics' code lets another call run (Swift actor reentrancy). Two
  ways to handle it: refuse suspension in a serial executor's code, or
  document it and lint it. The first is simpler and matches the
  "refuse the footgun" rule. That is a sub-question.
- **(c) A synchronized `object`** that takes a lock on every call (Java's
  `synchronized`). Rejected in advance: it hides a lock per call, and two
  such objects calling each other can deadlock without any visible lock.

Decides it: (b) is the only option with no lock and no OS thread, and D150
already holds every rule it needs. Cost: the runtime (a serial queue on the
pool, a fast path when idle), sema (D150 rules with `serial`), docs.
Self-hosting: a compiler server can keep its caches and interner in one
serial executor.

### P3 — A singleton with a lifecycle (it can fail, it suspends, it must close)

```veles
// today: made in main, then passed down or bound to a task-local
fun main() throws DbError {
  with db = try Db.connect(os.env("DATABASE_URL") ?: "postgres://localhost")
  try http.serve(listener, router(db))
}
```

- **(a) Leave it** *(recommended)*. A resource is made in `main` and handed
  down, either as a parameter or as a task-local binding (P4). The program
  shows what it opens and when; a test makes its own.
- **(b) Throwing and suspending module initializers.** `val db = try
  Db.connect(url)` at module level runs in a startup task before `main`; an
  error ends the program with its message and location; `Closeable` globals
  are closed after `main`, in reverse order. To limit the damage, this could
  be allowed only in the program's own package, never in a dependency
  (importing a library must never open a socket; that also matches D138's
  capability model).
- **(c) A `service` declaration** with `start`/`stop` and dependencies on
  other services (a Spring- or Guice-like container in the language). Too
  much policy for a language.

Decides it: a global made by I/O cannot be replaced in a test, and `veles
test` would connect to the database before every run. Go's `init()`
experience is the evidence: the advice there is "don't do I/O in init".
Cost of (a): docs only, as a chapter on "application structure" showing the
`main` → handler pattern with P4.

### P4 — Replaceable singletons: dependency injection without a framework

D72's task-locals already are the mechanism. A default lives in the global,
and a test or a request rebinds it for its task tree:

```veles
// today
val clock = TaskLocal<Clock>(fallback: SystemClock())
fun stamp(): string => clock.get().now().toString()

test "stamps use the clock" {
  with b = clock.bind(FixedClock(at: Instant.epoch))
  expect(stamp() == "1970-01-01T00:00:00Z")
}
```

- **(a) Leave it and document it** as the dependency-injection pattern:
  docs, a `veles explain` page, and std using it for its own clock, random
  source and HTTP transport.
- **(b) A `context val` declaration** *(recommended)*, sugar over D72:

  ```veles
  context val clock: Clock = SystemClock()          // the default; reads are plain
  fun stamp(): string => clock.now().toString()

  test "stamps use the clock" {
    with clock = FixedClock(at: Instant.epoch)      // rebinds for the rest of the block and every task started in it
    expect(stamp() == "1970-01-01T00:00:00Z")
  }
  ```

  The rules are D72's: the binding is scoped and immutable, follows `async`,
  the value has to be `Sendable`, and rebinding is refused where nothing may
  suspend. `with name = value`, where `name` is a `context val`, is the
  rebinding. That reuses `with`'s "for the rest of the block" meaning; the
  alternative spelling is `clock.bind(…)`. Hover on `clock` says "context:
  rebound by callers". The cost of a read is D72's (a walk of a handful of
  bindings). A hot loop reads it once into a local, and a lint can suggest
  that.
- **(c) Kotlin-style context parameters** (`fun stamp() context(clock:
  Clock)`). The dependency is visible in every signature on the path. D72
  rejected this ("every function on the path carries it"), and nothing new
  argues for reopening it.

Decides it: (b) turns the pattern D72 already allows into the one-line
obvious spelling, and gives Veles testable globals with no mocking framework.
Cost: parser (contextual `context` before `val`), sema (read as `get`, `with`
as `bind`), formatter, LSP, docs; std migrates `requestId`, the OTel current
span, and the log fields. Self-hosting: the compiler's diagnostic sink and
options could be context values instead of a parameter threaded through
every pass.

### P5 — A stateless instance of a trait: no `object` keyword

```veles
struct SystemClock {}
implement Clock for SystemClock {
  fun now(): Instant => time.now()
}
val clock: Clock = SystemClock()
```

A zero-field struct is already a zero-sized value; Kotlin's `object` and
Scala's `object` mostly exist because those languages have no top-level
functions or values. Veles has modules for namespaces, executors for state,
and P1/P2/P4 for the rest. **Recommendation: no `object` keyword.** Listed so
the question is closed rather than left open.

---

## Part 2 — Patterns that need no new feature

Each line is the idiom to teach (in the cheat sheet and a docs chapter on
patterns). The idiom here **is** the answer.

| Pattern | The Veles idiom |
|---|---|
| Builder | named arguments with defaults (D28), `copy` (D164), `apply`/`edit` blocks (D207): `Server(port: 80, tls: cfg)`, `base.copy(port: 81)` |
| Factory / smart constructor | `init` with parameters (D73); a `static fun` returning `This` (D175) or `T?`/`Result` when it can fail: `Email.parse(text)` (the D205 `Parsable` convention) |
| Prototype | `copy` (D164) |
| Strategy | function types and trait objects: `sort(by: (a, b) => …)` |
| Command / undo | a sealed trait of commands, each variant with its data; `when` is exhaustive |
| State machine | a sealed trait of states; `when ((state, event))` returning the next state |
| Visitor | `when` over a sealed type (closed hierarchies); a trait with default bodies (D53) for open ones |
| Template method | trait default bodies (D53) |
| Iterator | `Iterator` and its lazy adapters (D46); generators would be idea I1 |
| Null object | `T?` with `?.`/`?:`; `Default` (D119) |
| RAII / dispose | `with x = e` and `Closeable` (D100); `defer` stays rejected |
| Observer (async) | `Broadcast`, `Watch`, channels (D146) |
| Actor | a declared executor (D150), or P2's serial executor |
| Adapter | extension functions (D173); `implement Trait for Foreign` within coherence (D17) |
| Decorator (of functions) | typed decorators (D176): `annotation Retry(…) on fun = retrying` |
| Newtype | a one-field struct, positional (D163), with derived traits (D58): `struct UserId { value: i64 }` + `implement Hashable, Comparable for UserId {}` |
| Type class / static interface | trait statics and `This` (D175) |
| Constants namespace | `enum`, `static val`, `const` |

---

## Part 3 — Patterns where a proposition helps

Ideas, not yet worked out to decision depth. Each says what it would let a
program do. The user picks which ones get a full `veles-decide` write-up.

### R1 — Registries across modules (plugins, commands, routes, lints)

Go registers database drivers in `init()`; Rust uses the `inventory` and
`linkme` crates; Java uses `ServiceLoader`. Every large program wants "every
X declared anywhere in the program, collected at one place" (CLI
subcommands, HTTP routes, lint rules, test cases, codecs). The self-host
review hit it (F6: the settings registry, whose public extension point is
still open).

```veles
annotation Command(name: string) on fun

@Command(name: "build")
public fun build(args: List<string>) throws CliError { … }

// anywhere: every function in the program carrying @Command, at compile time
const commands = program.functions(annotated: Command)   // List<(Command, fun(List<string>) throws CliError)>
```

It is compile-time (D174's reflection extended from "the members of a type"
to "the declarations of the program"), so nothing runs at startup, the list
is a constant table, and "which commands exist" is answerable by the LSP. The
open questions: a library's declarations count only when it is imported;
the order is by package, then source order; and the signature has to be
checked against the annotation's declared `on fun(…)` shape.

### R2 — Delegation: `implement Trait for Wrapper by field`

The decorator and proxy patterns, and composition over inheritance, are
mostly forwarding boilerplate:

```veles
struct CountingWriter {
  inner: Writer
  var bytes: i64 = 0
}
implement Writer for CountingWriter by inner {      // every method forwards to inner, except the ones written here
  fun write(data: List<u8>) throws IoError {
    bytes += data.len()
    try inner.write(data)
  }
}
```

This is Kotlin's `by`. Go's struct embedding is the alternative, but it
promotes every method of the field into the outer type, which is implicit and
leaks the inner API. Delegation is per trait and explicit. Cost: sema
(synthesize the forwarders as D58 does), hover and `veles explain --derive`
show them.

### R3 — Consuming methods (typestate, use-after-close)

```veles
public struct Tx {
  …
  public consuming fun commit() throws DbError { … }    // after tx.commit(), tx is dead
}
val tx = try db.begin()
try tx.commit()
tx.exec(sql"…")            // error: 'tx' was consumed by 'commit' at 3:5
```

A GC language usually cannot express "this handle is finished". A
`consuming` method, checked by the same flow analysis that D63 uses for moves,
gives typestate (`Request<Unsent>` → `Request<Sent>`), refuses
use-after-close at compile time, and refuses double commits. That is the
project's "refuse a footgun at compile time" rule applied to resources.
Large: it needs aliasing rules (a consumed value copied into a field
earlier). A candidate for its own design batch.

### R4 — Interning (flyweight): `std` `Interner` and `Symbol`

A compiler, a JSON decoder with repeated keys, and an HTTP header table all
want strings compared as integers. Proposed std API: `Interner()`,
`intern(s): Symbol`, `Symbol.text`, with `Symbol` 4 bytes, `Hashable` and
`Comparable` by id. Thread-safe by design (a sharded table), so it can be a
module `val`, or a `lazy val` (P1). Directly useful for self-hosting.

### R5 — Arenas (object pools at the low level)

```veles
with arena = Arena(capacity: 1 << 20)
val nodes = arena.list<Node>()        // allocated in the arena; freed when the arena closes
```

Bulk allocation and free for a parse tree, a request's temporaries, or a
frame of a game. The hard part is safety under a GC: a reference out of the
arena that outlives it. That reference has to be refused, either by an
escape check (as the receiver analysis does) or by making arena values
non-`Sendable` and scope-bound. Idea-level; it overlaps with the measured
performance work (F10) and should be decided only with a benchmark that
needs it.

### R6 — `@memo` and `@once` in std (D176 decorators)

```veles
@memo
fun fib(n: i64): i64 => if (n < 2) n else fib(n - 1) + fib(n - 2)
```

Library-only, now that D176 exists: the cache is a module-level `Mutex`
(or, after P2, a serial executor), keyed by the `Hashable` arguments.
Choices: the bound (`maxEntries:`), and whether the function has to be pure
(the compiler knows effects, so a memoized function that does I/O could be
refused).

### R7 — Synchronous signals for single-thread UI (`Signal<T>`)

The observer pattern inside one executor (a GUI thread, `gl`): listeners
called synchronously, with no channel and no `Sendable`. `signal.on(f):
Subscription` (`Closeable`, so `with` unsubscribes); `signal.emit(v)`. A
std API for the UI track (§3.5 of the idioms menu), not a language feature.

### R8 — Derived traversals for trees (`implement Walk for Expr {}`)

The visitor pattern for the self-hosted compiler: an empty `implement` (D58)
of a std `Walk`/`Fold` trait derives "visit every child of these sealed
variants". D203's `T.members` and inline loops may already be enough to write
this as a library. **To probe before deciding anything.**

---

## Suggested order

1. F1, the race (decision-free, a soundness fix).
2. P1 `lazy val`, P4 `context val`, P2 serial executors: one batch, because
   all three are "global state", and the docs chapter describes them
   together.
3. P3, P5: confirm "leave it", then write the application-structure docs.
4. R1 (registries) and R2 (delegation): the next batch.
5. R4 (interner) with the self-host work; R3, R5 when a program needs them.
