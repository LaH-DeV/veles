// Float bit patterns (D93): what a code generator needs to write a double
// constant exactly, and what a binary format needs to store one.
use io { println }

/// LLVM's textual form of a double constant: the bit pattern in hex, 16
/// digits, so nothing is lost to decimal rounding.
fun llvmDouble(x: f64): string {
  val digits = x.toBits().toString(radix: 16).toUpper()
  "0x" + "0".repeat(16 - digits.len()) + digits
}

/// The same for a float; LLVM writes it as the double it widens to, so this
/// shows the raw 32 bits instead.
fun rawFloat(x: f32): string {
  val digits = x.toBits().toU64().toString(radix: 16).toUpper()
  "0x" + "0".repeat(8 - digits.len()) + digits
}

fun main() {
  println("1.5       ${llvmDouble(1.5)}")
  println("0.1       ${llvmDouble(0.1)}")
  println("-0.0      ${llvmDouble(-0.0)}")
  println("pi        ${llvmDouble(3.141592653589793)}")
  println("infinity  ${llvmDouble(1.0 / 0.0)}")

  // the pattern is exact: it goes back to the very same number
  val back = f64.fromBits(0x3FB999999999999A)
  println("0x3FB999999999999A is ${back}, and ${back == 0.1}")

  // a NaN keeps its payload; nothing canonicalises it
  val payload = f64.fromBits(0x7FF0000000000001)
  println("NaN payload kept: ${payload.isNaN()} ${payload.toBits() == 0x7FF0000000000001}")

  // -0.0 == 0.0, but they are different numbers underneath
  println("-0.0 == 0.0: ${-0.0 == 0.0}, same bits: ${(-0.0).toBits() == (0.0).toBits()}")

  // f32 is 32 bits wide, with a u32 pattern
  val single: f32 = 1.5
  println("f32 1.5   ${rawFloat(single)}")
  println("0x40490FDB is ${f32.fromBits(0x40490FDB)}")

  // sign, exponent and fraction of a double, taken apart by shifting
  val bits = (-6.25).toBits()
  val sign = bits >> 63
  val exponent = (bits >> 52) & 0x7FF
  val fraction = bits & 0xFFFFFFFFFFFFF
  println("-6.25 = sign $sign, exponent ${exponent} (unbiased ${exponent.wrapI64() - 1023}), fraction 0x${fraction.toString(radix: 16)}")
}
