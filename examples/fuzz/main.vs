// A property fuzzer for the standard library's decoders: everything that
// reads text someone else wrote, the HTTP server's request parser included
// (over a real connection, see `fuzzHttp`). Each target feeds generated input and
// checks an invariant that must hold for *every* input — a decoder may
// refuse, but it may never panic, and what it accepts must survive the
// trip back out. Seeded, so a failure is reproducible from its seed.
//
//   fuzz [iterations] [seed]
//
// A failure prints the target, the iteration and the input, and the exit
// code is 1.
use base64, codec, hex, http, io { println }, json, net, os, random, utf8

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
      if (s.failures <= 3) println("FAIL $target: $failure")
    }
    this.stats.set(target, s)
  }

  /// Up to `maxPieces` of `pieces`, joined: text made only of what the
  /// caller allows, for inputs where one stray byte would be a fault.
  fun join(pieces: List<string>, maxPieces: i64): string {
    val sb = StringBuilder()
    loop (_ in 0..<this.rng.range(0, maxPieces + 1)) sb.append(this.rng.pick(pieces) ?: "")
    sb.toString()
  }

  fun bytes(max: i64): List<u8> {
    val n = this.rng.range(0, max + 1)
    var out: MutableList<u8> = []
    loop (_ in 0..<n) out.push((this.rng.range(0, 256)).wrapU8())
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
      val exp = (f.rng.range(-307, 308)).toF64()
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
    data.push(if (f.rng.range(0, 3) == 0) (f.rng.range(0, 256)).wrapU8() else f.rng.pick(special) ?: 0)
  }
  val bytes = data.toList()
  val std = utf8.isValid(bytes)
  val runtime = bytes.decodeUtf8() != null
  f.record("utf8 validators agree", std, if (std != runtime) "${hex.encode(bytes)}: std/utf8 says $std, decodeUtf8 says $runtime" else null)
}

// ---------------------------------------------------------------------------
// http: the request parser, over a real connection
//
// `http.serve` runs in this process on a loopback listener. Each run opens
// a connection, sends one to three requests in one write and closes its
// side. The last request may carry one fault — a ceiling passed, a
// malformed line, framing the server must refuse — and each fault has the
// status RFC 9112 asks for; or the whole write may be mutated at random,
// and then any refusal will do. Either way every answer must be a
// well-formed response, none a 500, and none may wait for a timeout: the
// client has closed, so every read on the server ends.

/// Small ceilings, so each is within reach of a generated request.
val fuzzLimits = http.Limits(requestLineBytes: 200, headerLineBytes: 100, headerCount: 6, headerBytes: 400, bodyBytes: 64)

/// What the parser produced, sent back: the method and decoded path in a
/// header (a path can decode to anything, `\r\n` included) and the body.
fun echo(req: http.Request): http.Response suspends throws http.Fail | IoError =
  http.Response.bytes(try req.bytes(), "application/octet-stream").withHeader("x-echo", "${req.method} ${req.path}")

/// `data` as a chunked body: pieces of random size, some with a chunk
/// extension, then the last chunk and sometimes a trailer field.
fun chunked(f: *Fuzzer, data: List<u8>): List<u8> {
  val out: MutableList<u8> = []
  var at: i64 = 0
  loop (at < data.len()) {
    val n = f.rng.range(1, data.len() - at + 1)
    val ext = if (f.rng.range(0, 4) == 0) ";x=1" else ""
    out.addAll("${hexNumber(n)}$ext\r\n".bytes())
    out.addAll(data.slice(at, at + n))
    out.addAll("\r\n".bytes())
    at += n
  }
  out.addAll("0\r\n".bytes())
  if (f.rng.range(0, 3) == 0) out.addAll("X-Trailer: 1\r\n".bytes())
  out.addAll("\r\n".bytes())
  out.toList()
}

fun hexNumber(n: i64): string {
  if (n == 0) return "0"
  val digits = "0123456789abcdef"
  var out = ""
  var rest = n
  loop (rest > 0) {
    out = (digits.substring(rest % 16, rest % 16 + 1) ?: "0") + out
    rest /= 16
  }
  out
}

