// HTTP/1.1 request and response framing shared by the server and client.
use io
use time

// One line of the head, refusing a ceiling breach with the status the
// client should see rather than letting the read error escape.
fun headLine(c: io.Stream, max: i64, limit: Duration, tooLong: Fail): string? suspends throws Fail | IoError | Timeout {
  when (withTimeout(limit, () => try c.readLine(max: max))) {
    is Ok(line) => line
    is Err(e)   => when (e) {
      is io.TooLong => throw tooLong
      is Timeout    => throw e
      is IoError    => try rethrow(e)
    }
  }
}

// A line that is not UTF-8 is the client's mistake, answered with a 400;
// any other read error is the connection's, which closes without a word.
fun rethrow(e: IoError): Never throws Fail | IoError {
  if (e.kind == IoKind.InvalidData) throw badRequest("request is not valid UTF-8")
  throw e
}

// The wait for a request line is the idle wait of a kept-alive
// connection. It ends in the line; in null when the peer closed or the
// server began to stop; in a Timeout when the peer stayed silent — the
// quiet end of a connection, not a bad request — or in a 414.
fun requestLine(c: io.Stream, limits: Limits, drain: Drain): string? suspends throws Fail | IoError | Timeout {
  if (drain.stopping.load()) return null
  select {
    val line = async readLineOf(c, limits.requestLineBytes) => line
    sleep(limits.idleTimeout) => throw Timeout(limit: limits.idleTimeout)
    drain.wake.recv() => null
  }
}

// the line, with a line past `max` already the 414 it answers
fun readLineOf(c: io.Stream, max: i64): string? suspends throws Fail | IoError {
  when (c.readLine(max: max)) {
    is Ok(line) => line
    is Err(e)   => when (e) {
      is io.TooLong => throw Fail(status: Status.uriTooLong, text: "URI too long")
      is IoError    => try rethrow(e)
    }
  }
}

