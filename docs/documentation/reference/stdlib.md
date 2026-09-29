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
public trait Iterator {
  type Item
  fun next(): Item?                       // null at the end

  // lazy adapters: build a new iterator, do no work yet
  fun map<U>(f: fun(Item): U): MapIter<Self, U>
  fun filter(f: fun(Item): bool): FilterIter<Self>
  fun take(n: i64): TakeIter<Self>
  fun skip(n: i64): SkipIter<Self>
  fun enumerate(): EnumerateIter<Self>        // yields (i64, Item)
  fun zip<J: Iterator>(other: J): ZipIter<Self, J>   // yields (Item, J.Item)

  // terminal operations: pull until done
  fun toList(): List<Item>
  fun count(): i64
  fun fold<A>(init: A, f: fun(A, Item): A): A
  fun forEach(f: fun(Item))
  fun any(f: fun(Item): bool): bool
  fun all(f: fun(Item): bool): bool
  fun find(f: fun(Item): bool): Item?
  fun last(): Item?
}

public trait Iterable {
  type Iter: Iterator
  fun iterator(): Iter
  fun iter(): Iter = this.iterator()
}
```

`List<T>`, `MutableList<T>` and `Range<T>` implement `Iterable`;
`loop (x in c)` accepts any `Iterable` or `Iterator`. Maps iterate as
`(K, V)` tuples and sets as elements directly.

### Resources

```veles
// fragment
public trait Closeable { fun close() }     // for `with (r = ...) { }` (D43)
```

### Synchronisation (D35, D66)

Tasks run on several threads, so a `Mutex` is a real lock; an `Atomic`
of a number or a `bool` takes none — each operation is one processor
instruction, and `update` retries a compare-and-swap, so its function
may run more than once under contention (keep it free of side effects).
An `Atomic` of any other type is guarded by a lock. `withLock`'s
function cannot suspend; locking a `Mutex` again inside its own
`withLock` panics; the lock is released when the function panics. Copies of a `Mutex` or an
`Atomic` share the lock and the value. Module-level state that changes
must be a `val` holding one of these — a module-level `var` is an error.

```veles
// fragment
public struct Mutex<T> {
  init(value: T)                         // Mutex(value: 0)
  public fun withLock<R>(f: fun(*T): R): R
  public fun get(): T
  public fun set(value: T)
}

public struct Atomic<T> {
  init(value: T)                         // Atomic(value: 0)
  public fun load(): T
  public fun store(value: T)
  public fun swap(value: T): T           // the value it replaced
  public fun update(f: fun(T): T): T     // f of the value, stored as one step; the new value
}

public trait Sendable { }   // derived from the type's shape; a bound, never implemented by hand
```

Both are `Sendable` regardless of `T`. `Sendable` itself is the marker for
values with no shared mutable state — numbers, strings, immutable
collections, structs of such fields; never a mutable collection, a pointer
or a closure. It is what may cross a task boundary, and what a function may
duplicate: `MutableList<T>.repeat` and `fill` require it.

### Panics (D52)

```veles
// fragment
public struct Panic {                             // a task's panic, as seen by `gather`
  public message: string
  public location: string = ""                   // "main.vs:7:31"; empty inside the runtime
}
```

`panic(message)` raises one deliberately (D20). It never returns, so it
can stand in for a value: `val x = xs.at(i) ?: panic("index $i")`. A panic
prints its message and, on the next line, `at file:line:col in function`
relative to the package root (D64), then in a debug build a `called from`
line per call that led there (D81).

### Recursion depth — module `recursion` (D75)

```veles
// fragment
use recursion   // recursion.Depth(limit: 500), recursion.maxRecursionDepth
public val maxRecursionDepth: i64 = 1000         // the default bound
public fun tooDeepMessage(limit: i64): string    // "nesting deeper than 64"

