// Numeric conversions (D86): `toT()` says null when a value does not fit,
// `wrapT()` keeps the low bits. The program reads a byte count off the wire
// the way a protocol parser does: a length that must fit its field is
// checked, a checksum is meant to wrap.
use io

/// A 16-bit length field: the payload must fit, or the frame is refused.
fun lengthField(payload: i64): u16? = payload.toU16()

/// A one-byte checksum: the sum of the bytes, wrapped on purpose.
fun checksum(bytes: List<u8>): u8 {
  var sum = 0
  loop (b in bytes) {
    sum += b.toI64()
  }
  sum.wrapU8()
}

/// Generic code names its target as a type argument: the low `bits` of
/// `n` as whatever integer type the caller asks for.
fun lowBits<T>(n: i64): T = n.wrapTo<T>()

fun show(label: string, v: string) {
  io.println("$label = $v")
}

fun main() {
  show("length 500", "${lengthField(500)}")
  show("length 65535", "${lengthField(65535)}")
  show("length 65536", "${lengthField(65536)}")
  show("length -1", "${lengthField(-1)}")
  show("checksum", "${checksum([200, 100, 7])}")

  val low8: u8 = lowBits(0x1234)
  val low16: i16 = lowBits(0x12345)
  show("wrapTo", "$low8 $low16")

  // lossless: no `?`
  val small: u8 = 250
  show("u8 to i64", "${small.toI64() + 10}")
  show("i32 to f64", "${(7).wrapI32().toF64() / 2.0}")

  // integer to integer: the range decides
  val minusOne = -1
  show("i64 -1 to u8", "${minusOne.toU8()} wrap ${minusOne.wrapU8()}")
  show("u8 250 to i8", "${small.toI8()} wrap ${small.wrapI8()}")
  val big: u64 = 18446744073709551615
  show("u64 max to i64", "${big.toI64()} wrap ${big.wrapI64()}")

  // float to integer: truncates toward zero, null when out of range or NaN
  show("255.9 to u8", "${255.9.toU8()}")
  show("256.0 to u8", "${256.0.toU8()}")
  show("-0.9 to u8", "${(-0.9).toU8()}")
  show("-3.7 to i64", "${(-3.7).toI64()}")
  show("1e20 to i64", "${1e20.toI64()}")
  show("nan to i32", "${(0.0 / 0.0).toI32()}")
}
