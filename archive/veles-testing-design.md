# Testing — design note (decided 2026-09-27: the recommendation, spec D78; built)

Asked 2026-09-27, answering the proposal of a built-in `assert`:

> *"I was thinking about testing module with different devex
> assertions/helpers, however they should only be usable inside test
> function... on the other hand, I was thinking: do we need '@test' and
> other annotations for it? maybe we could add "test" keyword that would be
> synonymous to the 'fun' in terms of tests... but that could be weird...
> WE NEED TO INVESTIGATE THE TOPIC FURTHER"*

This note is that investigation: what testing is today, what the
languages Veles borrows from do, the five questions a design has to
answer, the options for each with the same program written in each, and a
recommended combination. Nothing is built until you choose.

---

## 1. Today

```veles
use io

error Mismatch<T> { expected: T, actual: T }

fun expectEq(expected: i64, actual: i64) throws Mismatch<i64> {
  if (expected != actual) throw Mismatch(expected, actual)
}

fun add(a: i64, b: i64): i64 = a + b

@test
fun additionWorks() throws Mismatch<i64> {
  try expectEq(4, add(2, 2))
}
```

```text
test additionWorks ... FAILED: Mismatch(expected: 4, actual: 5)
```

What works: `@test` on a parameterless function; it passes by returning
and fails by throwing or panicking; `veles test` runs them with
`--filter` and `--timeout`, prints a summary, and `veles build` leaves
them out. A test may suspend (`scope`, channels).

What does not:

1. **Every project writes its own assertions**, and each needs an error
   type, a `throws`, and a `try` at every use.
2. **A failure says what, never where.** A thrown error carries no
   location; the line above does not say which `expectEq` failed.
3. **Nothing shows the expression.** `Mismatch(expected: 4, actual: 5)` —
   of what?
4. **The name is an identifier**: `additionWorks`, where the reader wants
   a sentence ("adds two small numbers").
5. **No test-only surface**: a temp directory, capturing output, "this
   must throw", "this must panic" — each written by hand, and nothing
   stops test helpers leaking into the program.

## 2. What the others do

| | declares a test | asserts | scoped to tests? | where tests live |
|---|---|---|---|---|
| Go | `func TestX(t *testing.T)` | `t.Errorf`, `t.Fatalf` (libraries add more) | `testing` is an ordinary package | `x_test.go` files |
| Rust | `#[test] fn x()` | `assert!`, `assert_eq!` (macros: expression text + both sides + location) | `#[cfg(test)] mod tests` | same file, gated module; `tests/` for integration |
| Zig | `test "adds numbers" { ... }` | `try std.testing.expect(x)`, `expectEqual` | `test` blocks compile only under `zig test` | same file |
| Kotlin | `@Test fun x()` (JUnit) | `assertEquals`; power-assert plugin prints the expression tree | test source set | `src/test/` |
| Swift | `@Test func x()` (Swift Testing, 2024) | `#expect(a == b)` — a macro: captures the expression, both sides, location; `#require` stops the test | test target | test target |
| Elixir | `test "adds numbers" do ... end` | `assert a == b` (a macro: shows left and right) | `ExUnit.Case` modules | `test/` |
| JS (Jest/Vitest) | `test("adds numbers", () => { ... })` | `expect(x).toBe(y)` matchers | test files by name | `*.test.ts` |