public struct Depth {
  public limit: i64 = maxRecursionDepth
  public fun enter(): bool     // false at the limit, and then there is nothing to leave
  public fun leave()
  public fun depth(): i64
  public fun deepest(): i64    // the high-water mark
  public fun reset()
  public fun message(): string
}
```

Veles runs on the C stack and does not grow it, so a recursive walk over
input someone else wrote — a JSON document of 100 000 `[`, a source file of
50 000 `(` — has to be bounded or it is a crash rather than an error. There
is one default for that and one sentence to report reaching it, and every
decoder, encoder and parser in the library uses them.

A walk that already keeps a stack compares its length against a limit and
reports `tooDeepMessage(limit)`; that is what `std/json` does, with
`Options.maxDepth` (64) as its own policy. A walk whose depth is only its
call frames — a recursive descent parser — counts them with `Depth(limit: n)`,
where `enter` and `leave` pair like a push and a pop on every path out of
the body except a throw, which abandons the walk anyway. `examples/recursion`
is that parser, in forty lines.

### Errors (D4)

```veles
// fragment
public trait Error { fun message(): string = "$this" }   // what `error Name { }` implements
```

`error Name { fields; fun message() ... }` declares a struct with this
implement; only such types (or an explicit `implement Error for T`) can appear in
error position. A `message: string` field is used as the message when no
`message()` is written. `error Set = A | B` names a union (D45); a field
of an `error` may hold a union (a cause). `message()` is callable on an
error union directly.

```veles
// fragment
public error IoError { public path: string; public code: i64; public detail: string; public kind: IoKind = IoKind.Other }
public enum IoKind { Other = 0, NotFound, PermissionDenied, AlreadyExists, NotADirectory, IsADirectory, DirectoryNotEmpty,
  ConnectionRefused, ConnectionReset, ConnectionAborted, TimedOut, AddressInUse, AddressNotAvailable,
  BrokenPipe, Interrupted, InvalidInput, InvalidData }
```

`IoError` is what `fs`, `os` and `net` throw: `kind` is which failure,
portably (D76) — decide by it; `detail` is the system's description
("No such file or directory"), `path` the file, program or address
involved, `code` the platform's own error number (errno, or a Winsock code
on Windows), for logs; `message()` is `"detail: path"`.

### The operator traits

```veles
// fragment
public enum Ordering { Less = -1, Equal, Greater }             // what compareTo returns
public trait Comparable { fun compareTo(other: Self): Ordering } // <, <=, >, >=, sorted, min, max
public trait Equatable  { fun equals(other: Self): bool }     // ==, !=, contains, indexOf
public trait Hashable   { fun hash(): i64 }                   // map keys, set elements
public trait Display    { fun toString(): string }            // interpolation "$x"
```

A struct or sealed type has structural equality, hashing and
`Name(field: value)` text by default and replaces any of them with an
implement. `compareTo` returns `Ordering.Less`, `Ordering.Equal` or `Ordering.Greater`
(an enum whose values are -1, 0 and 1, so `< 0` still reads). Numbers and strings
implement `Comparable` (so `T: Comparable` bounds accept them); the
built-in types otherwise keep their own behaviour: lists, maps and sets
compare by content (`[1, 2] == [1, 2]`, a map equals a map with the same
entries in any order) whenever their elements do. A type with
`Equatable` needs `Hashable` to be a map key.

### Parsable

```veles
// fragment
public trait Parsable { static fun parse(s: string): Self? }   // i64.parse("42"), T.parse(s)
```

Construction from text, implemented for `i64`, `f64`, `bool` and `string`;
`null` when the text is not a value of the type. A `static fun` has no
receiver and is called on the type (`Point.origin()`, `Stack<i64>.of(1)`).

### Codable (D58)

```veles
// fragment
public trait Encodable { fun encode(to: codec.Encoder) throws EncodeError }
public trait Decodable { static fun decode(from: codec.Decoder): Self throws DecodeError }
public trait Codable : Encodable + Decodable { }   // `implement Codable` in a struct body derives both; `implement Codable for pkg.T` at top level
public error EncodeError { message, path }; public error DecodeError { problems: List<codec.Problem> }
```

The wire traits ([chapter 18](../18-codable-and-json.md)). Implemented
by the prelude for the numbers, `bool`, `string`, `T?`, `List`,
`MutableList`, `Map<string, V>` and `MutableMap<string, V>`; derived for
structs and sealed traits by an empty `implement`; automatic for enums.
`Comparable` is derived the same way. A struct field takes `@key`, `@skip`
and `@required`; a sealed trait `@tag`. Deriving needs no import; the
machinery below is module `codec` (D75).

### Module `codec`

```veles
// fragment
use codec
public trait Encoder { format(); enums(); keys(); beginObject(); key(name); endObject(); beginList(); endList(); writeI64/U64/F64/Bool/String/Null(v) }
public trait Decoder { format(); enums(); keys(); peek(): Kind; beginObject(); nextKey(): string?; endObject(); beginList(); hasNext(); endList(); readI64/U64/F64/Bool/String/Null(); skip(); path(); problem(msg); problemAt(path, msg); problems() }
public struct Problem { path, message; pointer() }; public struct Problems   // what a decoder records
public enum EnumStyle { Name, Number }; public enum DurationStyle { Seconds, Iso8601, Text, Nanos, Millis }; public enum KeyStyle { AsWritten, SnakeCase, CamelCase }; public enum Kind { Null, Bool, Int, Float, String, List, Object }
public sealed trait Value   // VNull, VBool, VInt, VFloat, VString, VList, VObject; get(k), at(i), asString/asI64/asF64/asBool(), isNull()
ValueEncoder.of(format, enums, keys, durations); ValueDecoder.of(v, format, enums, keys, durations)   // a Value as the sink / the source
```

A program imports `codec` to walk a document whose shape it does not know
(`codec.Value`), to pick a style (`json.Options(keys:
codec.KeyStyle.SnakeCase)`), or to write a format of its own.

### StringBuilder

```veles
// fragment
public struct StringBuilder {                 // StringBuilder() starts an empty one
  public fun append(s: string)
  public fun appendLine(s: string = "")
  public fun appendByte(b: u8)          // one UTF-8 byte, for code walking a string with byteAt
  public fun len(): i64
  public fun isEmpty(): bool
  public fun clear()
  public fun toString(): string
}
```

Builds text in linear time where repeated `+` would copy the whole string
each time.

### Deque and PriorityQueue

```veles
// fragment
public struct Deque<T> {                           // Deque<i64>() starts an empty one
  public fun addLast(x: T)
  public fun addFirst(x: T)
  public fun removeFirst(): T?
  public fun removeLast(): T?
  public fun first(): T?
  public fun last(): T?
  public fun at(i: i64): T?          // from the front; a negative i counts from the back
  public fun len(): i64
  public fun isEmpty(): bool
  public fun clear()
  public fun toList(): List<T>       // front to back; also Iterable and Display
}

public struct PriorityQueue<T> {
  init(compare: fun(T, T): Ordering)  // PriorityQueue<Job>(compare: (a, b) => ...)
  public static fun natural(): PriorityQueue<T>   // T: Comparable; smallest first
  public fun push(x: T)
  public fun pop(): T?               // the first element in the queue's order
  public fun peek(): T?
  public fun len(): i64
  public fun isEmpty(): bool
  public fun clear()
  public fun toList(): List<T>       // heap order: only the first is the smallest
}
```

Both are reference types like `MutableList` (D25): a `val` binding can
grow them and a callee shares the caller's. `Deque` is a ring buffer —
O(1) at both ends, amortised O(1) growth — and serves as a FIFO queue
(`addLast` / `removeFirst`), a stack, or a sliding window. `PriorityQueue`
is a binary heap: `push` and `pop` are O(log n). For the largest first,
`PriorityQueue<i64>(compare: (a, b) => b.compareTo(a))`.

### Result and Option

`Result<T, E>` with variants `Ok(value)` and `Err(error)`; `Option<T>`
with `Some(value)` and `None`, normally spelled `T?` and `null`. Both are
sealed types and match with `when`.

```veles
// fragment
r.getOrNull(): T?            // the value, or null
r.getOrDefault(fallback): T
r.errorOrNull(): E?          // the error, or null
results.oks(): List<T>       // on a List<Result<T, E>>: the successes, in order
results.errors(): List<E>    // the failures, in order
```

`Never` is the type of a call that does not return (`panic`, `os.exit`, a
function declared `: Never`); it fits anywhere, and code after it is
unreachable.

### Concurrency primitives

| | |
|---|---|
| `Channel<T>()` | a rendezvous: `send` completes when a receiver has the value |
| `Channel<T>(capacity: n)` | bounded channel; `send(v)`, `await recv(): T?`, `close()`, `closeAfter(n)` (closes itself after `n` more sends), `len()`; `trySend(v): bool` and `tryRecv(): T?` never wait (`false` when full, `null` when nothing is buffered) |
| `xs.mapConcurrent(f, workers: 4)`, `xs.forEachConcurrent(f, workers: 4)` | `List<T: Sendable>`: the worker pool — at most `workers` calls of `f` in flight, results in order; `f` is a `sendable fun` that may suspend and throw (then the call throws) |
| `await sleep(d: Duration)` | suspend for at least `d`; rounded up to the executor’s millisecond, so `Duration.zero` yields |
| `async f(...)`: `Task<T>` | start a task in the enclosing `scope`; `await task`; `task.cancel()` asks it to stop at its next suspension point (its `with` cleanups run, the scope still waits for it) |
| `withTimeout(limit: Duration, f): R throws E \| Timeout` | run the sendable `f` in a task of its own and throw `Timeout` (which carries the `limit` it reached) if it passes first — `f` is cancelled and has unwound by then; `f`'s own errors are rethrown |
| `TaskLocal(fallback: T)`: `TaskLocal<T: Sendable>` | a value that follows a task (D72): `tl.withValue(v, f): R throws E` binds `v` while `f` runs — there and in every task started inside, which keep it — and ends the binding with `f`, however `f` ends; `tl.get(): T` is the innermost binding, or `fallback` |

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
| `splitOnce(sep)` | `(string, string)?`: the text before and after the first `sep`, or `null` when it does not occur — `val (k, v) = line.splitOnce("=") ?: return` |
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
| `at(i)` | `T?`; null when out of range, or `T` where the index is known to be in range (D62). A negative `i` counts from the end (`at(-1)` is the last). A copy, like every read. Where it cannot fail, say why: `xs.at(i) ?: panic("…")` |
| `ref(i)` | `(*T)?`: a pointer to the element itself (`MutableList` only), or `*T` where the index is known. `xs.ref(i)?.bump()`, `xs.ref(i)?.n = 0` change the element; `loop (&x in xs)` visits every element this way |
| `indices()` | `0..<len()`; in `loop (i in xs.indices())` each `xs.at(i)` is a `T` |
| `atOrDefault(i, d)` | `T`; `at(i) ?: d` |
| `len()`, `isEmpty()`, `lastIndex()` | `lastIndex` is `len() - 1`: −1 when empty |
| `contains(x)`, `indexOf(x)`, `count(p)` | `bool`, `i64` (−1 if absent), `i64` |
| `first()`, `last()`, `min()`, `max()` | `T?`; `min`/`max` need `Comparable` elements |
| `take(n)`, `drop(n)`, `slice(from, to)` | new `List`, bounds clamped |
| `map(f)`, `filter(p)`, `fold(z, f)`, `forEach(f)`, `flatMap(f)` | eager; return `List` |
| `mapNotNull(f)`, `partition(p)` | `f` returns `U?`, nulls dropped; `(List<T>, List<T>)` of accepted and rest |
| `filterIs<V>()`, `filterNotNull()` | on a list of a sealed type: the elements of variant `V` as `List<V>`; on `List<T?>`: the present ones as `List<T>` |
| `any(p)`, `all(p)`, `find(p)` | |
| `zip(ys)`, `chunked(n)`, `windowed(n)`, `distinct()` | `List<(T, U)>`, `List<List<T>>`, `List<List<T>>`, `List<T>` |
| `enumerate()` | `List<(i64, T)>`: each element with its index, eager; `xs.iter().enumerate()` is the lazy form |
| `sorted()`, `sortedDescending()`, `sortedBy(key)`, `sortedByDescending(key)`, `reversed()` | new `List`; elements or `key` results must be `Comparable` (D48) |
| `sortedWith(compare)`, `minWith(compare)`, `maxWith(compare)` | any order: `compare(a, b)` negative when `a` comes first; sorts are stable, O(n log n) |
| `binarySearch(x)`, `binarySearchWith(compare)`, `binarySearchBy(key, target)` | `i64` index of the **first** match, −1 if absent, in O(log n). The list must already be in order; nothing checks, and a list that is not gives a wrong answer, never a panic. `compare(element)` says where the element sits relative to what is wanted |
| `lowerBound(x)`, `upperBound(x)`, `partitionPoint(pred)` | `i64` insertion points: first index not less than `x`, first greater than `x`, and the general form — the first index where `pred` stops holding. `upperBound − lowerBound` is how many times `x` occurs |
| `minBy(key)`, `maxBy(key)`, `distinctBy(key)` | `T?`, `T?`, `List<T>`; by a `Comparable` (or, for `distinctBy`, hashable) key |
| `sum()` | `List<i64>` and `List<f64>` only |
| `join(sep)` | `string` |
| `iter()` | lazy iterator |
| `toList()`, `toMutable()` | copies (D25) |
| `toSet()`, `toMutableSet()` | the distinct elements as a set (iterators have `toSet()` too) |
| `push(x)`, `pop(): T?`, `set(i, x)`, `clear()` | `MutableList` only |
| `insert(i, x)`, `removeAt(i): T`, `addAll(xs)`, `sort()` | `MutableList` only; `sort` is in place |
| `swap(i, j)`, `sortWith(compare)` | `MutableList` only; in place |
| `fill(x)` | `MutableList` only; overwrites every element, length unchanged |
| `MutableList<T>.repeat(x, count)`, `MutableList<T>.make(n, i => ...)` | statics: `count` copies of `x`, or `init(i)` called once per slot. `repeat` and `fill` need `T: Sendable` (D35): a mutable collection or pointer would be one value aliased by every slot, which is what `make` is for |

A `MutableList<T>` has every `List<T>` method; a `val` binding is enough to
call the mutating ones, since the list is a reference (D25).

### `Map<K, V>` and `MutableMap<K, V>`

| Method | Result |
|---|---|
| `get(k)` | `V?` |
| `getOrDefault(k, d)` | `V`: `get(k) ?: d`. A copy of the value. Where a key must be there, say why: `m.get(k) ?: panic("…")` |
| `ref(k)` | `(*V)?`: a pointer to the stored value (`MutableMap` only): `m.ref(k)?.bump()`, `m.ref(k)?.n += 1`; `loop ((k, &v) in m)` visits every value by reference |
| `containsKey(k)`, `len()`, `isEmpty()` | |
| `keys()`, `values()`, `entries()` | `List<K>`, `List<V>`, `List<(K, V)>` in insertion order |
| `toMap()`, `toMutable()` | copies |
| `set(k, v)`, `remove(k): bool`, `clear()` | `MutableMap` only |
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
`reversed()` (the last two are iterators that combine either way:
`(1..10).step(3).toList()`, `(0..10).reversed().step(3)` is 10, 7, 4, 1).

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
| `wrappingAdd/Sub/Mul(y)`, `saturatingAdd/Sub/Mul(y)` | every integer type | same type — the overflow policies other than the default panic (D21) |
| `checkedAdd/Sub/Mul(y)` | every integer type | `T?`: `null` on overflow |
| `countOnes()`, `leadingZeros()`, `trailingZeros()` | every integer type | same type |
| `toString(radix: i64 = 10)` | `i64`, `u64` | `string`; digits `0-9a-z`, radix 2 to 36 |
| `toFixed(digits)` | `f64`, `f32` | `string` with exactly that many decimals, rounded |

The bitwise operators `&`, `|`, `^`, `<<`, `>>` and `~` work on every integer
type; `&`, `<<`, `>>` bind like `*`, `|` and `^` like `+` (Go's grouping), and a
shift count at or beyond the width gives 0 (or the sign fill for a signed `>>`).

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
io.readAll(): string             // the rest of standard input
```

Each line goes out whole — tasks printing at once never interleave inside
one — and as it ends, whether standard output is a terminal or a pipe
(`docker logs`, the journal, `| grep`), so a service's log is live. Only
when it is redirected to a regular file are lines collected and written in
blocks, which is several times faster for bulk output.

## Module `os`

```veles
// fragment
use os
os.args(): List<string>                          // arguments, without the program name
os.program(): string                             // the program as invoked
os.env(name: string): string?                    // null when unset
os.exit(code: i64)                               // flushes output, ends the process
os.pid(): i64                                    // this process's id
os.hostname(): string throws IoError             // the host's network name
os.tempDir(): string                             // TMPDIR or /tmp; TMP/TEMP on Windows; no trailing separator
os.shutdownSignal(): os.Signal                    // suspends until SIGINT/SIGTERM (Ctrl+C/Break/close on Windows); arms on the first call, one signal per call
os.raiseSignal(sig: os.Signal)                    // as if it came from outside — for testing a shutdown path
// enum Signal { Interrupt = 2, Terminate = 15 }
os.run(program: string, args: List<string> = [], input: string = "", stderr: os.Stderr = os.Stderr.Capture): Output throws IoError   // no shell: each arg is one argument, verbatim; program found on PATH
// Output { code, stdout, stderr, ok() }; os.Stderr: Capture (apart, in stderr), Merge (into stdout), Inherit (to yours)
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
fs.readFile(path: string): string throws IoError           // must be UTF-8
fs.readBytes(path: string): List<u8> throws IoError
fs.writeBytes(path: string, bytes: List<u8>) throws IoError
fs.appendBytes(path: string, bytes: List<u8>) throws IoError
fs.writeFile(path: string, text: string) throws IoError    // replaces
fs.appendFile(path: string, text: string) throws IoError   // creates when missing
fs.exists(path: string): bool
fs.isFile(path: string): bool
fs.isDir(path: string): bool
fs.listDir(path: string): List<string> throws IoError      // names, sorted
fs.walk(root: string): List<string> throws IoError         // every file below root, depth-first in name order; links to directories not followed
fs.mkdir(path: string) throws IoError                      // with parents
fs.remove(path: string) throws IoError                     // a file or an empty directory
fs.rename(from: string, to: string) throws IoError
fs.cwd(): string throws IoError
```

## Module `net`

TCP over the task executor ([chapter 16](../16-networking.md)): every call
that waits suspends the task. Failures throw `IoError` with the address in
`path`.

```veles
// fragment
use net
net.listen(host: string = "127.0.0.1", port: i64 = 0): Listener throws IoError   // "" = every interface; 0 = any free port
net.connect(host: string, port: i64): Conn suspends throws IoError
listener.port(): i64                                     // the bound port
listener.accept(): Conn suspends throws IoError
conn.read(max: i64 = 65536): List<u8> suspends throws IoError   // what has arrived; [] at end of stream
conn.readExact(n: i64): List<u8> suspends throws IoError       // n bytes, fewer only at end of stream; n is the ceiling too
conn.readLine(max: i64): string? suspends throws IoError | TooLong   // without the newline; null at end of stream
error TooLong { message, limit }                               // a read hit the ceiling; `max` has no default, by design
conn.write(bytes: List<u8>) suspends throws IoError            // all of it
conn.writeText(text: string) suspends throws IoError
conn.shutdownWrite() throws IoError                            // half-close: the peer reads end of stream
conn.peer(): string                                            // "host:port"
// Listener and Conn are Closeable (use `with`) and Sendable (hand a Conn to `async handle(conn)`)
```

## Module `http`

An HTTP/1.1 server on `net` ([chapter 17](../17-http.md)).

```veles
// fragment
use http
val app = http.Router()
app.get("/users/{id}", req => ...)          // get / post / put / delete / any; `{name}` captures, a final `*` the rest
app.get("/static/*", http.files("./public"))   // index.html for a directory, `..` refused, type by extension
app.wrap(http.requestId()); app.wrap(http.timeout(Duration.seconds(1))); app.wrap(http.logging())   // first wrap = outermost; wraps the 404s too
type Middleware = sendable fun(Handler): Handler          // `next => req => ...`; req.withHeader(n, v) hands something to the handlers behind
http.serve(listener, app.handler(), limits: http.Limits(), log: true)   // forever, one task per connection; cancel its task to stop
http.serve(listener, h, stop: () => os.shutdownSignal(), grace: Duration.seconds(10))   // graceful: stop accepting, close idle, drain, cancel after grace
http.Limits(requestLineBytes: 8192, headerLineBytes: 8192, headerCount: 100, headerBytes: 65536,
            bodyBytes: 1048576, headerTimeout: Duration.seconds(10), bodyTimeout: Duration.seconds(30), idleTimeout: Duration.seconds(15))
// a byte ceiling answers 414 / 431 / 413 and closes; a time ceiling 408; idleTimeout just closes
type Handler = sendable fun(Request): Response suspends   // the stored form; `http.handler(h)` adapts a throwing h
error Fail { status, text }; http.notFound(text); http.badRequest(text); http.forbidden(text)   // thrown → that status; other errors → 500 + log; a panic → 500 + log
req.method; req.path; req.query; req.headers; req.header(name); req.body; try req.text(); req.param(name); req.peer
http.Response.text(s, status: http.Status.ok); .html(s); .json(s); .bytes(b, contentType); .empty(status); .redirect(url); resp.withHeader(n, v)
http.contentTypeOf(name); http.httpDate(t: time.Timestamp); http.percentDecode(s, plusIsSpace)
http.Status.notFound; http.Status(code: 418); s.code; s.reason(); s.isSuccess() / isRedirect() / isClientError() / isServerError(); "$s" is "404 Not Found"
http.Method.get / head / post / put / delete / patch / options / connect / trace; http.Method(name: "PROPFIND"); req.method == http.Method.post
http.Header.contentType, .location, .allow, .authorization, .cacheControl, ...   // lower-case names, as req.header() and withHeader() store them
// path matches, method does not → 405 + Allow; HEAD → the GET route, body dropped; OPTIONS → 204 + Allow; 1xx/204/304 never carry a body
http.call(handler, http.Method.get, "/notes/7?full=yes", body: "", headers: [:])   // in memory, no socket: same target parsing and panic boundary as serve
```

## Module `json`

JSON for anything Codable ([chapter 18](../18-codable-and-json.md)).

```veles
// fragment
use json
try json.encode(x); try json.pretty(x)                 // T: Encodable → text; EncodeError for a NaN or infinity
try json.decode<T>(text)                               // T: Decodable; DecodeError lists every problem with its path
try json.parse(text): codec.Value; try json.toValue(x); try json.fromValue<T>(v)
json.Options(keys: codec.KeyStyle.SnakeCase, enums: codec.EnumStyle.Number, durations: codec.DurationStyle.Iso8601, omitNulls: true, pretty: true, indent: "  ", maxDepth: 64, maxProblems: 100)
json.JsonEncoder(options); json.JsonDecoder.of(text, options)   // the Encoder / Decoder themselves
```

## Module `path`

Text only; nothing here touches the disk. `/` and `\` both separate on
input, output uses `/`.

```veles
// fragment
use path
path.join(parts: string...): string              // "a/b/c"; an absolute part starts over; join(xs...) for a list
path.dir(p: string): string                      // "a/b.vs" -> "a"; "" when no separator
path.base(p: string): string                     // "a/b.vs" -> "b.vs"
path.ext(p: string): string                      // ".vs" or ""
path.stem(p: string): string                     // "b"
path.isAbsolute(p: string): bool                 // "/x", "C:\x", "C:/x"
path.clean(p: string): string                    // "a/./b/../c" -> "a/c"; "" -> "."; lexical
path.within(root: string, p: string): bool       // p, cleaned, is root or inside it; both separators count
```

## Module `time`

`Duration` is in the prelude, not here: it is a *length* of time, and
`sleep` and `withTimeout` take one. `Timestamp` is the wall clock,
`Deadline` and `Stopwatch` the monotonic one — no raw monotonic reading is
handed out, so the two cannot be confused (D60).

```veles
// fragment
use time
time.now(): Timestamp                 // Timestamp.now(); .epoch, .ofSeconds/.ofMillis/.ofMicros(n)
t.toSeconds() / toMillis() / toMicros(): i64          // rounded down, so they work before 1970
t.subsecondMicros(): i64              // 0..999999
t + d / t - d: Timestamp;  t.since(earlier) / t.until(later): Duration
t.utc() / t.local() / t.at(offset): DateTime
"$t"                                  // RFC 3339 in UTC; Parsable and Codable are the same text

time.Offset.utc; Offset.of(hours, minutes = 0): Offset?; Offset.ofMinutes(m): Offset?   // ±18:00
time.Offset.local(at: Timestamp): Offset              // the host zone's offset at that instant
"$o"                                  // Z, +02:00, -05:30

DateTime(year:, month:, day:, hour: 0, minute: 0, second: 0, micros: 0, offset: Offset.utc)
d.timestamp(): Timestamp;  d.normalized(): DateTime   // out-of-range fields carry (month 13 = next January)
d.weekday(): Weekday (Monday … Sunday; .value is ISO, Monday = 1);  d.yearDay(): i64 (1 = Jan 1)
d.date(): string                      // 2026-09-24;  time(): 09:15:02;  "$d": full RFC 3339

time.parseRfc3339(s): Timestamp?      // strict, plus lower-case t/z, a space separator, expanded years
time.parseRfc3339Fields(s): DateTime? // the same, keeping the text's offset
time.formatRfc3339(t): string
time.formatHttp(t): string            // Sun, 06 Nov 1994 08:49:37 GMT (IMF-fixdate)
time.parseHttp(s): Timestamp?         // IMF-fixdate, RFC 850 and asctime (RFC 9110 §5.6.7)

time.Stopwatch.start(): Stopwatch     // elapsed(): Duration, reset()
time.Deadline.after(d): Deadline      // remaining(): Duration (never negative), expired(), extend(d), earlier(other)
time.monotonicNanos(): i64            // the raw reading, for a benchmark

time.daysFromCivil(y, m, d): i64;  time.civilFromDays(days): (i64, i64, i64)
time.isLeapYear(y): bool;  time.daysInMonth(y, m): i64
```

## Prelude type `Duration`

```veles
// fragment
Duration.zero; Duration.nanos/micros/millis/seconds/minutes/hours/days(n: i64)
Duration.ofSeconds(v: f64)            // a fractional count
d.toNanos() / toMicros() / toMillis() / toSeconds() / toMinutes() / toHours() / toDays(): i64   // truncate toward zero
d.asSeconds() / asMillis(): f64       // the fraction kept
a + b, a - b, d * n, d / n, -d (D71), d.abs(): Duration;  a.over(b): i64
d.over(o): i64                        // how many times o fits in d
d.isZero() / isNegative(): bool;  d.min(o) / d.max(o): Duration
"$d"                                  // 0s, 250ms, 1.5s, 2m30s, 1d1h, -90ms
Duration.parse(s): Duration?          // reads back exactly what "$d" writes; 1h30m, 250ms, 1.5s, -2m30s
```

## Module `log`

```veles
// fragment
use log { field }
log.debug(lazy msg, fields: Field...)   // also info, warn, error; the message is built only when the level is on
field<T: Encodable>(key, value): Field  // a named value; keeps its JSON type
log.setLevel(level: Level)              // Debug | Info | Warn | Error | Off; starts at Info or VELES_LOG
log.enabled(level: Level): bool
log.withFields(fields, f)               // fields on every line logged inside f, child tasks too
```

One line per call on standard error, written whole: text on a terminal,
one JSON object otherwise. `http.requestId()` binds the request id this
way. See [chapter 21](../21-logging.md).

## Module `random`

```veles
// fragment
use random
random.seed(n: i64)                   // reproducible from here on; unseeded, from the clock
random.range(lo: i64, hi: i64): i64   // lo..<hi
random.float(): f64                   // 0.0..<1.0
random.boolean(): bool
random.nextU64(): u64
random.pick<T>(xs: List<T>): T?       // null when empty
random.shuffle<T>(xs: MutableList<T>) // in place
random.Rng.seeded(n: i64): Rng        // an independent generator with the same methods
```

xoshiro256** seeded through splitmix64: fast and well distributed, not
cryptographic.

## Module `crypto`

```veles
// fragment
use crypto
crypto.sha256(data: List<u8>): Digest        // also sha384, sha512, sha1Legacy
crypto.digest<H: Hasher>(data: List<u8>): Digest
crypto.Sha256.start(): Sha256                // also Sha384, Sha512, Sha1
  h.update(data: List<u8>)                   // any number of times
  h.finish(): Digest                         // once; update after it panics
crypto.hmacSha256(key: List<u8>, message: List<u8>): Digest  // also 384, 512, Sha1Legacy
crypto.hmac<H: Hasher>(key: List<u8>, message: List<u8>): Digest
crypto.Hmac<Sha256>.start(key: List<u8>): Hmac<Sha256>       // update/finish as above
crypto.equalBytes(a: List<u8>, b: List<u8>): bool            // constant time
crypto.randomBytes(n: i64): List<u8>         // the OS CSPRNG; panics if it refuses
crypto.randomU64(): u64
crypto.uuidV4(): Uuid                        // 122 random bits
crypto.uuidV7(): Uuid                        // time-ordered, strictly increasing
```

`Digest`: `bytes()`, `len()`, `toHex()`, `toBase64Url()`, `prefix(n)`,
`Digest.of(bytes)`; `==` is constant time, `"$d"` is lower-case hex.

`Uuid`: `version()`, `timestamp()` (v7 only), `bytes()`, `isZero()`,
`Uuid.parse(text): Uuid?`, `Uuid.of(bytes)`, `Uuid.zero()`; `Comparable`,
so a list of v7 ids sorts into creation order.

`Hasher` is the trait behind the four digests: `start`, `algorithm`,
`blockSize`, `digestSize`, `update`, `finish`.

## Module `hex`

```veles
// fragment
use hex
hex.encode(bytes: List<u8>): string       // lower case
hex.encodeUpper(bytes: List<u8>): string
hex.decode(text: string): List<u8> throws hex.Invalid   // either case, nothing else
```

`hex.Invalid { message, position }` — `position` is the byte offset at fault.

## Module `base64`

```veles
// fragment
use base64
base64.encode(bytes: List<u8>): string       // standard alphabet, padded
base64.encodeUrl(bytes: List<u8>): string    // RFC 4648 §5, unpadded
base64.decode(text: string): List<u8> throws base64.Invalid
base64.decodeUrl(text: string): List<u8> throws base64.Invalid
base64.encodedLen(n: i64, pad: bool = true): i64
```

Both decoders take padded or unpadded input and refuse everything else —
the other alphabet, whitespace, an `=` in the middle, a non-canonical last
character. `base64.Invalid { message, position }`.

## Module `utf8`

```veles
// fragment
use utf8
utf8.decode(s: string, at: i64): utf8.Rune?        // null: out of range, or mid-character
utf8.decodeLast(s: string, before: i64): utf8.Rune?
utf8.decodeBytes(bytes: List<u8>, at: i64): utf8.Rune?   // null: also malformed
utf8.encodeTo(out: MutableList<u8>, code: i64): i64      // bytes appended, 1..4
utf8.encode(code: i64): List<u8>
utf8.char(code: i64): string                       // a one-character string
utf8.size(code: i64): i64?                         // how many bytes it takes
utf8.isScalar(code), utf8.isSurrogate(code)
utf8.isContinuation(b: u8), utf8.isStart(b: u8)
utf8.combineSurrogates(high: i64, low: i64): i64?  // for \uXXXX\uXXXX escapes
utf8.isValid(bytes: List<u8>), utf8.count(bytes: List<u8>)
utf8.maxCode, utf8.replacement, utf8.maxSize       // 0x10FFFF, 0xFFFD, 4
```

`utf8.Rune { code, size }` — the scalar value and its width, so
`i += r.size` steps to the next character.

Strings are indexed in bytes (D18) and are valid UTF-8 by construction, so
`decode` on one fails only for an index outside it or in the middle of a
character. Bytes carry no such promise, and `decodeBytes` is strict about
them in the sense Table 3-7 of the Unicode standard means: shortest form
only, no surrogates, nothing above U+10FFFF. Encoding a value that is not a
code point writes U+FFFD rather than failing. See `examples/utf8`.

## Module `jwt`

```veles
// fragment
use jwt
jwt.sign(claims: Claims, key: List<u8>, algorithm: Algorithm = Algorithm.HS256,
         keyId: string? = null): string throws EncodeError
jwt.verify(token: string, key: List<u8>, options: Options = Options()): Claims throws jwt.Invalid
jwt.readHeader(token: string): codec.Value throws jwt.Invalid   // unverified: for `kid`
jwt.now(): i64                                            // Unix seconds
```

`Claims { issuer, subject, audience, expiresAt, notBefore, issuedAt, id,
extra }` with `claim(name)`, `text(name)`, `number(name)`, `flag(name)`;
the three times are Unix **seconds**.

`Options { algorithm, audience, issuer, leeway, requireExpiry, now }` —
the algorithm is the caller's, never the token's.

`Algorithm`: `HS256`, `HS384`, `HS512`. `jwt.Invalid { message, reason }`
with `Reason`: `Malformed`, `UnsupportedAlgorithm`, `AlgorithmMismatch`,
`UnsupportedExtension`, `BadSignature`, `Expired`, `NotYetValid`,
`WrongAudience`, `WrongIssuer`, `MissingClaim`.

## Module `ffi`

Memory crossing into C (D69; [chapter 13](../13-memory-and-ffi.md)).

```veles
// fragment
use ffi
ffi.CString.of(s: string): CString throws ffi.NulByte   // malloc'd, NUL-terminated; Closeable (`with` frees)
c.ptr(): *raw u8                                         // panics after close
unsafe ffi.readString(p: *raw u8): string                // copies up to the NUL
unsafe ffi.readBytes(p: *raw u8, n: i64): List<u8>       // copies n bytes
ffi.alloc(n: i64): *raw u8;  unsafe ffi.free(p: *raw u8)  // zeroed, unmanaged memory
ffi.handle(value: T): Handle<T>                          // a Veles value as C's `void *userdata`; Closeable
h.ptr(): *raw u8;  unsafe ffi.Handle<T>.from(p): T       // the value back, in the callback
// ffi.CLayout: memory C can read as it is (derived from the shape, never implemented by hand)
// prelude: xs.withRaw(p => ...) on a List<T: ffi.CLayout> lends the elements for the closure, no copy
// language: extern "C" fun name(...) { }  and  &name: extern fun(...); p.cast<*raw T>() in unsafe
```

## Not yet in the bootstrap

TLS, UDP, a format-string module, iterating a directory tree, running a
program with its own stdin or a separate stderr capture. Each is a small
`extern "C"` binding away (see [chapter 13](../13-memory-and-ffi.md)) or
plain Veles on top of `fs`; the library grows with the compiler's own needs
setting the order.
