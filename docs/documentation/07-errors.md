# 7. Errors: `throws`, `throw`, `try`

Veles has no exceptions in the Java sense and no `Result` boilerplate in
the Rust sense. It has one mechanism that reads like the first and costs
like the second (D4): a function that can fail says `throws`, fails with
`throw`, and callers either handle the failure or pass it on with `try`.
Under the hood the function returns a `Result<T, E>` — an error is a
value in a register, not a stack unwind — but you rarely write that
type.

## Declaring failure

```veles
use io

error ParseError { text: string }
error RangeError { value: i64 }

fun parsePort(text: string): i64 throws ParseError | RangeError {
  val n = text.toInt() ?: throw ParseError(text)
  if (n < 0 || n > 65535) throw RangeError(value: n)
  n
}

fun main() {
  loop (t in ["80", "x", "70000"]) {
    when (val port = parsePort(t)) {
      is Ok  => io.println("$t -> $port")
      is Err => io.println("$t -> failed: $port")
    }
  }
}
```

Output:
```text
80 -> 80
x -> failed: ParseError(text: x)
70000 -> failed: RangeError(value: 70000)
```

- `error Name { ... }` declares an error: a struct that implements the
  prelude trait `Error`. Only errors can be thrown — `throw Dog()` on a
  plain struct is a compile error. Fields, methods and construction work
  exactly as for a struct.
- `throws A | B` lists what can go wrong. `throw e` leaves the function
  with that error; it works anywhere an expression can go, including on
  the right of `?:`.
- The **caller sees a `Result`**. `when (val port = parsePort(t))` gives
  it a name; `is Ok` / `is Err` tell the arms apart, and the compiler
  checks both are there.
- An `is Ok` / `is Err` test **smart-casts the value itself**: in the
  `is Ok` arm `port` is the `i64` inside, in the `is Err` arm it is the
  error — the same rule that turns a `T?` into a `T` after `!= null`. In
  an `if`, `r.ok` and `r.err` are the same tests spelled as properties:
  after `if (r.ok)` the name `r` is the value, and in the `else` branch
  the error. `Ok` and `Err` are handled, never held; `is Ok(v)` with a
  fresh name still works when you want one.

You cannot silently drop a failure. Calling `parsePort(t)` as a bare
statement is a compile error ("unused Result"), and so is binding the
result to a name you never read (`val r = parsePort(t)` with no `when`
on `r`) — either match on it or propagate it. This holds whether or not
the caller is `throws`: only `try` propagates. `val _ = parsePort(t)` is
the explicit way to say you really do not care.

## Propagating with `try`

Inside another `throws` function, `try expr` unwraps a success and
returns the error to *your* caller otherwise:

```veles
use io

error ParseError { text: string }

fun parseAll(items: List<string>): List<i64> throws ParseError {
  val out: MutableList<i64> = []
  loop (it in items) {
    val n = it.toInt() ?: throw ParseError(text: it)
    out.push(n)
  }
  out.toList()
}

fun total(items: List<string>): i64 throws ParseError {
  val numbers = try parseAll(items)
  numbers.fold(0, (a, b) => a + b)
}

fun main() {
  val r = total(["1", "2", "3"])
  if (r.ok) io.println("total $r") else io.println("bad input ${r.text}")
  when (val r = total(["1", "two"])) {
    is Ok  => io.println("total $r")
    is Err => io.println("bad input ${r.text}")
  }
}
```

Output:
```text
total 6
bad input two
```

`try` is only allowed in a function declared `throws`; using it
elsewhere is an error that tells you exactly that.

One `try` covers every call in the chain after it that can fail (D134):
in `try client.fetch(url).json<User>()` both `fetch` and `json` may
throw, each failure propagates where it happens, and the function's
error type is the union of both. A method that belongs to the `Result`
itself applies to the `Result` — `try parse(s).mapError(...)`,
`try parse(s) ?! e`. Arguments are not part of the chain: in
`try f(g()).h()`, `g()`'s `Result` is passed to `f` as a value. The old
spelling `try (try f()).g()` still compiles, with a warning that the
inner `try` is redundant and a fix that removes it.

## Letting the compiler work out the error type

