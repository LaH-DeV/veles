// A property fuzzer for the standard library's decoders: everything that
// reads text someone else wrote. Each target feeds generated input and
// checks an invariant that must hold for *every* input — a decoder may
// refuse, but it may never panic, and what it accepts must survive the
// trip back out. Seeded, so a failure is reproducible from its seed.
//
//   fuzz [iterations] [seed]
//
// A failure prints the target, the iteration and the input, and the exit
// code is 1.
use base64, codec, hex, http, io, json, os, random, utf8

struct Stats {
  var runs:     i64 = 0
  var accepted: i64 = 0
  var failures: i64 = 0
}

struct Fuzzer {
  var rng:   random.Rng
  var stats: MutableMap<string, Stats> = [:]

  fun record(target: string, accepted: bool, failure: string?) {
    var s = this.stats.get(target) ?: Stats()
    s.runs += 1
    if (accepted) s.accepted += 1
    if (failure != null) {
      s.failures += 1
      if (s.failures <= 3) io.println("FAIL $target: $failure")
    }
    this.stats.set(target, s)
  }

  fun bytes(max: i64): List<u8> {
    val n = this.rng.range(0, max + 1)
    var out: MutableList<u8> = []
    loop (_ in 0..<n) out.push(this.rng.range(0, 256) as u8)
    out.toList()
  }

  /// Text built from pieces that are likely to matter to the decoder under
  /// test, with the occasional arbitrary byte turned into a character.
  fun textFrom(pieces: List<string>, maxPieces: i64): string {
    val sb = StringBuilder()
    loop (_ in 0..<this.rng.range(0, maxPieces + 1)) {
      if (this.rng.range(0, 10) == 0) {
        sb.append(utf8.char(this.rng.range(1, 0x2FF)))
      } else {
        sb.append(this.rng.pick(pieces) ?: "")
      }
    }
    sb.toString()
  }
}

fun show(s: string): string = if (s.len() <= 80) "\"$s\"" else "\"${s.substring(0, 80) ?: s}...\" (${s.len()} bytes)"

// ---------------------------------------------------------------------------
// targets

/// base64: bytes survive encode/decode in both alphabets, and any text is
/// either refused or is exactly the canonical encoding of what it decodes
/// to (padding aside: both decoders take unpadded input).
fun fuzzBase64(f: *Fuzzer) {
  val data = f.bytes(40)
  val back = base64.decode(base64.encode(data)).getOrNull()
  val backUrl = base64.decodeUrl(base64.encodeUrl(data)).getOrNull()
  f.record("base64 round trip", true, if (back != data || backUrl != data) "bytes ${hex.encode(data)}" else null)

  val text = f.textFrom(["A", "Q", "w", "8", "+", "/", "-", "_", "=", "==", "AAAA", "QUJD", " ", "\n"], 12)
  when (val r = base64.decode(text)) {
    is Ok  => {
      val again = base64.encode(r).replace("=", "")
      f.record("base64 decode", true, if (again != text.replace("=", "")) "${show(text)} decoded to ${hex.encode(r)}, which encodes as \"$again\"" else null)
    }
    is Err => f.record("base64 decode", false, null)
  }
}

/// hex: the same two properties, case-insensitively on the way in.
fun fuzzHex(f: *Fuzzer) {
  val data = f.bytes(40)
  f.record("hex round trip", true, if (hex.decode(hex.encode(data)).getOrNull() != data) "bytes ${hex.encode(data)}" else null)

  val text = f.textFrom(["0", "9", "a", "f", "A", "F", "g", "00", "ff", " "], 12)
  when (val r = hex.decode(text)) {
    is Ok  => f.record("hex decode", true, if (hex.encode(r) != text.toLower()) "${show(text)} re-encodes as \"${hex.encode(r)}\"" else null)
    is Err => f.record("hex decode", false, null)
  }
}

/// json: any text is either refused with a DecodeError or parses to a
/// value that encodes, parses back to the same value, and encodes to the
/// same text again.
fun fuzzJson(f: *Fuzzer) {
  val pieces = ["{", "}", "[", "]", ":", ",", " ", "\"a\"", "\"k\":", "\"\"", "0", "-1", "12", "3.5", "-0.25e2", "1e400", "-", "1.", ".5", "01", "true", "false", "null", "nul", "\"\\n\\t\\\\\\\"\"", "\"\\u00e9\"", "\"\\ud83d\\ude00\"", "\"\\ud800\"", "\"\\u12\"", "\"x", "9223372036854775807", "9223372036854775808", "-9223372036854775808"]
  checkJson(f, f.textFrom(pieces, 14))
}

fun checkJson(f: *Fuzzer, text: string) {
  when (val r = json.parse(text)) {
    is Ok  => {
      val v: codec.Value = r
      when (val e = json.encode(v)) {
        is Ok  => {
          val first: string = e
          val again = json.parse(first).getOrNull()
          val second: string? = if (again == null) null else json.encode(again).getOrNull()
          val broken: string? = when {
            again == null   => "${show(text)} parsed, but its encoding ${show(first)} does not"
            second != first => "${show(text)} encodes as ${show(first)}, then as ${show(second ?: "")}"
            else            => null
          }
          f.record("json parse", true, broken)
        }
        is Err => f.record("json parse", true, "${show(text)} parsed, but does not encode: ${e.message()}")
      }
    }
    is Err => f.record("json parse", false, null)
  }
}