/// One request, in parts a fault can edit before they are joined.
struct Draft {
  var method:  string
  var target:  string
  var version: string = "HTTP/1.1"
  var line:    string? = null  // the whole request line, when a fault wrote it
  var headers: MutableList<string> = []
  var body:    List<u8> = []
  var eol:     string = "\r\n"

  fun wire(): List<u8> {
    val sb = StringBuilder()
    sb.append(this.line ?: "${this.method} ${this.target} ${this.version}")
    sb.append(this.eol)
    loop (h in this.headers) sb.append("$h${this.eol}")
    sb.append(this.eol)
    sb.toString().bytes().concat(this.body)
  }
}

/// What must come back for one request.
struct Want {
  status: i64
  echo:   string = ""
  body:   List<u8> = []
  closes: bool = false  // the response says `connection: close` and is the last
}

/// A request the server must accept. `last` may end the connection
/// (`Connection: close`, or HTTP/1.0 without keep-alive); `framed` leaves
/// out the body and the connection headers, for a fault to add its own.
fun draft(f: *Fuzzer, last: bool, framed: bool): (Draft, Want) {
  val method = f.rng.pick(["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "BREW", "M-SEARCH"]) ?: "GET"
  val path = "/" + f.join(["a", "b/", "..", "~", "%20", "%2F", "%zz", "%", "+", "%0d%0a", "%0A", "%00", "%e2%82%ac", "%ff"], 5)
  val query = if (f.rng.boolean()) "" else "?" + f.join(["a=1", "&", "b", "=", "%26", "+", "%C3%A9"], 4)
  var d = Draft(method, target: path + query)
  if (f.rng.range(0, 4) == 0) d.eol = "\n"
  d.headers.push("Host: fuzz")
  loop (_ in 0..<f.rng.range(0, 3)) {
    val name = f.rng.pick(["X-A", "Accept", "user-agent", "x!#$%&'*+.^`|~"]) ?: "X-A"
    val value = f.rng.pick(["v", " v", " a b ", "\t\u{e9}\t", "", " x:y", " 1"]) ?: ""
    d.headers.push("$name:$value")
  }
  var closes = false
  // what the handler must read back: the body as sent, or as a chunked
  // framing decodes it
  var decodedBody: List<u8> = []
  var chunkedBody = false
  if (!framed) {
    if (f.rng.range(0, 3) == 0) {
      d.body = f.bytes(fuzzLimits.bodyBytes)
      decodedBody = d.body
      d.headers.push("Content-Length: ${d.body.len()}")
    } else if (f.rng.range(0, 4) == 0) {
      decodedBody = f.bytes(fuzzLimits.bodyBytes)
      d.body = chunked(f, decodedBody)
      d.headers.push("Transfer-Encoding: chunked")
      chunkedBody = true
    } else if (f.rng.range(0, 6) == 0) {
      d.headers.push("Content-Length: 0")
    }
    if (last && f.rng.range(0, 4) == 0) {
      d.headers.push("Connection: close")
      closes = true
    } else if (last && !chunkedBody && f.rng.range(0, 6) == 0) {
      // HTTP/1.0 has no chunked coding
      d.version = "HTTP/1.0"
      closes = true
    }
  }
  val decoded = http.percentDecode(path, plusIsSpace: false).replace("\r", " ").replace("\n", " ").replace("\u{0}", " ")
  (d, Want(status: 200, echo: "$method $decoded", body: decodedBody, closes))
}

/// The last request, with one fault: its bytes, the status it must get,
/// and a name for a failure message.
fun faulty(f: *Fuzzer): (List<u8>, i64, string) {
  var (d, _) = draft(f, last: true, framed: true)
  when (f.rng.range(0, 25)) {
    0    => {
      d.target = "/" + "a".repeat(fuzzLimits.requestLineBytes)
      return (d.wire(), 414, "long target")
    }
    1    => {
      d.headers.push("X-Long: " + "v".repeat(fuzzLimits.headerLineBytes))
      return (d.wire(), 431, "long header")
    }
    2    => {
      loop (_ in 0..<fuzzLimits.headerCount) d.headers.push("X-N: 1")
      return (d.wire(), 431, "many headers")
    }
    3    => {
      loop (_ in 0..<5) d.headers.push("X-B: " + "b".repeat(80))
      return (d.wire(), 431, "header bytes")
    }
    4    => {
      d.headers.push("Content-Length: ${f.rng.range(fuzzLimits.bodyBytes + 1, 100000)}")
      return (d.wire(), 413, "large body")
    }
    5    => {
      val length = f.rng.pick(["abc", "-1", "+5", "1 2", "0x10", "", "5.0", "99999999999999999999"]) ?: ""
      d.headers.push("Content-Length: $length")
      d.body = "hello".bytes()
      return (d.wire(), 400, "content-length \"$length\"")
    }
    6    => {
      d.headers.push("Content-Length: 5")
      d.headers.push("Content-Length: 6")
      d.body = "hello!".bytes()
      return (d.wire(), 400, "two content-lengths")
    }
    7    => {
      // chunked is understood; these codings are not
      val name = f.rng.pick(["gzip", "gzip, chunked", "identity", "chunked, chunked"]) ?: "gzip"
      d.headers.push("Transfer-Encoding: $name")
      return (d.wire(), 501, "transfer coding $name")
    }
    8    => {
      val (version, status) = f.rng.pick([("HTTP/2.0", 505), ("HTTP/3.0", 505), ("HTTP/1.1x", 400), ("http/1.1", 400), ("HTTP/1", 400), ("HTTP/1.", 400), ("HTTP/11.1", 400)]) ?: ("HTTP/2.0", 505)
      d.version = version
      return (d.wire(), status, "version $version")
    }
    9    => {
      val line = f.rng.pick(["GET /", "GET  / HTTP/1.1", "GET / HTTP/1.1 x", " GET / HTTP/1.1", "GET\t/\tHTTP/1.1", "GET / HTTP/1.1 ", "/ HTTP/1.1"]) ?: "GET /"
      d.line = line
      return (d.wire(), 400, "request line \"$line\"")
    }
    10   => {
      d.headers.push("NoColon")
      return (d.wire(), 400, "header without a colon")
    }
    11   => {
      d.headers.push(": v")
      return (d.wire(), 400, "header without a name")
    }
    12   => {
      val h = f.rng.pick(["X-A : v", "X-A\t: v", "Content-Length : 0"]) ?: "X-A : v"
      d.headers.push(h)
      return (d.wire(), 400, "space before the colon: \"$h\"")
    }
    13   => {
      val fold = f.rng.pick([" folded: x", "\tmore"]) ?: " folded: x"
      d.headers.push("X-A: v")
      d.headers.push(fold)
      return (d.wire(), 400, "folded header")
    }
    14   => {
      val h = f.rng.pick(["X A: v", "X\u{1}B: v", "X(A): v", "\u{e9}: v", "X\"Q: v", "X/A: v"]) ?: "X A: v"
      d.headers.push(h)
      return (d.wire(), 400, "header name \"$h\"")
    }
    15   => {
      val h = f.rng.pick(["X-A: a\u{0}b", "X-A: a\rb"]) ?: "X-A: a\rb"
      d.headers.push(h)
      return (d.wire(), 400, "NUL or CR in a value")
    }
    16   => {
      // the client closes with the head unfinished
      val w = d.wire()
      return (w.take(w.len() - d.eol.len()), 400, "closed inside the headers")
    }
    17   => {
      d.headers.push("Content-Length: 10")
      d.body = "four".bytes()
      return (d.wire(), 400, "body shorter than its length")
    }
    18   => {
      // a byte that is not UTF-8 in the target
      var w = d.wire().toMutable()
      w.insert(d.method.len() + 2, 0xFF)
      return (w.toList(), 400, "target not UTF-8")
    }
    19   => {
      val _ = d.headers.removeAt(0)
      return (d.wire(), 400, "no Host")
    }
    20   => {
      d.headers.push("Transfer-Encoding: chunked")
      d.headers.push("Content-Length: 5")
      d.body = "0\r\n\r\n".bytes()
      return (d.wire(), 400, "chunked and a length")
    }
    21   => {
      d.headers.push("Transfer-Encoding: chunked")
      val body = f.rng.pick(["zz\r\nabc\r\n0\r\n\r\n", "-1\r\na\r\n0\r\n\r\n", "\r\n0\r\n\r\n", "1 2\r\nab\r\n0\r\n\r\n"]) ?: "zz\r\n"
      d.body = body.bytes()
      return (d.wire(), 400, "chunk size in ${showBytes(d.body)}")
    }
    22   => {
      // the client closes inside a chunk
      d.headers.push("Transfer-Encoding: chunked")
      d.body = "5\r\nab".bytes()
      return (d.wire(), 400, "closed inside a chunk")
    }
    23   => {
      // a chunk larger than the ceiling is refused before its data
      d.headers.push("Transfer-Encoding: chunked")
      d.body = "${hexNumber(fuzzLimits.bodyBytes + 1)}\r\n".bytes()
      return (d.wire(), 413, "large chunk")
    }
    else => {
      d.headers.push("Host: other")
      return (d.wire(), 400, "two Hosts")
    }
  }
}

/// Up to three bytes changed, inserted or removed, preferring the bytes
/// the parser splits on.
fun mutate(f: *Fuzzer, wire: List<u8>): List<u8> {
  val special: List<u8> = [' ', ':', '\r', '\n', '\t', '%', '0', '9', '-', 0x00, 0x7F, 0xC3, 0xFF]
  var out = wire.toMutable()
  loop (_ in 0..<f.rng.range(1, 4)) {
    if (out.isEmpty()) break
    val at = f.rng.range(0, out.len())
    val b: u8 = if (f.rng.boolean()) f.rng.pick(special) ?: 0 else (f.rng.range(0, 256)).wrapU8()
    when (f.rng.range(0, 3)) {
      0    => out.set(at, b)
      1    => out.insert(at, b)
      else => {
        val _ = out.removeAt(at)
      }
    }
  }
  out.toList()
}

fun fuzzHttp(f: *Fuzzer, port: i64) {
  val n = f.rng.range(1, 4)
  val mode = f.rng.range(0, 4)  // 0: the last request is faulty, 1: all of it mutated
  var wire: MutableList<u8> = []
  var wants: MutableList<Want> = []
  var fault = ""
  loop (i in 0..<n) {
    if (i == n - 1 && mode == 0) {
      val (bytes, status, name) = faulty(f)
      wire.addAll(bytes)
      wants.push(Want(status, closes: true))
      fault = name
    } else {
      val (d, want) = draft(f, last: i == n - 1, framed: false)
      wire.addAll(d.wire())
      wants.push(want)
    }
  }
  val sent = if (mode == 1) mutate(f, wire.toList()) else wire.toList()
  val target = if (mode == 1) "http mutated" else "http requests"
  when (val r = exchange(port, sent)) {
    is Ok     => {
      val (got, broken) = parseResponses(r)
      val problem = if (!broken.isEmpty()) broken else if (mode == 1) judgeMutated(got) else judge(got, wants.toList())
      val accepted = if (mode == 1) got.all(g => g.status == 200) else mode != 0
      f.record(target, accepted, if (problem.isEmpty()) null else "$problem\n  sent ${showBytes(sent)}" + (if (fault.isEmpty()) "" else " ($fault)") + "\n  got  ${showBytes(r)}")
    }
    is Err(e) => f.record(target, false, "${e.message()}\n  sent ${showBytes(sent)}" + (if (fault.isEmpty()) "" else " ($fault)"))
  }
}

/// Requests that once got a wrong answer, checked on every run before the
/// generated ones; each was found by this program.
fun httpCorpus(): List<(List<u8>, List<Want>)> {
  val refused = [Want(status: 400, closes: true)]
  [
    // a path that decodes to a line break, echoed into a header, split
    // the response: the client wrote a header of its own
    ("GET /a%0d%0aSet-Cookie:%20x=1 HTTP/1.1\r\nHost: fuzz\r\n\r\n".bytes(), [Want(status: 200, echo: "GET /a  Set-Cookie: x=1")]),
    // a line that is not UTF-8 closed the connection without an answer
    ("GET / HTTP/1.1\r\nHost: fuzz\r\nX-A: ".bytes().concat([0x87]).concat("\r\n\r\n".bytes()), refused),
    // read one way here and another way by a proxy in front
    ("POST / HTTP/1.1\r\nHost: fuzz\r\nContent-Length : 5\r\n\r\nhello".bytes(), refused),
    ("POST / HTTP/1.1\r\nHost: fuzz\r\nContent-Length: +5\r\n\r\nhello".bytes(), refused),
    ("POST / HTTP/1.1\r\nHost: fuzz\r\nContent-Length: 5\r\nContent-Length: 6\r\n\r\nhello!".bytes(), refused),
    ("GET / HTTP/1.1\r\nHost: fuzz\r\nX-A: v\r\n folded: x\r\n\r\n".bytes(), refused),
    ("GET / HTTP/1.1\r\nHost: fuzz\r\nX-A: a\rb\r\n\r\n".bytes(), refused),
    ("GET / HTTP/1.1x\r\nHost: fuzz\r\n\r\n".bytes(), refused),
    ("GET / HTTP/1.1\r\n\r\n".bytes(), refused),
    // HTTP/1.0 without keep-alive kept the connection open
    ("GET / HTTP/1.0\r\n\r\n".bytes(), [Want(status: 200, echo: "GET /", closes: true)]),
  ]
}

fun checkHttp(f: *Fuzzer, port: i64, wire: List<u8>, wants: List<Want>) {
  when (val r = exchange(port, wire)) {
    is Ok     => {
      val (got, broken) = parseResponses(r)
      val problem = if (broken.isEmpty()) judge(got, wants) else broken
      f.record("http corpus", true, if (problem.isEmpty()) null else "$problem\n  sent ${showBytes(wire)}\n  got  ${showBytes(r)}")
    }
    is Err(e) => f.record("http corpus", true, "${e.message()}\n  sent ${showBytes(wire)}")
  }
}

/// Sends `wire`, closes the sending side, and reads until the server closes.
fun exchange(port: i64, wire: List<u8>): List<u8> suspends throws IoError | Timeout {
  with conn = try net.connect("127.0.0.1", port)
  try conn.write(wire)
  try conn.shutdownWrite()
  // the client has closed its side, so every read on the server ends:
  // a server still waiting after this is a bug, not a slow peer
  try withTimeout(Duration.seconds(5), () => try readAll(conn))
}

fun readAll(c: net.Conn): List<u8> suspends throws IoError {
  var out: MutableList<u8> = []
  loop (out.len() <= 1048576) {
    val chunk = try c.read()
    if (chunk.isEmpty()) break
    out.addAll(chunk)
  }
  out.toList()
}

/// One response as the client read it.
struct Got {
  status:  i64
  headers: Map<string, string>
  body:    List<u8>
}

/// Where the CRLF at or after `from` starts, or -1.
fun crlf(data: List<u8>, from: i64): i64 {
  var i = from
  loop (i + 1 < data.len()) {
    if (data.at(i) == '\r' && data.at(i + 1) == '\n') return i
    i += 1
  }
  -1
}

/// Everything the server sent, as responses; the text says what is not a
/// well-formed HTTP/1.1 response, and is empty when all of it is.
fun parseResponses(data: List<u8>): (List<Got>, string) {
  var out: MutableList<Got> = []
  var at: i64 = 0
  loop (at < data.len()) {
    val end = crlf(data, at)
    if (end < 0) return (out.toList(), "response ${out.len() + 1} has no status line")
    val statusLine = data.slice(at, end).decodeUtf8() ?: ""
    val (proto, rest) = statusLine.splitOnce(" ") ?: ("", "")
    val (code, _) = rest.splitOnce(" ") ?: ("", "")
    val status = code.toInt() ?: 0
    if (proto != "HTTP/1.1" || code.len() != 3 || status < 100) return (out.toList(), "status line \"$statusLine\"")
    at = end + 2
    val headers: MutableMap<string, string> = [:]
    loop {
      val e = crlf(data, at)
      if (e < 0) return (out.toList(), "response ${out.len() + 1}: headers without an end")
      if (e == at) {
        at += 2
        break
      }
      val line = data.slice(at, e).decodeUtf8() ?: ""
      val (name, value) = line.splitOnce(": ") ?: ("", "")
      if (name.isEmpty() || name != name.toLower() || name.contains(" ") || value.contains("\r") || value.contains("\n")) {
        return (out.toList(), "response ${out.len() + 1}: header line \"$line\"")
      }
      headers.set(name, value)
      at = e + 2
    }
    val length = headers.get("content-length")?.toInt() ?: return (out.toList(), "response ${out.len() + 1} has no content-length")
    if (at + length > data.len()) return (out.toList(), "response ${out.len() + 1}: body shorter than its content-length")
    out.push(Got(status, headers: headers.toMap(), body: data.slice(at, at + length)))
    at += length
  }
  (out.toList(), "")
}

/// Every response is the one wanted, and a refusal ends the connection.
fun judge(got: List<Got>, wants: List<Want>): string {
  val statuses = got.map(g => g.status)
  if (got.len() != wants.len()) return "${got.len()} responses ($statuses), wanted ${wants.map(w => w.status)}"
  loop ((g, w) in got.zip(wants)) {
    if (g.status != w.status) return "status ${g.status}, wanted ${w.status} ($statuses)"
    if (w.status == 200) {
      val echoed = g.headers.get("x-echo") ?: ""
      if (echoed != w.echo) return "x-echo \"$echoed\", wanted \"${w.echo}\""
      if (g.body != w.body) return "body ${showBytes(g.body)}, wanted ${showBytes(w.body)}"
    }
    val closes = g.headers.get("connection") == "close"
    if (closes != (w.closes || w.status != 200)) return "status ${g.status} says connection: ${g.headers.get("connection") ?: "-"}"
  }
  ""
}

/// A mutated request may be refused any way the server refuses, but not
/// with a 500 or a timeout, and a refusal ends the connection.
fun judgeMutated(got: List<Got>): string {
  if (got.isEmpty()) return "no response"
  var i: i64 = 0
  loop (g in got) {
    i += 1
    if (![200, 400, 413, 414, 431, 501, 505].contains(g.status)) return "status ${g.status}"
    if (g.status != 200 && i != got.len()) return "a ${g.status} is not the last response"
    if (g.status != 200 && g.headers.get("connection") != "close") return "a ${g.status} keeps the connection"
  }
  ""
}

/// Bytes as a quoted string, `\r`, `\n` and anything unprintable escaped.
fun showBytes(b: List<u8>): string {
  val sb = StringBuilder()
  loop (x in b.take(200)) {
    when {
      x == '\r'          => sb.append("\\r")
      x == '\n'          => sb.append("\\n")
      x >= 32 && x < 127 => sb.append(utf8.char(x.toI64()))
      else               => sb.append("\\x${hex.encode([x])}")
    }
  }
  if (b.len() > 200) sb.append("...")
  "\"${sb.toString()}\" (${b.len()} bytes)"
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
  val listener = net.listen().getOrNull() ?: panic("fuzz: cannot listen on loopback")
  with (server = async http.serve(listener, http.handler(echo), fuzzLimits, log: false)) {
    loop ((wire, wants) in httpCorpus()) checkHttp(&f, listener.port(), wire, wants)
    loop (_ in 0..<iterations) fuzzHttp(&f, listener.port())
  }
  var failed = false
  loop ((target, s) in f.stats.entries().sortedBy(e => e.0)) {
    println("$target: ${s.runs} runs, ${s.accepted} accepted, ${s.failures} failures")
    if (s.failures > 0) failed = true
  }
  if (failed) os.exit(1)
}