Write `throws` with no type and the compiler infers the union of
everything the body can throw, including through `try` (D45):

```veles
use io

error NotFound { key: string }
error Invalid { reason: string }

fun lookup(key: string): string throws NotFound =
  if (key == "name") "veles" else throw NotFound(key)

fun validate(v: string): string throws Invalid =
  if (v.len() > 3) v else throw Invalid(reason: "too short")

// inferred: throws NotFound | Invalid
fun settingFor(key: string): string throws = try validate(try lookup(key))

fun main() {
  loop (k in ["name", "age"]) {
    when (val v = settingFor(k)) {
      is Ok  => io.println("$k = $v")
      is Err => when (v) {
        is NotFound(key) => io.println("no such key '$key'")
        is Invalid(reason) => io.println("invalid: $reason")
      }
    }
  }
}
```

Output:
```text
name = veles
no such key 'age'
```

Hover over `settingFor` in the editor and you will see the inferred
union. Inference is per function body; a **trait method** must declare
its error type explicitly (D40), because callers of the trait cannot see
any particular body.

A `main() throws` is allowed: an error that escapes it ends the program
with the error's `message()` printed and a non-zero exit code.

The other direction is checked too: a function written `throws` whose
body cannot fail — no `throw`, no `try` of anything that can — gets a
warning, because the clause makes every caller pay for a `try` that never
propagates. `veles check --fix` removes the clause and then the callers'
`try`. Public functions keep theirs: a library may declare `throws` today
so that adding a failure later is not a breaking change.

## Every error has a `message()`

You do not have to know an error's fields to report it. An `error`
declaration is a struct plus an implement of the prelude trait `Error`:

```veles
// fragment
public trait Error {
  fun message(): string = "$this"     // default: the value as `show` renders it
}
```

The default is inherited; write `fun message()` inside the
`error` body to replace the text. And because every member of an error
union implements `Error`, you can call `message()` on the union itself,
without matching first:

```veles
use io

error NotFound { key: string }
error Invalid {
  reason: string
  fun message(): string = "invalid: ${this.reason}"
}

fun lookup(key: string): string throws NotFound =
  if (key == "name") "veles" else throw NotFound(key)

fun validate(v: string): string throws Invalid =
  if (v.len() > 3) v else throw Invalid(reason: "too short")

fun settingFor(key: string): string throws = try validate(try lookup(key))

fun main() {
  loop (k in ["name", "age"]) {
    val r = settingFor(k)
    if (r is Ok) io.println("$k = $r")
    else io.println("$k: ${r.message()}")       // r: NotFound | Invalid
  }
}
```

Output:
```text
name = veles
age: NotFound(key: age)
```

Only errors can be thrown. A struct you cannot change (from another
package, say) becomes one with an explicit `implement Error for That { }`;
anything else — a `string`, a number — is rejected, so wrap it. A generic
`throws X` needs the bound `X: Error`. Note that `error` is a keyword
only at the start of a declaration; `is Err(error) => ...` still works.

If all you want is a per-throw text, declare a `message: string` field:
it becomes the message automatically, so `error Failed { message: string }`
is thrown as `Failed(message: "disk full")`. Any other method in an
`error` body is an ordinary method.

## Naming an error set, and wrapping a cause

A long `throws A | B | C` repeats itself across a module. Name it:

```veles
use io

error NotFound { key: string }
error Invalid { reason: string; fun message(): string = "invalid: ${this.reason}" }
error Timeout

error LookupErrors = NotFound | Invalid | Timeout

// context around a cause: an error's field may hold an error set
error ConfigError {
  file: string
  cause: LookupErrors
  fun message(): string = "${this.file}: ${this.cause.message()}"
}

fun setting(key: string): string throws LookupErrors =
  when (key) {
    "name" => "veles"
    "slow" => throw Timeout()
    "bad" => throw Invalid(reason: "too short")
    else => throw NotFound(key)
  }

fun load(file: string, key: string): string throws ConfigError {
  val r = setting(key)
  if (r is Err) throw ConfigError(file, cause: r)
  r
}

fun main() {
  loop (k in ["name", "slow", "age"]) {
    val r = load("app.toml", k)
    if (r is Ok) io.println("$k = $r")
    else when (r.cause) {
      is Timeout => io.println("$k: try again later")
      else => io.println(r.message())
    }
  }
}
```

