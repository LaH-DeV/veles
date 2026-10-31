# From Swift

Swift and Veles agree on more than they differ: optionals with `?.`,
values that are structs, protocols with default methods, typed errors,
structured concurrency and a `Sendable` rule. Most of the work is
learning new spellings for familiar ideas — and two rules that run the
other way: `??` means something else, and collections are shared, not
copied.

## At a glance

| Swift | Veles |
|---|---|
| `let x = 1`, `var y = 2` | `val x = 1`, `var y = 2` |
| `func add(_ a: Int, _ b: Int) -> Int` | `fun add(a: i64, b: i64): i64` |
| `name ?? "anonymous"` on an optional | `name ?: "anonymous"` |
| `(try? parse(s)) ?? -1` | `parse(s) ?? -1` |
| `if let w = firstWord(s) { … }` | `if (val w = firstWord(s)) { … }` |
| `guard let w = firstWord(s) else { return }` | `val w = firstWord(s) else return` |
| `enum Shape { case circle(r: Double) … }` | `sealed trait Shape` + `struct Circle : Shape { r: f64 }` |
| `enum Color: Int { case red = 1 }` | `enum Color { Red = 1 }` |
| `protocol P` + `extension P { default }` | `trait P { fun f() => default }` |
| `class C` | `struct C`; share it through a pointer `*C` or a `Mutex` |
| `throws(ParseError)` | `throws ParseError` |
| `async func f()`, `await f()` | `fun f()`, `f()` (suspension is inferred) |
| `async let a = f()` | `val a = async f()`, then `await a` |
| `withThrowingTaskGroup` | `scope { … }` |
| `"\(x)"` | `"$x"`, `"${expr}"` |

## `?:` for a missing value, `??` for a failure

The spelling to unlearn first. Swift's `??` gives the fallback for a
`nil` optional; in Veles that is `?:`, the Kotlin spelling. Veles keeps
`??` for the other case: the fallback when a call that `throws` failed,
which in Swift you write as `(try? f()) ?? x`
([chapter 6](../06-nullable-types.md), [chapter 7](../07-errors.md)).
`if let` and `guard let` become `if (val …)` and a `val` with `else`
that must leave.

```veles
use io

error NotANumber { text: string }

fun parse(text: string): i64 throws NotANumber => text.toInt() ?: throw NotANumber(text)

fun firstWord(line: string): string? {
  val word = line.split(" ").at(0) ?: return null
  if (word.isEmpty()) null else word
}

fun main() {
  val nickname: string? = null
  io.println(nickname ?: "anonymous")
  io.println("${parse("42") ?? -1} ${parse("x") ?? -1}")
  if (val word = firstWord("hello world")) io.println(word)
  io.println(firstWord(" leading") ?: "none")
}
```

Output:

```text
anonymous
42 -1
hello
none
```

A throwing call without `try` is not an error in Veles: it is a
`Result<T, E>` you can match, store or fall back from, so there is no
`try?` and no `try!`.

## Structs copy, collections are shared

As in Swift, a struct is a value: `var b = a` copies it and changing `b`
leaves `a` alone. Collections are where the languages part. A Swift array
is a value too (copied on write); a Veles `MutableList` is one object that
every name holding it sees. A `List`, which can never change, makes the
difference invisible, which is why it is the type to pass around; reach
for `MutableList` where the sharing is the point, and copy with
`toList()` where it is not ([chapter 4](../04-collections.md),
[chapter 5](../05-structs-and-methods.md#structs-are-values)).

```veles
use io

fun main() {
  val shared: MutableList<string> = ["a"]
  val alias = shared
  alias.push("b")
  io.println("$shared")
}
```

Output:

```text
[a, b]
```

## Enums with payloads are sealed types

A Swift `enum` does two jobs that Veles splits. Cases that carry data
become a `sealed trait` whose cases are structs, so each case is a type of
its own that a function can take or return; a `when` over it must cover
every case, as `switch` must. A Veles `enum` is the other job: a closed
set of named integer values, like a Swift enum with raw values
([chapter 9](../09-sealed-types.md)).

## Labels are the parameter names

A Veles parameter's name is its label, and there is no separate argument
label or `_`. A call may pass any argument by name, in any order, or by
position: `greet("Ada")`, `greet("Bob", punctuation: "?")`,
`greet(punctuation: ".", name: "Cy")` are all the same function
([chapter 3](../03-functions-and-control-flow.md)).

## Concurrency without `await` at every call

Veles infers which functions suspend, so there is no `async` on a
declaration and no `await` on a call. `await` is written only on the
waiting primitives — `sleep`, a channel's `recv`, a task handle — and on
the task you started with `async`. The structure is Swift's: a task
belongs to its block, `scope` is a task group, the first failure cancels
the siblings, and what crosses into a task must be `Sendable` (an
immutable value, a channel, a `Mutex`, an `Atomic`). Where Swift has an
actor, Veles has a `Mutex` around the state, or an `executor` declaration
for state that lives on one thread ([chapter 12](../12-concurrency.md)).

## Smaller differences

- A tracing collector instead of ARC: cycles are freed, and there is no
  `weak` or `unowned`.
- `s.len()` counts UTF-8 bytes, `s.charCount()` counts Unicode scalars
  (`unicodeScalars.count`); there is no grapheme-cluster count like
  Swift's `count` yet ([chapter 2](../02-values-and-strings.md)).
- There are no classes and no inheritance; a struct implements traits.
- Integer overflow panics in every build, as Swift traps; `+%` is `&+`.
- `defer` becomes `with`: a `Closeable` value is closed when its block ends.

## Read next

[Nullable types](../06-nullable-types.md), [errors](../07-errors.md),
[collections](../04-collections.md), [sealed types](../09-sealed-types.md)
and [concurrency](../12-concurrency.md).
