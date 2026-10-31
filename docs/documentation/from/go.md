# From Go

Much of what Go programmers value carries over: a garbage collector, a
module per directory, one formatter (`veles fmt`), no implicit numeric
conversions, strings that are UTF-8 bytes, channels and cheap tasks, and a
short list of features. What changes is mostly what the compiler checks
for you: errors you cannot drop, absence you cannot forget, tasks that
cannot outlive their caller, and data races refused before the program
runs.

## At a glance

| Go | Veles |
|---|---|
| `x := 1`, `var x int64` | `val x = 1` (never reassigned), `var x = 1` |
| `func add(a, b int64) int64` | `fun add(a: i64, b: i64): i64` |
| `(T, error)` and `if err != nil { return err }` | `fun f(): T throws E` and `try f()` |
| `v, ok := m[k]` | `m.get(k)` is a `V?`: `m.get(k) ?: fallback` |
| `nil` pointers, zero values | `T?` only where a value may be missing; `?.`, `?:` |
| `type S struct{…}`, methods declared apart | `struct S { … }` with its methods inside, receiver `this` |
| interfaces satisfied implicitly | `implement Trait` written out |
| `switch v := x.(type)` | `when (x) { is A => …; is B => … }`, exhaustive over a `sealed trait` |
| `go f()` | `async f()`, inside a block that waits for it |
| `defer f.Close()` | `with f = try fs.open(p)` |
| `[]T` | `List<T>` (never changes) or `MutableList<T>` |
| `len(s)`, `s[i]` on a string | `s.len()` (bytes, as in Go), `s.byteAt(i)` |
| exported by a capital letter | `public` (other packages), nothing (this package), `internal`, `private` |
| `for` | `loop (x in xs)`, `loop (cond)`, `loop { }`; ranges `0..<n` |

## Errors are thrown, and the compiler keeps track

A Go function returns `(T, error)` and every caller decides by hand what to
do with the error. A Veles function says `throws E`, and its caller either
passes the error on with `try` or handles it; there is no third choice,
because an unused result is a compile error. The error type is part of the
signature, so the compiler knows exactly what can arrive, and `when` over
the result needs no default arm ([chapter 7](../07-errors.md)).

```veles
use io

error BadPort { text: string }

fun parsePort(text: string): i64 throws BadPort {
  val n = text.toInt() ?: throw BadPort(text)
  if (n < 1 || n > 65535) throw BadPort(text)
  n
}

fun ports(texts: List<string>): List<i64> throws BadPort => try texts.map(t => try parsePort(t))

fun main() {
  io.println("${ports(["80", "443"]) ?? []}")
  when (val r = ports(["80", "http"])) {
    is Ok => io.println("$r")
    is Err => io.println("bad port: ${r.text}")
  }
}
```

Output:

```text
[80, 443]
bad port: http
```

`try` inside the lambda passes the error out of `map`, and the `try` in
front of `map` passes it out of `ports`: no wrapping, no `if` per call.
`?? []` is "the value, or this if it failed".

## A missing value is a type, not a zero

In Go, `ages["cy"]` is `0` whether Cy is missing or zero years old, a
struct you forgot to fill has zero fields, and a `nil` pointer waits for
the first dereference. In Veles a value that may be missing has a type
that says so, `T?`, and the compiler makes you decide what happens when
it is: `m.get(k)` is a `V?`, a constructor needs every field that has no
default, and a pointer cannot be null unless it is a `(*T)?`
([chapter 6](../06-nullable-types.md)).

```veles
use io

fun main() {
  val ages = ["ann": 31, "bob": 0]
  io.println("${ages.get("bob") ?: -1} ${ages.get("cy") ?: -1}")
}
```

Output:

```text
0 -1
```

## Tasks finish where they started

`go f()` returns at once, and nothing ties the goroutine to its caller: it
may outlive it, leak, or fail with nobody listening, which is why Go code
carries `sync.WaitGroup`, `errgroup` and a `context.Context` parameter
through every call. In Veles a task belongs to the block that started it,
and the block does not end before its tasks do. Under `awaitAll` or a
`scope`, the first failure cancels the other tasks
(`errgroup.WithContext`, built in), and cancellation needs
no parameter: a cancelled task stops at its next suspension point
([chapter 12](../12-concurrency.md)).

```veles
use io

fun fetch(id: i64): string {
  await sleep(Duration.millis(10 * (4 - id)))
  "page$id"
}

fun main() {
  val pages = [1, 2, 3].mapConcurrent(id => fetch(id), workers: 3)
  io.println("$pages")
}
```

Output:

```text
[page1, page2, page3]
```

There is no `go` statement and no colouring either: a function that waits
(on a timer, a socket, a channel) is called like any other, and the
compiler infers that it suspends. `await` is written only on the waiting
primitives themselves (`sleep`, `recv`, a task handle).

## Data races are refused at compile time

Go finds a data race with `-race`, at run time, if the test happens to
hit it. Veles refuses one before the program runs: what crosses into a
task must be `Sendable` — immutable values, channels, `Mutex`, `Atomic` —
and a `MutableList` or a struct holding one is not. A module-level `var`
is an error for the same reason; shared state lives behind `Atomic` or
`Mutex` (D35, D66).

## Numbers overflow loudly

As in Go, numeric types never convert implicitly, and an integer literal
is an `i64` unless the context says otherwise. Unlike Go, arithmetic that
overflows panics, in every build, so a size computed from input cannot
wrap into a small number that passes a check; `+%`, `-%`, `*%` wrap when
that is what you want. A conversion that can lose information returns a
`T?`: `300.toU8()` is null where Go's `uint8(x)` keeps the low bits
([chapter 2](../02-values-and-strings.md)).

## Smaller differences

- Methods live in the struct's body; inside them a bare field name means
  `this.field` ([chapter 5](../05-structs-and-methods.md)). A struct is a
  value: assigning it copies it, as in Go.
- `defer` becomes `with`: a value that implements `Closeable` is closed
  when its block ends, on every way out.
- `switch` becomes `when`, which is an expression and must cover every
  case of what it matches.
- Generic functions take bounds: `fun largest<T: Comparable>(xs: List<T>): T?`;
  each instance is compiled for its types ([chapter 8](../08-traits-and-generics.md)).
- There is no `recover`. A panic is a bug that ends the program; `gather`
  reports a task's panic as a value when one task must not take the
  others down.
- There is no struct embedding: share behaviour through a trait's default
  methods, or hold the other struct in a field.

## Read next

[Errors](../07-errors.md), [nullable types](../06-nullable-types.md),
[sealed types](../09-sealed-types.md) (Go's missing sum types) and
[concurrency](../12-concurrency.md), then
[memory and C](../13-memory-and-ffi.md) for pointers and `unsafe`.