Output:
```text
name = veles
slow: try again later
app.toml: NotFound(key: age)
```

`error LookupErrors = ...` is a name for the union, nothing more: it is
transparent (`throws LookupErrors | IoError` flattens), `when` matches its
members, and it can appear only where a union can — after `throws`, in
another set, or as the type of an `error`'s field. It is not a value type
you can bind or construct. In the editor, hovering a set — or a function
that `throws` one — spells out its members, each a link to its declaration.

## Errors are values

A `Result` can be stored and passed around like anything else. The type
is `Result<T, E>` with variants `Ok(value)` and `Err(error)`:

```veles
use io

error Oops { code: i64 }

fun risky(n: i64): i64 throws Oops = if (n > 0) n * 10 else throw Oops(code: n)

fun main() {
  val results = [risky(1), risky(0), risky(2)]      // List<Result<i64, Oops>>
  io.println("${results.oks()} ${results.errors()}")
  loop (e in results.errors()) io.println("failed with code ${e.code}")
  val [first, second, _] = results else panic("three calls, three results")
  io.println("${first.getOrNull()} ${second.getOrDefault(-1)} ${first.errorOrNull()}")
  val explicit: Result<i64, Oops> = Err(Oops(code: -1))
  io.println("$explicit")
}
```

Output:
```text
[10, 20] [Oops(code: 0)]
failed with code 0
10 -1 null
Err(error: Oops(code: -1))
```

A single result unwraps with `getOrNull()`, `getOrDefault(d)` and
`errorOrNull()`; a list of results splits with `oks()` and `errors()`, which
is how a batch that must not stop at the first failure reports and then
continues (`examples/dedup`). Note what does *not* work: narrowing does not
survive a call boundary, so `results.filter(r => r is Ok).map(r => ...)`
still sees `Result` inside the `map` — use `oks()`, or `mapNotNull(r => r.getOrNull())`.

### Changing the error on the way up: `?!` and `mapError`

An error from one layer is often the wrong error for the next: a
`ParseError` deep inside a request handler should reach the client as
"400 bad request", a missing map key should become "no such user". Two
tools, both leaving `try` as the one place where propagation happens:

- `x ?! e` — "or fail with `e`". A `T?` or a `Result<T, E1>` becomes a
  `Result<T, E2>` that fails with `e`; the right side is evaluated only on
  the failure path, and `try x ?! e` is read as `try (x ?! e)`.
- `r.mapError(f)` — the same with the old error in hand, when the new one
  should mention it.

```veles
use io

error NotFound { id: string; fun message(): string = "no user ${this.id}" }
error BadRequest { detail: string }
error ParseError { at: i64 }

fun parseAge(s: string): i64 throws ParseError =
  s.toInt() ?: throw ParseError(at: 0)

fun age(users: Map<string, string>, id: string): i64 throws NotFound | BadRequest {
  val text = try users.get(id) ?! NotFound(id)                       // absence → NotFound
  try parseAge(text).mapError(e => BadRequest(detail: "age at ${e.at}"))   // ParseError → BadRequest
}

fun main() {
  val users = ["ann": "41", "bob": "x"]
  loop (id in ["ann", "bob", "cid"]) {
    when (age(users, id)) {
      is Ok(n)  => io.println("$id is $n")
      is Err(e) => io.println("$id: ${e.message()}")
    }
  }
  val kept: Result<i64, BadRequest> = parseAge("z") ?! BadRequest(detail: "not a number")
  io.println("$kept")
}
```

Output:
```text
ann is 41
bob: BadRequest(detail: age at 0)
cid: no user cid
Err(error: BadRequest(detail: not a number))
```

`?!` next to `?:`: both read "the value, or…"; `?:` supplies a fallback
value and the expression is a `T`, `?!` supplies a failure and the
expression is a `Result` for `try` to propagate. The right side of `?!`
must be an error type — a value, never a thrown one; `?: throw e` remains
the spelling for "fail right here, not as a Result".

### Functions that never return

`os.exit` and `panic` never come back; their type is `Never`, the type with no
values, and a function of your own can declare it:

