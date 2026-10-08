# 2. Values, types and strings

## Naming a value

```veles
use io

fun main() {
  val greeting = "hello"     // cannot be reassigned
  var count = 0              // can be
  count += 1
  count = count * 10
  io.println("$greeting $count")
}
```

Output:
```text
hello 10
```

`val` introduces a name that is bound once; `var` one that can be
assigned again. Prefer `val`; reach for `var` only when you need to
change the binding, and the compiler will tell you when you try to assign
a `val`. A binding that is never read gets a warning; name it `_`
when you mean to discard the value.

Types are inferred from the initializer. You can always write one:

```veles
use io

fun main() {
  val n: i64 = 5
  var ratio: f64 = 0.5
  ratio = n.toF64() / 2.0
  io.println("$ratio")
}
```

## Numbers

| Type | Meaning |
|---|---|
| `i8` `i16` `i32` `i64` | signed integers of that many bits |
| `u8` `u16` `u32` `u64` | unsigned |
| `f32` `f64` | floating point |

An integer literal is `i64` unless the context asks for something else —
the same type `len()`, indices and counts use, so ordinary code has one
integer type and no casts. A literal with a decimal point or exponent is
`f64`. Literals adapt to an expected type, so `val x: u8 = 200` and
`val small: i32 = 7` both work without a cast, but `300` where a `u8` is
expected is an error because it does not fit. Reach for the narrower
types when layout or memory matters (struct fields in bulk data, C
interop, bit manipulation).

There is **no implicit conversion** between numeric types (D21). Mixing
`i32` and `i64` in one expression is an error; say what you mean with a
conversion method (D86). Each one names what it can lose:

```veles
use io

fun main() {
  val a = 7
  val b = 2
  io.println("${a / b} ${a % b} ${a.toF64() / b.toF64()}")
  val small: i32 = 7
  val wide = small.toI64() * 1000000000     // cannot lose: an i64
  val narrow = 300.wrapU8()                 // keeps the low 8 bits: 44
  val fits = 200.toU8()                     // may lose: a u8?, here 200
  val count = 300
  val big = count.toU8()                    // out of range: null
  io.println("$wide $narrow $fits $big ${-7 / 2}")
}
```

Output:
```text
3 1 3.5
7000000000 44 200 null -3
```

Integer division truncates towards zero. The conversions are methods on
every numeric type:

- `x.toT()` where nothing can be lost (`u8` to `i64`, an integer to a
  float, `f32` to `f64`) returns a `T`. Where a value could be lost
  (`i64` to `u8`, a float to an integer) it returns a `T?` that is null
  when the value does not fit. A float truncates toward zero and NaN is
  null: `1e20.toI64()` and `(-1.0).toU8()` are null, `2.9.toI32()` is 2.
- `x.wrapT()` between integers keeps the low bits, the conversion `+%`
  is to `+`: `300.wrapU8()` is 44, `(-1).wrapU8()` is 255. Generic code that only knows its target as a
  type parameter writes it as a type argument: `n.wrapTo<T>()`.

A literal that cannot fit is an error at compile time (`300.toU8()`
says to use `wrapU8()` if the low bits are what you want).

### Overflow is an error, unless you ask for wrapping

`i32` cannot hold 2147483648. In a debug build `2147483647 + 1` panics
at run time; in a `--release` build the check is dropped. When wrapping
is what you want — hashes, counters, checksums — the wrapping operators
`+%`, `-%`, `*%` say so in the source:

```veles
use io

fun main() {
  val max: i32 = 2147483647
  io.println("${max +% 1}")
}
```

Output:
```text
-2147483648
```

### Bits and formatting

Integers have the bitwise operators `&`, `|`, `^`, `<<`, `>>` and `~`.
`&` and the shifts bind like `*`, `|` and `^` like `+`, so `2 + 4 & 7`
is `2 + (4 & 7)` (grouping which reads the way you mean it). A
shift by the width or more gives 0 rather than something undefined.
Literals may be written in hex or binary: `0xFF`, `0b1010`.

To shape a number as text, ask it: `n.toString(radix: 16)` for hex and
`x.toFixed(2)` for two decimals, then the string methods for width —
`n.toString().padStart(6, "0")`.

