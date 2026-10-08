// D104: each bit operation is one LLVM intrinsic (fshl, fshr, bswap,
// bitreverse, copysign); a swap of one byte is no instruction at all
use io

fun rotl(x: u32, n: i64): u32 => x.rotateLeft(n)
fun rotr(x: i16, n: i64): i16 => x.rotateRight(n)
fun swap(x: u64): u64 => x.swapBytes()
fun swap8(x: u8): u8 => x.swapBytes()
fun rev(x: i64): i64 => x.reverseBits()
fun sign(x: f64, y: f64): f64 => x.copySign(y)
fun negative(x: f32): bool => x.isSignNegative()

fun main() {
  val a: u32 = 0x80000001
  val b: i16 = 3
  val c: u8 = 7
  io.println("${rotl(a, 4)} ${rotr(b, 1)} ${swap(1)} ${swap8(c)} ${rev(1)} ${sign(2.0, -0.0)} ${negative(-0.0)}")
}
