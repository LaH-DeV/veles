test "RFC 4648 section 10 base16 vectors" {
  val plain = ["", "f", "fo", "foo", "foob", "fooba", "foobar"]
  val encoded = ["", "66", "666F", "666F6F", "666F6F62", "666F6F6261", "666F6F626172"]
  loop ((text, expected) in plain.zip(encoded)) {
    expect(encodeUpper(text.bytes()) == expected)
    expect(encode(text.bytes()) == expected.toLower())
    expect((try decode(expected)).decodeUtf8() == text)
    expect((try decode(expected.toLower())).decodeUtf8() == text)
  }
}

test "all byte values round trip in both hex cases" {
  val buffer: MutableList<u8> = []
  loop (value in 0..<256) buffer.push(value.wrapU8())
  val bytes = buffer.toList()
  val lower = encode(bytes)
  val upper = encodeUpper(bytes)
  expect(lower.len() == 512)
  expect(upper == lower.toUpper())
  val decoded = try decode(lower)
  expect(decoded.len() == 256)
  loop (index in 0..<256) expect(decoded.at(index) == bytes.at(index))
  expect(encode(try decode(upper)) == lower)
  expect(encode(try decode("aBcD09eF")) == "abcd09ef")
}

test "hex rejects odd lengths prefixes separators and whitespace" {
  loop (text in ["0", "abc", "0x00", "gg", "00:1", " 00 ", "00\n0", "\t0", "\u{E9}"]) {
    expect(decode(text) is Err)
  }
}

test "hex errors point at an invalid byte or the incomplete end" {
  val texts = ["0", "abc", "g0", "0G", "00 0", "00\n0", "\u{E9}"]
  val positions = [1, 3, 0, 1, 2, 2, 0]
  loop ((text, position) in texts.zip(positions)) {
    expect(when (decode(text)) {
      is Err(error) => error.position == position
      is Ok(_)      => false
    })
  }
}