// One request from the connection, and whether it spoke HTTP/1.0; null
// when the peer closed between requests or the server is stopping. A
// malformed or oversized request is a Fail (400/413/414/431/501/505), a
// slow one a 408.
//
// The head is read strictly, as RFC 9112 asks of a server: wherever two
// readers could disagree about where a header or the body ends — a space
// before a colon, a folded line, two lengths, a stray CR or NUL — the
// request is refused rather than read one way, since a proxy in front of
// the server may have read it the other way.
fun readRequest(c: io.Stream, peer: string, limits: Limits, drain: Drain): (Request, bool)? throws Fail | IoError | Timeout {
  val first = try requestLine(c, limits, drain) ?: return null
  // from here the whole head is on one clock, so a peer cannot hold the
  // connection open by sending one header every few seconds
  val headDeadline = time.Deadline.after(limits.headerTimeout)
  val [method, target, version] = first.split(" ") else throw badRequest("malformed request line")
  if (!isToken(method) || target.isEmpty()) throw badRequest("malformed request line")
  // `HTTP/` DIGIT `.` DIGIT (RFC 9112 §2.3)
  val (protocol, number) = version.splitOnce("/") ?: ("", "")
  val [major, minor] = number.split(".") else throw badRequest("malformed HTTP version")
  if (protocol != "HTTP" || major.len() != 1 || minor.len() != 1 || !isDigits(major) || !isDigits(minor)) throw badRequest("malformed HTTP version")
  if (major != "1") throw Fail(status: Status.httpVersionNotSupported, text: "HTTP version not supported")
  val http10 = minor == "0"
  val tooManyHeaders = Fail(status: Status.requestHeaderFieldsTooLarge, text: "request header fields too large")
  val headers: MutableMap<string, string> = [:]
  var headerBytes: i64 = 0
  // lines, not entries: repeating one name costs the server the same and
  // the map would collapse them to a single key
  var headerLines: i64 = 0
  val headerTimeout = Fail(status: Status.requestTimeout, text: "request header timeout")
  loop {
    // the deadline is answered with a status, not with a bare Timeout: a
    // Timeout out of `readRequest` is the *idle* wait above, which closes
    // without a word, and a client that dribbles its headers has to be told
    if (headDeadline.expired()) throw headerTimeout
    val line = when (headLine(c, limits.headerLineBytes, headDeadline.remaining(), tooManyHeaders)) {
      is Ok(l)  => l ?: throw badRequest("connection closed inside the headers")
      is Err(e) => when (e) {
        is Timeout => throw headerTimeout
        is Fail    => throw e
        is IoError => throw e
      }
    }
    if (line.isEmpty()) break
    headerBytes += line.len()
    headerLines += 1
    if (headerBytes > limits.headerBytes) throw tooManyHeaders
    if (headerLines > limits.headerCount) throw tooManyHeaders
    // a line that starts with whitespace continues the one before it in
    // the obsolete folded form, which a server must refuse or unfold
    if (line.startsWith(" ") || line.startsWith("\t")) throw badRequest("folded header line")
    val (rawName, rawValue) = line.splitOnce(":") else throw badRequest("malformed header line")
    // the name runs up to the colon with nothing in between (RFC 9112
    // §5.1): `Content-Length : 5` is refused, not trimmed
    if (!isToken(rawName)) throw badRequest("malformed header name")
    if (rawValue.contains("\r") || rawValue.contains("\u{0}")) throw badRequest("CR or NUL in a header value")
    val name = rawName.toLower()
    val value = rawValue.trim()
    val earlier = headers.get(name)
    if (earlier != null) {
      if (name == "host") throw badRequest("more than one Host header")
      if (name == "content-length" && earlier != value) throw badRequest("conflicting content-length")
    }
    headers.set(name, if (name == "connection" && earlier != null) earlier + "," + value else value)
  }
  // HTTP/1.1 names the host it asks (RFC 9112 §3.2)
  if (!http10 && headers.get("host") == null) throw badRequest("missing Host header")
  // Where the body ends is what a proxy in front may have read differently,
  // so a request that says it two ways is refused (RFC 9112 §6.1); the body
  // itself is not read here but when the handler asks (see `Body`)
  val coding = headers.get("transfer-encoding")
  val declared = headers.get("content-length")
  // `Expect: 100-continue` holds the client back until the handler reads;
  // any other expectation cannot be met (RFC 9110 §10.1.1)
  val expect = headers.get("expect")
  if (expect != null && expect.toLower() != "100-continue") throw Fail(status: Status.expectationFailed, text: "expectation failed")
  val waiting = expect != null && !http10
  val body = if (coding != null) {
    if (http10) throw badRequest("transfer-encoding in an HTTP/1.0 request")
    if (declared != null) throw badRequest("both transfer-encoding and content-length")
    if (coding.toLower() != "chunked") throw Fail(status: Status.notImplemented, text: "unsupported transfer coding")
    Body.wire(conn: c, framing: 2, length: 0, waiting: waiting, limits: limits)
  } else if (declared != null) {
    // digits and nothing else: `+5`, `0x10` and `5.0` read differently
    // elsewhere
    if (!isDigits(declared)) throw badRequest("malformed content-length")
    val length = try declared.toInt() ?! badRequest("malformed content-length")
    if (length == 0) Body.none(limits) else Body.wire(conn: c, framing: 1, length: length, waiting: waiting, limits: limits)
  } else {
    Body.none(limits)
  }
  val (rawPath, rawQuery) = target.splitOnce("?") ?: (target, "")
  val req = Request(method: Method(name: method), path: percentDecode(rawPath, plusIsSpace: false), query: parseQuery(rawQuery), headers, body, peer, rawQuery)
  (req, http10)
}

// The characters a method or a header name may hold (RFC 9110 §5.6.2).
fun isToken(s: string): bool =>
  !s.isEmpty() && s.bytes().all(b => b.isAsciiAlphanumeric() || tokenMarks.contains(b))

val tokenMarks: List<u8> = "!#$%&'*+-.^_`|~".bytes()

fun isDigits(s: string): bool => !s.isEmpty() && s.bytes().all(b => b.isAsciiDigit())

fun parseQuery(text: string): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  if (text.isEmpty()) return out.toMap()
  loop (pair in text.split("&")) {
    if (pair.isEmpty()) continue
    // a key without `=` has the empty value
    val (k, v) = pair.splitOnce("=") ?: (pair, "")
    out.set(percentDecode(k, plusIsSpace: true), percentDecode(v, plusIsSpace: true))
  }
  out
}

