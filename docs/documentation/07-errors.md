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
  val n = text.toInt() ?: throw ParseError(text: text)
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
  numbers.fold(0 as i64, (a, b) => a + b)
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

`try` is a prefix that covers the whole expression after it, so
`try fs.readFile(p).lines()` asks for `lines()` on the `Result` and is refused;
unwrap first with `(try fs.readFile(p)).lines()`, or bind the value on its
own line.

## Letting the compiler work out the error type

Write `throws` with no type and the compiler infers the union of
everything the body can throw, including through `try` (D45):

```veles
use io

error NotFound { key: string }
error Invalid { reason: string }

fun lookup(key: string): string throws NotFound =
  if (key == "name") "veles" else throw NotFound(key: key)

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

## Every error has a `message()`

You do not have to know an error's fields to report it. An `error`
declaration is a struct plus an impl of the prelude trait `Error`:

```veles
// fragment
pub trait Error {
  fun message(): string = "$self"     // default: the value as `show` renders it
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
  fun message(): string = "invalid: ${self.reason}"
}

fun lookup(key: string): string throws NotFound =
  if (key == "name") "veles" else throw NotFound(key: key)

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
package, say) becomes one with an explicit `impl Error for That { }`;
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
error Invalid { reason: string; fun message(): string = "invalid: ${self.reason}" }
error Timeout

error LookupErrors = NotFound | Invalid | Timeout

// context around a cause: an error's field may hold an error set
error ConfigError {
  file: string
  cause: LookupErrors
  fun message(): string = "${self.file}: ${self.cause.message()}"
}

fun setting(key: string): string throws LookupErrors =
  when (key) {
    "name" => "veles"
    "slow" => throw Timeout()
    "bad" => throw Invalid(reason: "too short")
    else => throw NotFound(key: key)
  }

fun load(file: string, key: string): string throws ConfigError {
  val r = setting(key)
  if (r is Err) throw ConfigError(file: file, cause: r)
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
  val successes = results.filter(r => r.ok)
  io.println("${successes.len()} of ${results.len()} succeeded")
  val explicit: Result<i64, Oops> = Err(Oops(code: -1))
  io.println("$explicit")
}
```

Output:
```text
2 of 3 succeeded
Err(error: Oops(code: -1))
```

## When to throw, when to return `T?`

- **Nothing to return** but the situation is normal (key absent, list
  empty): return `T?`. The caller needs a fallback, not a diagnosis.
- **Something went wrong** and the caller might need to know *what*
  (bad input, I/O failure): `throws`.
- **The program is broken** (index out of range, arithmetic overflow, a
  case you believed impossible): a panic. Panics are not caught by
  `try`; they stop the current task ([chapter 12](12-concurrency.md)).

Next: [Traits and generics](08-traits-and-generics.md).
