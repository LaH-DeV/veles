# Standard library reference (bootstrap)

The bootstrap standard library is deliberately small: enough to write
the tutorials and the compiler's own tests. Everything here is embedded
in the compiler binary (`std/` in the repository). A few primitives on
strings and collections are implemented by the compiler; everything else —
the prelude, `io`, `os`, `fs`, `path` — is written in Veles.

## Always in scope (the prelude, D24)

### Iteration

```veles
// fragment
pub trait Iterator {
  type Item
  mut fun next(): Item?                       // null at the end

  // lazy adapters: build a new iterator, do no work yet
  fun map<U>(f: fun(Item): U): MapIter<Self, U>
  fun filter(f: fun(Item): bool): FilterIter<Self>
  fun take(n: i64): TakeIter<Self>
  fun skip(n: i64): SkipIter<Self>
  fun enumerate(): EnumerateIter<Self>        // yields (i64, Item)
  fun zip<J: Iterator>(other: J): ZipIter<Self, J>   // yields (Item, J::Item)

  // terminal operations: pull until done
  mut fun toList(): List<Item>
  mut fun count(): i64
  mut fun fold<A>(init: A, f: fun(A, Item): A): A
  mut fun forEach(f: fun(Item))
  mut fun any(f: fun(Item): bool): bool
  mut fun all(f: fun(Item): bool): bool
  mut fun find(f: fun(Item): bool): Item?
  mut fun last(): Item?
}

pub trait Iterable {
  type Iter: Iterator
  fun iterator(): Iter
  fun iter(): Iter = self.iterator()
}
```

`List<T>`, `MutableList<T>` and `Range<T>` implement `Iterable`;
`loop (x in c)` accepts any `Iterable` or `Iterator`. Maps iterate as
`(K, V)` tuples and sets as elements directly.

### Resources

```veles
// fragment
pub trait Closeable { mut fun close() }     // for `with (r = ...) { }` (D43)
```

### Synchronisation (D35)

```veles
// fragment
pub struct Mutex<T> {
  pub fun withLock<R>(f: fun(*T): R): R
  pub fun get(): T
  pub fun set(value: T)
}
pub fun <T> mutex(value: T): Mutex<T>

pub struct Atomic<T> {
  pub fun load(): T
  pub fun store(value: T)
  pub fun swap(value: T): T
}
pub fun <T> atomic(value: T): Atomic<T>
```

Both are `Sendable` regardless of `T`.

### Panics (D52)

```veles
// fragment
pub struct Panic { pub message: string }    // a task's panic, as seen by `gather`
```

`panic(message)` raises one deliberately (D20). It never returns, so it
can stand in for a value: `val x = xs.at(i) ?: panic("index $i")`.

### Errors (D4)

```veles
// fragment
pub trait Error { fun message(): string = "$self" }   // what `error Name { }` implements
```

`error Name { fields; fun message() ... }` declares a struct with this
impl; only such types (or an explicit `impl Error for T`) can appear in
error position. A `message: string` field is used as the message when no
`message()` is written. `error Set = A | B` names a union (D45); a field
of an `error` may hold a union (a cause). `message()` is callable on an
error union directly.

```veles
// fragment
pub error IoError { pub path: string; pub code: i64; pub detail: string }
```

`IoError` is what `fs` and `os` throw: `detail` is the system's description
("No such file or directory"), `path` the file or program involved, `code`
the platform error number; `message()` is `"detail: path"`.

### The operator traits

```veles
// fragment
pub trait Comparable { fun compareTo(other: Self): i64 }   // <, <=, >, >=, sorted, min, max
pub trait Equatable  { fun equals(other: Self): bool }     // ==, !=, contains, indexOf
pub trait Hashable   { fun hash(): i64 }                   // map keys, set elements
pub trait Display    { fun toString(): string }            // interpolation "$x"
```

A struct or sealed type has structural equality, hashing and
`Name(field: value)` text by default and replaces any of them with an
impl. `compareTo` is negative, zero or positive. Numbers and strings
implement `Comparable` (so `T: Comparable` bounds accept them); the
built-in types otherwise keep their own behaviour. A type with
`Equatable` needs `Hashable` to be a map key.

### Parsable

```veles
// fragment
pub trait Parsable { static fun parse(s: string): Self? }   // i64.parse("42"), T.parse(s)
```

Construction from text, implemented for `i64`, `f64`, `bool` and `string`;
`null` when the text is not a value of the type. A `static fun` has no
receiver and is called on the type (`Point.origin()`, `Stack<i64>.of(1)`).

### StringBuilder

```veles
// fragment
pub fun stringBuilder(): StringBuilder
pub struct StringBuilder {
  pub fun append(s: string)
  pub fun appendLine(s: string = "")
  pub fun len(): i64
  pub fun isEmpty(): bool
  pub fun clear()
  pub fun toString(): string
}
```

Builds text in linear time where repeated `+` would copy the whole string
each time.

### Result and Option

