// Integer and float arithmetic at every width, checked division, bitwise
// operators, and every kind of numeric cast.
use io

fun ints(a: i64, b: i64): i64 => a + b - a * b / (b + 1) % 7
fun wrapping(a: i64, b: i64): i64 => (a +% b) -% (a *% b)
fun narrow(a: i32, b: i32): i32 => a * b - a / b
fun unsigned(a: u32, b: u32): u32 => a / b + a % b
fun bits(a: u64, b: u64): u64 => (a & b) | (a ^ b) << 3 >> 1
fun negate(a: i64): i64 => ~a
fun floats(a: f64, b: f64): f64 => a * b + a / b
fun single(a: f32, b: f32): f32 => a * b

fun casts(x: i64, f: f64) {
  val a = x.wrapU8()
  val b = x.wrapI16()
  val c = a.toI64()
  val d = b.wrapU32()
  val e = x.toF64()
  val g = f.toI64()
  val h = f.toU8()
  val i = f.toF32()
  val j = i.toF64()
  val k = x.wrapU64().toF32()
  io.println("$a $b $c $d $e $g $h $i $j $k")
}

fun main() {
  io.println("${ints(40, 3)} ${wrapping(9, 4)} ${narrow(7, 2)} ${unsigned(17, 5)}")
  io.println("${bits(12, 10)} ${negate(5)} ${floats(7.5, 2.0)} ${single(1.5, 2.0)}")
  casts(300, 1e20)
  casts(-1, -3.7)
  saturating(4611686018427387904, 100, 200, -2147483648)
  saturating(3, -5, 7, 11)
}

fun saturating(a: i64, b: i8, c: u8, d: i32) {
  io.println("${a.saturatingMul(2)} ${a.saturatingMul(-3)} ${b.saturatingMul(-2)} ${c.saturatingMul(2)} ${d.saturatingMul(-1)}")
}
