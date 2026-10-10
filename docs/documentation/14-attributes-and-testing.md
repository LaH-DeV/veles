# 14. Attributes and tests

## Attributes

An attribute is `@name` or `@name(args)` on the line before a
declaration (D51). The compiler knows each one; an unknown one is an
error, not a silent no-op.

| Attribute | On | Effect |
|---|---|---|
| `@deprecated("why")` | any declaration | a warning at every use, with the message |
| `@mustUse` | a function returning a value | an error if a call's result is discarded |
| `@inline` / `@noinline` | a function | a hint to the optimiser (`--release`) |
| `@template` | a function `(parts: List<string>, values: List<V>)` | the function a template literal `name"…"` calls ([chapter 2](02-values-and-strings.md)) |
| `@key("k")` / `@key(json: "k", db: "c")` | a field, an enum member, a sealed variant | its name on the wire, for every format or by format ([chapter 18](18-codable-and-json.md)) |
| `@skip` / `@skip(json)` | a field with a default | left out of the wire form, everywhere or in one format |
| `@required` | a nullable field | the key must be present even though the value may be null |
| `@tag("kind")` / `@tag("type", content: "value")` | a sealed trait | the key that names the variant; with `content`, the key the fields go under |

```veles
use io

@deprecated("use greet")
fun hello(name: string): string => "hello $name"

fun greet(name: string): string => "hi $name"

@mustUse
fun important(): i64 => 7

@inline
fun tiny(x: i64): i64 => x + 1

fun main() {
  io.println("${greet("ann")} ${important()} ${tiny(1)}")
}
```

Output:
```text
hi ann 7 2
```

Change `greet("ann")` to `hello("ann")` and the compiler prints
`warning: 'hello' is deprecated: use greet` with the call site. Write
`important()` as a bare statement and it refuses: `result of 'important'
must be used (@mustUse)`.

## Writing tests

A test is `test "what it checks" { ... }`, next to the code it tests, in
the same module — so it can call private functions (D78). The name is a
sentence, not an identifier; it is what the report prints. A test has no
signature: nothing calls it, it takes nothing, and it may throw and
suspend without saying so.

Inside a test, a small vocabulary is in scope:

| | |
|---|---|
| `expect(cond)` | records a failure when `cond` is false, **and the test goes on** — one run reports every broken expectation. For a comparison it shows both sides. |
| `require(x)` | the value of a `T?` or a `Result`; when there is none it records why and **ends the test**. |
| `expectThrows<E>(() => ...)` | a failure unless the function throws (an `E`, when one is named) |
| `expectPanics(() => ...)` | a failure unless the function panics (it runs in a task of its own) |
| `fail("why")` | records `why` and ends the test |

```veles
use io

error RangeError { value: i64 }

fun parsePort(text: string): i64 throws RangeError {
  val n = text.toInt() ?: 0
  if (n < 0 || n > 65535) throw RangeError(value: n)
  n
}

fun wordCount(text: string): i64 => text.split(" ").filter(w => !w.isEmpty()).len()

test "counts words" {
  expect(wordCount("one two three") == 3)
  expect(wordCount("  a   b ") == 2)
  expect(wordCount("") == 0)
}

test "parses ports and refuses the rest" {
  expect(require(parsePort("8080")) == 8080)
  expectThrows<RangeError>(() => parsePort("70000"))
}

test "tasks work in tests" {
  val ch = Channel<i64>(capacity: 1)
  scope {
    ch.send(41)
    val v = await ch.recv()
    expect((v ?: 0) + 1 == 42)
  }
}

fun main() {
  io.println("${wordCount("hello brave new world")}")
}
```

Output:
```text
4
```

`veles run` leaves the tests out; `veles test <dir>` runs each one as its
own task and prints a line per test:

```text
test counts words ... ok
test parses ports and refuses the rest ... ok
test tasks work in tests ... ok

3 passed, 0 failed
```

A failure says where, what was written, and what each side was — without
the test writing any of it. Drop the `filter` from `wordCount`, so empty
pieces count as words, and the report is:

```text
test counts words ... FAILED
  main.vs:15:3: expect(wordCount("  a   b ") == 2)
      left:  7
      right: 2
  main.vs:16:3: expect(wordCount("") == 0)
      left:  1
      right: 0
test parses ports and refuses the rest ... ok
test tasks work in tests ... ok

2 passed, 1 failed: counts words
```

Strings are shown quoted, so `""` and `" "` are visible; a `require` that
finds nothing says `was null`, or `threw: RangeError(value: 99999)`. The
summary line names every test that failed, and the exit code is non-zero.
A panic in one test does not stop the others.

What a test prints — `io.println` for a quick look at a value, from the
test or from the code it calls — is kept while it runs. A test that passes
drops it, so a green run reads as its verdicts alone; a test that fails
shows it under its failure, in the order it was written (standard output
and error together):

```text
test counts words ... FAILED
  main.vs:15:3: expect(wordCount("  a   b ") == 2)
      left:  7
      right: 2
  output:
    pieces: ["", "", "a", "", "", "b", ""]
```

A test that times out shows what it printed before the deadline, which is
often the only clue to where it hung.

Tests run at once, as tasks on the runtime's threads, so a suite takes
about as long as its slowest tests rather than all of them added up. The
report does not change with it: each test's lines — its verdict, what it
recorded, what it printed — come together, in declaration order, whichever
test finished first. Tests cannot race on memory (a module-level `var` is
behind a lock, D66, and only Sendable values cross between tasks), but
they can still step on each other's toes — two tests resetting one
counter, binding one port; `--jobs 1` runs them one at a time (D80).

