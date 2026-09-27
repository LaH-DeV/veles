| `@key("k")` / `@key(json: "k", db: "c")` | a field, an enum member, a sealed variant | its name on the wire, for every format or by format ([chapter 18](18-codable-and-json.md)) |
| `@skip` / `@skip(json)` | a field with a default | left out of the wire form, everywhere or in one format |
| `@required` | a nullable field | the key must be present even though the value may be null |
| `@tag("kind")` / `@tag("type", content: "value")` | a sealed trait | the key that names the variant; with `content`, the key the fields go under |
# 14. Attributes and the test runner

An attribute is `@name` or `@name(args)` on the line before a
declaration (D51). The bootstrap compiler knows five; an unknown one is
an error, not a silent no-op.

| Attribute | On | Effect |
|---|---|---|
| `@test` | a function with no parameters | included and run by `veles test`; excluded from `veles build`/`run` |
| `@deprecated("why")` | any declaration | a warning at every use, with the message |
| `@mustUse` | a function returning a value | an error if a call's result is discarded |
| `@inline` / `@noinline` | a function | a hint to the optimiser (`--release`) |

```veles
use io

@deprecated("use greet")
fun hello(name: string): string = "hello $name"

fun greet(name: string): string = "hi $name"

@mustUse
fun important(): i64 = 7

@inline
fun tiny(x: i64): i64 = x + 1

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

Tests are ordinary functions marked `@test`, living in the module they
test. A test **passes when it returns** and **fails when it throws or
panics**. Since a `throws` function can fail with any struct, an
assertion is just a function that throws a descriptive one:

```veles
use io

error Expected { what: string, expected: string, actual: string }

fun expectEq<T>(what: string, expected: T, actual: T) throws Expected {
  if (expected != actual) throw Expected(what, expected: "$expected", actual: "$actual")
}

fun wordCount(text: string): i64 {
  var n = 0
  var inWord = false
  loop (i in 0..<text.len()) {
    val c = text.substring(i, i + 1) ?: ""
    if (c == " ") {
      inWord = false
    } else if (!inWord) {
      inWord = true
      n += 1
    }
  }
  n
}

@test
fun countsWords() throws Expected {
  try expectEq("simple", 3, wordCount("one two three"))
  try expectEq("extra spaces", 2, wordCount("  a   b "))
  try expectEq("empty", 0, wordCount(""))
}

@test
fun tasksWorkInTests() throws Expected {
  val ch = Channel<i64>(capacity: 1)
  scope {
    ch.send(41)
    val v = await ch.recv()
    try expectEq("channel", 42, (v ?: 0) + 1)
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

`veles test <dir>` on that module prints one line per test:

```text
test countsWords ... ok
test tasksWorkInTests ... ok

2 passed, 0 failed
```

A failing assertion prints the thrown value — `FAILED:
Expected(what: simple, expected: 3, actual: 2)` — the summary line names
every test that failed, and the exit code is non-zero. Each test runs as
its own task on the executor, so tests may use `scope`, `async`, channels
and `sleep` freely, and a panic in one test does not stop the others.

Two flags shape a run:

- `--filter text` runs only the tests whose name contains `text`
  (`veles test . --filter Words`); the summary counts the rest as
  filtered out. A filter that matches nothing is an error, so a typo in
  CI fails instead of passing with no tests run.
- `--timeout 30s` bounds each test (default `10m`, `0` for no bound).
  A test still running at its deadline is reported —
  `FAILED: timed out after 30s` — and ends the run, since a task busy in a
  loop cannot be stopped from outside; the report says how many tests
  after it did not run.

Generic assertion helpers like `expectEq<T>` work for any `T` that
supports `==` and interpolation — which is every struct, number,
string, tuple and list.

## Conventions the compiler enforces so you need not

Looking back over the tutorials, a good deal of what a style guide would
say is already a compile error in Veles:

- an unused `Result` (chapter 7), a `when` missing a variant (9), a
  `val` that is assigned (2), a bare field assigned (5);
- a `T?` used as a `T` (6), a `MutableList` handed to another task (12),
  a C call outside `unsafe` (13);
- an import cycle (11), a private name used from another module (11).

The remaining conventions are few: `camelCase` for functions and values,
`CapitalCase` for types, two-space indentation, and a `veles.toml` at the
root of anything you intend to publish.

That is the end of the tutorial track. The [cheat sheet](reference/cheatsheet.md)
condenses it to one page.
