// std/utf8 against the vectors every UTF-8 decoder is judged by: the
// boundary code point of each length, the overlong forms, the surrogates,
// the values past U+10FFFF, and the truncated sequences. The shape of the
// table is Markus Kuhn's "UTF-8 decoder capability and stress test" and
// Table 3-7 of the Unicode standard; a decoder that accepts any of the
// rejected rows is one an attacker can smuggle a character past.

use hex, io { println }, utf8

fun u(code: i64): string = "U+${code.toString(radix: 16).toUpper().padStart(4, "0")}"

/// One row: the bytes, then what `decodeBytes` made of them.
fun row(label: string, bytes: List<u8>) {
  val shown = hex.encode(bytes).padEnd(10)
  val r = utf8.decodeBytes(bytes, 0)
  val outcome = if (r == null) "rejected" else "${u(r.code)} size ${r.size}"
  println("  ${label.padEnd(26)} $shown $outcome")
}

fun roundTrip(code: i64) {
  val bytes = utf8.encode(code)
  val back = utf8.decodeBytes(bytes, 0)
  val ok = back != null && back.code == code && back.size == bytes.len()
  println("  ${u(code).padEnd(10)} ${hex.encode(bytes).padEnd(10)} ${if (ok) "round trip" else "BROKEN"}")
}

fun main() {
  println("-- the boundary of each length --")
  loop (code in [0, 0x7F, 0x80, 0x7FF, 0x800, 0xFFFF, 0x10000, 0x10FFFF]) {
    roundTrip(code)
  }

  println("-- accepted --")
  row("ASCII 'A'", [0x41])
  row("U+00E9 e-acute", [0xC3, 0xA9])
  row("U+20AC euro", [0xE2, 0x82, 0xAC])
  row("U+10348 gothic hwair", [0xF0, 0x90, 0x8D, 0x88])
  row("U+FFFD replacement", [0xEF, 0xBF, 0xBD])

  println("-- overlong: a shorter form exists --")
  row("NUL as two bytes", [0xC0, 0x80])
  row("U+007F as two bytes", [0xC1, 0xBF])
  row("NUL as three bytes", [0xE0, 0x80, 0x80])
  row("U+07FF as three bytes", [0xE0, 0x9F, 0xBF])
  row("NUL as four bytes", [0xF0, 0x80, 0x80, 0x80])
  row("U+FFFF as four bytes", [0xF0, 0x8F, 0xBF, 0xBF])

  println("-- surrogates: not code points --")
  row("U+D800 high", [0xED, 0xA0, 0x80])
  row("U+DBFF high last", [0xED, 0xAF, 0xBF])
  row("U+DC00 low", [0xED, 0xB0, 0x80])
  row("U+DFFF low last", [0xED, 0xBF, 0xBF])
  row("U+D7FF just below", [0xED, 0x9F, 0xBF])
  row("U+E000 just above", [0xEE, 0x80, 0x80])

  println("-- past U+10FFFF, and bytes that are never UTF-8 --")
  row("U+110000", [0xF4, 0x90, 0x80, 0x80])
  row("five-byte lead", [0xF8, 0x88, 0x80, 0x80, 0x80])
  row("0xFE", [0xFE])
  row("0xFF", [0xFF])

  println("-- truncated and stray --")
  row("U+20AC less a byte", [0xE2, 0x82])
  row("U+10348 less a byte", [0xF0, 0x90, 0x8D])
  row("lone continuation 80", [0x80])
  row("lone continuation BF", [0xBF])
  row("lead with no follower", [0xC3])

  println("-- a string, forwards and backwards --")
  val text = "aé€𐍈"
  println("  bytes ${text.len()}, characters ${text.charCount()}")
  var i = 0
  val forwards: MutableList<string> = []
  loop (i < text.len()) {
    val r = utf8.decode(text, i) ?: panic("a string is valid UTF-8")
    forwards.push("${u(r.code)}/${r.size}")
    i += r.size
  }
  println("  forwards  ${forwards.join(" ")}")
  var j = text.len()
  val backwards: MutableList<string> = []
  loop (j > 0) {
    val r = utf8.decodeLast(text, j) ?: panic("a string is valid UTF-8")
    backwards.push("${u(r.code)}/${r.size}")
    j -= r.size
  }
  println("  backwards ${backwards.join(" ")}")
  println("  decode at 1 is the whole character: ${utf8.decode(text, 1) != null}")
  println("  decode at 2 splits one:            ${utf8.decode(text, 2) == null}")
  println("  decode at the end:                 ${utf8.decode(text, text.len()) == null}")
  println("  decodeLast before 0:               ${utf8.decodeLast(text, 0) == null}")

  println("-- encoding a value that is not a code point --")
  val notCodePoints = [("negative", -1), ("high surrogate", 0xD800), ("low surrogate", 0xDFFF), ("past U+10FFFF", 0x110000)]
  loop ((label, code) in notCodePoints) {
    val bytes = utf8.encode(code)
    println("  ${label.padEnd(16)} ${hex.encode(bytes)} is ${u(utf8.decodeBytes(bytes, 0)?.code ?: -1)}, size ${utf8.size(code)}")
  }
  println("  isScalar: ${utf8.isScalar(0x41)} ${utf8.isScalar(0xD800)} ${utf8.isScalar(0x10FFFF)} ${utf8.isScalar(0x110000)}")
  println("  size:     ${utf8.size(0x41)} ${utf8.size(0x7FF)} ${utf8.size(0x800)} ${utf8.size(0x10000)} ${utf8.size(0xD800)}")
  println("  char:     ${utf8.char(0x20AC)} ${utf8.char(0x1F600)}")

  println("-- surrogate pairs, as the hex escapes of other formats spell them --")
  println("  D83D DE00 -> ${u(utf8.combineSurrogates(0xD83D, 0xDE00) ?: -1)}")
  println("  D800 DC00 -> ${u(utf8.combineSurrogates(0xD800, 0xDC00) ?: -1)}")
  println("  DBFF DFFF -> ${u(utf8.combineSurrogates(0xDBFF, 0xDFFF) ?: -1)}")
  println("  low first: ${utf8.combineSurrogates(0xDE00, 0xD83D) == null}")
  println("  not a pair: ${utf8.combineSurrogates(0x41, 0x42) == null}")

  println("-- whole buffers --")
  val good = text.bytes()
  val bad: List<u8> = [0x61, 0xC3, 0x28, 0x62]
  println("  isValid good ${utf8.isValid(good)}, count ${utf8.count(good)}")
  println("  isValid bad  ${utf8.isValid(bad)}, count ${utf8.count(bad)}")
  println("  isValid empty ${utf8.isValid([])}, count ${utf8.count([])}")
}