/// Decodes `%XX` escapes (and `+` as a space in query strings); text that
/// does not decode to UTF-8 is returned as it came.
public fun percentDecode(s: string, plusIsSpace: bool): string {
  if (!s.contains("%") && !(plusIsSpace && s.contains("+"))) return s
  val bytes = s.bytes()
  val out: MutableList<u8> = []
  var i: i64 = 0
  loop (i < bytes.len()) {
    val b = bytes.at(i)
    if (b == '%' && i + 2 < bytes.len()) {
      val hi = bytes.at(i + 1)?.digitValue(16)
      val lo = bytes.at(i + 2)?.digitValue(16)
      if (val high = hi && val low = lo) {
        out.push((high * 16 + low).wrapU8())
        i += 3
        continue
      }
    }
    if (plusIsSpace && b == '+') out.push(' ') else out.push(b)
    i += 1
  }
  out.toList().decodeUtf8() ?: s
}

// `headOnly` is the answer to a HEAD request: every header a GET would
// get, `content-length` included, and no body (RFC 9110 §9.3.2).
fun writeResponse(c: io.Stream, peer: string, resp: Response, close: bool, headOnly: bool = false, http10: bool = false) suspends throws IoError {
  val head = StringBuilder()
  head.append("HTTP/1.1 ${resp.status.code} ${resp.status.reason()}\r\n")
  var hasType = false
  var hasDate = false
  loop ((name, value) in resp.headers.entries()) {
    // the server frames the response itself; a second length or a
    // connection header of the handler's would contradict it
    if (!isToken(name) || framing.contains(name.toLower())) continue
    if (name.toLower() == "content-type") hasType = true
    if (name.toLower() == "date") hasDate = true
    // a value often comes from the request — a redirect's location, an
    // echoed path that decoded to `\r\n` — and a line break in it would
    // let the client write the rest of the response (RFC 9110 §5.5)
    head.append("$name: ${value.replace("\r", " ").replace("\n", " ").replace("\u{0}", " ")}\r\n")
  }
  loop (cookie in resp.cookies) {
    head.append("set-cookie: ${setCookieLine(cookie)}\r\n")
  }
  val bodyless = hasNoBody(resp.status)
  val producer = if (bodyless) null else resp.stream
  if (!bodyless) {
    if (!hasType && (!resp.body.isEmpty() || producer != null)) head.append("content-type: application/octet-stream\r\n")
    // a streamed body has no length to announce: chunks say where it ends,
    // and an HTTP/1.0 client, which has no chunks, reads to the close
    if (producer == null) {
      head.append("content-length: ${resp.body.len()}\r\n")
    } else if (val n = resp.streamLength) {
      head.append("content-length: $n\r\n")
    } else if (!http10) {
      head.append("transfer-encoding: chunked\r\n")
    }
  }
  if (!hasDate) head.append("date: ${httpDate(time.now())}\r\n")
  head.append(if (close) "connection: close\r\n" else "connection: keep-alive\r\n")
  head.append("\r\n")
  val bytes = head.toString().bytes()
  if (val p = producer) {
    try c.write(bytes)
    if (headOnly) return
    val chunked = !http10 && resp.streamLength == null
    val out = BodyWriter.toConn(c, chunked: chunked, length: resp.streamLength)
    val complete = produce(p, out) && out.missing() == 0
    // the head has gone, so a failure cannot be a 500: the connection ends
    // without the last chunk, and the client sees a body that stops short
    if (!complete) throw IoError(path: peer, code: 0, detail: "the response body could not be completed", kind: IoKind.Other)
    if (chunked) try c.write("0\r\n\r\n".bytes())
    return
  }
  try c.write(if (headOnly || bodyless) bytes else bytes.concat(resp.body))
}

const framing = ["content-length", "transfer-encoding", "connection"]

// 1xx, 204 and 304 never carry a body, and a 1xx or 204 may not even say
// `content-length` (RFC 9110 §6.4.1, §8.6); a 304's length would be the
// 200's, which a handler returning `empty(304)` does not know.
fun hasNoBody(status: Status): bool => status.isInformational() || status in [Status.noContent, Status.notModified]

/// A time in the format HTTP dates use: `Sun, 06 Nov 1994 08:49:37 GMT`.
///
/// The format itself lives in `std/time`, with the parser for the two
/// obsolete forms a recipient must also accept (`time.parseHttp`) —
/// `Last-Modified` and `If-Modified-Since` are two ends of one conversation
/// and belong in one place.
public fun httpDate(t: time.Timestamp): string => time.formatHttp(t)
