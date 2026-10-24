// Format specs (D171) on every integer type, not only i64: a narrower
// integer widens before it is written in its radix.

test "a spec formats every integer type" {
  val b: u8 = 0xE9
  val n: i32 = -255
  val u: u32 = 4000000000
  val s: usize = 17
  val w: i16 = 42
  expect("${b:02X}" == "E9")
  expect("${b:x}" == "e9")
  expect("${n:08x}" == "-00000ff")
  expect("${n:+}" == "-255")
  expect("${u:,}" == "4,000,000,000")
  expect("${s:#b}" == "0b10001")
  expect("${w:>5}" == "   42")
}

test "u64 and i64 are written as they are" {
  val big: u64 = 18446744073709551615
  expect("${big:x}" == "ffffffffffffffff")
  expect("${-1:x}" == "-1")
}