```veles
use io

fun main() {
  val flags: u8 = 0b1010
  io.println("${flags & 0b0010} ${flags | 1} ${flags << 4} ${(255).toString(radix: 2)} ${(-1).toString(radix: 16)}")
  io.println("${(2.0 / 3.0).toFixed(3)} ${(42).toString().padStart(6, "0")} ${(1234.56).toFixed(0)}")
}
```

Output:
```text
2 11 160 11111111 -1
0.667 000042 1235
```

A float's bits are one call away, without `unsafe` (D93): `x.toBits()` is
the IEEE 754 pattern as a `u64` (`u32` for an `f32`), and `f64.fromBits(bits)`
gives the number back. Nothing is rounded and a NaN keeps its payload, so the
pair is exact; it is what writes a constant the way LLVM wants it, or stores a
float in a binary format.

```veles
use io

fun main() {
  val bits = (1.5).toBits()
  io.println("${bits.toString(radix: 16)} ${f64.fromBits(bits)}")
  io.println("${(-0.0).toBits() == (0.0).toBits()} ${-0.0 == 0.0}")
}
```

Output:
```text
3ff8000000000000 1.5
false true
```

## Text

`string` is immutable UTF-8 text. Concatenate with `+`, compare with
`==` and `<`, and build strings with **interpolation**: `$name` for a
plain name, `${expr}` for anything else. Any value can be interpolated —
numbers, booleans, lists, your own structs.

```veles
use io

fun main() {
  val name = "Veles"
  val year = 2026
  io.println("$name was born in $year; next year is ${year + 1}")
  val s = "héllo"
  io.println("${s.len()} bytes, starts with h: ${s.startsWith("h")}, has ll: ${s.contains("ll")}")
  io.println("${s.substring(0, 1) ?: "?"} ${"abc" < "abd"} ${"a" + "b" == "ab"}")
  io.println("${s.substring(1, 2)} ${s.substring(4, 9)} ${s.substring(3, 6)}")
  io.println("${s.charCount()} characters: ${s.chars()}")
}
```

Output:
```text
Veles was born in 2026; next year is 2027
6 bytes, starts with h: true, has ll: true
h true true
null null llo
5 characters: [h, é, l, l, o]
```

Two things worth knowing early:

