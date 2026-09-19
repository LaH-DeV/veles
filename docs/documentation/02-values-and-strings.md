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
  ratio = n as f64 / 2.0
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
`i32` and `i64` in one expression is an error; say what you mean with
`as`:

```veles
use io

fun main() {
  val a = 7
  val b = 2
  io.println("${a / b} ${a % b} ${a as f64 / b as f64}")
  val small: i32 = 7
  val wide = small as i64 * 1000000000
  val narrow = 300 as u8        // wraps: 44
  io.println("$wide $narrow ${-7 / 2}")
}
```

Output:
```text
3 1 3.5
7000000000 44 -3
```

Integer division truncates towards zero. `as` between numeric types
converts and, when narrowing, wraps. A float cast to an integer drops
the fraction and saturates: `1e20 as i64` is the largest `i64`,
`-1.0 as u8` is 0, and NaN becomes 0.

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
is `2 + (4 & 7)` (Go's grouping, which reads the way you mean it). A
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
  `"héllo".chars()` is `[h, é, l, l, o]`.
- `substring(from, to)` takes byte offsets and returns `string?` — null
  when the offsets fall outside the text or would cut a multi-byte
  character, so a parser can probe `s.substring(pos, pos + 4)` near the
  end without a bounds check. The `?: "?"` after
  it supplies a fallback; nullable types get [a chapter of their own](06-nullable-types.md).

There is no separate character type; a one-character string is what you
use. When you walk a string byte by byte (`s.byteAt(i)`), `'"'` is the
`u8` of one ASCII character, so `b == '"'` and `b >= '0' && b <= '9'` read
as intended; a non-ASCII character is not a byte literal.

Escapes in string literals: `\n`, `\t`, `\\`, `\"`, `\$` (a literal
dollar) and `\u{1F600}` for a code point.

## Booleans

`bool` is `true` or `false`. `&&` and `||` short-circuit; `!` negates.
Comparison operators `== != < <= > >=` work on numbers and strings and,
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

fun divmod(a: i64, b: i64): (i64, i64) = (a / b, a % b)

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
  (xs.atOrPanic(0), xs.atOrPanic(2)) = (xs.atOrPanic(2), xs.atOrPanic(0))
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

Next: [Functions and control flow](03-functions-and-control-flow.md).