Three flags shape a run:

- `--filter text` runs only the tests whose name contains `text`
  (`veles test . --filter ports`); the summary counts the rest as
  filtered out. A filter that matches nothing is an error, so a typo in
  CI fails instead of passing with no tests run.
- `--jobs n` runs at most `n` tests at once (default: one per thread of
  the runtime, `VELES_THREADS` or the machine's cores); `--jobs 1` runs
  them in order, one after another.
- `--timeout 30s` bounds each test (default `10m`, `0` for no bound).
  A test still running at its deadline is reported —
  `FAILED: timed out after 30s` — and ends the run (whatever the test was
  doing is not trusted to unwind cleanly); the report says how many other
  tests had not finished.

### Helpers and test files

A check used by several tests is a `test fun`: it may use the vocabulary,
only test code can call it, and a build leaves it out.

```veles
// fragment
test fun expectSorted(xs: List<i64>) {
  loop (i in 1..<xs.len()) {
    val before = xs.at(i - 1) ?: 0
    expect(before <= xs.at(i))
  }
}

test "sorts" {
  expectSorted([3, 1, 2].sorted())
}
```

A failure inside a helper says where the helper is, and which line of the
test called it — through helpers that call helpers, innermost first, into
a helper the test launched with `async`, and for a panic inside a helper
as well as for a failed `expect`:

```text
test sorts ... FAILED
  main.vs:4:5: expect(before <= xs.at(i))
      called from main.vs:9:3
```

When a module's tests outgrow its files, move them to a file named
`*.test.vs` in the same directory (`parser.test.vs` next to `parser.vs`).
Everything in it is test code: it sees the module's private names, may
use the vocabulary, and is never loaded by `veles build` or `veles run`.
It holds tests, `test fun` helpers and the types and values the tests need —
a plain `fun` there is an error whose fix writes `test fun` (a function the
program uses belongs in a module file) — and the module's own files cannot
name any of it: a type or helper declared in a test file is an error when a
non-test file uses it.

### Suites

A suite groups tests under a name. There are two ways to have one, and
they nest:

- `suite "name" { ... }` holds tests, other suites and `test fun`
  helpers. A helper declared in a suite is visible only inside it, so two
  suites can each have their own `expectParses`.
- A `*.test.vs` file is a suite by itself, named after the file:
  everything in `config.test.vs` is in the suite `config`.

```veles
// fragment
suite "parser" {
  test fun expectParses(text: string, want: i64) {
    expect(parseInt(text) == want)
  }

  test "parses ints" {
    expectParses("42", 42)
  }

  suite "rejects" {
    test "letters" {
      expect(parseInt("x4") == null)
    }
    test "empty input" {
      expectParses("", 0)
    }
  }
}
```

The report groups a suite's tests under its name, indented, with tests
outside any suite first; each failure stays under its test:

```text
test adds ... ok
parser
  test parses ints ... ok
  rejects
    test letters ... ok
    test empty input ... FAILED
      main.vs:5:5: expect(parseInt(text) == want)
          left:  null
          right: 0

3 passed, 1 failed: parser / rejects / empty input
```

A test's full name is its suites' names and its own, joined with ` / `:
that is what the summary lists and what `--filter` matches, so
`veles test --filter "parser / rejects"` runs one suite. Names need to
differ only within a suite — `parser / rejects / empty input` and
`router / empty input` are two tests.

### Invariants: `assert`

Outside tests, the vocabulary is unknown — a program does not "expect".
What a program has is invariants, and `assert(cond, "why")` states one
anywhere. The reason is required, as `panic`'s message is: an invariant
that breaks in production must say what was meant to hold. When `cond`
is false it panics with the reason, then the report `expect` gives; the
reason is only built then, so interpolating into it costs nothing on the
happy path.

```veles
use io

fun average(xs: List<i64>): i64 {
  assert(!xs.isEmpty(), "an average needs at least one number")
  xs.sum() / xs.len()
}

fun main() {
  io.println("${average([2, 4, 6])}")
}
```

Output:
```text
4
```

A comparison shows both sides, as in a test:

```text
panic: n is 3, so the total is off
      assert(n * 2 == 7)
      left:  6
      right: 7
  at main.vs:12:3
```

The pre-D78 form, `@test fun name() { }`, is an error whose fix writes
the new one — `veles check --fix` turns `@test fun parsesDates()` into
`test "parses dates"`.

## Conventions the compiler enforces so you need not

Looking back over the tutorials, a good deal of what a style guide would
say is already a compile error in Veles:

- an unused `Result` (chapter 7), a `when` missing a variant (9), a
  `val` that is assigned (2), a bare field assigned (5);
- a `T?` used as a `T` (6), a `MutableList` handed to another task (12),
  a C call outside `unsafe` (13);
- an import cycle (11), a private name used from another module (11);
- test code called from a program, and a test's vocabulary used outside
  a test (this chapter).

The remaining conventions are few: `camelCase` for functions and values,
`CapitalCase` for types, two-space indentation, and a `veles.toml` at the
root of anything you intend to publish.

That is the end of the tutorial track. The [cheat sheet](reference/cheatsheet.md)
condenses it to one page.