`Result<T, E>` with variants `Ok(value)` and `Err(error)`; `Option<T>`
with `Some(value)` and `None`, normally spelled `T?` and `null`. Both are
sealed types and match with `when`.

### Concurrency primitives

| | |
|---|---|
| `Channel<T>(capacity: n)` | bounded channel; `send(v)`, `await recv(): T?`, `close()`, `len()` |
| `await sleep(ms: i64)` | suspend for at least `ms` milliseconds |
| `async f(...)`: `Task<T>` | start a task in the enclosing `scope`; `await task` |

## Built-in methods

The compiler provides a small set of primitives on each built-in type;
the rest of the methods below are ordinary Veles in the prelude, added
with `extend` blocks (`std/prelude/string.vs`, `list.vs`) — go-to-definition
opens them. Both kinds are called the same way.

### `string`

| Method | Result |
|---|---|
| `len()` | `i64`, in bytes (D18) |
| `charCount()` | `i64`, Unicode scalar values |
| `chars()` | `List<string>`, one string per code point |
| `isEmpty()` | `bool` |
| `startsWith(s)`, `endsWith(s)`, `contains(s)` | `bool` |
| `indexOf(part, from: 0)`, `lastIndexOf(part)` | `i64` byte index, −1 if absent |
| `substring(from, to)` | `string?` — byte offsets; null if a boundary splits a character |
| `byteAt(i)`, `bytes()` | `u8` (panics out of range), `List<u8>` |
| `trim()`, `trimStart()`, `trimEnd()` | ASCII whitespace removed |
| `split(sep)`, `lines()` | `List<string>`; `lines` drops `\r` and a final empty line |
| `replace(old, new)`, `repeat(n)` | `string` |
| `toUpper()`, `toLower()` | ASCII letters only |
| `padStart(width, pad: " ")`, `padEnd(width, pad: " ")` | `string` |
| `toInt()`, `toF64()` | `i64?`, `f64?` |
| `+`, `==`, `<` … | concatenation and comparison |

Interpolation `"$x ${expr}"` accepts any value. `List<u8>.decodeUtf8()` is
the way back from `bytes()`: `string?`, null when the bytes are not valid
UTF-8.

### `List<T>` and `MutableList<T>`

| Method | Result |
|---|---|
| `xs[i]` | `T`; panics when out of range |
| `at(i)` | `T?`; a negative `i` counts from the end (`at(-1)` is the last) |
| `len()`, `isEmpty()` | |
| `contains(x)`, `indexOf(x)`, `count(p)` | `bool`, `i64` (−1 if absent), `i64` |
| `first()`, `last()`, `min()`, `max()` | `T?`; `min`/`max` need `Comparable` elements |
| `take(n)`, `drop(n)`, `slice(from, to)` | new `List`, bounds clamped |
| `map(f)`, `filter(p)`, `fold(z, f)`, `forEach(f)`, `flatMap(f)` | eager; return `List` |
| `any(p)`, `all(p)`, `find(p)` | |
| `zip(ys)`, `chunked(n)`, `windowed(n)`, `distinct()` | `List<(T, U)>`, `List<List<T>>`, `List<List<T>>`, `List<T>` |
| `sorted()`, `sortedDescending()`, `sortedBy(key)`, `reversed()` | new `List`; elements or `key` results must be `Comparable` (D48) |
| `sum()` | `List<i64>` and `List<f64>` only |
| `join(sep)`, `joinToString(sep)` | `string` |
| `iter()` | lazy iterator |
| `toList()`, `toMutable()` | copies (D25) |
| `push(x)`, `pop(): T?`, `clear()` | `MutableList` only |
| `insert(i, x)`, `removeAt(i): T`, `addAll(xs)`, `sort()` | `MutableList` only; `sort` is in place |

A `MutableList<T>` has every `List<T>` method; a `val` binding is enough to
call the mutating ones, since the list is a reference (D25).

### `Map<K, V>` and `MutableMap<K, V>`

| Method | Result |
|---|---|
| `m[k]`, `get(k)` | `V?` |
| `containsKey(k)`, `len()`, `isEmpty()` | |
| `keys()`, `values()`, `entries()` | `List<K>`, `List<V>`, `List<(K, V)>` in insertion order |
| `toMap()`, `toMutable()` | copies |
| `m[k] = v`, `set(k, v)`, `remove(k): bool`, `clear()` | `MutableMap` only |
| `forEach((k, v) => ...)`, `mapValues(v => ...)`, `filter((k, v) => ...)` | iterate; new `Map` |
| `getOrPut(k, () => v)` | `V` — `MutableMap` only: stores `v` when `k` is absent |

Keys must be hashable: scalars, strings, tuples and structs of hashable
fields.

### `Set<T>` and `MutableSet<T>`

