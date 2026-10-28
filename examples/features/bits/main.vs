// D104: rotate, swap and reverse on every integer width, and the sign and
// neighbours of floats — the edges of each, which must agree in debug and
// release builds.
use io { println }

fun ints() {
  // rotation keeps every bit: what leaves one end comes back at the other
  val a: u8 = 0b1000_0001
  val b: u8 = 0x0F
  println("u8  ${a.rotateLeft(1)} ${a.rotateRight(1)} ${b.rotateLeft(12)} ${b.rotateLeft(-1)}")
  val minI8: i8 = -128
  val oneI8: i8 = 1
  println("i8  ${minI8.rotateLeft(1)} ${oneI8.rotateLeft(-1)} ${minI8.reverseBits()}")
  val c: u16 = 0x00FF
  val d: u16 = 0x1234
  val oneU16: u16 = 1
  println("u16 ${c.rotateLeft(8)} ${d.swapBytes()} ${oneU16.reverseBits()}")
  val minI16: i16 = -32768
  val e: i16 = 0x0102
  println("i16 ${minI16.rotateRight(15)} ${e.swapBytes()}")
  val f: u32 = 0x12345678
  val top: u32 = 0x80000000
  val oneU32: u32 = 1
  println("u32 ${f.swapBytes()} ${oneU32.reverseBits()} ${top.rotateLeft(33)}")
  val minI32: i32 = -2147483648
  val g: i32 = 0x7F
  println("i32 ${minI32.rotateLeft(1)} ${g.swapBytes()}")
  val h: u64 = 0x0102030405060708
  val oneU64: u64 = 1
  println("u64 ${h.swapBytes()} ${oneU64.reverseBits()} ${oneU64.rotateRight(1)}")
  val minI64 = -9223372036854775807 - 1
  println("i64 ${minI64.rotateLeft(1)} ${1.rotateLeft(64)} ${1.rotateLeft(-64)} ${(-1).reverseBits()} ${1.rotateLeft(-1) == minI64}")
  val oneUsize: usize = 1
  val oneIsize: isize = 1
  println("usize ${oneUsize.rotateLeft(63)} isize ${oneIsize.swapBytes()}")
  // one byte has nothing to swap; reversing twice is the identity
  val byte: u8 = 0xAB
  println("u8 swap ${byte.swapBytes()} twice ${byte.reverseBits().reverseBits()}")
}

fun floats() {
  val nan = 0.0 / 0.0
  val inf = 1.0 / 0.0
  println("sign ${1.5.isSignNegative()} ${(-1.5).isSignNegative()} ${(-0.0).isSignNegative()} ${0.0.isSignNegative()} ${(-nan).isSignNegative()}")
  println("copySign ${3.0.copySign(-1.0)} ${(-3.0).copySign(0.0)} ${2.0.copySign(-0.0)} ${inf.copySign(-1.0)}")
  println("nextUp ${0.0.nextUp() == f64.fromBits(1)} ${(-0.0).nextUp() == f64.fromBits(1)} ${1.0.nextUp() - 1.0 == 2.0.pow(-52.0)} ${inf.nextUp() == inf} ${nan.nextUp().isNaN()}")
  println("nextDown ${0.0.nextDown() == -f64.fromBits(1)} ${1.0.nextDown() < 1.0} ${(-inf).nextDown() == -inf} ${inf.nextDown() < inf}")
  println("max ${f64.fromBits(0x7FEFFFFFFFFFFFFF).nextUp() == inf} ${(-inf).nextUp() == -f64.fromBits(0x7FEFFFFFFFFFFFFF)}")
  val one: f32 = 1.0
  val two: f32 = -2.0
  println("f32 ${one.nextUp() > one} ${one.copySign(two)} ${(-one).isSignNegative()} ${one.nextUp().nextDown() == one}")
}

fun main() {
  ints()
  floats()
}
