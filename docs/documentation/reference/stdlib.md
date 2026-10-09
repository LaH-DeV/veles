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
  fun iter(): Iter => this.iterator()
}
```

`List<T>`, `MutableList<T>` and `Range<T>` implement `Iterable`;
`loop (x in c)` accepts any `Iterable` or `Iterator`. Maps iterate as
`(K, V)` tuples and sets as elements directly.

### Resources

```veles
// fragment
public trait Closeable { fun close() }     // for `with r = ...` and `with (r = ...) { }` (D43/D100)
```

### Synchronisation (D35, D66)

Tasks run on several threads, so a `Mutex` is a real lock; an `Atomic`
of a number or a `bool` takes none — each operation is one processor
instruction, and `update` retries a compare-and-swap, so its function
may run more than once under contention (keep it free of side effects).
An `Atomic` of any other type is guarded by a lock. `withLock`'s
function cannot suspend; locking a `Mutex` again inside its own
`withLock` panics; the lock is released when the function panics. For
more than one expression, `with n = m.lock()` holds the lock to the end
of the block, with the same rules: nothing in the block may suspend, `n`
cannot leave it, and `lock()` is usable only as a `with` value (D107). Copies of a `Mutex` or an
`Atomic` share the lock and the value. Module-level state that changes
must be a `val` holding one of these — a module-level `var` is an error.

```veles
// fragment
public struct Mutex<T> {
  init(value: T)                         // Mutex(value: 0)
  public fun withLock<R>(f: fun(*T): R): R
  public fun lock(): Locked<T>           // only as `with n = m.lock()`: n is a *T, held to the block's end (D107)
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
public const maxRecursionDepth: i64 = 1000       // the default bound
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

Veles runs on the C stack and does not grow it — it is 256 MB per thread
(`VELES_STACK`, chapter 7), and running out is a `stack overflow` panic that
cannot be caught — so a recursive walk over input someone else wrote, a JSON
document of 100 000 `[` or a source file of 50 000 `(`, has to be bounded to
answer with an *error* instead. There is one default for that and one
sentence to report reaching it, and every decoder, encoder and parser in the
library uses them.

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
public trait Error { fun message(): string => "$this" }   // what `error Name { }` implements
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

### Default (D119)

```veles
// fragment
public trait Default { static fun default(): Self }   // i64.default(), T.default()
```

A value to start from, for generic code that has to make one: `0` for every
number, `false`, `""`, empty `List`/`Map`/`Set` and their mutable forms,
`null` for `T?`, `Duration.zero`, a tuple (up to eight elements) of
`Default` types, and an `Array<T, N>` of `N` copies of `T`'s default. A struct derives it with an empty `implement Default`:
each field takes its declared default, or else its type's `default()`; a
field with neither is an error naming it, and a generic struct's type
parameters get the `Default` bound they need. A sealed trait, a `Secret` and
a value holding a task are not derived (write `static fun default(): Self`
by hand); an enum cannot implement it (D57).

### Codable (D58)

```veles
// fragment
public trait Encodable { fun encode(to: codec.Encoder) throws EncodeError }
public trait Decodable { static fun decode(from: codec.Decoder): Self throws DecodeError; static fun schema(format: string, keys: codec.KeyStyle): codec.Schema }   // schema: derived beside a derived decode (D125); the default is one opaque value
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
public enum EnumStyle { Name, Number }; public enum DurationStyle { Seconds, Iso8601, Text, Nanos, Millis }; public enum KeyStyle { AsWritten, SnakeCase, CamelCase, UpperSnake }; public enum Kind { Null, Bool, Int, Float, String, List, Object }
public sealed trait Value   // VNull, VBool, VInt, VFloat, VString, VList, VObject; get(k), at(i), asString/asI64/asF64/asBool(), isNull()
ValueEncoder.of(format, enums, keys, durations); ValueDecoder.of(v, format, enums, keys, durations)   // a Value as the sink / the source
public struct Schema { kind: SchemaKind, what: string, secret: bool, fields: List<SchemaField>; element(): Schema?; asSecret() }   // the shape of a Decodable (D125): what `config` asks for by name
public struct SchemaField { name, key, schema: Schema, required: bool, nullable: bool, fallback: string? }   // key: as decode matches it for the format and key style
public enum SchemaKind { Bool, Int, Float, Text, List, Object, Opaque, Unsupported }   // Unsupported: a Map, a sealed family
```

A program imports `codec` to walk a document whose shape it does not know
(`codec.Value`), to pick a style (`json.Options(keys:
codec.KeyStyle.SnakeCase)`), or to write a format of its own.

### StringBuilder

```veles
// fragment
public struct StringBuilder {                 // StringBuilder() starts an empty one
  public const fun append(s: string)
  public const fun appendLine(s: string = "")
  public const fun appendByte(b: u8)          // one UTF-8 byte, for code walking a string with byteAt
  public const fun reserve(n: i64)           // room for n bytes in all: appending up to n never grows it (D105)
  public const fun len(): i64
  public const fun isEmpty(): bool
  public const fun clear()
  public const fun toString(): string
}
```

Builds text in linear time where repeated `+` would copy the whole string
each time. Every method is a `const fun`: a constant's function may build text
with one (D113).

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
  public fun reserve(n: i64)         // room for n elements in all (D105)
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
| `race { ch.send(v) => … }` | a send arm (D108): ready when `ch` can take `v`; when it wins `v` is in the channel, when another arm wins it was not sent; on a closed channel it panics |
| `ch.toList(): List<T>`, `ch.forEach(f)` | receive until `ch` is closed and drained; an error from `f` stops `forEach` and is thrown (D110) |
| `Semaphore(permits: n)` | at most `n` holders (D110): `acquire(): Permit` waits for one (cancellable), `tryAcquire(): Permit?` does not, `available(): i64`; a `Permit` is `Closeable` — `with sem.acquire()` — and a second close does nothing; `n < 1` panics |
| `retry(times, f, delay: Duration.zero): R throws E` | call `f` until it returns, waiting `delay` after each thrown error, `times` calls in all, then throw the last error; a panic is not retried, cancellation stops it at the wait; `times < 1` panics (D110) |
| `xs.mapConcurrent(f, workers: 4)`, `xs.forEachConcurrent(f, workers: 4)` | `List<T: Sendable>`: the worker pool — at most `workers` calls of `f` in flight, results in order; `f` is a `sendable fun` that may suspend and throw (then the call throws) |
| `await sleep(d: Duration)` | suspend for at least `d`; rounded up to the executor’s millisecond, so `Duration.zero` yields |
| `async f(...)`: `Task<T>` | (`T` is `f`'s value even when `f` throws — its error fails the scope; in `gather` the handle is `Task<Result<T, E>>`; a handle stays in its block, D141) start a task in the enclosing `scope` — or, as `with t = async f(...)`, in the background until the block ends, which cancels then joins it (D100); `await task`; `task.cancel()` asks it to stop at its next suspension point (its `with` cleanups run, the scope still waits for it) |
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
| `byteAtUnchecked(i)` | `u8`; only in `unsafe`, checked in a debug build only (D114) |
| `trim()`, `trimStart()`, `trimEnd()` | ASCII whitespace removed |
| `split(sep)`, `lines()` | `List<string>`; `lines` drops `\r` and a final empty line |
| `splitOnce(sep)` | `(string, string)?`: the text before and after the first `sep`, or `null` when it does not occur — `val (k, v) = line.splitOnce("=") ?: return` |
| `replace(old, new)`, `repeat(n)` | `string` |
| `toUpper()`, `toLower()` | ASCII letters only |
| `padStart(width, pad: " ")`, `padEnd(width, pad: " ")` | `string` |
| `toInt()`, `toF64()` | `i64?`, `f64?` |
| `+`, `==`, `<` … | concatenation and comparison |

The string functions that are written in Veles — `trim`, `trimStart`,
`trimEnd`, `split`, `splitOnce`, `lines`, `replace`, `repeat`, `toUpper`,
`toLower`, `capitalize`, `padStart`, `padEnd`, `indexOf`, `lastIndexOf` — are
`const fun`s, as are the built-in ones (`len`, `byteAt`, `substring`, `chars`,
`bytes`, `startsWith`, `toInt`…) and the integer and exact float operations of
[Numbers](#numbers) (`abs`, `min`, `max`, `pow`, `mod`, `rotateLeft`,
`wrappingAdd`, `sqrt`, `floor`…): a constant's function may use them (D113).
`sin`, `cos`, `exp`, `log`, `pow` on floats and the other C library functions
are not: the compiler does not guess which library the program links.

Interpolation `"$x ${expr}"` accepts any value. `List<u8>.decodeUtf8()` is
the way back from `bytes()`: `string?`, null when the bytes are not valid
UTF-8.

### Endian bytes (D130)

```veles
// fragment
x.toBeBytes(): Array<u8, N>; x.toLeBytes()            // every integer type; N = 1, 2, 4, 8 (isize, usize: 8)
u32.fromBeBytes(a: Array<u8, 4>): u32; u32.fromLeBytes(a)   // likewise on every integer type
list.readU16Be(offset: i64): u16?                      // List<u8>; U16, U32, U64, I16, I32, I64 × Be, Le; null when the bytes are not all there
out.pushU32Be(x: u32); out.pushI16Le(x: i16)           // MutableList<u8>; the same twelve
```

### `List<T>` and `MutableList<T>`

| Method | Result |
|---|---|
| `at(i)` | `T?`; null when out of range, or `T` where the index is known to be in range (D62). A negative `i` counts from the end (`at(-1)` is the last). A copy, like every read. Where it cannot fail, say why: `xs.at(i) ?: panic("…")` |
| `atUnchecked(i)`, `setUnchecked(i, v)` | `T`, `()`; only in `unsafe`, `0 <= i < len`, checked in a debug build only (D114); `set…` on `MutableList` |
| `ref(i)` | `(*T)?`: a pointer to the element itself (`MutableList` only), or `*T` where the index is known. `xs.ref(i)?.bump()`, `xs.ref(i)?.n = 0` change the element; `loop (&x in xs)` visits every element this way |
| `indices()` | `0..<len()`; in `loop (i in xs.indices())` each `xs.at(i)` is a `T` |
| `atOrDefault(i, d)` | `T`; `at(i) ?: d` |
| `len()`, `isEmpty()`, `lastIndex()` | `lastIndex` is `len() - 1`: −1 when empty |
| `contains(x)`, `indexOf(x)`, `count(p)` | `bool`, `i64` (−1 if absent), `i64` |
| `first()`, `last()`, `min()`, `max()` | `T?`; `min`/`max` need `Comparable` elements |
| `take(n)`, `drop(n)`, `slice(from, to)` | new `List`, bounds clamped |
| `map(f)`, `filter(p)`, `fold(z, f)`, `forEach(f)`, `flatMap(f)` | eager; return `List`. These and `any`, `all`, `find`, `count`, `indexOfFirst`, `mapNotNull`, `partition` are prelude Veles generic over what the function does: they throw what it throws (`xs.map(a => try parse(a))` is a `Result` until `try`) and suspend only when it suspends (D116) — `xs.map(x => fetch(x))` waits for each in turn |
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
| `push(x)`, `pop(): T?`, `set(i, x)`, `clear()`, `reserve(n)` | `MutableList` only; `reserve` makes room for `n` in all, so pushing up to `n` never grows the list (D83) |
| `insert(i, x)`, `removeAt(i): T`, `addAll(xs)`, `sort()` | `MutableList` only; `sort` is in place |
| `swap(i, j)`, `sortWith(compare)` | `MutableList` only; in place |
| `fill(x)` | `MutableList` only; overwrites every element, length unchanged |
| `incrementAt(i)` | `MutableList<i64>` only; increments one element in place, panics out of range; negative indexes count from the end |
| `updateAt(i, transform): T` | `MutableList<T>` only; transforms and replaces one element, returning the new value. The synchronous callback runs once and must not mutate the same list; panics out of range, negative indexes count from the end |
| `MutableList<T>.repeat(x, count)`, `MutableList<T>.make(n, i => ...)` | statics (the `<T>` may be left out when the arguments or the expected type give it, D137): `count` copies of `x`, or `init(i)` called once per slot. `repeat` and `fill` need `T: Sendable` (D35): a mutable collection or pointer would be one value aliased by every slot, which is what `make` is for |

A `MutableList<T>` has every `List<T>` method; a `val` binding is enough to
call the mutating ones, since the list is a reference (D25).

The prelude helpers that take no function — `atOrDefault`, `take`, `drop`,
`concat`, `indices`, `enumerate`, `zip`, `distinct`, `chunked`, `windowed`,
`lastIndex`, `toSet`, `toMutableSet`, `sortedDescending`, `min`, `max`, `sum`,
and `MutableList`'s `repeat`, `fill`, `swap`, `insert`, `removeAt`, `sort` — are
`const fun`s: a constant's function may use them (D113). So are the ones that
take a function (`map`, `filter`, `fold`, `forEach`, `any`, `all`, `find`,
`count`, `indexOfFirst`, `flatMap`, `mapNotNull`, `partition`, `sortedBy`,
`sortedByDescending`, `sortedWith`, `sortWith`, `minBy`, `maxBy`, `minWith`,
`maxWith`, `distinctBy`, `partitionPoint`, `binarySearch`, `binarySearchBy`,
`binarySearchWith`, `lowerBound`, `upperBound`, `make`, `updateAt`) and a `Map`'s
`mapValues`, `filter`, `forEach` and `getOrPut`, when the function given is
one the compiler can run.

### `Map<K, V>` and `MutableMap<K, V>`

| Method | Result |
|---|---|
| `get(k)` | `V?` |
| `getOrDefault(k, d)` | `V`: `get(k) ?: d`. A copy of the value. Where a key must be there, say why: `m.get(k) ?: panic("…")` |
| `ref(k)` | `(*V)?`: a pointer to the stored value (`MutableMap` only): `m.ref(k)?.bump()`, `m.ref(k)?.n += 1`; `loop ((k, &v) in m)` visits every value by reference |
| `containsKey(k)`, `len()`, `isEmpty()` | |
| `keys()`, `values()`, `entries()` | `List<K>`, `List<V>`, `List<(K, V)>` in insertion order |
| `toMap()`, `toMutable()` | copies |
| `set(k, v)`, `remove(k): bool`, `clear()`, `reserve(n)` | `MutableMap` only; after `reserve(n)` inserting up to `n` entries never grows or rehashes (D105) |
| `forEach((k, v) => ...)`, `mapValues(v => ...)`, `filter((k, v) => ...)` | iterate over the entries as they were when the call starts; new `Map`. Prelude Veles: they throw and suspend as the function does (D116) |
| `getOrPut(k, () => v)` | `V` — `MutableMap` only: stores `v` when `k` is absent |

Keys must be hashable: scalars, strings, tuples and structs of hashable
fields.

### `Array<T, N>` (D121)

`N` elements stored inline, a value: `val b = a` copies it. `len()` (the constant `N`),
`isEmpty()`, `at(i)` (`T?`; a `T` for a constant index in range or one the bounds facts prove),
`first()`, `last()`, `indices()` (`0..<N`), `set(i, x)` on a `var`, `atUnchecked(i)` and
`setUnchecked(i, x)` (`unsafe`), `toList()`, `toMutable()`; `loop (x in a)` over a copy and
`loop (&x in a)` in place. `map`, `filter`, `forEach`, `fold`, `any`, `all` and `find` read the
array where it lies; every other read-only `List` method runs on a copy. Created from a literal
where an array type is expected, `Array<T, N>.make(x)` (`N` copies), `xs.toArray<N>()`
(`Array<T, N>?`, null unless the list holds exactly `N`) or `Array<T, N>.default()`. `==`, hash and
text are a list's; it is `Codable` (a list of exactly `N`), `Sendable` when `T` is, and
`withRaw(f)` lends C its storage when `T` is `CLayout`. `N` is a constant argument
(`Array<u8, 4 * 16>`) or a `<const N: i64>` parameter.

### `Set<T>` and `MutableSet<T>`

`contains(x)`, `len()`, `isEmpty()`, `toList()`, `toSet()`, `toMutable()`,
`union(s)`, `intersect(s)`, `difference(s)`, `isSubsetOf(s)`; on `MutableSet`:
`add(x): bool`, `remove(x): bool`, `clear()`, `reserve(n)` (D105). Construct with `Set<T>()` /
`MutableSet<T>()`, or with a list literal where a set type is expected:
`val s: Set<i64> = [1, 2]`.

### `Range<T>`

Fields `lo`, `hi`, `inclusive`; iterable. `len()`, `isEmpty()` (D106), `contains(x)`, `step(n)` and
`reversed()` (the last two are iterators that combine either way:
`(1..10).step(3).toList()`, `(0..10).reversed().step(3)` is 10, 7, 4, 1). All of
them, and looping over a range or a step, work in a `const fun` (D113).

### Numbers

Each is a single machine instruction (an LLVM intrinsic), not a runtime call.

| Method | On | Result |
|---|---|---|
| `sqrt()`, `abs()`, `floor()`, `ceil()`, `round()`, `trunc()` | `f64`, `f32` | same type; `round` is half away from zero |
| `pow(y)`, `min(y)`, `max(y)`, `mod(y)`, `clamp(lo, hi)`, `sign()` | `f64`, `f32` | same type; `mod` is Euclidean (never negative for `y > 0`) |
| `log()`, `log2()`, `log10()`, `exp()`, `sin()`, `cos()`, `tan()`, `atan2(x)`, `hypot(y)` | `f64`, `f32` | same type; radians |
| `isNaN()`, `isFinite()`, `isInfinite()` | `f64`, `f32` | `bool` |
| `isSignNegative()`, `copySign(y)` | `f64`, `f32` | the sign bit — true for `-0.0` and a negative NaN; this magnitude with `y`'s sign (D104) |
| `nextUp()`, `nextDown()` | `f64`, `f32` | the nearest representable number above / below (IEEE 754); NaN stays NaN, an infinity in its own direction stays (D104) |
| `abs()`, `min(y)`, `max(y)`, `mod(y)`, `clamp(lo, hi)`, `sign()` | every integer type | same type; `abs` on an unsigned type is the identity; `mod` is Euclidean where `%` truncates (`(-7).mod(3)` is 2, `-7 % 3` is -1) |
| `pow(n)` | every integer type | same type; panics on overflow or `n < 0` |
| `wrappingAdd/Sub/Mul(y)`, `saturatingAdd/Sub/Mul(y)` | every integer type | same type — the overflow policies other than the default panic (D21) |
| `checkedAdd/Sub/Mul(y)` | every integer type | `T?`: `null` on overflow |
| `countOnes()`, `leadingZeros()`, `trailingZeros()` | every integer type | same type |
| `rotateLeft(n)`, `rotateRight(n)` | every integer type | same type; bits rotated by `n` modulo the width, a negative `n` the other way (D104) |
| `swapBytes()`, `reverseBits()` | every integer type | same type; byte order (big- ↔ little-endian; identity on 8-bit types) or bit order reversed (D104) |
| `toString(radix: i64 = 10)` | `i64`, `u64` | `string`; digits `0-9a-z`, radix 2 to 36 |
| `toFixed(digits)` | `f64`, `f32` | `string` with exactly that many decimals, rounded |
| `toBits()` | `f64`, `f32` | the IEEE 754 bit pattern, `u64` / `u32`; nothing rounded, a NaN keeps its payload (D93) |
| `f64.fromBits(bits)`, `f32.fromBits(bits)` | static | the number with that pattern; the inverse, and every pattern is a number |

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
trait io.Stream : Closeable + Sendable   // net.Conn, fs.File; effects declared (D40, D128); objects cross tasks
  read(max: i64 = 65536): List<u8> suspends throws IoError         // [] at the end
  readExact(n: i64): List<u8> suspends throws IoError              // fewer only at the end
  readLine(max: i64): string? suspends throws IoError | TooLong    // no default for `max`, by design
  write(bytes: List<u8>) suspends throws IoError
  writeText(text: string) suspends throws IoError
  shutdownWrite() suspends throws IoError                          // nothing on a file
error io.TooLong { message, limit }                                // a read hit the ceiling
```

`io.Stream` is usable as a type (`fun summarize(s: io.Stream)`, a trait
object whose suspending calls park the task) and as a bound
(`fun lines<S: io.Stream>(s: S)`, one copy per type).

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
fs.stat(path: string): Stat throws IoError                 // Stat { size: i64, modified: Timestamp, isDir: bool, isFile() }; follows links
fs.open(path: string, mode: FileMode = FileMode.Read): File throws IoError   // enum FileMode { Read, Write /* creates, empties */, Append }; Closeable (D97)
file.read(max: i64 = 65536): List<u8> throws IoError       // from where the last read ended; [] at the end
file.readAt(offset: i64, max: i64): List<u8> throws IoError  // anywhere; does not move `read`'s place
file.write(bytes: List<u8>) throws IoError; file.size(): i64 throws IoError
file.readExact(n), file.readLine(max), file.writeText(text), file.shutdownWrite()   // the rest of io.Stream (D128), which File implements
fs.listDir(path: string): List<string> throws IoError      // names, sorted
fs.walk(root: string): List<string> throws IoError         // every file below root, depth-first in name order; links to directories not followed
fs.mkdir(path: string) throws IoError                      // with parents
fs.remove(path: string) throws IoError                     // a file or an empty directory
fs.rename(from: string, to: string) throws IoError
fs.cwd(): string throws IoError
fs.writeAtomic(path: string, bytes: List<u8>) throws IoError   // D130: temp file beside it, synced, renamed over; permissions kept; Windows: not while another handle has it open
fs.copy(from: string, to: string) throws IoError               // a piece at a time, replaces `to`; onto itself does nothing
fs.lines(path: string, max: i64): Lines throws IoError         // Iterator of Result<string, IoError>; a line past max is an error item (InvalidData); closes at the end or after an error; Lines is Closeable; max <= 0 panics
file.seek(offset: i64) throws IoError                          // next read / write at offset (Append always writes at the end); negative panics
file.sync() throws IoError                                     // data on the disk (fsync / FlushFileBuffers)
file.lock(): Lock throws IoError; file.tryLock(): Lock? throws IoError   // exclusive, advisory (flock; one byte past any file on Windows); Lock is Closeable; the file closing lets go
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
conn.readLine(max: i64): string? suspends throws IoError | io.TooLong   // without the newline; null at end of stream
                                                               // `Conn` implements io.Stream (below it, `io.TooLong` is the ceiling error)
conn.write(bytes: List<u8>) suspends throws IoError            // all of it
conn.writeText(text: string) suspends throws IoError
conn.shutdownWrite() throws IoError                            // half-close: the peer reads end of stream
conn.peer(): string                                            // "host:port"
// Listener and Conn are Closeable (use `with`) and Sendable (hand a Conn to `async handle(conn)`)
```

## Module `tls`

TLS over any `io.Stream` ([chapter 16](../16-networking.md)): SChannel on Windows, OpenSSL elsewhere (D128).
Failures throw `IoError` whose `detail` starts `tls:`.

```veles
// fragment
use tls
tls.connect(host: string, port: i64, options: tls.Options = tls.Options()): tls.Conn suspends throws IoError
tls.client(stream: io.Stream, serverName: string, options: tls.Options = tls.Options(), address: string = ""): tls.Conn suspends throws IoError   // the handshake over a stream that is already connected; owns it
tls.Options(roots: string = "", serverName: string = "", alpn: List<string> = [], dangerouslyAcceptAnyCertificate: bool = false)
                                                          // roots: PEM text trusted instead of the system's; serverName: for SNI and the check, default the host
conn.protocol(): string?                                  // the ALPN protocol agreed
conn.peer(): string
// Conn implements io.Stream and Closeable; one task may read while another writes
tls.Certificate.load(certPath: string, keyPath: string, alpn: List<string> = []): tls.Certificate throws IoError
tls.Certificate.fromPem(chain: string, key: Secret<string>, alpn: List<string> = []): tls.Certificate throws IoError
cert.replace(chain: string, key: Secret<string>, alpn: List<string> = []) throws IoError   // for connections accepted from now on
tls.listen(cert: tls.Certificate, host: string = "127.0.0.1", port: i64 = 0): tls.Listener throws IoError
tls.wrap(listener: net.Listener, cert: tls.Certificate): tls.Listener
listener.accept(): tls.Conn suspends throws IoError      // the handshake runs on the connection's first read or write
listener.port(): i64
tls.reloading(certPath: string, keyPath: string, every: Duration, alpn: List<string> = []): tls.CertificateReloader throws IoError   // holds a task (D111): `with`-bind it
reloader.certificate(): tls.Certificate
```

## Module `db`

PostgreSQL, written in Veles over `std/net`, `std/tls` and `std/crypto` ([chapter 24](../24-databases.md)).

```veles
// fragment
use db
sql"select id from users where name = ${name}"        // a Sql: the text keeps $1, $2; values travel apart
db.ident(name: string): Sql?                              // a quoted table or column name, null unless letters, digits, _ and dots
Sql.dangerouslyRaw(text: string): Sql                     // migrations and DDL: the text is the query
sql.text(): string  sql.valueCount(): i64                 // the text with placeholders; how many values
trait db.Param { fun toArg(): db.Arg }                    // implemented for the numbers, bool, string, List<u8>, Timestamp, Duration, Uuid, T?, Secret, Sql
db.open(url: Secret<string>, size: i64 = 10, timeout: Duration = 30s, connectTimeout: Duration = 10s, application: string = "veles", checkEvery: Duration = 30s, idleTimeout: Duration = 5m): db.Pool suspends throws db.DbError   // holds a task: `with`-bind it
pool.query<T: Decodable>(statement: Sql, timeout: Duration? = null): List<T> suspends throws db.DbError | DecodeError
pool.queryOne<T: Decodable>(statement: Sql, timeout: Duration? = null): T? suspends throws db.DbError | DecodeError
pool.exec(statement: Sql, timeout: Duration? = null): i64 suspends throws db.DbError   // rows affected
pool.ping() suspends throws db.DbError
pool.begin(timeout: Duration? = null): db.Tx suspends throws db.DbError   // `with tx = try pool.begin()`
tx.query / tx.queryOne / tx.exec                          // as on the pool
tx.commit() suspends throws db.DbError                    // not committed when the block ends: abandoned, the database rolls back
tx.rollback() suspends throws db.DbError                  // the connection returns to the pool
// DbError: kind (Connection, Closed, Timeout, Auth, Server, Protocol, Config), text, code (SQLSTATE), detail, hint, severity
//   isUniqueViolation() isForeignKeyViolation() isConstraintViolation() isRetryable() isCancelled()
// Pool and Tx are Closeable
```

## Module `http`

An HTTP/1.1 server and client on `net` ([chapter 17](../17-http.md)).

```veles
// fragment
use http
val app = http.Router()
app.get("/users/{id}", req => ...)          // get / post / put / delete / any; `{name}` captures, a final `*` the rest
app.get("/static/*", http.files("./public"))   // ETag + Last-Modified + 304/412, one Range (206/416), `no-cache`, index.html, dotfiles 404, `..` refused
http.files(dir, maxAge: null, immutable: false, index: ["index.html"], dotfiles: false, redirect: true, etag: true, lastModified: true)   // D96; immutable needs a maxAge
app.wrap(http.requestId()); app.wrap(http.timeout(Duration.seconds(1))); app.wrap(http.logging())   // first wrap = outermost; wraps the 404s too
app.wrap(http.cors(origins: ["https://app.example.com", "https://*.example.org"], methods: [...], headers: [...], expose: [...], credentials: false, maxAge: null))   // D99: nothing allowed by default; preflight answered before the router; Vary: Origin; "*" + credentials panics
app.wrap(http.guard(req => null))   // Response? — null lets on, a Response answers, a thrown Fail answers (D99)
app.wrap(http.basicAuth("realm", (user, pass) => ok)); app.wrap(http.bearer(token => ok, realm: null))   // 401 + WWW-Authenticate; basicAuth puts the user in x-remote-user (Header.remoteUser); compare secrets with crypto.equalBytes
type Middleware = sendable fun(Handler): Handler          // `next => req => ...`; req.withHeader(n, v) hands something to the handlers behind
http.serve(listener, app.handler(), limits: http.Limits(), log: true)   // forever, one task per connection; cancel its task to stop
http.serve(listener, h, stop: () => os.shutdownSignal(), grace: Duration.seconds(10))   // graceful: stop accepting, close idle, drain, cancel after grace
// health (D133, chapter 17): /healthz runs no check; /readyz runs them all at once under their timeouts, 503 + failing names, details logged not returned
val health = http.Health(); health.check("db", () => try pool.ping(), timeout: Duration.seconds(2)); app.wrap(health.endpoints(live: "/healthz", ready: "/readyz"))
http.serve(listener, h, tls: cert)   // HTTPS: each connection is secured with the tls.Certificate (chapter 16); the handshake runs in the connection's task
http.serve(listener, h, health: health, stop: …)   // flips health.stopping() when the stop begins: /readyz is 503 from then; health.stopping() is public too; probe answers are left out of the request log
http.Limits(requestLineBytes: 8192, headerLineBytes: 8192, headerCount: 100, headerBytes: 65536,
            bodyBytes: 1048576, headerTimeout: Duration.seconds(10), bodyTimeout: Duration.seconds(30), idleTimeout: Duration.seconds(15),
            connections: 10000)   // connections: served at once, 0 = no limit; a full server stops accepting (backlog waits) (D99)
// a byte ceiling answers 414 / 431 / 413 and closes; a time ceiling 408; idleTimeout just closes
type Handler = sendable fun(Request): Response suspends   // the stored form; `http.handler(h)` adapts a throwing h
error Fail { status, text }; http.notFound(text); http.badRequest(text); http.forbidden(text)   // thrown → that status; other errors → 500 + log; a panic → 500 + log
req.method; req.path; req.query; req.headers; req.header(name); req.param(name); req.peer
try req.bytes(max: null); try req.text(max: null)   // the body, read now, at most max (Limits.bodyBytes): 413 over it; kept, so form()/text() can follow (D97)
req.stream(max: n): Body                            // body.read(max = 65536): List<u8> (empty at the end); body.readAll(); body.length(): i64? (null when chunked); `max` has no default
try req.multipart(max: n, maxParts: 100): Multipart // try form.next(): Part?; part.name / .filename / .contentType / .headers; try part.read(max) / .bytes(max:) / .text(max:) / .saveTo(path, max:)
// Transfer-Encoding: chunked decoded; Expect: 100-continue answered on the first read; an unread body is drained (64 KiB) or the connection closes
http.Response.text(s, status: http.Status.ok); .html(s); .json(s); .bytes(b, contentType); .empty(status); .redirect(url); resp.withHeader(n, v)   // contentType: a MediaType or a string
http.Response.stream(http.MediaType.eventStream, out => { try out.writeText("data: x\n\n") }, length: null)   // sent as produced: chunked, or content-length when `length` is given; out.write(bytes) / out.writeText(s) (D97)
http.MediaType.html / .json / .png / ...; http.MediaType.ofExtension(".png"); http.MediaType(name: "application/vnd.api+json"); m.essence()   // D97
http.httpDate(t: time.Timestamp); http.percentDecode(s, plusIsSpace)
http.Status.notFound; http.Status(code: 418); s.code; s.reason(); s.isSuccess() / isRedirect() / isClientError() / isServerError(); "$s" is "404 Not Found"
http.Method.get / head / post / put / delete / patch / options / connect / trace; http.Method(name: "PROPFIND"); req.method == http.Method.post
http.Header.contentType, .location, .allow, .authorization, .cacheControl, ...   // lower-case names, as req.header() and withHeader() store them
// path matches, method does not → 405 + Allow; HEAD → the GET route, body dropped; OPTIONS → 204 + Allow; 1xx/204/304 never carry a body
with srv = try http.testServer(handler, limits: http.Limits())   // D130: a real listener on a free loopback port for the `with` block; srv.url, srv.port(); log off
http.call(handler, http.Method.get, "/notes/7?full=yes", body: "", headers: [:])   // in memory, no socket: same target parsing and panic boundary as serve
// the client (D127, chapter 17): one request, or a Client with a pool of keep-alive connections
http.fetch(url, method: http.Method.get, headers: [:], body: null /* Payload? */, timeout: null /* Duration?, default the client's 30s, covers the body too */, redirect: true, retry: 0)   // throws FetchError; url is absolute http://…
http.get(url, headers:, timeout:, redirect:, retry:); http.head(…); http.put(url, body:, …); http.delete(url, …); http.post(url, body:, headers:, timeout:, redirect:); http.patch(…)   // the shared default client
with client = http.Client(timeout: Duration.seconds(30), headers: [:], maxRedirects: 10, maxIdlePerHost: 8, idleTimeout: Duration.seconds(30))   // same methods; close() drops idle connections
http.Payload.text(s); .json(value) /* throws EncodeError */; .form([("a", "1"), ("a", "2")]); .bytes(data, contentType: http.MediaType.octetStream)   // data, contentType
res.status; res.headers /* lower-case, repeats joined by ", " */; res.setCookies; res.ok /* 2xx */; res.url /* after redirects */; res.header(name); res.length() /* Content-Length */
try res.text(max: 64 MiB); try res.bytes(max:); try res.json<T>(max:) /* throws FetchError | DecodeError */; res.stream() /* ClientBody: read(max: 65536), length() */; try res.ensureSuccess() /* throws StatusError { url, status } */; res.close()
// error FetchError { url, detail, kind: FetchKind }; enum FetchKind { InvalidRequest, Unsupported, Connect, Timeout, Closed, Io, Protocol, TooManyRedirects, TooLarge }
// redirects (301/302/303/307/308): GET/HEAD only, authorization/cookie/proxy-authorization dropped across hosts; retry: idempotent methods only, 100 ms doubling backoff; https → Unsupported until std/tls
// cookies (D94): one Set-Cookie line each; value percent-encoded; a bad name/path/domain, SameSite.None without secure, a broken __Host-/__Secure- name panic at the call
http.Cookie(name:, value:, path: "/", domain: null, maxAge: null /* Duration?, null = session */, secure: false, httpOnly: true, sameSite: http.SameSite.Lax)   // enum SameSite { Lax, Strict, None }
resp.withCookie(c); resp.withoutCookie(name, path: "/", domain: null, secure: false); resp.cookies      // Max-Age=0 to forget
req.cookie(name); req.cookies()                                                                         // string?; Map<string, string>, first of a repeated name
// forms and query strings (D94): urlencoded, repeats kept
try req.form<T>(keys: KeyStyle.AsWritten); try req.query<T>(keys:)    // T: Decodable, flat; 400 "invalid form:"/"invalid query:" + one line per problem; 415 for another content type
try req.formFields(); req.queryFields(); req.rawQuery                 // http.Fields: pairs; .get(name) first; .all(name); .names(); Fields.parse(text)
try req.formValue(name); try req.formValues(name)                     // string?; List<string>
http.compress(minBytes: 1024, level: 6); http.decompressRequests(max: n)   // D124: gzip textual responses ≥ minBytes for Accept-Encoding: gzip (Vary, weak ETag, streams too); opens gzip request bodies up to max (413/400/415)
// http.files serves name.gz beside name to a client that accepts gzip, unless it asks for a range
http.FormDecoder.of(fields, keys:)                                    // the codec.Decoder behind them, format name "form"
```

## Module `compress`

DEFLATE and gzip in Veles (D124, [chapter 17](../17-http.md)).

```veles
// fragment
use compress
compress.gzip(bytes: List<u8>, level: i64 = 6): List<u8>                       // RFC 1952; level 0 (stored) to 9, else a panic at the caller
compress.gunzip(bytes: List<u8>, max: i64 = 67108864): List<u8> throws CompressError  // members joined; CRC and length checked
compress.deflate(bytes, level = 6): List<u8>; compress.inflate(bytes, max = 67108864): List<u8> throws CompressError   // raw RFC 1951
error CompressError { message, kind: CompressKind }   // enum CompressKind { Corrupt, Truncated, TooLarge } — TooLarge: the output passed max
var w = compress.GzipWriter(to: stream, level: 6)       // io.Stream out: try w.write(bytes); try w.writeText(s); try w.flush() (sync); try w.finish() (trailer; `to` stays open)
var r = compress.GzipReader(from: stream, max: 67108864)  // try r.read(max = 65536): List<u8> (empty at the end) throws IoError | CompressError; try r.readAll()
var e = compress.GzipEncoder(level: 6)                  // no stream: e.push(bytes): List<u8>, e.flush(), e.finish() hand back what is ready
```

`max` is a ceiling on the **output**, 64 MiB unless said otherwise: input
that would decompress past it is a `TooLarge`, so a small file cannot become a
large allocation. Corrupt or cut-short input is a `CompressError`, never a
panic. The encoder picks per block the smallest of stored, fixed and dynamic
Huffman coding, matches with hash chains (lazy from level 4), and writes
incompressible data stored, so the output is at most a few bytes more than the
input. The decoder resumes where the input ends, which is what lets
`GzipReader` work on a stream that arrives a few bytes at a time.

## Module `config`

Settings read into a struct (D125, [chapter 22](../22-configuration.md)).

```veles
// fragment
use config
config.load<T: Decodable>(files: List<string> = [], prefix: string = ""): T throws config.Error   // environment, then files last to first, then field defaults
config.describe<T: Decodable>(prefix: string = ""): List<config.Variable>                          // what T reads: for --help and docs
error Error { problems: List<codec.Problem> }   // one problem per variable or file, path = the variable (`DB_URL: not set (expected text)`)
struct Variable { name: string, what: string, required: bool, secret: bool, default: string? }
// DATABASE_URL for databaseUrl; DB_POOL_SIZE for db.poolSize; @key("X") as written, @key(env: "X") for the environment only
// files: *.json by field name (typed, nested), anything else dotenv (NAME=value, #, export, '…', "…" with escapes; no interpolation)
```

Values: numbers, `bool` (`true`/`false` only), `string`, `Secret<T>`, `Duration`
(`30s`), `Timestamp` (RFC 3339), enums by member name, `List<T>` (comma
separated), `T?` (`null` when unset); a field with a default needs no
variable, any other is required. An empty variable is a value. A struct field
whose variables are all unset takes its default (or `null`). A `Map` field
(or a list of structs) panics at the first call, naming the field: mark it
`@skip` with a default. A `Secret`'s value is never in a message.

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
time.ticker(every: Duration): Ticker  // .ticks: Channel<Timestamp> (holds one; a slow reader misses ticks); Closeable: stops and closes ticks (D110)

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

Everything above but `Duration.zero` (a `static val`) is a `const fun`, so a
constant may be a `Duration`: `const TIMEOUT = Duration.seconds(30)` (D113).

## Prelude type `Secret<T>` (D112)

```veles
// fragment
Secret.of(v): Secret<T>               // T = string or List<u8> only; a private copy of v
s.expose(): T                         // a fresh copy; panics at the caller after close()
s.len(): i64                          // bytes; the length is not secret
"$s"                                  // [redacted] — so are expect captures and derived Display
s == t                                // constant time in the contents; never hashed or ordered
s.close()                             // zeroes the bytes now; the collector zeroes them when freed
```

`Decodable` (a config file or the environment fills it), not `Encodable`:
deriving `Encodable` over a `Secret` field asks for `@skip` or a hand-written
`encode`. `Sendable`. What `expose()` returns is ordinary memory.

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

Once `otel` runs, each line is also an OTLP log record (fields as attributes, severity, the ids of the current span) and, inside a span, carries `trace_id`.

## Module `otel`

```veles
// fragment
use otel
with tel = try otel.start(service: "notes", exporter: http.otlp(endpoint: "http://localhost:4318", headers: ["authorization": Secret.of(key)]), interval: Duration.seconds(10), resource: [otel.attr("service.version", "1.2")], sampleRatio: 1.0, maxQueue: 2048)   // D126; throws StartError when one is running; before start everything is a no-op
tel.flush(): bool; tel.shutdown(): bool   // send what is queued now / send it and stop recording — call before the program ends: a `with` cannot suspend, so closing only warns about what was lost
with span = otel.span("load note", attrs: [otel.attr("note.id", id)], kind: otel.SpanKind.Internal, parent: null)   // current span to the end of the block, tasks started inside see it; kinds Internal | Server | Client | Producer | Consumer
span.set(attr); span.event(name, attrs); span.ok(); span.fail(error); span.failWith(text); span.rename(name); span.end(); span.discard(); span.context(): SpanContext; span.handle: SpanHandle
otel.inSpan("name", () => try work(), attrs:, kind:)   // ok on return, failed (and rethrown) on a thrown error
otel.current(): SpanHandle?; otel.traceparent(): string?; otel.traceId(): string?; otel.active(): bool; otel.dropped(): i64
otel.SpanContext.parse(text): SpanContext?   // W3C traceparent, strictly; ctx.traceparent(), traceIdHex(), spanIdHex(), sampled
otel.attr(key, value)   // value: string | bool | i64 | i32 | f64 | List<string>
val m = otel.meter("notes"); m.counter(name, unit:, description:).add(n, attrs); m.upDownCounter(…).add(n); m.gauge(…).set(x: f64); m.histogram(name, buckets: otel.defaultBuckets).record(x /* f64 | i64 | Duration (seconds) */, attrs)
trait otel.Exporter : Sendable { fun export(signal: Signal, body: List<u8>) suspends throws ExportError }   // Signal: Traces | Metrics | Logs; otel.signalPath(signal)
error ExportError { message, retryable: bool = true }; error StartError { message }
http.otlp(endpoint, headers: Map<string, Secret<string>> = [:], timeout: 10s, gzip: true): otel.Exporter   // OTLP/HTTP protobuf to <endpoint>/v1/{traces,metrics,logs}; http only until E1
```

Finished spans and log records queue up to `maxQueue` each and are dropped, and counted (`otel.dropped()`, the metric `otel.dropped`), when the collector cannot be reached after four tries; an instrument holds at most 2000 attribute sets. `http.serve` opens a server span per request and `http.fetch` a client span, both continuing the W3C `traceparent`. See [chapter 23](../23-observability.md).

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
crypto.hmacSha256(key: Secret<List<u8>>, message: List<u8>): Digest  // also 384, 512, Sha1Legacy
crypto.hmac<H: Hasher>(key: Secret<List<u8>>, message: List<u8>): Digest
crypto.Hmac<Sha256>.start(key: List<u8>): Hmac<Sha256>       // update/finish as above
crypto.equalBytes(a: List<u8>, b: List<u8>): bool            // constant time
crypto.randomBytes(n: i64): List<u8>         // the OS CSPRNG; panics if it refuses
crypto.randomU64(): u64
crypto.uuidV4(): Uuid                        // 122 random bits
crypto.uuidV7(): Uuid                        // time-ordered, strictly increasing
crypto.hashPassword(password: Secret<string>): string        // Argon2id, PHC string, 19 MiB / 2 passes / 1 lane, random salt
crypto.verifyPassword(password: Secret<string>, hash: string): bool   // constant time; false for anything that is not an Argon2id hash or asks for too much
```

`Digest`: `bytes()`, `len()`, `toHex()`, `toBase64Url()`, `prefix(n)`,
`Digest.of(bytes)`; `==` is constant time, `"$d"` is lower-case hex.

`Uuid`: `version()`, `timestamp()` (v7 only), `bytes()`, `isZero()`,
`Codable` (its text form), `Uuid.parse(text): Uuid?`, `Uuid.of(bytes)`, `Uuid.zero()`; `Comparable`,
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
`encode` and `encodeUpper` are `const fun`s (D113); `decode` throws, which a
`const fun` cannot yet.

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
character. `base64.Invalid { message, position }`. The encoders and
`encodedLen` are `const fun`s (D113); the decoders throw, which a `const fun`
cannot yet.

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
code point writes U+FFFD rather than failing. See `examples/utf8`. Every
function here is a `const fun` (D113).

## Module `jwt`

```veles
// fragment
use jwt
jwt.sign(claims: Claims, key: Secret<List<u8>>, algorithm: Algorithm = Algorithm.HS256,
         keyId: string? = null): string throws EncodeError
jwt.verify(token: string, key: Secret<List<u8>>, options: Options = Options()): Claims throws jwt.Invalid
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

UDP, a format-string module, iterating a directory tree, running a
program with its own stdin or a separate stderr capture. Each is a small
`extern "C"` binding away (see [chapter 13](../13-memory-and-ffi.md)) or
plain Veles on top of `fs`; the library grows with the compiler's own needs
setting the order.
