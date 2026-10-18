# 14. Attributes, annotations and tests

## Attributes

An attribute is `@name` or `@name(args)` on the line before a
declaration (D51). The compiler knows each lowercase one; an unknown
lowercase name is an error, not a silent no-op. (A capitalised `@Name` is an
annotation a library defined — [below](#annotations-of-your-own).)

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

## Annotations of your own

The `@names` above belong to the compiler: they are lowercase and reserved, and
each changes how code is compiled or checked. A library can also define
**annotations** (D174), and they are the opposite kind of thing: typed data on
a declaration that **changes nothing by itself**. A library reads it, while the
program is compiled, to write the code or the table you would otherwise write
by hand — a validation rule, a flag's short name, a column's name. An annotation
does not wrap a function or alter what it does, with one declared exception ([decorators](#decorators)), and
nothing is looked up when the program runs.

An annotation is declared like a function's header, followed by where it may
stand and, when a trait reads it, which one:

```veles
// fragment
/// The smallest value a number field may hold.
annotation Min(value: i64) on field for Validate

annotation Hidden on field for Validate                  // no parameters: `@Hidden`
annotation Route(path: string, method: string = "GET") on fun
annotation Key(name: string) on field, enumMember, variant
```

- **The name is a type's** — `Min`, not `min` — so it never clashes with the
  compiler's own lowercase ones. It is imported like any name
  (`use validate { Min }`) or written through its module, `@validate.Min(18)`.
- **The parameters are the arguments.** `@Min(18)` constructs a `Min`, with the
  same rules as a struct: named arguments, defaults, and one bare argument when
  there is exactly one parameter. The arguments are constants.
- **`on` lists the places**: `field`, `struct`, `fun`, `trait`, `enum`,
  `enumMember` and `variant`. An annotation anywhere else is an error that
  names the places.
- **`for` names the readers.** Put an annotation on a type that does not
  implement any of them and the compiler warns: nothing would read it.

### Reading them

A library reads the annotations of a type through `T.fields`, a constant list
of [`FieldInfo`](reference/stdlib.md#reflection-d174) — the name, doc comment,
type, default and annotations of each field, known while the program is
compiled — and through `T.annotated<A>()`, the fields that carry `A` together
with the annotation's value. Both are constants, and there are two ways to use
them. A library typically uses both.

A **`const fun`** reads them as data and builds a table the compiler stores in
the program: nothing is computed when the program starts.

An **`inline loop`** is unrolled once per field, so each copy knows the field's
own type. `f.get(x)` reads the field (`f.get(this)` in a trait's default body:
the type said `implement` and so opted in, whatever the field's visibility),
`f.Type` is its type, and `T.build(f => …)` makes a `T` by giving a value for
each field. `inline loop ((f, a) in T.annotated<A>())` visits only the fields
that carry `A`, with `a` the annotation, which is how one body can compare a
number field with `<` and measure a text field with `len()`.

```veles
use io

annotation Min(value: i64) on field for Validate
annotation MaxLen(value: i64) on field for Validate

trait Validate {
  fun problems(): List<string> {
    val out: MutableList<string> = []
    inline loop ((f, min) in This.annotated<Min>()) {
      if (f.get(this) < min.value) out.push("${f.name} must be at least ${min.value}")
    }
    inline loop ((f, most) in This.annotated<MaxLen>()) {
      if (f.get(this).len() > most.value) out.push("${f.name} is longer than ${most.value}")
    }
    out.toList()
  }

  static const fun rules(): List<string> {
    val out: MutableList<string> = []
    loop ((f, min) in This.annotated<Min>()) out.push("${f.name} >= ${min.value}")
    loop ((f, most) in This.annotated<MaxLen>()) out.push("${f.name} <= ${most.value} bytes")
    out.toList()
  }
}

struct Signup {
  @MaxLen(5)
  name: string
  @Min(18)
  age: i64
  note: string = ""
  implement Validate
}

const RULES: List<string> = Signup.rules()

fun main() {
  io.println("${RULES}")
  io.println("${Signup(name: "Alexander", age: 12).problems()}")
  io.println("${Signup(name: "Al", age: 30).problems()}")
}
```

Output:
```text
[age >= 18, name <= 5 bytes]
[age must be at least 18, name is longer than 5]
[]
```

When one field needs several questions, walk all the fields and ask each one
(`inline loop (f in This.fields)`): `f.has<A>()` and `f.attr<A>()` are
constants there, `if (f.has<A>())` keeps only the branch that applies, and
`if (val a = f.attr<A>())` binds the annotation. `examples/annotations` does
this for a `Skip` annotation.

The rules the compiler holds you to:

- `f.attr<A>()` and `f.has<A>()` exist while the program is compiled. In an
  `inline loop` they are constants; on a `FieldInfo` value they work in a
  `const fun` only. A function that reads annotations, called from code that
  runs when the program does, is an error that says to call it in a constant
  (`const RULES = Signup.rules()`).
- An `inline loop` walks `T.fields` or `T.annotated<A>()` and nothing else, and
  has no `break`, `continue` or label: it is not iterating, it is being written
  out. A generic body that has one is checked once for each type it is used with.
- Reading a field with `f.get(x)` follows the usual visibility, except in a
  trait's default body for the type that implemented the trait — so a library
  cannot read what a type keeps private unless the type asked it to.
- `This` is a receiver in a trait's default body: `This.fields`,
  `This.annotated<A>()`, `This.build(…)`, `This.default()`.
- Annotations on enum members, variants and functions can be declared and used
  but not yet read ([checklist §11](../../veles-checklist.md)).

`examples/annotations` is a small validation library used from a program.

### Decorators

One kind of annotation does change what a function does, and says so in its
declaration (D176): it names an ordinary generic function that **wraps** the
functions it stands on. The function's body becomes a lambda handed to the
wrapper, and the wrapper's result is the function's result:

```veles
use io { println }

error Flaky {
  n: i64
}

annotation Retry(times: i64, delay: Duration = Duration.zero) on fun = retryCall
annotation Logged(label: string) on fun = logged

fun retryCall<R, E>(a: Retry, body: fun(): R suspends throws E): R suspends throws E =>
  try retry(a.times, () => try body(), delay: a.delay)

fun logged<A, R>(a: Logged, args: A, body: fun(A): R): R {
  println("[${a.label}] called with $args")
  body(args)
}

val attempts = Atomic(value: 0)

@Retry(times: 3)
fun flaky(): string throws Flaky {
  val n = attempts.add(1)
  if (n < 3) throw Flaky(n)
  "succeeded on attempt $n"
}

@Logged(label: "add")
fun add(a: i64, b: i64): i64 => a + b

fun main() {
  println(flaky() catch (e) { "gave up" })
  println("${add(2, 3)}")
}
```

Output:
```text
succeeded on attempt 3
[add] called with (2, 3)
5
```

- **There is no macro.** `flaky()` means `retryCall(Retry(times: 3), () => <the body>)`.
  The wrapper is a function anyone can read, call and test, and `throws` and
  suspension flow through it by inference: a function wrapped by a retrying
  wrapper suspends, and its errors are the wrapper's.
- **Two shapes of wrapper**, told apart by the number of parameters:
  `(a: Name, body: fun(): R)` sees the annotation and the body;
  `(a: Name, args: A, body: fun(A): R)` also receives the function's arguments,
  as a tuple (the parameter itself for one, `()` for none), for a log line or a
  cache key. The first parameter is the annotation.
- **Several decorators nest**, the first written outermost, as in Python.
- **Where**: any function, method or trait default with a body. The annotation
  is declared `on fun` and nothing else; its arguments are built on each call, so
  they need not be constants. State, such as a cache, lives in a module-level
  `Mutex`, `Atomic` or `Lazy`.
- `examples/decorators` has a retry, a logger and a decorator pair.

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

A test or a suite may also carry a **label**, an identifier written before
the sentence: `test listMethods "incrementAt updates an element" { }`,
`suite Lexer "tokenising" { }`. Either part may stand alone (`test
listMethods { }`). The report and `--filter` use `label: sentence`, so a
filter matches both; a label, like a sentence, is unique within its suite.

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