```veles
// fragment
fun usage(): Never {
  io.println("usage: tool <dir>")
  os.exit(2)
}
```

A `Never` value fits anywhere (`val n = xs.first() ?: usage()` is an `i64`), a
`when` whose arms all end in one is itself `Never`, and code after such a
statement is reported as unreachable.

## Falling back and bailing out: `??`, `val ... else` and `catch`

`try` passes a failure up. Often the right answer is closer: a default,
a value worked out from the error, or leaving the function (or the loop
iteration) right here. Three forms say that in one line.

`r ?? fallback` is the value of a `Result`, or the fallback when it is an
`Err` — what `?:` is to a nullable. The fallback may be a value or something
that leaves (`return`, `continue`, `throw`); it does not see the error.

`val x = r else ...` binds the value and runs the `else` otherwise — and
the `else` **must leave**, since `x` does not exist on that path. It works
on a `Result`, on a nullable, and on a pattern:
`val Circle(radius) = shape else return 0.0`. It does not see the error
either.

`r catch (e) { ... }` is the one that does: the value of a `Result`, or what
the block yields when it is an `Err`, with the error bound to `e` (D98). The
block yields the same type or leaves (`return`, `continue`, `throw`), and
binds nothing when written `catch { ... }`.

```veles
use io

error Invalid { line: string }

fun number(line: string): i64 throws Invalid = line.trim().toInt() ?: throw Invalid(line)

fun total(lines: List<string>): i64 {
  var sum = 0
  loop (line in lines) {
    val n = number(line) else continue             // skip what is not a number
    sum += n
  }
  sum
}

fun firstOrReport(lines: List<string>): string {
  val first = lines.first() else return "empty"    // a nullable
  val n = number(first) catch (e) {                // a Result, with its error
    return "not a number: '${e.line}'"
  }
  "first is $n"
}

fun main() {
  val lines = ["3", "x", " 4 "]
  io.println("${total(lines)}")
  io.println(firstOrReport(lines))
  io.println(firstOrReport(["y"]))
  io.println(firstOrReport([]))
  io.println("${number("x") ?? 0} ${number("x") catch (e) { -e.line.len() }}")
}
```

Output:
```text
7
first is 3
not a number: 'y'
empty
0 -1
```

The two operators are split by what is on their left: `?:` for a
nullable, `??` for a `Result`. Writing the other one is an error whose
quick fix swaps it, so the operator always says which kind of "maybe" is
being unwrapped. (D61)

## Several calls, one handler: `do { } catch (e) { }`

`catch` after a call answers for that call. When several calls in a row
should fail into the same place and the function should carry on afterwards,
put them in a `do` block and say what happens in its `catch` (D98). One call
followed by methods takes `try` and `catch` around the whole chain:
`try parse(s).len() catch (e) { -1 }` — `try` marks the call that can fail,
and `catch` says where the failure goes, with no parentheses needed:

```veles
use io

error Invalid { line: string }
error Negative { value: i64 }

fun number(line: string): i64 throws Invalid = line.trim().toInt() ?: throw Invalid(line)

fun positive(n: i64): i64 throws Negative {
  if (n < 0) throw Negative(value: n)
  n
}

// one handler for the calls in the block; this function does not throw
fun describe(a: string, b: string): string {
  do {
    val x = try positive(try number(a))
    val y = try positive(try number(b))
    "sum ${x + y}"
  } catch (e) {
    "cannot add: ${e.message()}"
  }
}

// in a loop, `continue` and `break` are the loop's own
fun total(lines: List<string>): i64 {
  var sum = 0
  loop (line in lines) {
    do {
      sum += try positive(try number(line))
      if (sum > 100) break
    } catch (e) {
      io.println("skipped '${line.trim()}'")
      continue
    }
  }
  sum
}

fun main() {
  io.println(describe("2", "3"))
  io.println(describe("2", "x"))
  io.println(describe("2", "-4"))
  io.println("${total(["5", "x", "-1", "7", "500", "9"])}")
}
```

Output:
```text
sum 5
cannot add: Invalid(line: x)
cannot add: Negative(value: -4)
skipped 'x'
skipped '-1'
512
```

