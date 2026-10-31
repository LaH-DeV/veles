test "RFC 4648 section 10 standard and URL vectors" {
  val plain = ["", "f", "fo", "foo", "foob", "fooba", "foobar"]
  val encoded = ["", "Zg==", "Zm8=", "Zm9v", "Zm9vYg==", "Zm9vYmE=", "Zm9vYmFy"]
  loop ((text, expected) in plain.zip(encoded)) {
    expect(encode(text.bytes()) == expected)
    expect((try decode(expected)).decodeUtf8() == text)
    val unpadded = expected.replace("=", "")
    expect(encodeUrl(text.bytes()) == unpadded)
    expect((try decode(unpadded)).decodeUtf8() == text)
    expect((try decodeUrl(expected)).decodeUtf8() == text)
    expect((try decodeUrl(unpadded)).decodeUtf8() == text)
  }
}

test "binary vectors distinguish standard and URL alphabets" {
  val bytes: List<u8> = [0xFB, 0xFF, 0xFF, 0xFF]
  expect(encode(bytes) == "+////w==")
  expect(encodeUrl(bytes) == "-____w")
  expect(encode(try decode("+////w==")) == "+////w==")
  expect(encodeUrl(try decodeUrl("-____w")) == "-____w")
}

test "encodedLen agrees with both encoders across tail lengths" {
  val bytes: MutableList<u8> = []
  loop (length in 0..256) {
    val snapshot = bytes.toList()
    expect(encodedLen(length) == encode(snapshot).len())
    expect(encodedLen(length, pad: false) == encodeUrl(snapshot).len())
    if (length < 256) bytes.push(length.wrapU8())
  }
}

test "all byte values survive both base64 alphabets" {
  val bytes: MutableList<u8> = []
  loop (value in 0..<256) bytes.push(value.wrapU8())
  val snapshot = bytes.toList()
  val standard = try decode(encode(snapshot))
  val url = try decodeUrl(encodeUrl(snapshot))
  expect(standard.len() == bytes.len())
  expect(url.len() == bytes.len())
  loop (index in 0..<bytes.len()) {
    expect(standard.at(index) == bytes.at(index))
    expect(url.at(index) == bytes.at(index))
  }
}

test "decoders reject malformed padding and noncanonical tail bits" {
  loop (text in ["A", "AAAAA", "=", "====", "Zg=", "Zg===", "Z===", "Zm9v=", "Zm=8", "AB", "AAB", "Zh==", "Zm9="]) {
    expect(decode(text) is Err)
    expect(decodeUrl(text) is Err)
  }
}

test "decoders reject foreign alphabets whitespace and non ASCII" {
  loop (text in ["-w", "_w"]) expect(decode(text) is Err)
  loop (text in ["+w", "/w"]) expect(decodeUrl(text) is Err)
  loop (text in ["Z g==", "Zg==\n", "Z\tg==", "Zg\r\n", "Zg!", "\u{E9}"]) {
    expect(decode(text) is Err)
    expect(decodeUrl(text) is Err)
  }
}

test "invalid positions identify bad bytes incomplete input and unused bits" {
  val texts = ["Z!", "A", "Zh==", "Zm9=", "Zg=", "AA=A"]
  val positions = [1, 1, 1, 2, 3, 2]
  loop ((text, position) in texts.zip(positions)) {
    expect(when (decode(text)) {
      is Err(error) => error.position == position
      is Ok(_)      => false
    })
    expect(when (decodeUrl(text)) {
      is Err(error) => error.position == position
      is Ok(_)      => false
    })
  }
}
