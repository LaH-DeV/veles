# From Kotlin

Of the languages here, Kotlin is the closest: `fun`, `val`/`var`, `when`,
`?:` and `?.`, lambdas with an implicit receiver, `sealed` hierarchies and
data-class-like structs all mean what you expect. The differences are
mostly removals — no classes or inheritance, no `!!`, no unchecked
exceptions, no `suspend` keyword — and one strengthening: an immutable
`List` really cannot change. Veles compiles to a native executable; there
is no JVM.

## At a glance

| Kotlin | Veles |
|---|---|
| `val x = 1` (an `Int`) | `val x = 1` (an `i64`: integer literals are 64-bit) |
| `fun add(a: Long, b: Long): Long = a + b` | `fun add(a: i64, b: i64): i64 => a + b` |
| `data class User(val name: String)` | `struct User { name: string }`: `==`, printing, hashing, `copy` |
| `user.copy(age = 32)` | `user.copy(age: 32)` |
| `sealed interface Shape` + `data class Circle(…) : Shape` | `sealed trait Shape` + `struct Circle : Shape { … }` |
| `interface I { fun f() = … }` + `class C : I` | `trait I { fun f() => … }` + `implement I` in `C`'s body |
| `open class`, `abstract class`, `: Base()` | none: traits with default methods, or a field |
| `companion object { fun of() }` | `static fun of()`; `object` → a module-level `val` |
| `fun String.initials()` | `extend string with fun initials()`; other files `use m extend` |
| `x!!` | none: `x ?: panic("why it cannot be null")`, or a pattern |
| `x?.let { … }` | `if (val v = x) { … }` |
| `@Throws`, unchecked exceptions | `throws E` in the signature, `try` at the call |
| `runCatching { f() }.getOrElse { d }` | `f() ?? d` |
| `suspend fun f()` | `fun f()` (suspension is inferred) |
| `coroutineScope { launch { … } }` | `scope { async … }` |
| `List<T>` / `MutableList<T>` | `List<T>` (immutable) / `MutableList<T>` |

## What a struct gives you

A `struct` is the data class: built by field name, compared with `==`,
printed with its fields, usable as a map key, and copied with `copy`. Two
differences: a struct is a value, so assigning it copies it (no shared
mutable object, nothing like `===`), and fields are fixed unless declared
`var`. `apply { }` works as in Kotlin, on a `var` binding
([chapter 5](../05-structs-and-methods.md)).

```veles
use io

struct User {
  name: string
  age: i64
  var visits: i64 = 0
}

extend string with fun initials(): string => split(" ").map(w => w.slice(0, 1)).join("")

fun main() {
  val ann = User(name: "Ann Lee", age: 31)
  val older = ann.copy(age: 32)
  var counted = older
  counted.apply { visits += 1 }
  io.println("$ann")
  io.println("$counted ${ann == older} ${ann.name.initials()}")
}
```

Output:

```text
User(name: Ann Lee, age: 31, visits: 0)
User(name: Ann Lee, age: 32, visits: 1) false AL
```

An extension function reaches other files only when they ask for it
(`use m extend`), so a module cannot change what `string` offers to code
that did not import it (D173).

## Immutable means immutable

Kotlin's `List` is a read-only view: the `MutableList` behind it may still
change under its reader. A Veles `List` cannot change at all, so it can be
shared between tasks without a lock, and a `MutableList` becomes one only
by a move (returning a list you built) or a copy (`toList()`)
([chapter 4](../04-collections.md)).

## No `!!`, and exceptions are checked

There is no operator that turns a null into a crash without a reason.
Where you know a value is there, say why (`?: panic("the header always has
three parts")`), or let the compiler see it: a list pattern
(`val [first, ..rest] = xs else return`) or an index it can prove in range
returns the value itself ([chapter 6](../06-nullable-types.md)).

Errors are values declared with `error`, a function that can fail says
`throws`, and a call passes the error on with `try` or handles it — the
discipline `Result` gives you in Kotlin, without the wrapping
([chapter 7](../07-errors.md)).

## Coroutines without `suspend`

Structured concurrency is the default, as with `coroutineScope`: a task
started with `async` belongs to its block, which waits for it, and a
failure cancels its siblings. What changes is the marking: there is no
`suspend` modifier, because the compiler infers which functions suspend,
and no `runBlocking`, because `main` may suspend too. `await` is written
only on the waiting primitives (`sleep`, a channel's `recv`) and on task
handles. Data that crosses into a task must be `Sendable` — an immutable
value, a channel, a `Mutex`, an `Atomic` — which is checked, not a
convention ([chapter 12](../12-concurrency.md)).

## Smaller differences

- Integer literals are `i64` and doubles are `f64`; there is no implicit
  widening between numeric types, and overflow panics instead of wrapping
  ([chapter 2](../02-values-and-strings.md)).
- `s.len()` counts UTF-8 bytes, not UTF-16 units.
- `when` arms are separated by newlines and use `=>`; `is` patterns can
  take a struct apart: `is Circle(r) => …`.
- `for (x in xs)` is `loop (x in xs)`; `while (c)` is `loop (c)`; ranges
  are `0..<n` (until) and `0..n` (inclusive).
- Visibility: nothing means the package, `public` other packages,
  `internal` the module (a directory), `private` the type.

## Read next

[Collections](../04-collections.md), [errors](../07-errors.md),
[traits and generics](../08-traits-and-generics.md) (no inheritance: what
to use instead) and [concurrency](../12-concurrency.md).
