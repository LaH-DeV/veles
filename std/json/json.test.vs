use json

test fun rejectsU64(text: string): bool {
  json.decode<u64>(text) catch (_) {
    return true
  }
  false
}

test fun rejectsJson(text: string): bool {
  json.parse(text) catch (_) {
    return true
  }
  false
}

test "u64 JSON decoding spans the full unsigned range" {
  val zero = try json.decode<u64>("0")
  val signedMax = try json.decode<u64>("9223372036854775807")
  val aboveSigned = try json.decode<u64>("9223372036854775808")
  val max: u64 = 18446744073709551615
  val encoded = try json.encode(max)
  val decoded = try json.decode<u64>(encoded)

  expect(zero == 0)
  expect(signedMax == 9223372036854775807)
  expect(aboveSigned == 9223372036854775808)
  expect(encoded == "18446744073709551615")
  expect(decoded == max)

  expect(rejectsU64("-1"))
  expect(rejectsU64("18446744073709551616"))
}

test "JSON number grammar rejects leading zeros and missing digits" {
  expect(rejectsJson("01"))
  expect(rejectsJson("-01"))
  expect(rejectsJson("1."))
  expect(rejectsJson("1e"))
  expect(rejectsJson("1e+"))
}