- `len()` counts **bytes**, not characters (D18): `"héllo"` is 6 bytes.
  Veles does not pretend a string is an array of characters, so there
  is no character-at-index either (`s.byteAt(i)` gives a byte). When you do need characters, `charCount()`
  counts them and `chars()` gives them as a `List<string>` of
  one-character strings: `"héllo".charCount()` is 5 and
  `"héllo".chars()` is `[h, é, l, l, o]`. Both build a whole list; when you
  need to walk characters *without* building one — a scanner, a lexer, a
  validator over bytes off a socket — `use utf8` steps one at a time:
  `utf8.decode(s, i)` gives the code point at a byte offset and how wide it
  is, `utf8.encodeTo(out, code)` writes one back
  ([reference](reference/stdlib.md#module-utf8)).
- `substring(from, to)` takes byte offsets and returns `string?` — null
  when the offsets fall outside the text or would cut a multi-byte
  character, so a parser can probe `s.substring(pos, pos + 4)` near the
  end without a bounds check. The `?:` after
  it supplies a fallback; nullable types get [a chapter of their own](06-nullable-types.md).

There is no separate character type; a one-character string is what you
use. When you walk a string byte by byte (`s.byteAt(i)`), `'"'` is the
`u8` of one ASCII character, so `b == '"'` and `b >= '0' && b <= '9'` read
as intended; a non-ASCII character is not a byte literal.

Escapes in string literals: `\n`, `\t`, `\\`, `\"`, `\$` (a literal
dollar) and `\u{1F600}` for a code point.

### Template literals

A name written **straight before** a string — no space — hands the string to
a function marked `@template`, in two lists: the pieces of text, and the
values you interpolated. The function decides what a value means. Joining
them into one string is exactly what it need not do, so the result can be a
type a plain string can never be: markup whose values were escaped, a query
whose values travel apart from its text (D129).

```veles
use io { println }

trait Markup {
  fun markup(): string
}

implement Markup for string {
  fun markup(): string => this.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
}

implement Markup for i64 {
  fun markup(): string => "$this"
}

struct Html {
  text: string
}

// `html"a ${x} b"` calls html(["a ", " b"], [x]): one more piece than values
@template
fun html(parts: List<string>, values: List<Markup>): Html {
  val out = StringBuilder()
  loop ((i, part) in parts.enumerate()) {
    out.append(part)
    val v = values.at(i)
    if (v != null) out.append(v.markup())
  }
  Html(text: out.toString())
}

fun main() {
  val comment = "1 < 2 & 3 > 2"
  val page = html"<p>${comment}</p><p>${7} votes</p>"
  println(page.text)
}
```

Output:
```text
<p>1 &lt; 2 &amp; 3 &gt; 2</p><p>7 votes</p>
```

The rules: the parameters are `(parts: List<string>, values: List<V>)` —
`V` is usually a trait, so each value's type must implement it (a value that
does not is an error at its `${…}`); the pieces are what the source
says, never input, and there is always one more than the values; `tag"…"` is
the call `tag(pieces, values)` and has the function's result type, which is
not a `string`. A name that is not `@template` cannot tag a literal, and a
space (`html "…"`) is an error that says so. A module's function is tagged
`db.sql"…"`. `examples/templates` writes an `html` and an `sql` tag; a
database module (`std/db`) uses the same to make injection impossible.

## Booleans

`bool` is `true` or `false`. `&&` and `||` short-circuit; `!` negates.
Comparison operators `==`, `!=`, `<`, `<=`, `>`, `>=` work on numbers and strings and,
as you will see, on structs and tuples too.

```veles
use io

fun main() {
  val x = 10
  val inRange = x >= 0 && x < 100
  io.println("$inRange ${!inRange} ${x == 10 || x == 11}")
}
```

Output:
```text
true false true
```

## Tuples

A tuple groups a few values without naming a type: `(1, "one")` has type
`(i64, string)`. Read elements with `.0`, `.1`, or destructure:

```veles
use io

fun divmod(a: i64, b: i64): (i64, i64) => (a / b, a % b)

fun main() {
  val pair = divmod(17, 5)
  val (q, r) = divmod(17, 5)
  io.println("${pair.0} ${pair.1} $q $r ${pair == (3, 2)}")
}
```

Output:
```text
3 2 3 2 true
```

Destructuring also works as an *assignment* to existing places. The right
side is evaluated completely before anything is stored, so a swap needs no
temporary — and the same goes for list elements and struct fields:

```veles
use io

fun main() {
  var a = 1
  var b = 2
  (a, b) = (b, a)
  var xs = mut [10, 20, 30]
  // exchanges two elements; panics if either index is out of range
  xs.swap(0, 2)
  var fibA = 0
  var fibB = 1
  loop (_ in 0..<10) (fibA, fibB) = (fibB, fibA + fibB)
  io.println("$a $b $xs $fibA")
}
```

Output:
```text
2 1 [30, 20, 10] 55
```

Only `=` destructures; `(a, b) += (1, 1)` is an error.

## Constants

Module-level `val` and `const` hold values shared by the whole module:

```veles
use io

const limit = 3
val banner = "== report =="

fun main() {
  io.println("$banner $limit")
}
```

Output:
```text
== report == 3
```

A `const` is computed when the program is compiled (D113). It can use other
constants, arithmetic, interpolation, `len()`, conversions, `if` and `when`,
tuples, enum members and structs, and it can be a read-only `List`, `Map` or
`Set` — a table laid out once in the binary, never built at start-up. Every
use of a constant is its value written in, so it also works as a `when`
pattern. `static assert(cond, "why")` checks a constant condition at compile
time:

```veles
use io

const KB: i64 = 1024
const PAGE = KB * 4
const LABEL = "page of ${PAGE} bytes"
const PRIMES: List<i64> = [2, 3, 5, 7]
const PORTS: Map<string, i64> = ["http": 80, "https": 443]

static assert(PAGE == 4096, "a page is four kilobytes")

fun main() {
  io.println("$LABEL, ${PRIMES.at(2)}, ${PORTS.get("https") ?: 0}")
  when (8192) {
    PAGE => io.println("one page")
    else => io.println("not one page")
  }
}
```

Output:
```text
page of 4096 bytes, 5, 443
not one page
```

What would fail at run time fails the build instead, in every profile:
`const BIG: i64 = 9223372036854775807 + 1` is a compile error (write `+%` to
wrap on purpose), and so are a division by zero and `PRIMES.at(9)`. With a
constant index, `PRIMES.at(2)` is an `i64`, not an `i64?`: the compiler has
read it. A constant cannot read a `val`, and the only functions it can call are
the ones declared `const fun` (next) — compute anything else with `val`.
Module-level `val`s are computed before `main`, in the order they need each
other.

### Functions the compiler runs

A constant may call a function declared `const fun`. The compiler runs it
while it compiles, and the result is laid out in the binary like any other
constant table: nothing is computed when the program starts. A `const fun` is
still an ordinary function — call it at run time and it computes the same
thing.

```veles
use io

const fun crc32Table(): List<u32> {
  val table: MutableList<u32> = []
  loop (n in 0..<256) {
    var c = n.wrapU32()
    loop (_ in 0..<8) {
      c = if (c & 1 == 1) 0xEDB88320 ^ (c >> 1) else c >> 1
    }
    table.push(c)
  }
  table.toList()
}

const CRC: List<u32> = crc32Table()

fun main() {
  io.println("${CRC.len()} entries; the last is ${CRC.at(255)}")
}
```

Output:
```text
256 entries; the last is 755167117
```

A `const fun` may use its parameters, locals and loops, `if` and `when`,
recursion, structs, tuples and arrays as values, a `MutableList`, `MutableMap`
or `StringBuilder` it builds and returns as a `List`, `Map` or `string`, its own
methods, other `const fun`s and the standard library's: the string functions
(`trim`, `split`, `replace`, `indexOf`, `toUpper`, `padStart`…),
`StringBuilder`, the integer operations (`abs`, `pow`, `rotateLeft`,
`wrappingAdd`, `countOnes`…), `sqrt`, rounding, `min` and `max`, and an enum's
`values()`, `parse` and `toString`. A `const fun` can be generic, a method
can be one (`const fun bumped(): Version`), and it may take, make and call
lambdas — `xs.map(x => x * 2)`, `words.sortedBy(w => w.len())`, a lambda that
changes a `var` it captured — so long as each lambda keeps the same rules.

It may not do input or output, suspend or start a task, use `unsafe`, read a
module-level `val` or `var`, call through a trait object, or
`throw`. Each is an error naming the rule, and the rules are checked where the
function is declared — not where a constant uses it — so a change inside a
`const fun` cannot quietly break a constant in another file. The one rule a
generic `const fun` cannot settle at its declaration is whether its type
argument's own methods qualify: `xs.max()` calls `T`'s `compareTo`, which is a
`const fun` for numbers and strings but may be an ordinary one for your type.
Such a call works at run time as always; a constant that needs it is refused,
at the constant, naming the method to mark.

When it fails, the build fails at the constant that called it, with the message
and the chain of calls: a `panic`, an overflow, an index out of range, a
division by zero. A function that never ends is stopped too: evaluating one
constant has a budget of 10 million steps (`veles build --const-steps 50000000`
raises it) and 4096 nested calls, and running past either is an error naming
the constant.

The results are what the program would compute. Integers are checked as
everywhere (a constant never wraps unless you write `+%` or `wrapU32()`), an
`f32` is rounded after every operation, and `"${x}"` prints as it does at run
time. The one family left out is `sin`, `cos`, `exp`, `log`, `pow` and the other
functions of the C library: they differ between platforms in the last bit, and a
constant must not depend on which library the program is linked with — compute
those at run time.

#### How it works

The compiler checks a `const fun` like any other function, and then runs the
checked function in an interpreter of its own — the same code, not a second
language. That is why the results equal the program's, and why the rules are
what they are: the interpreter holds values, not memory, so it cannot follow
a raw pointer (`unsafe`), call C, read a file, wait for a task, or read a
`val` that the program computes only when it starts.

#### What it may call

| A `const fun` may call | It may not call |
|---|---|
| other `const fun`s, generic or not, and `const fun` methods | an ordinary `fun` — mark it `const` if it qualifies |
| a struct's constructor and its `init` block; lambdas, and `const fun`s passed as values | a call through a trait object (`val s: Shape = …; s.area()`) |
| std's helpers that take a function: `map`, `filter`, `fold`, `forEach`, `any`, `all`, `find`, `count`, `flatMap`, `mapNotNull`, `partition`, `sortedBy`, `sortedWith`, `minBy`, `maxBy`, `distinctBy`, `binarySearch`…, `MutableList.make`; a `Map`'s `mapValues`, `filter`, `forEach`, `getOrPut` | a lambda that does any of the things on this side |
| `MutableList`, `MutableMap`, `MutableSet`: `push`, `pop`, `set`, `get`, `at`, `remove`, `clear`, `addAll`, `slice`, `reserve`, `toList`… | anything that suspends: `sleep`, channels, `async`, `await` |
| `string`: `len`, `substring`, `split`, `startsWith`, `contains`, `toInt`, `bytes`, and std's `const fun` string helpers (`trim`, `replace`, `indexOf`, `padStart`, `toUpper`, `repeat`, `lines`…) | `io`, `fs`, `os`, `net`, `time`, `random` |
| `StringBuilder`, interpolation, `toString`, `compareTo` | `unsafe`, `extern` functions |
| integer and float arithmetic, conversions (`toI64()`, `wrapU8()`…), `abs`, `min`, `max`, `sqrt`, rounding | `sin`, `cos`, `exp`, `log`, `pow` on floats |
| an enum's `values()`, `parse`, `toString`; `panic` | `throw` (a failure in the compiler is a `panic`) |
| `List` helpers that take no function: `take`, `drop`, `concat`, `distinct`, `chunked`, `windowed`, `enumerate`, `zip`, `min`, `max`, `sum`, `sortedDescending`, `toSet`; `MutableList`'s `insert`, `removeAt`, `swap`, `fill`, `sort`, `repeat` | a `Result` (`Ok`, `Err`) or another sealed variant as a value |
| ranges: `len`, `contains`, `step`, `reversed`, and looping over them | the decoders that throw: `hex.decode`, `base64.decode` |
| `Duration`: its constructors, operators, accessors, `toString` and `parse`; `hex.encode`, `base64.encode`, `base64.encodeUrl`; `std/utf8` | |

If you call something on the right, the error names the rule at the function's
declaration, so you find out when you write the `const fun`, not when someone
uses it.

#### When it fails

A `panic` while a constant is computed stops the build. The error is at the
constant, and lists the calls that led to the panic, innermost first:

```veles
// fragment
const fun digit(c: string): i64 {
  val n = "0123456789".indexOf(c)
  if (n < 0) panic("'$c' is not a digit")
  n
}

const fun parsePort(text: string): i64 {
  var port: i64 = 0
  loop (c in text.split("")) port = port * 10 + digit(c)
  port
}

const PORT: i64 = parsePort("80a0")
```

```text
app\main.vs:13:19: error: panic in a constant: 'a' is not a digit
  at app\main.vs:3:14
  in const fun 'digit', called at app\main.vs:9:49
  in const fun 'parsePort', called at app\main.vs:13:19
  const PORT: i64 = parsePort("80a0")
                    ^^^^^^^^^^^^^^^^^
```

A loop that never ends is caught by the step budget instead:
`error: evaluating the constant 'STUCK' took more than 10000000 steps; make it
cheaper, or raise the budget with '--const-steps n'`.

#### `const fun`, `const` or `val`?

- A plain `const` when the value is a literal or a little arithmetic on other
  constants: `const PAGE = KB * 4`.
- A `const fun` when building the value needs a loop or a helper — a lookup
  table, a parsed version string, a precomputed list. It costs build time (each
  constant has its step budget) and binary size (the table is stored in the
  executable), and nothing at start-up.
- A module-level `val` when the value needs something only the running
  program has (the environment, a file, the clock), or a function the compiler
  cannot run. It is computed once, before `main`.

The standard library marks its helpers `const fun` where they qualify, so a
program can use them in constants; a function that is not marked cannot be
called from one, even if its body would qualify, because the mark is the
promise that it keeps qualifying.

Next: [Functions and control flow](03-functions-and-control-flow.md).
