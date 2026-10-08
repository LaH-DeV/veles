// D21 per build profile. This source is two fixtures: `overflow` builds it
// as debug — an overflow panics, and `gather` turns the panic into a value
// (D52) — and `overflow-release` as release, where `+` wraps. In both, `+%`
// wraps, `as` truncates only where it is written, and an inclusive range
// ends at the type's maximum.
use io

fun add(a: i8, b: i8): i8 => a + b
fun sub(a: u8, b: u8): u8 => a - b
fun mul(a: i64, b: i64): i64 => a * b
fun neg(a: i32): i32 => -a
fun quot(a: i64, b: i64): i64 => a / b
fun absolute(a: i16): i16 => a.abs()
fun bump(a: i8): i8 {
  var x = a
  x += 1
  x
}
fun wrapped(a: i8, b: i8): i8 => a +% b
fun narrow(x: i64): u8 => x.wrapU8()

fun sumUpTo(hi: u8): i64 {
  var n = 0
  loop (i in 250..hi) {
    n += i.toI64()
  }
  n
}

// ranges reaching the ends of their type: none may wrap, loop for ever or
// overflow while measuring itself
fun edges(lo: i8, hi: i8, top: u8) {
  io.println("(250..255).iter() yields ${((top - 5)..top).iter().count()}")
  io.println("(-128..127).len() = ${(lo..hi).len()}, (0..<-128).len() = ${((lo +% 127 +% 1)..<lo).len()}")
  io.println("(-128..127).step(100) = ${(lo..hi).step(100).toList()}")
  io.println("reversed then step = ${(lo..hi).reversed().step(100).toList()}, step then reversed = ${(lo..hi).step(100).reversed().toList()}")
}

fun call(f: sendable fun(): i64): i64 => f()

fun attempt(what: string, f: sendable fun(): i64) {
  val outcome = gather {
    async call(f)
  }
  when (val r = outcome) {
    is Ok(v)  => io.println("$what = $v")
    is Err(e) => io.println("$what: ${e.message}")
  }
}

fun main() {
  attempt("127 + 1 (i8)", () => (add(127, 1)).toI64())
  attempt("0 - 1 (u8)", () => (sub(0, 1)).toI64())
  attempt("MIN * -1", () => mul(-9223372036854775807 - 1, -1))
  attempt("-MIN (i32)", () => (neg(-2147483647 - 1)).toI64())
  attempt("MIN / -1", () => quot(-9223372036854775807 - 1, -1))
  attempt("MIN.abs() (i16)", () => (absolute(-32768)).toI64())
  attempt("x += 1 at 127 (i8)", () => bump(127).toI64())
  io.println("127 +% 1 (i8) = ${wrapped(127, 1)}")
  io.println("300 as u8 = ${narrow(300)}, -1 as u8 = ${narrow(-1)}")
  io.println("sum of 250..255 (u8) = ${sumUpTo(255)}")
  edges(-128, 127, 255)
}
