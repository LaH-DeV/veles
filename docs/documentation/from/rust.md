# From Rust

Rust readers will recognise the shape of Veles: `Result` and `Option`
(spelled `T?`), traits implemented explicitly, generics compiled per
type, closed sum types with exhaustive matching, `unsafe` blocks around
raw pointers and C. The difference is underneath: memory is managed by a
garbage collector, so there are no lifetimes, no borrow checker and no
`clone()` to satisfy one, and data races are refused by what may cross
into a task rather than by unique ownership.

## At a glance

| Rust | Veles |
|---|---|
| `let x = 1;`, `let mut y = 2;` | `val x = 1`, `var y = 2` |
| `fn add(a: i64, b: i64) -> i64` | `fun add(a: i64, b: i64): i64` |
| `Option<T>`, `Some(x)`, `None` | `T?`: a value or `null` |
| `opt.unwrap_or(d)`, `opt.map(f)` | `opt ?: d`, `opt?.f()` |
| `Result<T, E>` + `f()?` | `fun f(): T throws E` + `try f()` |
| `enum E { A, B }` for the errors of one function | `throws A \| B`, a union written in the signature |
| `res.unwrap_or(d)` | `res ?? d` |
| `enum Shape { Circle { r: f64 }, … }` | `sealed trait Shape` + `struct Circle : Shape { r: f64 }` |
| `enum Color { Red = 1 }` | `enum Color { Red = 1 }` |
| `impl Trait for T` | `implement Trait` in `T`'s body, or `implement Trait for T` |
| `fn largest<T: PartialOrd>(…)` | `fun largest<T: Comparable>(…)` |
| `Box<dyn Trait>` | a trait object: `val s: Shape = Circle(r: 1.0)` |
| `Vec<T>`, `&[T]` | `MutableList<T>`, `List<T>` (immutable) |
| `impl Drop` | `implement Closeable` + `with x = …` |
| `move \|\| …`, `FnMut` | `() => …`: captures by reference, no annotations |
| `tokio::spawn`, `.await` | `async f()` in a scope; calls that wait need no `.await` |
| `Send` | `Sendable` |
| `macro_rules!`, derive macros | `const fun`, annotations read at compile time, `implement Codable` |

## Errors: `try` is `?`, and unions replace the error enum

`try f()` does what `f()?` does: on failure, the enclosing function
returns the error. What changes is the error type. Instead of an enum per
function and `From` conversions, a signature lists what it can throw —
`throws Empty | TooLong` — or says a bare `throws` and lets the compiler
infer the union from what the body throws and passes on. A call
without `try` is a `Result` value, to match, store or fall back from with
`??` ([chapter 7](../07-errors.md)).

```veles
use io

error Empty { }
error TooLong { len: i64 }

fun check(name: string): string throws Empty | TooLong {
  if (name.isEmpty()) throw Empty()
  if (name.len() > 8) throw TooLong(name.len())
  name
}

fun main() {
  loop (name in ["ann", "", "bartholomew"]) {
    when (val r = check(name)) {
      is Ok => io.println("ok $r")
      is Err => io.println("rejected: ${r.message()}")
    }
  }
}
```

Output:

```text
ok ann
rejected: Empty
rejected: TooLong(len: 11)
```

## No lifetimes, no borrow checker

Values are kept alive by the collector for as long as something reaches
them, so a function returns whatever it likes, a struct holds any value,
and a closure captures the variables around it by reference — reads see
later writes, writes are seen outside — with no `move`, `Fn`/`FnMut` or
`Rc<RefCell<…>>` ([chapter 10](../10-closures-and-iterators.md)).

```veles
use io

fun largest<T: Comparable>(xs: List<T>): T? {
  val [first, ..rest] = xs else return null
  rest.fold(first, (best, x) => if (x > best) x else best)
}

fun main() {
  io.println("${largest([3, 9, 4]) ?: -1} ${largest(["b", "a"]) ?: "-"}")
  var total = 0
  val add = (n: i64) => { total += n }
  add(2)
  add(3)
  io.println("$total")
}
```

Output:

```text
9 b
5
```

A struct is a value, copied when assigned; a `MutableList`, a `Mutex` and
a pointer `*T` are references, and several names may hold one. Aliasing is
allowed — what is refused is sharing mutable data between tasks.

## Data races: `Sendable` instead of `&mut`

Rust prevents data races by allowing one `&mut` at a time. Veles allows
many names for a `MutableList` within one task and refuses it at the
task boundary: what an `async` captures or a channel carries must be
`Sendable` — immutable values, channels, `Mutex`, `Atomic` — which is
checked like `Send`. A module-level `var` is an error for the same reason
(D35, D66). Within a task, the order of your statements is the only
order there is.

## Async without a runtime crate

The scheduler is part of the language, so there is no executor to choose
and no `.await` on each call: the compiler infers which functions suspend,
and their callers wait for them. A task started with `async` belongs to
its block, which waits for it; leaving the block early, or the first
failure under `awaitAll`, cancels the others, which stop at their next
suspension point — the `try_join!` behaviour, everywhere. Tasks run on a
pool of threads ([chapter 12](../12-concurrency.md)).

## Overflow and conversions

Arithmetic overflow panics in every build — Rust panics only in debug and
wraps in release — and `+%`, `-%`, `*%` are `wrapping_add` and friends.
`x.toU8()` returns a `u8?`, like `u8::try_from(x).ok()`; `x.wrapU8()` is
`x as u8` ([chapter 2](../02-values-and-strings.md)).

## Smaller differences

- `match` is `when`; arms are `pattern => value`, one per line.
- Methods live in the struct's body and take no `self` parameter: the
  receiver is `this`, and a bare field name means `this.field`.
- Inherent methods go in the struct's body; `extend T { … }` adds more
  outside it, for a type your package declares, and an extension function
  (`extend string with fun …`) reaches a type you did not write.
- Macros do not exist. Compile-time work is a `const fun`, a struct's
  fields are read at compile time through annotations (D174), and the wire
  code for JSON comes from `implement Codable` ([chapter 18](../18-codable-and-json.md)).
- `unsafe`, raw pointers (`*raw T`) and `extern "C"` work much as in Rust,
  with a `// SAFETY:` comment expected above each block
  ([chapter 13](../13-memory-and-ffi.md)).

## Read next

[Errors](../07-errors.md), [traits and generics](../08-traits-and-generics.md),
[closures](../10-closures-and-iterators.md),
[concurrency](../12-concurrency.md) and [memory and C](../13-memory-and-ffi.md).