Three things stand out. **Every modern design captures the expression**
(Rust, Swift, Elixir, power-assert): the failure shows what was written
and what each side was. **The newer designs name tests with a string**
(Zig, Swift's `@Test("…")`, Elixir, Jest). **Test-only code is gated by
where it lives**, a file or a block, not by a runtime check.

## 3. The five questions

### Q1. How is a test declared?

**(a) `@test fun name()` — today.**
Familiar from Kotlin/Swift; a test is an ordinary function a tool treats
specially. Costs: a name that must be an identifier, a signature to write
(`throws ...` today), and an attribute whose only job is to say "test".

**(b) `test fun name()` — a modifier, your first idea.**
```veles
// fragment
test fun additionWorks() {
  expect(add(2, 2) == 4)
}
```
Reads like `static fun`, `mut fun`. It is still a function — it can be
named in hover and references, and can take a `throws` — but nothing may
call it. The "weird" part you felt: a function nobody calls, with a
parameter list that must be empty.

**(c) `test "name" { … }` — a block with a sentence for a name.**
```veles
// fragment
test "adds two small numbers" {
  expect(add(2, 2) == 4)
}
```
Zig and Elixir. No signature at all: it cannot be called, cannot take
parameters, may throw and suspend without saying so (its effects are
inferred like a lambda's), and the report prints the sentence. `test` is
contextual: only `test "` at top level starts one, so a function or a
variable named `test` stays legal. `--filter` matches the sentence.

Edge cases for (c): two tests with the same name in a module → an error
(the report would be ambiguous); interpolation in the name → refused (a
name is fixed at compile time); hover shows the name, go-to-definition
from a failure line lands on it; the formatter treats it like a function
body; the LSP offers "run this test" as a code lens (later).

### Q2. Where do assertions live, and who may call them?

**(a) A built-in `assert` everywhere** (my first proposal). One word,
tests and invariants alike. Your objection holds: test helpers should not
be available to production code, and the right failure behaviour
differs — a test wants to keep going and report *all* failed
expectations, production wants to stop.

**(b) A `testing` module anyone can import.** No compiler work, but it
leaks into programs, needs `try`, and loses the location.

**(c) Test-only built-ins, in scope only inside tests** — your
direction. Inside a test (whatever Q1 says), a small vocabulary is in
scope; outside, the same names are "unknown" with a hint:

```veles
// fragment
test "parses a port" {
  expect(parsePort("80") == 80)                  // soft: records, the test goes on
  val cfg = require(load("app.toml"))            // hard: a failure ends the test here; unwraps T? / Result
  expectThrows<RangeError>(() => parsePort("70000"))
  expectPanics(() => [1].at(5) ?: panic("no"))
  fail("not written yet")
}
```

- `expect(cond)` **captures the expression** (the compiler knows it): the
  source text, and for a comparison both sides' values; for `a == b` on
  lists, maps and structs a **diff** of the two. It does not stop the test,
  so one run reports every broken expectation (Swift's `#expect`).
- `require(x)` stops the test on failure and unwraps: a `T?` gives `T`, a
  `Result<T, E>` gives `T` (Swift's `#require`, Rust's `.unwrap()` in a
  test). This is what makes `throws` unnecessary in tests.
- `expectThrows<E>(f)`, `expectPanics(f)`, `fail(why)`.
- The location is the call's, always (the compiler writes it, so §11's
  std-panic caveat does not apply).

Because they are compiler-known, "only inside tests" is a checked rule,
not a convention — and hover documents each like `panic`.

### Q3. How are test helpers written?

A helper that sets something up and asserts (`expectValidJson(text)`)
must itself be able to call `expect`. Options:

**(a) Helpers are ordinary functions that return a value; only tests
assert.** Simple, strict; loses reusable assertions.

**(b) A test-only function form** — `test fun expectValidJson(text:
string)` (Q1 b's modifier, here for helpers): callable only from tests
and other test functions, may use the test vocabulary, left out of
`build`.

**(c) Test-only files** — every declaration in `name.test.vs` (Go's
`_test.go`, Jest's `.test.ts`) is test-only: it may use the test
vocabulary, it sees the module's private declarations (it is part of
the module), and `build`/`run` never load it. Tests and their helpers
live together, and a production file cannot reach them by accident.

(b) and (c) combine well: small modules keep a test next to the code
(`test "…" { }` in the same file), a module with many tests moves them to
`x.test.vs`, and helpers are `test fun` anywhere or plain `fun` in a test
file.

### Q4. Fixtures and resources

Setup/teardown is `with` (D43) already — a test does not need
`beforeEach`:

```veles
// fragment
test "writes and reads back" {
  with (dir = testing.tempDir()) {      // removed when the test ends, however it ends
    try fs.writeFile(path.join(dir.path, "a.txt"), "hi")
    expect(try fs.readFile(path.join(dir.path, "a.txt")) == "hi")
  }
}
```

The test-only vocabulary grows by library, not by keyword:
`testing.tempDir()`, `testing.captureOutput(f)`, an in-memory clock
later. `http.call` (already in std) is the pattern for servers.

### Q5. Table-driven tests

The common shape is a loop over cases. With soft `expect`, a loop needs
nothing new — every failing case is reported, each with its values:

```veles
// fragment
test "parses ports" {
  loop ((text, want) in [("80", 80), ("0", 0), ("65535", 65535)]) {
    expect(parsePort(text) == want)          // "parsePort(text) == want — left: 0, right: 80, text = "80""
  }
}
```

Showing the loop variables in the failure (`text = "80"`) is a nicety of
expression capture: the compiler knows which names the condition reads.

## 4. Invariants in ordinary code

Separate from testing, and worth deciding together: a program sometimes
wants "this cannot happen" checked. Today that is
`if (!ok) panic("why")`. A built-in `check(cond)` (Kotlin's `check`,
Swift's `precondition`) with the same expression capture as `expect`,
but panicking, always on, available everywhere, would make the test
vocabulary and the invariant vocabulary two different words for two
different jobs: `expect` records and continues (tests only), `check`
stops the program (anywhere). Or leave invariants as `panic`.

## 5. Recommendation

- **Q1 (c)** `test "sentence" { }` — the name is prose, there is no
  signature to get wrong, and "a function nobody calls" disappears. Keep
  `@test fun` parsing with an error and a fix to the new form (the house
  rule for removed syntax); the one decision this changes is D-level
  syntax.
- **Q2 (c)** `expect` / `require` / `expectThrows` / `expectPanics` /
  `fail`, compiler-known, in scope only in tests, with expression capture,
  diffs and the caller's location.
- **Q3 (b)+(c)** `test fun` for helpers anywhere, `*.test.vs` files for
  suites, both left out of `build`.
- **Q4** fixtures are `with`; a small `testing` library for temp
  directories and output capture, importable only from test code.
- **§4** `check(cond)` for invariants, sharing `expect`'s capture.

The one reason: a failure should say *what was expected, what happened,
and where* without the author writing any of it — which only the
compiler can do, and which is why the vocabulary is built in and scoped
rather than a library.

## 6. Cost

Parser (`test "…" { }`, `test fun`), AST dump and formatter, sema (test
scope for the vocabulary, the `*.test.vs` loader rule, effects of a test
block inferred), codegen (expression capture: evaluate each operand once,
format both sides; soft failures counted per test), the runner (report
all failures of a test, the sentence as the name), LSP (hover for the
vocabulary; later a "run test" lens), docs 14 rewritten, `examples/testing`
and every `@test` in the tree migrated by `--fix`. Roughly the size of
D61 (let-else).

> **Amended 2026-09-27:** §4's `check(cond)` is `assert(cond, "why")`, the reason required (user). Tests became D78.