- **The block is an expression.** Its value is the block's last expression,
  or what the handler yields; the handler yields the same type or leaves
  (`return`, `break`, `continue`, `throw`), exactly like a `??` handler.
- **`try` stays.** A failed `try` — or a `throw` — anywhere in the block goes
  to the handler; a call that can fail and is not marked with `try` is still an
  error, so every place a failure can start is visible.
- **`e` is the union of the errors the block can raise**, so `e.message()`
  works and `when (e) { is Invalid => ... is Negative => ... }` tells them
  apart. The function around it does not have to be `throws`: the errors the
  handler catches are not its own.
- **It is a block, not a lambda.** `return` leaves the function, `break` and
  `continue` the loop around the `do`, and a call that suspends needs nothing
  declared. That is what the immediately called closure `(() => { ... })()`,
  which does the same job, cannot do.
- **The handler is outside the block.** A `try` or `throw` in it is the
  function's (or the next `do` out): `catch (e) { throw Wrapped(cause: e) }`
  makes the function `throws Wrapped`.
- **What it does not catch.** A panic still ends the task (D52). A `scope`
  that launches a child which can fail passes the error to the function, past
  the handler, so that is refused inside a `do`: put the scope in its own
  function and `try` it. A block with nothing in it that can fail is an
  error, since the `catch` could never run.

Use `??`, `val ... else` or a plain `catch` when one call needs its own answer, and `do` when
several share one.

## When to throw, when to return `T?`

- **Nothing to return** but the situation is normal (key absent, list
  empty): return `T?`. The caller needs a fallback, not a diagnosis.
- **Something went wrong** and the caller might need to know *what*
  (bad input, I/O failure): `throws`.
- **The program is broken** (index out of range, arithmetic overflow, a
  case you believed impossible): a panic. Panics are not caught by
  `try`; they stop the current task ([chapter 12](12-concurrency.md)).
  A panic prints its message and where it happened, relative to the
  package root. Had the list pattern in the `Result` example above not
  matched, the program would have stopped with:

  ```text
  panic: three calls, three results
    at main.vs:11:41
  ```

  There is no shorthand like Kotlin's `!!`: a read that cannot fail says
  why with `?: panic("…")`, and the location comes for free (D64).

  A debug build (`veles run`, `veles test`) adds the calls that led there,
  innermost first, so the report reads like a stack trace (D81):

  ```text
  panic: index 7 is out of range
    at main.vs:4:24 in parse
    called from main.vs:9:12 in load
    called from main.vs:13:19 in main
  ```

  The chain is complete through functions that suspend; a task started
  with `async` begins its own chain at the function it runs. A release
  build (`--release`) keeps no chain — that is what makes calls free — and
  prints the location with a note that a debug build shows the rest.

  A panic that says you called something wrongly — `xs.swap(0, 7)` on a
  three-element list, `list.chunked(0)`, `255.toString(radix: 1)` — points at
  **your** call, not at the line inside the standard library that noticed
  (D88). Those functions are marked `@caller_location`, which hands them
  the site they were called from; both build profiles report it:

  ```text
  panic: swap: index 7 out of bounds for list of length 3
    at main.vs:6:3 in main
  ```

  A recursion that never ends is a panic too, not a silent death (D92). Every
  thread that runs Veles code — the one that starts `main` and each task
  worker — has a stack of 256 MB, reserved when the thread starts and
  committed only as the calls reach it, so a million frames are unremarkable.
  When it is used up anyway the process prints and exits with the panic exit
  code, 101:

  ```text
  panic: stack overflow
    the stack is 256 MB; VELES_STACK=<megabytes> sets it
    in depth
    called from main.vs:5:11 in depth
    called from main.vs:5:11 in depth
    called from main.vs:5:11 in depth
    (2795412 calls deep; the chain keeps the first 4096)
  ```

  The chain is a debug build's; a release build says so instead, as above.
  `VELES_STACK=64` runs with 64 MB (1 to 4096); a value that is not a number
  is reported and ignored. A stack overflow cannot be caught and does not
  unwind, since there is no stack left to unwind with — a walk over input
  someone else wrote should still bound its own depth
  (`recursion.Depth`, stdlib reference), so that the user gets an error message
  rather than a panic.

Next: [Traits and generics](08-traits-and-generics.md).
