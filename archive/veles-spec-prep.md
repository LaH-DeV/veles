# Veles — state review and spec preparation (2026-09-30)

A working file. Its goal: every open design question is answered and written
into `veles-spec.md` / `veles-plan.md` precisely enough that an implementing
agent can build the rest of the plan without asking. Each batch below is
prepared here, asked, then recorded the usual way (`veles-decide`); a batch
that is recorded is struck here. When every batch is struck, this file moves
to `archive/`.

---

## 1. Where Veles stands

- **Spec** v0.58, decisions D1–D99; §9 of the checklist holds 13 open
  questions (Q1, Q2, Q5–Q7, Q9–Q14, Q17, Q20; notes I2/I3 are inside Q13).
- **Compiler** green on Windows and Linux (WSL): build, vet, every test, the
  examples under 1/2/4/8 threads, sanitizers, GC poison.
- **Plan**: track A done except A7 (macOS, the user's M4, later). Track B
  done except B8 (blocked on a language question, batch 5 below). C1
  (`std/log`) done; C2 (`std/http` server) done up to compression. S1–S3
  done. Track E (TLS, PostgreSQL, reactor, self-hosted front end, GC,
  release engineering, packages, compiler speed) not started.
- **What an agent can already do unsupervised**: all of track A7/E3/E5/E6/E8
  and the decision-free checklist items (performance, runtime, tooling).
- **What stops it**: every std API still to write (C2 compression onward, C3
  `std/config`, C4 observability, C5 `std/fs`, E1 TLS, E2 PostgreSQL, the
  HTTP client) and every §9 question is a decision. Batches 1–8 below turn
  them into spec entries.

## 2. Findings of this review

### 2.1 Stale or contradictory spec text — fixed in the spec 2026-09-30

1. The header still says "v0.58 — language design complete; D58 adds the
   derivation story…, D59 the standard library's cryptography. Remaining
   work is … C ABI FFI" — FFI is built (D67/D69), design is not complete.
2. §6.2 points at `veles-manifest.md`, which does not exist.
3. §7 "Not yet designed" lists what is designed and built: closures, the
   loop syntax, `Mutex`/`Atomic`, the task API, C FFI, formatter and LSP.
   Really open there: `Mutex.withLock` as a `with` resource (batch 3) and how
   collection literals are wired (D41).
4. §4b "Variadic C functions — excluded" contradicts Q7 ("varargs calls
   (`printf`)" open). Re-asking Q7 must say what is new since.
5. §4b "`testing` — stdlib, with a test build mode" is superseded by D78.
6. §5 describes generated module interface files as "load-bearing"; they
   are not built (a package is checked whole). Say so and point at E8.
7. D43 says "This raises the stakes on §6.1. The panic mechanism is still
   undecided" — decided (D49 and its v0.29 addendum). D43 also says "no
   `defer`"; the lexer reserves `defer` — batch 1 settles it.

### 2.2 Bug found while probing

- A program whose only diagnostic is the needless-`throws` warning (D45)
  prints `see: veles explain type-mismatch` — the warning is sorted into the
  wrong family (`source/family.go`). Pin with a case in
  `TestEveryDiagnosticHasAFamily` or a driver test.

### 2.3 Decision-free clean-up

- `expect(when (early) { is Ok => false  is Err => true })` in
  `std/http/limits.test.vs` is `expect(early is Err)` today (checked: `x is
  Err` is a `bool`). Look for the pattern elsewhere.

### 2.4 Where the nesting comes from (measured)

Ordinary code is flat: outside tests, nine files have any line 6+ levels
deep, a few lines each (the most: `Duration.toString`, 18), and those are
`if`s and `when` arms inside loops inside methods. The towers are
in code that holds **resources and tasks at once** — tests and servers:

| Source | Count in std + examples |
|---|---|
| `with (x = …) { }` — one level per resource opened after other work | 58 blocks in 19 files |
| `scope { }` — one level to be allowed to write `async` | 29 |
| a background task (`serve`) that must be `cancel()`ed by hand before the scope can end | every server test |

Structural base, not re-proposed: a method is written inside `implement X {
}` inside `struct S { }` (D23, the user's choice), so its body starts three
levels in.

---

## 3. The decision queue

In this order; each batch is one message of questions.

| Batch | Questions | Why now |
|---|---|---|
| ~~**1. Less nesting**~~ | ~~N1 resources, N2 background tasks, N3 migration, N4 spelling~~ | **decided 2026-09-30: D100, plan B10** |
| ~~2. Small syntax~~ | ~~Q13, Q17, Q20, Q12, empty literals, §6.1~~ | **decided 2026-09-30: D101–D106, plan B11** |
| ~~3. Tests and concurrency~~ | ~~Q1, P11 helpers, `Mutex` guard, `with expr`, Q2~~ | **decided 2026-10-01: D107–D111, plan B12/B13** (Q2 by the user's Go-style idea, D111) |
| ~~4. Safety and build~~ | ~~Q9, Q11, Q10, the unused-`Closeable` warning~~ | **decided 2026-10-01: D112–D115, plan B13/B14** |
| ~~5. Types and effects~~ | ~~suspension-generic functions, Q5, Q6, `Default`~~ | **decided 2026-10-01: D116–D119, plan B15** |
| ~~6. FFI~~ | ~~Q7~~ | **decided 2026-10-01: D120–D123, plan B16; the by-value struct ABI bug found → plan A8** |
| ~~7. std APIs ahead of the plan~~ | ~~compress, config, observability, fs, tls, db, client~~ | **decided 2026-10-01: D124–D130, plan C2–C8, E1, E2; health endpoints to confirm** |
| ~~8. Packages~~ | ~~Q14, M7~~ | **decided 2026-10-01: D131, D132, plan E7** |

Every recorded answer must give the implementer: the rule, the examples, the
edge cases, what it touches (lexer … LSP, std, docs), the tests that pin it,
and the migration.

---

## 4. Batch 1 — less nesting, without less safety (decided: D100)

*Kept as the brief that was asked. Answers: N1 C, N2 B, N4 A (all
recommended); N3: "just migrate all of our codebase, not needed support for
later" — the tree is migrated, no warning is kept.*

### The problem

User, 2026-09-30: "are we able to make Veles safe, performant, intuitive, but
less indented by default? … I do not want to have to create 'towers of
terrors' when programming safely and 'properly'."

The test that prompted it (`std/http/limits.test.vs`) is five levels deep for
four resources and one background task:

```vs
test "at the connection limit a new connection waits until one closes" {
  with (listener = try net.listen()) {
    scope {
      val server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
      val port = listener.port()
      with (first = try net.connect("127.0.0.1", port)) {
        try first.writeText(limitedGet(false))
        expect((try limitedRead(first)).startsWith("HTTP/1.1 200"))
        with (second = try net.connect("127.0.0.1", port)) {
          try second.writeText(limitedGet(true))
          val early = withTimeout(Duration.millis(300), () => try limitedRead(second))
          expect(when (early) {
            is Ok  => false
            is Err => true
          })
          try first.shutdownWrite()
          expect((try limitedRead(second)).startsWith("HTTP/1.1 200"))
        }
      }
      server.cancel()
    }
  }
}
```

Two different causes: (1) a `with` block is the only way to close a resource
safely, so each resource opened *after other statements* costs a level
(`with (a = …, b = …)` only helps when both are opened together — it works
today, a later binding may use an earlier one); (2) `async` is only allowed
inside `scope`, and a task that never ends by itself (a server) must be
cancelled by hand — **forgetting `server.cancel()` hangs the test forever**,
a footgun the compiler does not refuse today.

### N1 — Closing a resource without a new level

**A. Leave it.** Blocks only.

**B. `defer`** (Go, Zig, Swift; the word is already reserved):
```vs
val listener = try net.listen()
defer listener.close()
```
Runs any statement at block exit. Pros: general (restore a flag, log on
exit). Cons: two statements where one does; forgetting the `defer` compiles
— exactly the leak `with` exists to prevent; a failing `close()` in a
`defer` has no good answer (Go drops it, Zig needs `errdefer`); any code can
run invisibly at `}`; a second way to do D43's job.

**C. A `with` statement: the rest of the block is its body** (C# 8 `using
var`, Rust's drop at the end of a scope, Kotlin has none):
```vs
with listener = try net.listen()
val port = listener.port()
with first = try net.connect("127.0.0.1", port)
```
It *is* D43: `with x = e` followed by statements S… in a block means exactly
`with (x = e) { S… }`, so every D43 rule holds unchanged — closes on every
exit (end, `return`, `break`/`continue`, failed `try`, `throw`, panic,
cancellation), in reverse order, the body's error wins over a close error,
cleanup shielded from cancellation (D47). Pros: one line per resource, the
same keyword and meaning, impossible to open without arranging the close,
zero cost (it is a parser/checker desugaring; IR identical). Cons: the close
is at the `}` of the enclosing block, not written there — mitigated by
tooling: hover on `with` says where it closes, and an inlay hint at that `}`
lists what closes there (`closes second, first, listener`).

Edge cases (for C):
- Allowed in any braced block of statements: function, test, lambda, loop
  body (closes each iteration), `if`/`when` arm, `do`, `scope`. Refused at
  module level, in `fun f() = expr`, and as a braceless `if`/`loop` body
  (nothing follows it) — each with an error naming the block form.
- Mixed with block-form `with`s and `scope`s in one block: every cleanup is
  run last-registered-first.
- A block whose last expression is its value: the value is computed, then
  the resources close (D43 v0.29 rule).
- Returning the resource, storing it in a field, or capturing it in an
  escaping lambda or `async`: an error "closed when the block ends" (checked
  for both forms; add it to the block form if missing).
- Generics, nullables: the value must be `Closeable` (a `T?` is refused, as
  in the block form — unwrap first). Threads: nothing new.
- Formatter: kept as written, never converted. Hover/inlay as above.
- Low level: `close()` can still be called by hand on a plain `val`; the
  block form stays for closing *before* the block ends.
- Self-hosting: the compiler opens few resources; neutral, it removes one
  reason for a helper function.

**Recommendation: C.** Deciding reason: it removes the level without adding
a way to forget the close — the flat form is still the safe form.

### N2 — A task that runs in the background of a block

**A. Leave it**: `scope { … server.cancel() }` — a level, plus a manual
cancel whose omission hangs.

**B. `with t = async f(…)`**: a task bound by `with` is a *background child*
of the rest of the block. It is fail-fast like a `scope` child (if it throws
or panics, the rest of the block is cancelled at its next suspension point
and the error propagates from the block); at the end of the block it is
**cancelled, then joined** — closing a task means stopping it — unless it
already finished; `await t` waits for it and takes its value. `async` is then
allowed in two places: inside `scope`/`gather`, and as the value of a `with`.
```vs
with listener = try net.listen()
with server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
```
Edge cases: its error types join the enclosing function's inferred `throws`
as a scope child's do; the `Cancelled` from its own close is not an error; a
failure while being cancelled follows D43's suppression rule; captures follow
the `async` Sendable rules (D35); the join at the end is shielded cleanup
(D47); inside a `scope` body it is cancelled at the end of its own block,
before the scope joins its other children; inside `gather` it is not one of
the gathered results. Also works in block form: `with (t = async f()) { }`.
Swift precedent: an `async let` that is never awaited is cancelled and
awaited implicitly when its scope ends.

**C. `scope` as a statement**: a bare `scope` line makes the rest of the
block a scope. Removes the level but keeps the manual cancel and its hang.

**D. Implicit scopes** (Swift `async let` everywhere): `async` allowed in any
block, every block silently a scope. Least syntax; but any `}` or `return`
may wait for tasks invisibly, and D34's visible boundary is gone.

**Recommendation: B.** Deciding reason: it turns today's forgettable
`server.cancel()` into the default, so the hang becomes unwritable — the
footgun is refused rather than documented. `scope { }` stays for tasks that
should be *waited for*.

### N3 — What happens to code written the old way

**A. A warning with a fix**: a statement-position `with (…) { }` or a
`scope` whose only job was a background task, when it is the last statement
of its block, is flattened by `veles check --fix` / the LSP quick fix; std,
examples, docs migrated in the same change. One spelling per situation, as
elsewhere in Veles (the `name: name` warning).
**B. Quick fix only**, no warning; the tree migrated by hand where it reads
better. **C. No migration.**

**Recommendation: A** — the project's habit is one spelling per situation,
and a block form that ends its block adds a level for nothing.

### N4 — Spelling of the statement

**A. `with f = try fs.open(p)`** — the block form without its parentheses and
braces. **B. `with val f = try fs.open(p)`** — echoes `if (val x = …)` (D95).
**C. `val f = with try fs.open(p)`** — reads as a value modifier.
**Recommendation: A** — `with` already means "bind and close", and the two
forms differ only in where the body is.

### The motivating test, with every recommendation

```vs
test "at the connection limit a new connection waits until one closes" {
  with listener = try net.listen()
  with server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
  val port = listener.port()
  with first = try net.connect("127.0.0.1", port)
  try first.writeText(limitedGet(false))
  expect((try limitedRead(first)).startsWith("HTTP/1.1 200"))
  // the first connection is kept alive and holds the only place
  with second = try net.connect("127.0.0.1", port)
  try second.writeText(limitedGet(true))
  expect(withTimeout(Duration.millis(300), () => try limitedRead(second)) is Err)
  // the first one ends: the server closes it and accepts the second
  try first.shutdownWrite()
  expect((try limitedRead(second)).startsWith("HTTP/1.1 200"))
}   // closes second, first; cancels and joins server; closes listener
```

One level instead of five; same close order as today; the server cannot be
left running.

### Cost

Parser (statement form in block statement lists; `with` + `async`), checker
(desugar to the block form; `async` placement rule; escape check; N2's
cancel-at-end flag on the scope lowering — the scope runtime already cancels
children when a body leaves early), no codegen or runtime change expected
for N1, a small one for N2 (cancel-then-join at normal end). Formatter, LSP
(hover, inlay hint at `}`, quick fix, completion), TextMate grammar
unchanged (same keyword). Conformance cases, an example
(`examples/with` extended), docs chapters 7 and 12, cheat sheet, errors
reference. N3 A: the lint and the migration of ~58 `with` and the server
`scope`s.

---

## 5. Batch 2 — small syntax and std consistencies (decided: D101–D106)

*Kept as asked. Every recommended option was chosen, including both small
ones in B2.8.*

Checked against the compiler on 2026-09-30 (probe programs in the
scratchpad). Found on the way, decision-free, go to the plan whatever the
answers: (1) `when (m = parse(s))` gives two parser errors and no hint —
one error naming `val` is enough; (2) "`'async' launches a direct call of a
named function or method`" names no fix; (3) the empty-list error suggests
`val xs: List<i32> = []` even for a `var` that is pushed to — it should
suggest `MutableList<i64>`; (4) `val small: i8 = id(12)` fails (a type
argument is not inferred from the expected type, §11) — making it compile
only accepts more programs, so it is a plan item, not a question.

### B2.1 (Q13 / notes I2) — which heads write `val`

Today: `if (val x = e && …)` (D95) and `when (val n = e)` need `val`;
`with (f = e)`, `with f = e` (D100), `loop (x in c)`, `catch (e)` do not.

- **A. Keep it, and state the rule** (recommended): *a head that holds an
  expression writes `val` to bind in it; a head that always binds does not.*
  `if` and `when` take any expression, so `val` marks the binding; `with`,
  `loop`, `catch` can only bind. The parser's error for `when (m = …)` names
  `val` with a fix. Deciding reason: `if (x = e)` is C's `=`/`==` bug when
  `x` already exists.
- **B. Drop `val` from `if` / `when`**: `when (n = parse(s))`,
  `if (c = req.cookie("id") && …)`. Shorter; reads as assignment; clashes
  with an existing `var n`.
- **C. `val` everywhere a name is introduced**: `loop (val x in xs)`,
  `with (val f = …)`, `catch (val e)`. Uniform; noisy, and contradicts D100's
  spelling chosen yesterday.

### B2.2 (Q13 / notes I3) — changing a list while looping over it

Today (probed): a value loop reads by index against the live length —
`loop (x in xs) { if (x < 3) xs.push(x + 10) }` visits the appended items
(and `xs.push(x)` in every iteration never ends); removing an earlier item
skips one (`[1,2,3,4,5]`, remove at 0 when x == 2 → visits 1, 2, 4, 5).
Memory-safe, meaning unspecified. By-reference loops already warn. Maps: an
insert during the loop is kept; whether it is visited is unspecified.

- **A. Refuse it** (recommended; Rust refuses at compile time, Java/C# throw
  at run time): a structural change (`push`, `insert`, `removeAt`, `clear`,
  `remove`, `set` on a map key not present, …) to the collection a loop
  walks, written in the body on the same name, is a **compile error** with
  fixes "loop over `xs.toList()`" (a copy) or "collect the changes and apply
  them after the loop"; replacing an element (`xs.set(i, v)`, `*x = v`) stays
  allowed. A change made through another path (a function that reaches the
  same list) is caught at run time: the list/map/set header gets a
  modification count, the loop compares it each step and panics "the list
  changed while a loop walked it" at the loop's line. Cost: one 8-byte field
  per collection, an increment per structural change, a compare per
  iteration (measured with veles-bench before it lands; LLVM hoists it out of
  loops that make no calls). Covers Map, Set, Deque.
- **B. Refuse at compile time only**; indirect changes keep today's live
  behaviour, documented.
- **C. Snapshot**: the loop walks the collection as it was when it started
  (Swift's value semantics). Intuitive; costs a copy per loop over a
  `MutableList` unless copy-on-write is added to the runtime.
- **D. Document today's live-index behaviour.**

Self-hosting: a compiler's worklists do "push while walking" on purpose —
with A they write a condition loop (`loop (i < work.len()) { … i += 1 }`)
or pop from a `Deque`, which states the intent. Low level: the condition
loop is always available.

### B2.3 (Q17a) — `gather` with one task

`gather { async work(9) }` is a 1-tuple, read with `.0`; the 1-tuple exists
nowhere else in the language. A one-task `gather` is the idiom for "run this
and see whether it panicked" (D52).

- **A. One task yields its `Result` itself** (recommended): `when (gather
  { async f() }) { is Ok … is Err … }`. Tuples start at two elements, as
  everywhere else.
- **B. Leave it** (`.0`).
- **C. A prelude function** `attempt(f): Result<T, Panic | E>` wrapping the
  one-task gather; `gather` unchanged.

### B2.4 (Q17b) — `async f(x)` on a function value

Refused today; `withTimeout` and the worker pools go through a trampoline
`fun call(f) = f()`. Function types carry their effects (D40) and
sendability (D35 v0.28), so nothing unsafe is left to refuse.

- **A. Allow it when the value's type is `sendable fun`** (recommended); a
  plain `fun` value is refused with the fix "declare it `sendable fun(…)`".
  A router or pool storing handlers launches them directly.
- **B. Allow any function value**: unsound — a non-sendable closure's
  captures would cross tasks.
- **C. Leave it**, with the error naming the trampoline.

### B2.5 (Q20) — integer and float bit operations

Existing: `countOnes`, `leadingZeros`, `trailingZeros` (Rust's names,
camel-cased), `toBits`/`fromBits` (D93).

- **A. Rust's names, camel-cased** (recommended): integers `rotateLeft(n)`,
  `rotateRight(n)`, `swapBytes()`, `reverseBits()` on all ten integer
  types, `n: i64` taken modulo the width (a negative rotates the other way,
  as Go); floats `copySign(y)`, `isSignNegative()` (next to `isNaN`,
  `isFinite`), `nextUp()`, `nextDown()`. Consistent with what exists.
- **B. Go's**: `bits.RotateLeft`-style free functions in a `bits` module
  (`bits.reverseBytes(x)`). Groups them, but breaks with the existing
  methods.
- **C. Java's**: `byteSwap`/`reverse`/`signum`-flavoured names.

Out of scope, noted: reading/writing integers as big/little-endian bytes
(`u32.fromBytesBE`), which a binary-format reader wants — its own API
question in batch 7.

### B2.6 (Q12) — capacity hints

D83 gave `MutableList.reserve(n)`. `StringBuilder` wraps a byte list but has
no `reserve`; `MutableMap`/`MutableSet`/`Deque` have none either.

- **A. `reserve(n)` on every growable container** (recommended):
  `StringBuilder` (bytes), `MutableMap`, `MutableSet`, `Deque` — one name,
  D83's meaning (room for at least n in total).
- **B. `StringBuilder.reserve` only.**
- **C. Leave it.**

### B2.7 — `var xs = []` with no annotation

Today: an error asking for an annotation. Kotlin and Swift also require it;
Rust and TypeScript infer from later use.

- **A. Keep the annotation, and let the tool write it** (recommended): the
  error's fix is computed from the first later use that fixes the type
  (`xs.push(1)` → `var xs: MutableList<i64> = []`), applied by `check --fix`
  and the editor. The type stays visible where the variable is declared.
- **B. Infer from later use** (Rust): no annotation; hover shows the type.
  Needs inference variables that live across statements in the checker — a
  larger change — and an error at the second use when two uses disagree.
- **C. Leave it** (the fix text only).

### B2.8 — two small ones

- **`Range.isEmpty()`** (Q17c): the prelude has it privately as
  `holdsNothing`; every collection has `isEmpty`. Recommended: add it.
- **`cond => x => x * 2` in a `when` arm** (spec §6.1, D33): recommended —
  the formatter prints the lambda in parentheses (`cond => (x => x * 2)`);
  no warning, nothing to fix by hand.

---

## 6. Batch 3 — tests and concurrency (D107–D110 decided; Q2 open)

*Answers 2026-10-01: B3.3 guard + `withLock` (recommended); B3.4 send arms
(recommended, after an explanation of what a send arm is); B3.5 all four
helpers; B3.6 `with expr` (recommended). B3.1/B3.2 (Q2): asked twice, the
second time with the suite-level alternative —*

```veles
suite "a server limited to one connection" {
  with listener = try net.listen()      // run fresh around each test below
  with server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
  test "the second connection waits" { … }
}
```

*— and the user is not sure yet ("Are lending functions like function
generators in js"). For the next ask: a lending function resembles a JS
generator that yields exactly once and whose `.return()` the language calls
for you when the caller's block ends — closer to Python's
`@contextmanager`. In plain words: "a helper that sets something up, hands
it to you for the rest of your block, and tidies up after you." Lead with
the suite form (no new syntax), then lenders only for parameterized setup
and locks.*

Checked 2026-10-01. Today: `Mutex.withLock<R>(f: fun(*T): R)` (43 uses,
nearly all one expression: `notes.withLock(n => n.all())`); `TaskLocal.withValue(v,
f)` and `log.withFields(fields, f)` take a lambda; `Channel` has `send`,
`recv`, `trySend`, `tryRecv`, `close`, `closeAfter`, `len`; `race` arms are
receives, `await t`, and `sleep(d)`; there is no public test helper
(`testing.tempDir()` from the D78 design was never built).

### B3.1 (Q2) — setup and teardown shared between tests

User, 2026-09-27: "the setup for before and after isn't good... (no new
keyword for them either)" — so no `beforeEach`, no suite-level blocks re-run
per test.

With D100 a *plain* resource fixture is already one line: a helper returns a
`Closeable` struct, the test writes `with db = try testDb()`. What still cannot
be a helper is a fixture that **starts a task** — the server every HTTP test
starts — because a task cannot outlive the function that started it
(structured concurrency, D3). So `std/http/limits.test.vs` repeats the same
two lines in every test.

- **A. A lending function** (recommended): a function declared `with fun`
  whose last statement is `yield v` lends `v` to the caller's `with`; its own
  `with`s — resources and background tasks — stay open until the *caller's*
  block ends, then close in reverse order. Python's `@contextmanager`, Ruby's
  `yield` to a block, without a lambda:
  ```veles
  with fun serving(limits: Limits): i64 {
    with listener = try net.listen()
    with server = async serve(listener, limitedOk(), limits: limits, log: false)
    yield listener.port()
  }

  test "at the connection limit a new connection waits until one closes" {
    with port = try serving(Limits(connections: 1))
    with first = try net.connect("127.0.0.1", port)
    ...
  }
  ```
  Rules: exactly one `yield`, as the body's last statement (no `yield` in a
  branch, no `return`); a `throw` or failed `try` before it is the call's
  error (with `try` at the call; the lender's opened `with`s close, the
  caller's block never runs); cleanup is only the lender's `with`s — any other
  "after" code is a `Closeable`, so there is no "what if the caller failed"
  question; the lent value obeys D100 part 3 (cannot leave the caller's
  block); a lender is called only as a `with` value (else an error with the
  fix), cannot be a function value, `async`ed, recursive through `with`, or a
  trait method (for now); methods allowed (`with fun lock()`); generic
  allowed. Lowering: the body is checked once, then expanded at each `with`
  site in HIR (its `with`s join the caller's cleanup stack) — no coroutine,
  no allocation, the same IR as writing it out. Hover: "lends `i64` for a
  block". It also turns the lambda APIs flat: `with tl.bound(v)`, `with
  log.fields(…)`, `with n = m.lock()` (B3.3) would be lenders.
  Self-hosting: scoped compiler state (`with scope = checker.enter()`) reads
  naturally; neutral otherwise.
- **B. The pattern only**: document helpers returning `Closeable` structs;
  a task-starting fixture stays two `with` lines per test.
- **C. Leave it open.**

### B3.2 — the lender's spelling (if A)

- **A. `with fun name(…): T { … yield v }`** (recommended): the declaration
  says how it is called; `yield` is already reserved and is the word Python
  and Ruby use for exactly this. A later generator feature could share it (in
  Python a context manager *is* a generator).
- **B. `with fun … { … lend v }`**: a word no other feature will want; less
  familiar.
- **C. No marker on the declaration**, `yield` alone makes it a lender:
  shorter; the kind of function is not visible in its first line.

### B3.3 — `Mutex` as a `with` guard (spec §7, open since D43)

- **A. Add `with p = m.lock()` and keep `withLock` for one expression**
  (recommended): `lock()` lends `*T` (a lender, or compiler-known if B3.1 is
  not A); the rest of the block — or the block of the block form — holds the
  lock, and a suspension inside it is a compile error ("the lock is held until
  the end of the block; use `with (p = m.lock()) { }` to release it
  earlier"), the D35 rule `withLock` gets from its lambda type today; the
  pointer cannot escape (D100 part 3); locking the same mutex again panics as
  now. `withLock` stays for `val all = notes.withLock(n => n.all())`.
- **B. The guard replaces `withLock`**, migrated (one spelling; the
  one-expression uses get longer: `val all = with (n = notes.lock()) { n.all() }`).
- **C. `withLock` only.**

### B3.4 (Q1) — send arms in `race`

Asked 2026-09-27, answered "idk yet". Nothing new since, except that D100
makes the workaround shorter to read.
```veles
race {
  queue.send(line)          => { }
  sleep(Duration.millis(50)) => dropped += 1
}
```
- **A. Add them**: a send arm is ready when the channel has room; a losing
  send arm never sent (Go's `select`). No task per send.
- **B. Leave it**: `withTimeout(d, () => queue.send(line))`, or `trySend` in
  a loop.
- **C. Still undecided** (it stays in §9).

Recommendation: **A** — a bounded send with a deadline is the backpressure
primitive (a log shipper, a metrics queue), and today's workaround starts a
task per send.

### B3.5 (notes P11) — which concurrency helpers to fix now

So an agent can add them when a program needs one, without asking:

| Helper | Signature (recommended) |
|---|---|
| `retry` | `retry<R, E>(times: i64, f: fun(): R suspends throws E, delay: Duration = Duration.zero): R throws E` — the last error when every try fails; `delay` between tries |
| `Semaphore` | `Semaphore(permits: n)`; `with sem.acquire()` (a `Permit` that is `Closeable`); `trySend`-like `tryAcquire(): Permit?` |
| channel drains | `ch.forEach(f)` until closed; `ch.toList()` |
| `ticker` | `ticker(every: Duration): Ticker` — `Closeable`, `recv()` gives the tick's `Timestamp`, usable as a `race` arm, no task behind it |
| later, when a program needs them | `filterConcurrent`, `firstConcurrent`, channel-to-channel stages, `awaitAll` |

### B3.6 — `with e` without a name

`with sem.acquire()`, `with tl.bound(v)`, `with log.fields(…)` bind nothing
the block uses.
- **A. Allow `with expr`** (recommended; Python's `with lock:`): the value is
  only closed (or lent and ignored). `with _ = expr` stays legal.
- **B. Require `with _ = expr`.**

### Q2 again — the user's idea, 2026-10-01

User: "I'm not convinced for lend functions, maybe returned reference to
something should work like in golang? and that would work with `with` no?"

Yes, and it fits better than a lender. For a *resource* it works today: a
helper returns a `Closeable` struct and the caller writes `with db = try
testDb()` (Go: `db := testDb(t); defer db.Close()`). What Go does that Veles
cannot yet is `httptest.NewServer`: return a value with a **running task**
inside, because Go's goroutines are unstructured. One rule makes it work
without giving up structure — *a value that holds a `Task` must be bound by a
`with`, and its tasks become background children of that `with`'s block*
(D100 part 2, moved one call up):

```veles
struct TestServer {
  listener: net.Listener
  server: Task<()>
  fun port(): i64 = this.listener.port()
  implement Closeable { fun close() { this.listener.close() } }
}

fun serving(limits: Limits): TestServer throws IoError {
  val listener = try net.listen()
  TestServer(listener, server: async serve(listener, limitedOk(), limits: limits, log: false))
}

test "at the connection limit a new connection waits until one closes" {
  with srv = try serving(Limits(connections: 1))
  with first = try net.connect("127.0.0.1", srv.port())
  ...
}   // srv's task is cancelled and joined, then srv.close() closes the listener
```

Rules: a type with a `Task` field (directly, or through a field of such a
type) is *task-holding*; a task-holding value must be bound by `with` where
it is received (else an error with the fix), cannot be stored or returned
past that block (D100 part 3), and its tasks are fail-fast children of the
block, cancelled and joined at its end **before** the value's own `close()`;
`async` outside `scope`/`gather` is allowed where its `Task` goes straight
into such a value that the function returns; if the function fails between
the `async` and the return, the task is cancelled and joined first. Hover
shows "holds a task: bind it with `with`". std can then offer Go's
`httptest` shape: `with srv = try http.testServer(handler)`.

---

## 7. Batch 4 — safety and build (decided: D112–D115)

*Answers 2026-10-01: Q2 → Go-style values (D111, recommended of 4); Q9 → also
zero when collected (D112); Q10 → expressions, tables **and** compile-time
functions, marked `const fun` (D113); Q11 → `atUnchecked` in `unsafe` only
(D114); the never-closed warning (D115).*

Checked 2026-10-01: `const` takes a literal or an operator expression over
literals (`1 << 20`, `-128`); a reference to another `const` (`KB * 1024`),
`+` on strings, a list or `NAME.len()` is "'const' requires a compile-time
constant". `--release` is `-O2`, wrapping overflow (D21), bounds checks on;
there is no unchecked element access. `Secret<T>` does not exist; keys and
HMAC pads are plain GC memory (§11).

### B4.1 (Q9) — `Secret<T>`

The leaks that happen in practice: a password printed by a log line, a token
encoded into a JSON debug dump, a key compared with a short-circuiting `==`.

- **A. Redaction, and zeroing on `close()`** (recommended): `Secret.of(v)`;
  `s.expose(): T` is the only way to the value (grep-able); `toString()` and
  interpolation print `[redacted]`; **not `Encodable`** — deriving `Codable`
  on a struct with a `Secret` field is an error naming `@skip` or a
  hand-written encode — but **`Decodable`** (a config file or the
  environment fills it); `==` constant-time for `Secret<string>` /
  `Secret<List<u8>>`; `Closeable`: `close()` zeroes the bytes the `Secret`
  owns (so `with key = Secret.of(…)` wipes it at the end). Honest limit:
  copies made by `expose()` are ordinary memory.
- **B. A + zeroing when collected**: the collector wipes a secret's bytes
  when it frees them (a header flag, a `memset` at sweep). Covers the
  forgotten `close()`; costs a GC change and still cannot reach copies.
- **C. Leave it.**

Deciding reason for A: the real leak is printing and serializing, which A
makes a compile error or `[redacted]`; GC zeroing promises more than it can
keep. Low level: `expose()` and `withRaw` on the bytes. Comes before
`std/config` (C3), whose `DATABASE_URL` is the first secret most programs
read.

### B4.2 (Q10) — what `const` may hold, and `static assert`

- **A. Constant expressions over consts** : arithmetic, bitwise,
  comparison and logic on numbers and `bool`, string `+`, `len()` of a const
  string, `toT()`/`wrapT()` (an overflow is a compile error), `if`
  expressions, and other consts (no cycles); no function calls.
- **B. A + constant tables** (recommended): `const ROUND: List<u32> = [ … ]`
  and maps of constants, stored read-only in the binary — no start-up cost,
  usable in `when` patterns where elements are. The SHA round constants,
  keyword tables and the UAX #31 table of the self-hosted lexer are exactly
  this.
- **C. Compile-time functions** (Zig `comptime`, Rust `const fn`): a marked
  function runs in the compiler. Most powerful; an interpreter inside the
  compiler.

And **`static assert(cond, "why")`** at module level or in a body, `cond` a
constant expression, checked at compile time (`static assert(HEADER_SIZE ==
16, "the wire header is 16 bytes")`) — recommended with A or B; the reason is
required, as in D78's `assert`.

### B4.3 (Q11) — turning bounds checks off

D21's overflow policy is not re-asked (no new evidence). The question is
only bounds checks.

- **A. Never globally; a local unchecked read in `unsafe`** (recommended):
  `unsafe { xs.atUnchecked(i) }` / `xs.setUnchecked(i, v)` (with the
  `// SAFETY:` lint), Rust's `get_unchecked`; plus the checks the compiler
  already removes when it can prove the index (D62 B). Every unchecked access
  is visible and auditable at its line.
- **B. A + a loud profile** `--release-unchecked`: every check off in every
  module including dependencies, a warning on every build, marked in
  `--version` of the binary (Swift `-Ounchecked`, Zig ReleaseFast).
- **C. Leave it**: checks always, no unchecked access (raw pointers via
  `withRaw` remain the escape).

Deciding reason for A: a global switch turns every index in every dependency
into undefined behaviour at once; a hot loop needs one unchecked read, not a
program without checks.

### B4.4 — a `Closeable` never closed

Today nothing notices `val f = try fs.open(p)` with no `with` and no
`close()`. With D100 the fix is one word.

- **A. A warning with a fix** (recommended): a local holding a `Closeable`
  that is never `with`-bound, closed, returned, stored, passed as an argument
  or captured — and a call whose `Closeable` result is discarded — warns
  "'f' is never closed", the fix turning `val f =` into `with f =`. Passing
  counts as handing it off, so `serve(listener)` is not flagged. Measured
  over std and examples before it lands.
- **B. An error**: the same analysis; refuses legitimate hand-offs it cannot
  see.
- **C. Nothing.**

---

## 8. Batch 5 — types and effects (decided: D116–D119)

*Answers 2026-10-01: B5.1, B5.3, B5.4 recommended; B5.2 "Both" (run time on
trait objects and `T implements X` at compile time).*

Checked 2026-10-01 (probes in the scratchpad):

- **Errors are already generic.** `fun mapV<T, U, E>(xs: List<T>, f: fun(T):
  U throws E): List<U> throws E` called with `x => x * 2` needs no `try`
  (`E` is inferred empty); with `x => try check(x)` it throws `Bad`.
- **Suspension is not.** A parameter `f: fun(T): U suspends` makes every
  call suspend, even with a pure lambda: `m.withLock(p => mapS(xs, x => x +
  *p))` is refused, and the suspension spreads to every caller (`log.withFields`
  and `TaskLocal.withValue` make all their callers suspending today). And the
  built-in adapters refuse a suspending lambda outright: `[1, 2].map(x =>
  slow(x))` is an error. This, not speed, is what B8 is blocked on.
- **`is Trait`** is refused with a wrong message: `x is Display` for `x: i64`
  says "'i64' can never be 'Display'" (it is). Decision-free fix: say that
  `is` cannot test a trait, whatever is decided below.

### B5.1 — suspension that follows the argument

- **A. Suspension follows the argument** (recommended; Swift's proposed
  `reasync`, the way `throws E` already works here): a `fun(…) suspends`
  parameter means *may* suspend; a call suspends only when an argument it
  passes there does. The function is compiled at most twice — a plain
  instance and a coroutine instance — and each call picks one (D8's
  stenciling gains one bit). No new syntax; a call that suspended before
  only stops suspending when nothing it passes can, so no program loses a
  suspension point it relied on. Then the eager adapters move into the
  prelude as ordinary Veles (B8 unblocked, `xs.map(x => slow(x))` works,
  sequentially), `withFields`/`withValue` stop making pure callers
  suspend, and the lowered Go adapters — most of the IR size measured
  under B2 — go away.
- **B. An explicit marker**: `suspends(f)` or `fun(…) suspends?` on the
  parameter. Same machinery, visible at the declaration; one more thing to
  write on every higher-order function.
- **C. Leave it**: the adapters stay lowered in Go, B8 stays blocked.

Self-hosting: a Veles compiler's passes are written as higher-order
functions over trees (visitors, `map` over lists of nodes); with A they are
written once and stay plain where nothing suspends.

### B5.2 (Q5) — `is Trait`

Use: Go's "optional interface" upgrade — `if (w is Flusher) w.flush()` in a
streaming response, `if (e is HasStatus) e.status()` when mapping errors.

- **A. At run time, on trait objects** (recommended): `s is Named` on a
  trait-object value tests whether its concrete type implements `Named`,
  and narrows `s` to `Named` inside (`when` arms too). The program is linked
  whole, so the compiler emits one table of (type, trait) → method table;
  the check is a lookup by the type id the box already carries (its method
  table). Kotlin `is`, Swift `as?`, Go's type assertion. On a concrete type
  the answer is static (a warning that it is always true/false).
- **B. At compile time, in generics**: `if (T implements Display)` resolved
  per instantiation. Complements A; does not cover trait objects.
- **C. Both.**
- **D. Neither**: keep refusing, with a correct message (sealed traits and
  `when` cover closed sets; an open set adds a method to the trait).

### B5.3 (Q6) — user-defined derivation

D58 derives `Codable`/`Comparable` in the compiler; a library cannot derive
its own trait (a `Hash` for a custom map, a `Validate`, a DB row mapper
beyond `Codable`). std's own needs are covered today.

- **A. Not now; the direction is compile-time reflection** (recommended):
  once D113's `const fun` evaluator exists, a trait's author writes the
  derivation once as ordinary Veles that walks a type's fields at compile
  time (Zig's `@typeInfo` + `inline for`) — no macro language. Designed when
  a library asks.
- **B. Not now; the direction is macros** (Rust `proc_macro`): a plugin turns
  syntax into syntax. Most general; a second language for authors, slow
  builds, unreadable errors.
- **C. Design it now.**

### B5.4 — `Default`

Field defaults cover structs; generic code has no way to say "a zero value of
`T`" (`MutableList<T>.make(n, …)`, `m.getOrPut(k, …)`, a buffer of `T`).

- **A. A prelude trait `Default { static fun default(): Self }`**
  (recommended): implemented for the numbers (`0`), `bool` (`false`),
  `string` (`""`), the collections (empty), `T?` (`null`), tuples of
  `Default`; a struct opts in with an empty `implement Default` (D58's rule:
  every field has a default or is `Default`); `T.default()` under `T:
  Default`.
- **B. Automatic**: every struct whose fields all have defaults is
  `Default` without writing it.
- **C. Leave it.**

---

## 9. Batch 6 — FFI (Q7) (decided: D120–D123)

*Answers 2026-10-01: every recommended option.*

Checked 2026-10-01:

- **Bug, decision-free:** a C function returning a struct by value is called
  with the LLVM aggregate as is — `declare %S.main.LDivT @lldiv(i64, i64)` —
  with no C ABI lowering (clang does that in its front end, not LLVM). On
  Windows `lldiv` (16 bytes, returned through a hidden pointer) **segfaults**;
  `div` (8 bytes) printed the right answer by luck, and on SysV `{i32, i32}`
  is returned differently from C's packed `rax`. Arguments by value have the
  same hole. Plan item (track A): a C ABI classifier per target (Win64,
  SysV x86-64, AAPCS64) for extern calls, `extern fun` pointers and exported
  `extern "C" fun`s, a test calling `div`/`lldiv`/a struct-taking function on
  each platform; until it lands, a by-value `extern struct` in an extern
  signature is an error naming the pointer form.
- `extern struct` is used nowhere in std or the examples; there is no fixed-
  size array type (C's `char name[16]`), no union, no packed layout.
- What is coming: OpenSSL and libpq (E1/E2) use opaque pointers and arrays of
  pointers — they need none of this. SChannel's structs are natural layout.
  The reactor (E3), if any of it is written in Veles: `struct epoll_event` is
  **packed** on x86-64 Linux and holds a **union**; `OVERLAPPED` holds a
  union; `sockaddr_in` has `sin_zero[8]`. POSIX `open` and `fcntl` are
  **variadic** — new since §4b excluded varargs.

### B6.1 — layout: packed, alignment, unions, transparent wrappers

- **A. Compiler-known attributes, plus `extern union`** (recommended; D51
  attributes are for exactly what the compiler implements):
  ```veles
  @packed
  extern struct EpollEvent {
    events: u32
    data: EpollData
  }
  extern union EpollData { ptr: *raw (); fd: i32; u64: u64 }
  @align(16) extern struct Frame { … }       // and @align(n) on a field
  @transparent struct Fd { raw: i32 }         // passed to C exactly as its one field
  ```
  A union's fields are read only in `unsafe` (which one is live is C's
  business); `@transparent` is allowed on any one-field struct (Rust's
  `repr(transparent)`), so a binding can give `i32` handles their own type
  at no cost.
- **B. One layout clause**: `extern(packed, align: 16) struct …`, `extern
  union`; `@transparent` as in A. Groups layout in one place; a second
  syntax for what attributes already express.
- **C. Leave it** (bindings pad by hand; no unions).

### B6.2 — fixed-size arrays

C's `u8 name[16]` inside a struct, and — beyond FFI — a value that lives
inline without a heap allocation (a SHA-256 state, a small stack buffer, a
lexer's lookahead).

- **A. A general value type `Array<T, N>`** (recommended): `N` a constant
  (D113); stored inline, copied like a struct; `a.at(i)` checked (free for a
  constant index or a proven one, D62), `a.set(i, v)` on a `var`, `len()` a
  constant, `loop (x in a)`, a literal `[…]` where the type is expected,
  `a.toList()`; legal in `extern struct` (C's `T x[N]`) and everywhere else;
  functions generic over the length: `fun sum<const N: i64>(a: Array<i64,
  N>)` (a *const generic parameter*). Rust `[T; N]`, Go `[N]T`, Swift
  `InlineArray`.
- **B. Spelled `[T; N]`** — Rust's spelling, same semantics as A.
  (Brackets are literal syntax only today, D25.)
- **C. Only inside `extern struct`**, accessed through a raw pointer in
  `unsafe`; no general type.
- **D. Leave it.**

### B6.3 — `.d.vs` declaration files

The idea (notes #16): a file kind holding only declarations, for typed,
shareable bindings. Since D67 an ordinary package can hold `extern` blocks,
so a binding is already shareable as a package.

- **A. Drop the idea** (recommended): a binding is an ordinary package — by
  convention a raw `-sys`-style package of `extern` blocks plus a safe
  wrapper package on top (Rust's `-sys` crates, Go's cgo packages); a future
  `veles bindgen` writes `.vs`. The generated `builtins.vs` stub stays
  reference-only. Notes #16 leaves the file.
- **B. A declaration-only file kind** `.d.vs`: checked to contain no bodies;
  used by bindgen output and the builtins stub.
- **C. Later.**

### B6.4 — calling variadic C functions

§4b excluded them ("the varargs calling convention differs per platform").
New: POSIX `open(path, flags, mode)` and `fcntl(fd, F_SETFL, flags)` are
variadic, and *calling* one from LLVM IR is the easy half — LLVM implements
each platform's convention for a call; the hard half (defining one, `va_list`)
is not needed.

- **A. Calls only** (recommended): `extern "C" { fun fcntl(fd: i32, cmd:
  i32, ...): i32 }`; the extra arguments must be numbers, `bool` or raw
  pointers (C's default promotions applied: `f32`→`f64`, integers narrower
  than `i32` widened), never a struct or a Veles value; defining a variadic
  function stays impossible. `printf` becomes callable (still `unsafe`).
- **B. Keep them excluded**: bind a fixed-arity C wrapper (a `.c` file in
  `[native]`).

---

## 10. Batch 7 — the std APIs ahead of the plan (decided: D124–D130)

*Answers 2026-10-01: B7.1 default 64 MiB; B7.2 recommended; B7.3
OpenTelemetry, then all signals + protobuf; B7.4 both spellings; B7.5
recommended; B7.6 the tagged literal, after an explanation; B7.7 all four.
Open: the health endpoints (bundled with B7.3's option A, not taken).*

So tracks C and E1/E2 run without stopping. Each module's surface below is
the recommendation; the questions are where a real choice lies. Existing
shape to follow: required limits on anything a peer controls (`readLine(max:)`,
2026-09-22), errors collected (`DecodeError.problems`), `with` for anything
that must be released (D100), `Secret` for credentials (D112), derive-driven
decoding (D58).

### Round 1

**B7.1 `std/compress`** (D99: written in Veles; gzip first).
`compress.gzip(bytes, level: 6): List<u8>`, `compress.gunzip(bytes, max:):
List<u8> throws CompressError`, `deflate`/`inflate` the same, streaming
`compress.GzipWriter(out)` / `GzipReader(input, max:)`. `http.compress()`
middleware: gzip when the client accepts it, the body is ≥ 1 KiB, the type is
textual (`text/*`, JSON, JS, SVG, XML) and not already encoded; adds `Vary:
Accept-Encoding`; streamed bodies compressed as they stream; `http.files`
serves a `x.gz` sibling when present. The question: **the decompression
limit** (a 10 KB gzip can expand to 10 GB).
- **A. `max:` required** (recommended; as `readLine`): the caller states the
  ceiling; exceeding it is `CompressError.TooLarge`.
- **B. A default** (64 MiB) with `max:` to change it.
- **C. No limit.**

**B7.2 `std/config`** (C3: typed, every missing key at once).
- **A. Derived from a struct, environment first** (recommended):
  ```veles
  struct Config {
    port: i64 = 8080
    databaseUrl: Secret<string>        // DATABASE_URL
    @key("LOG_LEVEL") level: log.Level = log.Level.Info
    db: DbConfig                       // DB_POOL_SIZE, DB_TIMEOUT ...
    implement Decodable
  }
  val cfg = try config.load<Config>()                    // environment only
  val cfg = try config.load<Config>(files: [".env"])     // a dotenv file under it
  ```
  Field names map to `SCREAMING_SNAKE` (`databaseUrl` → `DATABASE_URL`),
  nested structs prefix (`db.poolSize` → `DB_POOL_SIZE`), `@key` overrides;
  values parse by type (`Duration` as `"30s"`, enums by name, lists
  comma-separated); a field default is used when the variable is absent;
  every problem reported at once (`config.Error.problems`, the `DecodeError`
  shape) naming the variable; optional `files:` (dotenv format; JSON by
  extension), the real environment always winning; `prefix: "APP_"`.
- **B. Environment only**, same derive, no files.
- **C. Getters**: `config.string("PORT")`, `config.i64("PORT", default:)`
  — no struct, problems found one at a time.

**B7.3 Observability** (C4).
- **A. Prometheus-shaped metrics + health + traceparent** (recommended):
  `val requests = metrics.counter("http_requests_total", help: "…", labels:
  ["method", "status"])`, `requests.labels("GET", "200").inc()`; `gauge`
  (`set`/`add`), `histogram(buckets:)` (`observe(Duration)` or `f64`); the
  registry is global and lock-free per series; `http.metrics()` serves the
  text format; runtime metrics (GC pauses and heap from E5, tasks,
  connections, threads) registered automatically; `http.serve` records
  request count/latency by route. `http.health(checks: [("db", () => try
  pool.ping())])` serves `/healthz` (200 while the process serves) and
  `/readyz` (each check with a timeout, 503 when one fails or the server is
  stopping). `requestId()` reads or creates a W3C `traceparent` and puts the
  trace id in the log fields (D91).
- **B. OpenTelemetry-shaped**: meters, instruments, exporters, OTLP push.
  The industry's direction; much larger, needs the HTTP client and protobuf.
- **C. Counters and gauges only**, no labels or histograms.

**B7.4 HTTP client** (§5.3; TLS from E1).
- **A. Go's shape**: a `http.Client(timeout:, maxRedirects:, pool:)` with
  `client.send(req)`, and `http.get(url)` / `http.post(url, body)` shortcuts
  over a shared default client.
- **B. `fetch`'s shape** (recommended; the user likes JS's ergonomics):
  `val res = try http.fetch("https://api.example.com/users", method:
  Method.post, headers: [...], json: payload, timeout: Duration.seconds(5))`
  then `try res.json<User>()`, `res.text()`, `res.stream()` (the D97 body);
  a `Client` only to share a pool or defaults (`client.fetch(...)`).
- **C. Both spellings.**
Common to all: connection pooling per host, a total timeout by default (30 s)
and per-phase timeouts, redirects followed for GET/HEAD only (≤ 10), the body
read lazily with a required `max` for whole reads, `HTTP_PROXY`/`HTTPS_PROXY`/
`NO_PROXY`, retries only when asked (`retry:`, idempotent methods only,
D110's `retry`), TLS verification on.

### Round 2

**B7.5 A stream trait shared by `net.Conn` and `tls.Conn`.**
- **A. `io.Stream`** (recommended): `read(max)`, `readExact(n)`,
  `readLine(max)`, `write(bytes)`, `writeText`, `shutdownWrite`, `close`;
  `net.Conn`, `tls.Conn`, `fs.File` (read/write part) implement it; `http`'s
  server and client take a `Stream`, so HTTPS is the same code as HTTP.
- **B. Separate types**; `http` has a TLS path of its own.
TLS API either way: `tls.connect(host, port, options)`, `tls.listen(…, cert:
tls.Certificate.load(cert, key))` with `tls.CertificateReloader` for rotation,
verification on by default with `dangerouslyAcceptAnyCertificate: true` as
the only opt-out (loud by name); backends as planned (SChannel on Windows,
OpenSSL elsewhere).

**B7.6 SQL injection by construction** (`std/db`, E2).
- **A. A tagged string literal `sql"…"`** (recommended; JS tagged templates,
  Rust sqlx, Scala doobie): `pool.query<User>(sql"select * from users where
  id = ${id} and age > ${min}")` — each `${…}` becomes a bound parameter
  (`$1`, `$2`), never text; `query` takes a `db.Sql`, and a plain `string`
  is refused, so concatenating input into SQL cannot compile. A language
  feature: a prefix on a string literal names a type implementing a
  prelude `Template` trait that receives the literal parts and the values;
  `html"…"` (escaping) and `regex"…"` could use it later.
- **B. A constant query plus arguments**: `pool.query<User>("select … where
  id = $1", id)` with the SQL required to be a compile-time constant (D113).
  No new syntax; placeholders counted by hand.
- **C. A plain string plus arguments**, documented.
The rest of `std/db`, recommended: `db.Pool(url: Secret<string>, size:
10)`; `query<T: Decodable>(sql): List<T>`, `queryOne<T>(sql): T?`,
`exec(sql): i64` (rows affected), rows decoded by the derive with column
names as keys (`@key(db: "user_id")`, D58); `with tx = try pool.begin()` —
`tx.commit()` explicit, rolled back on close if not committed; statement and
connection timeouts; health `pool.ping()`; libpq binding first.

**B7.7 Small pre-approvals** (multi-select):
- `with srv = try http.testServer(handler)` (D111): a real listener on a
  free port, `srv.url`, `srv.port()`.
- Endian bytes (D104's names, D121's arrays): `x.toBeBytes()` /
  `toLeBytes(): Array<u8, N>`, `u32.fromBeBytes(a)` / `fromLeBytes(a)` on
  every integer type, and `bytes.readU32Be(offset): u32?`-style readers on
  `List<u8>` for every width and both orders.
- `std/fs` additions: `fs.writeAtomic(path, bytes)` (temp file, fsync,
  rename), `with lock = try file.lock()` / `tryLock()` (exclusive advisory:
  `flock` / `LockFileEx`), `file.sync()`, `file.seek(offset)`,
  `fs.lines(path)` read lazily (each line bounded by `max:`), `fs.copy`.
- Password hashing: `crypto.hashPassword(Secret<string>): string` /
  `verifyPassword` with argon2id through a binding (§5.5).

---

## 11. Batch 8 — packages (Q14, M7, notes #3 and #17) (decided: D131, D132)

*Answers 2026-10-01: B8.3 no lockfile; B8.4 scripts follow the manifest; B8.1
after two rounds `package.vs` with target-only conditions (the user's
reasoning moved the recommendation); B8.2 decentralized with all four
refinements.*

Today: `veles.toml` with `[package]` (name, version), `[dependencies]`
(paths only: `mathlib = "../mathlib"` or `{ path = "…" }`), `[format]`,
`[native]` (D67). The spec has M7 (Go's minimal version selection, no
lockfile needed), "`veles.sum` mandatory", a "decentralized registry model",
v2+ in the import path — none built. A dependency is imported by its
manifest name (`use mathlib`, M6). Scripts (`.vss`) cannot use dependencies
yet (M1 amendment, open). The user's notes: "I do not like how the
dependencies in toml file are written..." (#3) and the idea of an own format
"parsed by the compiler" (#17).

### B8.1 — the manifest's format

The same package three ways (name, a path dependency, a published one,
native libraries, formatter options):

**A. TOML, flattened** — one string per dependency, `source@version`:
```toml
[package]
name = "notes"

[dependencies]
mathlib = "../mathlib"
httputil = "github.com/acme/httputil@1.4.2"

[native]
libs = ["pq"]

[format]
indent = 2
```

**B. An own line format, `veles.mod`** (recommended; Go's `go.mod` shape):
```
package notes

require mathlib   ../mathlib
require httputil  github.com/acme/httputil  1.4.2

native libs pq
native pkg-config libpq

format indent 2
```
One statement per line, `keyword arguments…`, `#` comments; the tools
rewrite it (`veles add`, `veles update`), so the order and spacing are
canonical. Trivial to parse — which the self-hosted compiler will have to
do; TOML needs a full TOML parser written in Veles.

**C. Veles syntax**, a `package.vs` with a declarative block:
```veles
package notes {
  require("mathlib", path: "../mathlib")
  require("httputil", "github.com/acme/httputil", "1.4.2")
  native(libs: ["pq"])
  format(indent: 2)
}
```
Familiar syntax; but M1 says the resolver never parses Veles source, and a
manifest that looks like code invites code (conditions, loops) it cannot
run.

Edge cases for all: `veles.toml` migrated by `veles migrate-manifest` (or
by hand: three example packages and the templates); the package `version`
line disappears (versions are git tags, B8.2); `veles new` writes the new
file; the LSP and `veles fmt` read `format` from it.

### B8.2 — where packages come from

- **A. Decentralized, Go's model** (recommended; what the spec already says):
  a package's identity is its repository path (`github.com/acme/httputil`);
  versions are git tags (`v1.4.2`); `veles` fetches with git (any host), or
  through an optional proxy/cache (`VELES_PROXY`); M7's minimal version
  selection; a v2+ package has `/v2` at the end of its path, so v1 and v2 can
  coexist; `veles.sum` records a hash of every module version fetched and is
  checked on every build (and committed). No service to run. Commands:
  `veles add <path>[@version]` (latest tag when none), `veles update
  [name|--all]` (the explicit upgrade M7 requires), `veles remove`,
  `veles deps` (the graph and why each version), `veles vendor`
  (copy everything into `vendor/` for offline builds).
- **B. A central registry** (crates.io, npm): short names, `veles publish`,
  an index. Better discovery; a service to build, run and secure.
- **C. Both**: decentralized now, a central index of names → paths later
  for search only.

### B8.3 — a lockfile

- **A. None** (recommended; M7): with minimal version selection the
  manifest's `require` lines *are* the build list, and `veles.sum` makes
  the bytes reproducible. Go has no lockfile for this reason.
- **B. A lockfile anyway** (Cargo): an exact resolved list, redundant with
  MVS, one more file to merge.

### B8.4 — dependencies in a script (`.vss`)

A one-file program has no manifest (M1).
- **A. `require` lines at the top of the script** (recommended), the same
  statement as the manifest's:
  ```veles
  require httputil github.com/acme/httputil 1.4.2
  use httputil
  ```
  checked into `veles.sum` beside the script (`script.vss.sum`).
- **B. Scripts stay std-only**; use a package when you need dependencies.

### Batch 8, second round (the user asked for full examples and more ideas)

Answers so far: B8.3 no lockfile; B8.4 "probably 'require' but should depend
on our normal module decision" — scripts use the manifest's own dependency
statement, whatever B8.1 becomes.

**One realistic manifest in each format** — a service with a path
dependency, two published ones (one on its second major version), a
test-only dependency, native libraries and formatter options:

A. TOML, flattened
```toml
[package]
name = "notes"
description = "A small notes service"
license = "MIT"

[dependencies]
mathlib  = "../mathlib"
httputil = "github:acme/httputil 1.4.2"
pg       = "github:veles-db/pg 2.1.0"

[test-dependencies]
fakeclock = "codeberg:lah/fakeclock 0.3.0"

[native]
libs = ["pq"]
pkg-config = ["libpq"]

[format]
indent = 2
max_blank_lines = 1
```

B. `veles.mod`, a line format
```
package notes
description "A small notes service"
license MIT

require mathlib    ../mathlib
require httputil   github:acme/httputil   1.4.2
require pg         github:veles-db/pg     2.1.0
require test fakeclock  codeberg:lah/fakeclock  0.3.0

native libs pq
native pkg-config libpq

format indent 2
format max-blank-lines 1
```

C. `package.vs`, one typed Veles constant (checked by the compiler, D113)
```veles
const package = Package(
  name: "notes",
  description: "A small notes service",
  license: "MIT",
  require: [
    path("mathlib", "../mathlib"),
    github("httputil", "acme/httputil", "1.4.2"),
    github("pg", "veles-db/pg", "2.1.0"),
  ],
  testRequire: [codeberg("fakeclock", "lah/fakeclock", "0.3.0")],
  native: Native(libs: ["pq"], pkgConfig: ["libpq"]),
  format: Format(indent: 2, maxBlankLines: 1),
)
```

In every case the code says only `use httputil`, `use pg` — the paths live
in the manifest alone (M6), unlike Go where every import line carries
`github.com/…/v2/…`.

| | A. TOML | B. `veles.mod` | C. `package.vs` |
|---|---|---|---|
| Reads as | config | a list of statements | code |
| Noise | quotes, `=`, sections | none | quotes, commas, parens |
| Known from | Cargo, pyproject | Go | Swift's `Package.swift`, Zig's `build.zig.zon` |
| Editor | a TOML extension | our LSP (small grammar) | our LSP for free: completion, hover, errors from the `Package` struct |
| Tools rewrite it (`veles add`) | a TOML writer that keeps comments | line edits | the formatter on a Veles file |
| Self-hosted compiler needs | a full TOML parser in Veles | ~100 lines | nothing new (its own parser + D113) |
| Risk | — | a format nobody knows yet | looks like code, invites logic the resolver must refuse |

Recommendation stays **B**: the least noise, and the reader sees exactly the
dependency list; C is the strongest alternative (typed and checked by the
compiler with no new parser).

**Decentralized, without Go's look.** What makes Go's model look awful is
mostly avoidable:

1. *Paths only in the manifest* — already true in Veles (`use httputil`).
2. *Host shorthands*: `github:acme/httputil`, `gitlab:`, `codeberg:`,
   `sourcehut:`; any other git host in full (`git.example.com/team/lib`).
   npm and Deno use the same `host:` specifiers.
3. *The major version in the version, not the path*: identity is (path,
   major); `pg 2.1.0` is just written, never `github.com/veles-db/pg/v2`.
   Two majors side by side only under two local names (`require pg1 …
   1.9.0` and `require pg … 2.1.0`). MVS runs per (path, major).
4. *No pseudo-versions*: a dependency is a tag (`1.4.2`; tags `v1.4.2` and
   `1.4.2` both read), a path, or an explicit `commit 3f2a9c1` pin — never
   `v0.0.0-20240101123456-3f2a9c1d2e4f`. A library depending on a commit
   gets a warning (its users cannot do MVS over it).
5. *Publishing is tagging*: nothing to upload. A search index (names →
   paths, nothing more) can come later without changing a manifest.
