test "Unicode table 3-7 scalar boundaries encode and decode" {
  val codes = [0, 0x7F, 0x80, 0x7FF, 0x800, 0xD7FF, 0xE000, 0xFFFF, 0x10000, 0x10FFFF]
  val vectors: List<List<u8>> = [
    [0x00], [0x7F], [0xC2, 0x80], [0xDF, 0xBF],
    [0xE0, 0xA0, 0x80], [0xED, 0x9F, 0xBF], [0xEE, 0x80, 0x80], [0xEF, 0xBF, 0xBF],
    [0xF0, 0x90, 0x80, 0x80], [0xF4, 0x8F, 0xBF, 0xBF],
  ]
  loop ((code, bytes) in codes.zip(vectors)) {
    expect(isScalar(code))
    expect(size(code) == bytes.len())
    expect(isValid(bytes))
    expect(count(bytes) == 1)
    expect(decodeBytes(bytes, 0)?.code == code)
    expect(decodeBytes(bytes, 0)?.size == bytes.len())
    val encoded = encode(code)
    expect(encoded.len() == bytes.len())
    loop (index in 0..<bytes.len()) expect(encoded.at(index) == bytes.at(index))
    val text = char(code)
    expect(text.len() == bytes.len())
    expect(decode(text, 0)?.code == code)
    expect(decodeLast(text, text.len())?.code == code)
    val output: MutableList<u8> = [0x41]
    expect(encodeTo(output, code) == bytes.len())
    expect(output.len() == bytes.len() + 1)
    expect(output.at(0) == 0x41)
    loop (index in 0..<bytes.len()) expect(output.at(index + 1) == bytes.at(index))
  }
}

test "RFC 3629 rejects overlong surrogates out of range and malformed sequences" {
  val invalid: List<List<u8>> = [
    [0x80], [0xBF], [0xC0, 0x80], [0xC1, 0xBF],
    [0xE0, 0x80, 0x80], [0xE0, 0x9F, 0xBF],
    [0xED, 0xA0, 0x80], [0xED, 0xBF, 0xBF],
    [0xF0, 0x80, 0x80, 0x80], [0xF0, 0x8F, 0xBF, 0xBF],
    [0xF4, 0x90, 0x80, 0x80], [0xF5, 0x80, 0x80, 0x80],
    [0xF8, 0x88, 0x80, 0x80, 0x80], [0xFE], [0xFF],
    [0xC2, 0x41], [0xE1, 0x80, 0x41], [0xF1, 0x80, 0x80, 0x41],
  ]
  loop (bytes in invalid) {
    expect(!isValid(bytes))
    expect(decodeBytes(bytes, 0) == null)
    expect(bytes.decodeUtf8() == null)
  }
}

test "every truncated multibyte sequence is rejected" {
  loop (code in [0x80, 0x800, 0x10000, 0x10FFFF]) {
    val bytes = encode(code)
    val prefix: MutableList<u8> = []
    loop (byte in bytes) {
      val truncated = prefix.toList()
      expect(decodeBytes(truncated, 0) == null)
      if (!truncated.isEmpty()) expect(!isValid(truncated))
      prefix.push(byte)
    }
  }
}

test "single bytes and continuation classification cover all 256 values" {
  loop (value in 0..<256) {
    val byte = value.wrapU8()
    val bytes: List<u8> = [byte]
    expect(isValid(bytes) == (value < 0x80))
    expect(isContinuation(byte) == (value >= 0x80 && value <= 0xBF))
    expect(isStart(byte) == !isContinuation(byte))
    expect(count(bytes) == 1)
  }
}

test "invalid scalar values encode as replacement without appending invalid bytes" {
  loop (code in [-9223372036854775807 - 1, -1, 0xD800, 0xDBFF, 0xDC00, 0xDFFF, 0x110000, 9223372036854775807]) {
    expect(!isScalar(code))
    expect(size(code) == null)
    expect(char(code) == "\u{FFFD}")
    val bytes = encode(code)
    expect(bytes.len() == 3)
    expect(bytes.at(0) == 0xEF)
    expect(bytes.at(1) == 0xBF)
    expect(bytes.at(2) == 0xBD)
  }
}

test "UTF16 surrogate pairs cover both bounds and reject incorrect halves" {
  expect(combineSurrogates(0xD800, 0xDC00) == 0x10000)
  expect(combineSurrogates(0xDBFF, 0xDFFF) == maxCode)
  expect(combineSurrogates(0xD83D, 0xDE00) == 0x1F600)
  loop (code in [-1, 0, 0xD7FF, 0xDC00, 0xE000, 9223372036854775807]) {
    expect(combineSurrogates(code, 0xDC00) == null)
  }
  loop (code in [-1, 0, 0xDBFF, 0xE000, 9223372036854775807]) {
    expect(combineSurrogates(0xD800, code) == null)
  }
  expect(isSurrogate(0xD800))
  expect(isSurrogate(0xDFFF))
  expect(!isSurrogate(0xD7FF))
  expect(!isSurrogate(0xE000))
}

test "forward and reverse cursors require character boundaries" {
  val text = "A\u{A2}\u{20AC}\u{1F600}"
  val starts = [0, 1, 3, 6]
  val ends = [1, 3, 6, 10]
  val codes = [0x41, 0xA2, 0x20AC, 0x1F600]
  loop ((start, code) in starts.zip(codes)) {
    expect(decode(text, start)?.code == code)
    expect(decodeBytes(text.bytes(), start)?.code == code)
  }
  loop ((end, code) in ends.zip(codes)) expect(decodeLast(text, end)?.code == code)
  loop (offset in [2, 4, 5, 7, 8, 9]) {
    expect(decode(text, offset) == null)
    expect(decodeBytes(text.bytes(), offset) == null)
    expect(decodeLast(text, offset) == null)
  }
  loop (offset in [-9223372036854775807 - 1, -1, 10, 9223372036854775807]) {
    expect(decode(text, offset) == null)
    expect(decodeBytes(text.bytes(), offset) == null)
  }
  expect(decodeLast(text, 0) == null)
  expect(decodeLast(text, -1) == null)
  expect(decodeLast(text, 9223372036854775807)?.code == 0x1F600)
}

test "empty buffers and malformed byte counting make progress" {
  val empty: List<u8> = []
  expect(isValid(empty))
  expect(count(empty) == 0)
  expect(decodeBytes(empty, 0) == null)
  expect(decode("", 0) == null)
  expect(decodeLast("", 0) == null)
  val malformed: List<u8> = [0xE2, 0x82, 0x41, 0xC0, 0xAF, 0xF0, 0x9F, 0x92]
  expect(count(malformed) == 8)
  val mixed: List<u8> = [0x61, 0xE2, 0x82, 0xAC, 0xFF, 0xF0, 0x9F, 0x98, 0x80]
  expect(!isValid(mixed))
  expect(count(mixed) == 4)
}