`contains(x)`, `len()`, `isEmpty()`, `toList()`, `toSet()`, `toMutable()`,
`union(s)`, `intersect(s)`, `difference(s)`, `isSubsetOf(s)`; on `MutableSet`:
`add(x): bool`, `remove(x): bool`, `clear()`. Construct with `Set<T>()` /
`MutableSet<T>()`, or with a list literal where a set type is expected:
`val s: Set<i64> = [1, 2]`.

### `Range<T>`

Fields `lo`, `hi`, `inclusive`; iterable. `len()`, `contains(x)`, `step(n)` and
`reversed()` (the last two are iterators: `(1..10).step(3).toList()`).

### Numbers

Each is a single machine instruction (an LLVM intrinsic), not a runtime call.

| Method | On | Result |
|---|---|---|
| `sqrt()`, `abs()`, `floor()`, `ceil()`, `round()`, `trunc()` | `f64`, `f32` | same type; `round` is half away from zero |
| `pow(y)`, `min(y)`, `max(y)`, `mod(y)`, `clamp(lo, hi)`, `sign()` | `f64`, `f32` | same type; `mod` is Euclidean (never negative for `y > 0`) |
| `log()`, `log2()`, `log10()`, `exp()`, `sin()`, `cos()`, `tan()`, `atan2(x)`, `hypot(y)` | `f64`, `f32` | same type; radians |
| `isNaN()`, `isFinite()`, `isInfinite()` | `f64`, `f32` | `bool` |
| `abs()`, `min(y)`, `max(y)`, `mod(y)`, `clamp(lo, hi)`, `sign()` | every integer type | same type; `abs` on an unsigned type is the identity; `mod` is Euclidean where `%` truncates (`(-7).mod(3)` is 2, `-7 % 3` is -1) |
| `pow(n)` | every integer type | same type; panics on overflow or `n < 0` |
| `wrappingAdd/Sub/Mul(y)`, `saturatingAdd/Sub(y)` | every integer type | same type — the overflow policies other than the default panic (D21) |
| `checkedAdd/Sub/Mul(y)` | every integer type | `T?`: `null` on overflow |
| `countOnes()`, `leadingZeros()`, `trailingZeros()` | every integer type | same type |

`f64` is an IEEE 754 double, so `NaN` and the infinities are ordinary values
of the type: `0.0 / 0.0`, `inf - inf` and `(-1.0).sqrt()` produce `NaN`, and it
propagates through arithmetic. `NaN == NaN` is false — test with `isNaN()`.
They print as `NaN`, `inf` and `-inf`.

Every built-in method is documented in the editor: hover shows its signature
and description, and go-to-definition opens `builtins.vs`, a declarations-only
file the language server writes next to the standard library sources (they
are embedded in the compiler; the copies live under the user cache directory).

## Module `io`

```veles
// fragment
use io
io.println(s: string)
io.print(s: string)
io.eprintln(s: string)           // standard error
io.readLine(): string?           // null at end of input
```

## Module `os`

```veles
// fragment
use os
os.args(): List<string>                          // arguments, without the program name
os.program(): string                             // the program as invoked
os.env(name: string): string?                    // null when unset
os.exit(code: i64)                               // flushes output, ends the process
os.run(program: string, args: List<string> = []): Output throws IoError
// Output { code: i64, stdout: string, fun ok(): bool }
os.ioError(code: i64, path: string): IoError     // an IoError for a platform error number
```

`run` waits for the program, captures its standard output and lets its
standard error through; a non-zero exit is reported in `Output.code` —
only a program that cannot be started throws.

## Module `fs`

Every call that can fail throws `IoError` (below).

```veles
// fragment
use fs
fs.readFile(path: string): string throws IoError
fs.writeFile(path: string, text: string) throws IoError    // replaces
fs.appendFile(path: string, text: string) throws IoError   // creates when missing
fs.exists(path: string): bool
fs.isFile(path: string): bool
fs.isDir(path: string): bool
fs.listDir(path: string): List<string> throws IoError      // names, sorted
fs.mkdir(path: string) throws IoError                      // with parents
fs.remove(path: string) throws IoError                     // a file or an empty directory
fs.rename(from: string, to: string) throws IoError
fs.cwd(): string throws IoError
```

## Module `path`

Text only; nothing here touches the disk. `/` and `\` both separate on
input, output uses `/`.

```veles
// fragment
use path
path.join(a: string, b: string): string          // "a/b"; an absolute b wins
path.joinAll(parts: List<string>): string
path.dir(p: string): string                      // "a/b.vs" -> "a"; "" when no separator
path.base(p: string): string                     // "a/b.vs" -> "b.vs"
path.ext(p: string): string                      // ".vs" or ""
path.stem(p: string): string                     // "b"
path.isAbsolute(p: string): bool                 // "/x", "C:\x", "C:/x"
```

## Not yet in the bootstrap

Network I/O, formatting beyond interpolation, time, random numbers,
reading a file as bytes, iterating a directory tree. Each is a small
`extern "C"` binding away (see [chapter 13](../13-memory-and-ffi.md)) or
plain Veles on top of `fs`; the library grows with the compiler's own needs
setting the order.