/// json, the other way round: a generated value — extreme integers, floats
/// from the whole exponent range, text with escapes and non-ASCII — must
/// encode, and parse back to a value equal to itself.
fun fuzzJsonValues(f: *Fuzzer) {
  val v = randomValue(f, 3)
  when (val e = json.encode(v)) {
    is Ok  => {
      val text: string = e
      val back = json.parse(text).getOrNull()
      f.record("json value round trip", true, if (back != v) "$v encodes as ${show(text)}, which reads back as ${back ?: codec.VNull()}" else null)
    }
    is Err => f.record("json value round trip", true, "$v does not encode: ${e.message()}")
  }
}

fun randomValue(f: *Fuzzer, depth: i64): codec.Value {
  val ints: List<i64> = [0, -1, 1, 9223372036854775807, -9223372036854775807 - 1, 4503599627370496, -9007199254740993]
  val strs = ["", "a", "é", "日本", "😀", "\"", "\\", "\n\t\r", "\u{1}", "\u{7f}", "</script>", "\u{2028}"]
  when (f.rng.range(0, if (depth <= 0) 5 else 7)) {
    0    => return codec.VNull()
    1    => return codec.VBool(value: f.rng.boolean())
    2    => return codec.VInt(value: if (f.rng.boolean()) f.rng.pick(ints) ?: 0 else f.rng.range(-1000000, 1000000))
    3    => {
      // a mantissa in [1, 10) times a power of ten anywhere an f64 can
      // hold, so the shortest-text printer and the parser are both
      // exercised at the edges of the range
      val mantissa = 1.0 + f.rng.float() * 9.0
      val exp = f.rng.range(-307, 308) as f64
      val sign = if (f.rng.boolean()) -1.0 else 1.0
      return codec.VFloat(value: sign * mantissa * 10.0.pow(exp))
    }
    4    => {
      val sb = StringBuilder()
      loop (_ in 0..<f.rng.range(0, 4)) sb.append(f.rng.pick(strs) ?: "")
      return codec.VString(value: sb.toString())
    }
    5    => {
      var items: MutableList<codec.Value> = []
      loop (_ in 0..<f.rng.range(0, 4)) items.push(randomValue(f, depth - 1))
      return codec.VList(items: items.toList())
    }
    else => {
      var fields: MutableMap<string, codec.Value> = [:]
      loop (_ in 0..<f.rng.range(0, 4)) fields.set(f.rng.pick(strs) ?: "", randomValue(f, depth - 1))
      return codec.VObject(fields: fields.toMap())
    }
  }
}

/// percentDecode: never panics, and text with nothing to decode is itself.
fun fuzzPercent(f: *Fuzzer) {
  val text = f.textFrom(["%", "%2", "%20", "%zz", "%e2%82%ac", "%ff", "%C3", "+", "a", "/", "%%"], 10)
  val plain = !text.contains("%") && !text.contains("+")
  val out = http.percentDecode(text, plusIsSpace: true)
  f.record("percentDecode", true, if (plain && out != text) "${show(text)} became ${show(out)}" else null)
}

/// utf8: two independent validators — std/utf8's Table 3-7 decoder and the
/// runtime's decodeUtf8 — must agree on every byte string, and count must
/// agree with decoding one code point at a time.
fun fuzzUtf8(f: *Fuzzer) {
  // bias towards the interesting bytes: continuation, lead, and the edges
  val special: List<u8> = [0x7F, 0x80, 0xBF, 0xC0, 0xC1, 0xC2, 0xDF, 0xE0, 0xED, 0xEF, 0xF0, 0xF4, 0xF5, 0xFF, 0x9F, 0xA0, 0x8F, 0x90]
  var data: MutableList<u8> = []
  loop (_ in 0..<f.rng.range(0, 12)) {
    data.push(if (f.rng.range(0, 3) == 0) f.rng.range(0, 256) as u8 else f.rng.pick(special) ?: 0)
  }
  val bytes = data.toList()
  val std = utf8.isValid(bytes)
  val runtime = bytes.decodeUtf8() != null
  f.record("utf8 validators agree", std, if (std != runtime) "${hex.encode(bytes)}: std/utf8 says $std, decodeUtf8 says $runtime" else null)
}

fun main() {
  val args = os.args()
  val iterations = args.at(0)?.toInt() ?: 500
  val seed = args.at(1)?.toInt() ?: 1
  var f = Fuzzer(rng: random.Rng.seeded(seed))
  // inputs that once broke a decoder, checked on every run before the
  // generated ones: each was found by this program
  loop (text in ["1e400", "-1e400", "1e99999999999999999999", "[1e-99999999999999999999]", "-1.9223372036854775807", "{\"a\": 0.1, \"b\": 1.7976931348623157e308}"]) {
    checkJson(&f, text)
  }
  loop (_ in 0..<iterations) {
    fuzzBase64(&f)
    fuzzHex(&f)
    fuzzJson(&f)
    fuzzJsonValues(&f)
    fuzzPercent(&f)
    fuzzUtf8(&f)
  }
  var failed = false
  loop ((target, s) in f.stats.entries().sortedBy(e => e.0)) {
    io.println("$target: ${s.runs} runs, ${s.accepted} accepted, ${s.failures} failures")
    if (s.failures > 0) failed = true
  }
  if (failed) os.exit(1)
}
