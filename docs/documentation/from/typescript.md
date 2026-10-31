# From TypeScript

The syntax will look familiar — `val`/`var` for `const`/`let`, `=>`
lambdas, `?.`, template strings written `"$name ${expr}"` — but the
contract underneath is different. Types are checked and then trusted: there
is no `any`, no cast that lies, and no value whose type is wrong at run
time. A program compiles to a native executable, runs its tasks on every
core, and a function that waits does not change the type of its callers.

## At a glance

| TypeScript | Veles |
|---|---|
| `const x = 1`, `let y = 2` | `val x = 1`, `var y = 2` |
| `function add(a: number, b: number): number` | `fun add(a: i64, b: i64): i64` |
| `(x) => x * 2` | `x => x * 2` |
| `` `${name} is ${age}` `` | `"$name is $age"`, `"${expr}"` for an expression |
| `number` | `i64` and `f64` (and `i8`…`u64`, `f32`) |
| `null`, `undefined` | `null` only, in a `T?` |
| `a ?? b`, `a?.b` | `a ?: b`, `a?.b` |
| `type Shape = { kind: "circle"; … } \| …` | `sealed trait Shape` + one `struct` per case |
| `interface I { … }`, satisfied by shape | `trait I { … }`, `implement I` written out |
| `class C { … }` | `struct C { … }`: fields and methods, no inheritance |
| `async function f(): Promise<T>` | `fun f(): T` (suspension is inferred) |
| `await Promise.all([f(), g()])` | `awaitAll(async f(), async g())` |
| `throw` anything, catch `unknown` | `throws E` in the signature, `try`, a typed `catch` |
| `import { x } from "./m"` | `use m` (a module is a directory), then `m.x` |
| `arr.length`, `arr[i]` | `xs.len()`, `xs.at(i)` (a `T?`) |

## Integers are integers

`number` is a double: `7 / 2` is `3.5`, and integers are exact only up to
2^53. Veles has integer and float types, an integer literal is an `i64`,
and integer division stays integral. Overflow panics rather than losing
precision silently, and mixing an `i64` with an `f64` needs a conversion
you write (`n.toF64()`). Floats are IEEE doubles, as in JavaScript.

```veles
use io

fun main() {
  io.println("${7 / 2} ${7.0 / 2.0} ${0.1 + 0.2}")
  io.println("${9007199254740993 + 0}")
}
```

Output:

```text
3 3.5 0.30000000000000004
9007199254740993
```

The same addition in TypeScript prints `9007199254740992`.

## `==` compares contents, and types are nominal

Two structs, two lists or two maps are `==` when their contents are equal
— there is no reference identity to trip over, and no `===`. In the other
direction, a struct is its own type: a `Point` and a `Vector` with the same
fields do not convert into each other, and a value satisfies a trait only
when its type says `implement` ([chapter 8](../08-traits-and-generics.md)).

```veles
use io

struct Point {
  x: i64
  y: i64
}

fun main() {
  io.println("${Point(x: 1, y: 2) == Point(x: 1, y: 2)} ${[1, 2] == [1, 2]}")
}
```

Output:

```text
true true
```

## Unions are sealed types

A discriminated union becomes a `sealed trait` whose cases are structs.
There is no `kind` field to write and keep in step: the case is the type,
`when` takes it apart, and leaving a case out is a compile error, the job
`assertNever` does by convention in TypeScript
([chapter 9](../09-sealed-types.md)).

```veles
use io

sealed trait Event
struct Click : Event { x: i64, y: i64 }
struct Key : Event { code: string }

fun describe(e: Event): string => when (e) {
  is Click(x, y) => "click at $x,$y"
  is Key(code) => "key $code"
}

fun main() {
  val events: List<Event> = [Click(x: 3, y: 4), Key(code: "Enter")]
  loop (e in events) io.println(describe(e))
}
```

Output:

```text
click at 3,4
key Enter
```

Ad-hoc unions such as `string | number` do not exist; errors are the
exception, where `throws ParseError | IoError` names the errors a function
can throw ([chapter 7](../07-errors.md)).

## Waiting does not colour your functions

In TypeScript `async` spreads: a function that awaits returns a `Promise`,
so its callers must be `async` too, and forgetting an `await` gives you a
`Promise` where you wanted a value. In Veles a function that waits is
called like any other; the compiler infers that it suspends and its
callers simply wait for it. Concurrency is asked for explicitly with
`async`, and `await` appears only on the waiting primitives (`sleep`, a
channel's `recv`, a task handle). Tasks run on every core, not on one event
loop, which is why what they share must be immutable or behind a `Mutex`
([chapter 12](../12-concurrency.md)).

```veles
use io

fun user(id: i64): string {
  await sleep(Duration.millis(10))
  "user$id"
}

fun main() {
  io.println(user(7))   // no await: the call waits
  val (a, b) = awaitAll(async user(1), async user(2))
  io.println("$a $b")
}
```

Output:

```text
user7
user1 user2
```

## Errors are part of the signature

`throw` in TypeScript accepts anything and `catch` receives `unknown`.
Only values declared with `error` can be thrown in Veles, a function that
can fail says `throws` (with the errors listed, `throws ParseError`, or
bare, for the compiler to infer them), and a caller must handle the error
or pass it on with `try`: the compiler will not let it go unnoticed.

## Smaller differences

- `string.length` counts UTF-16 units; `s.len()` counts UTF-8 bytes, and
  `s.charCount()` counts characters ([chapter 2](../02-values-and-strings.md)).
- `xs.at(i)` returns a `T?` instead of `undefined` past the end, and the
  compiler makes you say what happens then. A negative index counts from
  the end, as `Array.prototype.at` does.
- `List<T>` cannot change at all — not `readonly` by convention but by
  type — and `MutableList<T>` can ([chapter 4](../04-collections.md)).
- No `this` binding rules: a lambda captures the variables around it, and
  in a method `this` is always the receiver.
- No `undefined` field and no optional-field object literals: a struct is
  built with every field that has no default, by name.

## Read next

[Values and types](../02-values-and-strings.md),
[nullable types](../06-nullable-types.md), [errors](../07-errors.md),
[sealed types](../09-sealed-types.md), and
[concurrency explained from scratch](../concurrency-explained.md) before
[the concurrency chapter](../12-concurrency.md).
