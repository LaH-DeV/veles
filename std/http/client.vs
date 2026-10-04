// The HTTP/1.1 client (D127): `fetch` and the Go spellings, on a pool of
// keep-alive connections. The server's wire code is reused where the two
// sides agree (header tokens, chunk sizes, percent-encoding); everything a
// client has to distrust — a status line, a framing it did not ask for, a
// body of any size — is read here under limits of its own.
use io, json as js, net, time

// ---------------------------------------------------------------------------
// errors

/// Why a request failed before it had an answer — or while its body was read.
public enum FetchKind {
  /// The URL, a header or the method is not one the client will send.
  InvalidRequest
  /// A scheme or feature this build does not have (`https` until TLS lands).
  Unsupported
  /// No connection could be made.
  Connect
  /// The whole-request time ran out.
  Timeout
  /// The server closed the connection without answering.
  Closed
  /// The connection failed while sending or reading.
  Io
  /// The answer is not HTTP the client accepts.
  Protocol
  /// More redirects than `maxRedirects`.
  TooManyRedirects
  /// A body longer than the ceiling the caller allowed.
  TooLarge
}

/// A request that produced no usable answer. A status of 404 or 500 is not
/// one — it is a `ClientResponse`; `ensureSuccess()` turns it into a
/// `StatusError` when that is what the caller wants.
public error FetchError {
  public url:    string
  public detail: string
  public kind:   FetchKind
  fun message(): string = "${this.detail}: ${this.url}"
}

/// A non-2xx answer that the caller declared an error with `ensureSuccess()`.
public error StatusError {
  public url:    string
  public status: Status
  fun message(): string = "${this.url} answered ${this.status}"
}

fun invalid(url: string, detail: string): FetchError =
  FetchError(url: shown(url), detail, kind: FetchKind.InvalidRequest)

fun protocolError(url: string, detail: string): FetchError =
  FetchError(url, detail, kind: FetchKind.Protocol)

fun timedOut(url: string): FetchError =
  FetchError(url, detail: "the request timed out", kind: FetchKind.Timeout)

fun ioFailure(url: string, e: IoError): FetchError =
  FetchError(url, detail: "the connection failed (${e.detail})", kind: FetchKind.Io)

fun closedWithoutAnswer(url: string): FetchError =
  FetchError(url, detail: "the server closed the connection without answering", kind: FetchKind.Closed)

// text that came from a caller or the wire, safe to put in a message or a log
fun shown(text: string): string {
  val one = text.replace("\r", "?").replace("\n", "?").replace("\u{0}", "?")
  if (one.len() > 200) (one.substring(0, 200) ?: one) + "…" else one
}

// ---------------------------------------------------------------------------
// URLs

// A parsed absolute `http` or `https` URL: where to connect and what to ask for.
struct Url {
  secure: bool
  // lower-case; an IPv6 literal without its brackets
  host: string
  port: i64
  // path and query, always starting with `/`, non-ASCII bytes percent-encoded
  target: string

  fun scheme(): string = if (this.secure) "https" else "http"

  fun defaultPort(): i64 = if (this.secure) 443 else 80

  // what `Host` says: the port left out when it is the scheme's own
  fun authority(): string {
    val name = if (this.host.contains(":")) "[${this.host}]" else this.host
    if (this.port == this.defaultPort()) name else "$name:${this.port}"
  }

  fun origin(): string = "${this.scheme()}://${this.authority()}"

  fun text(): string = this.origin() + this.target
}

fun parseUrl(text: string): Url throws FetchError {
  // a space or a control character would let a URL end the request line early
  // and write the rest of the request itself
  if (text.bytes().any(b => b <= 32 || b == 127)) {
    throw invalid(text, "a URL cannot hold spaces or control characters")
  }
  val (scheme, rest) = text.splitOnce("://") else throw invalid(text, "a URL starts with http:// or https://")
  val lower = scheme.toLower()
  if (lower != "http" && lower != "https") throw invalid(text, "only http and https URLs can be fetched")
  val secure = lower == "https"
  val (noFragment, _) = rest.splitOnce("#") ?: (rest, "")
  val slash = noFragment.indexOf("/")
  val query = noFragment.indexOf("?")
  var end = noFragment.len()
  if (slash >= 0 && slash < end) end = slash
  if (query >= 0 && query < end) end = query
  val authority = noFragment.substring(0, end) ?: ""
  val tail = noFragment.substring(end, noFragment.len()) ?: ""
  if (authority.contains("@")) {
    throw invalid(text, "a URL with a user name is not supported; send an authorization header")
  }
  var host = authority
  var port: i64 = if (secure) 443 else 80
  if (authority.startsWith("[")) {
    val close = authority.indexOf("]")
    if (close < 0) throw invalid(text, "an IPv6 address needs its closing bracket")
    host = authority.substring(1, close) ?: ""
    val after = authority.substring(close + 1, authority.len()) ?: ""
    if (!after.isEmpty()) {
      if (!after.startsWith(":")) throw invalid(text, "malformed host")
      port = try portOf(text, after.substring(1, after.len()) ?: "")
    }
    if (host.isEmpty() || !host.bytes().all(b => hexValue(b) >= 0 || b == ':' || b == '.')) {
      throw invalid(text, "malformed IPv6 address")
    }
  } else {
    val colon = authority.indexOf(":")
    if (colon >= 0) {
      host = authority.substring(0, colon) ?: ""
      port = try portOf(text, authority.substring(colon + 1, authority.len()) ?: "")
    }
    if (host.isEmpty()) throw invalid(text, "a URL needs a host")
    if (!host.bytes().all(b => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '.' || b == '-' || b == '_')) {
      throw invalid(text, "a host name is letters, digits, dots and hyphens; write an international name in its punycode form")
    }
  }
  val target = if (tail.isEmpty()) "/" else if (tail.startsWith("?")) "/$tail" else tail
  Url(secure, host: host.toLower(), port, target: encodeTarget(target))
}

fun portOf(url: string, text: string): i64 throws FetchError {
  if (!isDigits(text) || text.len() > 5) throw invalid(url, "malformed port")
  val n = try text.toInt() ?! invalid(url, "malformed port")
  if (n < 1 || n > 65535) throw invalid(url, "a port is 1 to 65535")
  n
}

// bytes outside ASCII go on the wire as `%XX`, as a browser sends them
fun encodeTarget(target: string): string {
  if (target.bytes().all(b => b < 128)) return target
  val digits = "0123456789ABCDEF".bytes()
  val out: MutableList<u8> = []
  loop (b in target.bytes()) {
    if (b < 128) {
      out.push(b)
    } else {
      val n = b.toI64()
      out.push('%')
      out.push(digits.at(n / 16) ?: '0')
      out.push(digits.at(n % 16) ?: '0')
    }
  }
  out.toList().decodeUtf8() ?: target
}

// `location` as a redirect means it, resolved against the URL that sent it
// (RFC 3986 §5)
fun resolve(base: Url, location: string): Url throws FetchError {
  val (loc, _) = location.trim().splitOnce("#") ?: (location.trim(), "")
  if (hasScheme(loc)) return try parseUrl(loc)
  if (loc.startsWith("//")) return try parseUrl("${base.scheme()}:$loc")
  if (loc.isEmpty()) return base
  val target = if (loc.startsWith("/")) {
    loc
  } else if (loc.startsWith("?")) {
    pathOf(base.target) + loc
  } else {
    directoryOf(base.target) + loc
  }
  try parseUrl(base.origin() + removeDots(target))
}

fun hasScheme(text: string): bool {
  val i = text.indexOf("://")
  if (i <= 0) return false
  (text.substring(0, i) ?: "").bytes().all(b => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '+' || b == '-' || b == '.')
}

fun pathOf(target: string): string {
  val q = target.indexOf("?")
  if (q < 0) target else (target.substring(0, q) ?: target)
}

// the path up to and including its last `/`
fun directoryOf(target: string): string {
  val path = pathOf(target)
  val i = path.lastIndexOf("/")
  if (i < 0) "/" else (path.substring(0, i + 1) ?: "/")
}

// `.` and `..` segments resolved, as a server would before routing
fun removeDots(target: string): string {
  val path = pathOf(target)
  val query = target.substring(path.len(), target.len()) ?: ""
  val parts = path.split("/")
  val out: MutableList<string> = []
  loop ((i, part) in parts.enumerate()) {
    if (i == 0) continue
    val last = i == parts.len() - 1
    if (part == ".") {
      if (last) out.push("")
    } else if (part == "..") {
      val _ = out.pop()
      if (last) out.push("")
    } else {
      out.push(part)
    }
  }
  "/" + out.join("/") + query
}

// ---------------------------------------------------------------------------
// what a request carries

/// A request body with the content type that goes with it. Built by the
/// statics, so the type is never forgotten and never contradicts the bytes:
///
/// ```veles
/// http.post(url, body: http.Payload.text("hello"))
/// http.post(url, body: try http.Payload.json(note))
/// http.post(url, body: http.Payload.form([("name", "ann"), ("tag", "a"), ("tag", "b")]))
/// ```
public struct Payload {
  public data: List<u8>
  /// The `Content-Type` it is sent with, unless the request sets one itself.
  public contentType: string?

  /// Raw bytes with their type (`application/octet-stream` when not said).
  public static fun bytes(data: List<u8>, contentType: MediaType = MediaType.octetStream): Payload =
    Payload(data, contentType: contentType.name)

  /// UTF-8 text (`text/plain` when not said).
  public static fun text(text: string, contentType: MediaType = MediaType.text): Payload =
    Payload(data: text.bytes(), contentType: contentType.name)

  /// `value` as JSON (`application/json`).
  public static fun json<T: Encodable>(value: T): Payload throws EncodeError =
    Payload(data: (try js.encode(value)).bytes(), contentType: MediaType.json.name)

  /// `name=value` pairs as a urlencoded form, repeats kept.
  public static fun form(fields: List<(string, string)>): Payload {
    val parts = fields.map(f => percentEncode(f.0) + "=" + percentEncode(f.1))
    Payload(data: parts.join("&").bytes(), contentType: MediaType.form.name)
  }
}

// ---------------------------------------------------------------------------
// the answer

// Where a body's read has got to, behind a Mutex so the response and every
// copy of it see one place.
struct WireState {
  // 0 no body, 1 Content-Length, 2 chunked, 3 until the connection closes
  var framing: i64 = 0
  // bytes left of the length, or of the chunk being read
  var remaining: i64 = 0
  // Content-Length; -1 when it is not known
  var declared: i64 = -1
  var done:     bool = false
  // the connection may carry another request once this body is read
  var reusable: bool = false
}

// Where a finished connection goes: back to the client's pool.
struct Keep {
  pool: Pool
  key:  string
  max:  i64
  ttl:  Duration

  fun store(c: net.Conn) {
    this.pool.put(this.key, c, this.max, this.ttl)
  }
}

/// The body of an answer, read a piece at a time (`res.stream()`); `bytes`,
/// `text` and `json` on the response read it whole. Reading it to the end
/// hands the connection back for the next request; `close()` (or the end of
/// a `with`) gives it up half read.
public struct ClientBody {
  conn:     net.Conn?
  state:    Mutex<WireState>
  deadline: time.Deadline
  keep:     Keep?
  url:      string

  // no body at all: the connection is free again already
  static fun none(conn: net.Conn, keep: Keep, reusable: bool, url: string, deadline: time.Deadline): ClientBody {
    if (reusable) keep.store(conn) else conn.close()
    ClientBody(conn: null, state: Mutex(value: WireState(done: true)), deadline, keep: null, url)
  }

  static fun wire(conn: net.Conn, keep: Keep, framing: i64, length: i64, reusable: bool, url: string, deadline: time.Deadline): ClientBody =
    ClientBody(
      conn,
      state: Mutex(value: WireState(framing, remaining: length, declared: if (framing == 1) length else -1, reusable)),
      deadline,
      keep,
      url,
    )

  /// How many bytes the body has, when the answer said: its `Content-Length`.
  public fun length(): i64? {
    val declared = this.state.get().declared
    if (declared < 0) null else declared
  }

  /// Up to `max` bytes, as soon as any have arrived; empty at the end of the
  /// body. A body cut short or framed wrongly is a `Protocol` error, and the
  /// request's total time still applies.
  public fun read(max: i64 = 65536): List<u8> suspends throws FetchError {
    val st = this.state.get()
    if (st.done || max <= 0) return []
    val c = this.conn ?: return []
    when (withTimeout(this.deadline.remaining(), () => try this.pull(c, max))) {
      is Ok(chunk) => chunk
      is Err(e)    => {
        this.abandon()
        when (e) {
          is Timeout    => throw FetchError(url: this.url, detail: "the request timed out while its body was read", kind: FetchKind.Timeout)
          is IoError    => throw ioFailure(this.url, e)
          is FetchError => throw e
        }
      }
    }
  }

  // the rest of the body, refusing more than `max` bytes
  fun collect(max: i64): List<u8> suspends throws FetchError {
    val declared = this.state.get().declared
    if (declared > max) {
      this.abandon()
      throw FetchError(url: this.url, detail: "the body is $declared bytes, more than the $max allowed", kind: FetchKind.TooLarge)
    }
    val out: MutableList<u8> = []
    loop {
      val chunk = try this.read(65536)
      if (chunk.isEmpty()) break
      if (out.len() + chunk.len() > max) {
        this.abandon()
        throw FetchError(url: this.url, detail: "the body is more than the $max bytes allowed", kind: FetchKind.TooLarge)
      }
      out.addAll(chunk)
    }
    out.toList()
  }

  fun pull(c: net.Conn, max: i64): List<u8> suspends throws FetchError | IoError {
    val st = this.state.get()
    if (st.framing == 1) {
      val chunk = try c.read(max: max.min(st.remaining))
      if (chunk.isEmpty()) throw protocolError(this.url, "the connection closed before the whole body arrived")
      this.state.withLock(s => {
        s.remaining -= chunk.len()
      })
      if (st.remaining - chunk.len() == 0) this.finish()
      return chunk
    }
    if (st.framing == 2) return try this.pullChunked(c, max)
    // the body ends where the connection does
    val chunk = try c.read(max: max)
    if (chunk.isEmpty()) this.finish()
    chunk
  }

  // one piece of a chunked body: the chunk header when none is open, then
  // what has arrived of the chunk's data
  fun pullChunked(c: net.Conn, max: i64): List<u8> suspends throws FetchError | IoError {
    var remaining = this.state.get().remaining
    if (remaining == 0) {
      val header = try this.line(c, 1024)
      val size = parseChunkSize(header) ?: throw protocolError(this.url, "malformed chunk size")
      if (size == 0) {
        try this.trailers(c)
        this.finish()
        return []
      }
      remaining = size
    }
    val chunk = try c.read(max: max.min(remaining))
    if (chunk.isEmpty()) throw protocolError(this.url, "the body ended inside a chunk")
    remaining -= chunk.len()
    // the line break that ends a chunk's data
    if (remaining == 0 && !(try this.line(c, 2).isEmpty())) throw protocolError(this.url, "malformed chunk end")
    this.state.withLock(s => {
      s.remaining = remaining
    })
    chunk
  }

  // trailer fields follow the last chunk; they are read and dropped
  fun trailers(c: net.Conn) suspends throws FetchError | IoError {
    var lines: i64 = 0
    loop {
      val line = try this.line(c, 8192)
      if (line.isEmpty()) return
      lines += 1
      if (lines > 32) throw protocolError(this.url, "too many trailer fields")
    }
  }

  fun line(c: net.Conn, max: i64): string suspends throws FetchError | IoError {
    when (c.readLine(max: max)) {
      is Ok(l)  => l ?: throw protocolError(this.url, "the connection closed inside the body framing")
      is Err(e) => when (e) {
        is io.TooLong => throw protocolError(this.url, "a framing line is too long")
        is IoError    => throw e
      }
    }
  }

  // the body is read: the connection goes back to the pool, or is closed
  fun finish() {
    val (first, reusable) = this.state.withLock(s => {
      val was = s.done
      s.done = true
      (!was, s.reusable)
    })
    if (!first) return
    val c = this.conn ?: return
    val k = this.keep
    if (reusable && k != null) k.store(c) else c.close()
  }

  // the body will not be read to its end: the connection cannot be reused
  fun abandon() {
    val first = this.state.withLock(s => {
      val was = s.done
      s.done = true
      !was
    })
    if (!first) return
    val c = this.conn ?: return
    c.close()
  }

  implement Closeable {
    fun close() {
      this.abandon()
    }
  }
}

/// What a request was answered with. A 404 or a 500 is an answer, not an
/// error (as in `fetch` and Go): look at `status`, or call `ensureSuccess()`.
/// Read the body whole — `text()`, `bytes()`, `json<T>()` — or a piece at a
/// time with `stream()`; a response whose body is never read holds its
/// connection, so one used only for its status is closed (`with res = ...`
/// does it).
public struct ClientResponse {
  public status: Status
  /// Lower-case names; a header sent twice has its values joined by `, `
  /// (`Set-Cookie` is in `setCookies`).
  public headers:    Map<string, string>
  public setCookies: List<string>
  /// The URL that answered, after any redirects.
  public url: string
  /// Whether the status is 2xx.
  public ok: bool
  body:      ClientBody

  /// A header by name, case-insensitively.
  public fun header(name: string): string? = this.headers.get(name.toLower())

  /// How many bytes the body has, when the answer said (its `Content-Length`);
  /// null for a chunked body or one that runs to the close.
  public fun length(): i64? = this.body.length()

  /// This response, or a `StatusError` when the status is not 2xx — the body
  /// of an error answer is dropped.
  public fun ensureSuccess(): ClientResponse throws StatusError {
    if (this.ok) return this
    this.body.abandon()
    throw StatusError(url: this.url, status: this.status)
  }

  /// The whole body; more than `max` bytes (64 MiB when not said) is a
  /// `TooLarge` error — before the first byte when `Content-Length` says so.
  public fun bytes(max: i64 = defaultBodyMax): List<u8> suspends throws FetchError = try this.body.collect(max)

  /// The whole body as UTF-8 text.
  public fun text(max: i64 = defaultBodyMax): string suspends throws FetchError =
    try (try this.bytes(max)).decodeUtf8() ?! protocolError(this.url, "the body is not valid UTF-8")

  /// The whole body read as JSON into a `T`.
  public fun json<T: Decodable>(max: i64 = defaultBodyMax): T suspends throws FetchError | DecodeError =
    try js.decode<T>(try this.text(max))

  /// The body a piece at a time, for one too large to hold.
  public fun stream(): ClientBody = this.body

  // a body that is read and dropped so that the connection can be reused;
  // too long to be worth it and it is closed
  fun discard() suspends {
    when (this.body.collect(65536)) {
      is Ok(_)  => { }
      is Err(_) => this.body.abandon()
    }
  }

  implement Closeable {
    fun close() {
      this.body.abandon()
    }
  }
}

/// The default ceiling on a body read whole: 64 MiB.
const defaultBodyMax: i64 = 64 * 1024 * 1024

// ---------------------------------------------------------------------------
// the connection pool

struct Idle {
  conn:    net.Conn
  expires: time.Deadline
}

fun newIdle(): Mutex<MutableMap<string, MutableList<Idle>>> {
  val empty: MutableMap<string, MutableList<Idle>> = [:]
  Mutex(value: empty)
}

// Idle connections by `scheme://host:port`. Nothing runs in the background: a
// connection that sat too long is found out, and closed, when the next
// request asks for one.
struct Pool {
  idle: Mutex<MutableMap<string, MutableList<Idle>>> = newIdle()

  // the most recently used live connection to `key`, or null
  fun take(key: string): net.Conn? {
    val expired: MutableList<net.Conn> = []
    val found = this.idle.withLock(m => {
      var out: net.Conn? = null
      val list = m.get(key)
      if (list != null) {
        loop {
          val item = list.pop() ?: break
          if (item.expires.expired()) {
            expired.push(item.conn)
          } else {
            out = item.conn
            break
          }
        }
      }
      out
    })
    loop (c in expired) {
      c.close()
    }
    found
  }

  // keeps `c` for `key`, unless `max` are already waiting there
  fun put(key: string, c: net.Conn, max: i64, ttl: Duration) {
    val kept = this.idle.withLock(m => {
      val entry = Idle(conn: c, expires: time.Deadline.after(ttl))
      val existing = m.get(key)
      if (existing != null) {
        if (existing.len() < max) {
          existing.push(entry)
          true
        } else {
          false
        }
      } else if (max > 0) {
        val list: MutableList<Idle> = [entry]
        m.set(key, list)
        true
      } else {
        false
      }
    })
    if (!kept) c.close()
  }

  fun closeAll() {
    val all: MutableList<net.Conn> = []
    this.idle.withLock(m => {
      loop (list in m.values()) {
        loop (item in list) {
          all.push(item.conn)
        }
      }
      m.clear()
    })
    loop (c in all) {
      c.close()
    }
  }
}

// ---------------------------------------------------------------------------
// the client

/// A client with its own settings and its own pool of connections. The
/// functions `http.fetch`, `http.get`, … use a shared default client; make
/// one of your own for a different timeout, headers every request carries,
/// or a pool of its own, and close it (`with client = ...`) to drop its idle
/// connections.
///
/// ```veles
/// with client = http.Client(timeout: Duration.seconds(5), headers: ["user-agent": "notes/1.0"])
/// val users = try client.get("http://api.example.com/users").json<List<User>>()
/// ```
public struct Client {
  /// The whole request, body included, from the first byte sent to the last
  /// read; each call may set its own.
  public timeout: Duration = Duration.seconds(30)
  /// Sent with every request; the request's own headers win.
  public headers: Map<string, string> = [:]
  /// How many redirects one request follows.
  public maxRedirects: i64 = 10
  /// How many idle connections are kept per `scheme://host:port`.
  public maxIdlePerHost: i64 = 8
  /// How long an idle connection is kept; a longer-idle one is closed when
  /// the pool is next asked for it.
  public idleTimeout: Duration = Duration.seconds(30)
  private pool:       Pool = Pool()

  /// One request. `url` is absolute (`http://…`); `headers` are added to the
  /// client's; `body` goes with a `Content-Length`. A redirect (301, 302,
  /// 303, 307, 308) is followed for `GET` and `HEAD` only — any other method
  /// gets the redirect back as its answer — and `authorization` and `cookie`
  /// headers are dropped when it leaves the host. `retry` is how many more
  /// times to try after a failure to connect, a lost connection, a timeout or
  /// a 502/503/504 — only for an idempotent method (GET, HEAD, PUT, DELETE,
  /// OPTIONS, TRACE), waiting 100 ms, 200 ms, 400 ms… between tries.
  public fun fetch(
    url: string,
    method: Method = Method.get,
    headers: Map<string, string> = [:],
    body: Payload? = null,
    timeout: Duration? = null,
    redirect: bool = true,
    retry: i64 = 0,
  ): ClientResponse suspends throws FetchError {
    if (retry < 0) panic("fetch: retry cannot be negative, got $retry")
    val retries = if (isIdempotent(method)) retry else 0
    var attempt: i64 = 0
    loop {
      val outcome = this.run(url, method, headers, body, timeout ?: this.timeout, redirect)
      when (outcome) {
        is Ok(res) => {
          if (attempt >= retries || !retryableStatus(res.status)) return res
          res.discard()
        }
        is Err(e)  => {
          if (attempt >= retries || !retryable(e)) throw e
        }
      }
      attempt += 1
      await sleep(backoff(attempt))
    }
  }

  /// `fetch` with the method `GET`.
  public fun get(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.get, headers: headers, timeout: timeout, redirect: redirect, retry: retry)

  /// `fetch` with the method `HEAD`: the headers a `GET` would get, no body.
  public fun head(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.head, headers: headers, timeout: timeout, redirect: redirect, retry: retry)

  /// `fetch` with the method `POST`.
  public fun post(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.post, headers: headers, body: body, timeout: timeout, redirect: redirect)

  /// `fetch` with the method `PUT`.
  public fun put(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.put, headers: headers, body: body, timeout: timeout, redirect: redirect, retry: retry)

  /// `fetch` with the method `PATCH`.
  public fun patch(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.patch, headers: headers, body: body, timeout: timeout, redirect: redirect)

  /// `fetch` with the method `DELETE`.
  public fun delete(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
    try this.fetch(url, method: Method.delete, headers: headers, timeout: timeout, redirect: redirect, retry: retry)

  // one try of a request: connect or reuse, send, read the head, follow redirects
  fun run(text: string, method: Method, headers: Map<string, string>, payload: Payload?, timeout: Duration, redirect: bool): ClientResponse suspends throws FetchError {
    val deadline = time.Deadline.after(timeout)
    var url = try parseUrl(text)
    var sent = merged(this.headers, headers)
    var hops: i64 = 0
    loop {
      val res = try this.once(url, method, sent, payload, deadline)
      if (!redirect || !followsRedirect(method, res.status)) return res
      val location = res.header(Header.location) ?: return res
      res.discard()
      if (hops >= this.maxRedirects) {
        throw FetchError(url: shown(text), detail: "more than ${this.maxRedirects} redirects", kind: FetchKind.TooManyRedirects)
      }
      hops += 1
      val next = try resolve(url, location)
      // credentials belong to the host they were given for
      if (next.origin() != url.origin()) sent = withoutCredentials(sent)
      url = next
    }
  }

  // the request on a pooled connection when there is one — and, if that had
  // gone stale, again on a new one, for a method that may be repeated — or
  // on a new connection
  fun once(url: Url, method: Method, headers: Map<string, string>, payload: Payload?, deadline: time.Deadline): ClientResponse suspends throws FetchError {
    if (url.secure) {
      throw FetchError(url: url.text(), detail: "https is not available yet: std has no TLS", kind: FetchKind.Unsupported)
    }
    // refused before any connection is made or reused
    val bytes = try requestBytes(url, method, headers, payload)
    val key = url.origin()
    val keep = Keep(pool: this.pool, key, max: this.maxIdlePerHost, ttl: this.idleTimeout)
    val pooled = this.pool.take(key)
    if (pooled != null) {
      when (exchange(pooled, url, method, bytes, deadline, keep)) {
        is Ok(res) => return res
        is Err(e)  => {
          pooled.close()
          // the server may have closed it while it sat idle
          if (!(isIdempotent(method) && (e.kind == FetchKind.Io || e.kind == FetchKind.Closed))) throw e
        }
      }
    }
    val fresh = try open(url, deadline)
    when (exchange(fresh, url, method, bytes, deadline, keep)) {
      is Ok(res) => res
      is Err(e)  => {
        fresh.close()
        throw e
      }
    }
  }

  implement Closeable {
    fun close() {
      this.pool.closeAll()
    }
  }
}

fun isIdempotent(m: Method): bool =
  m == Method.get || m == Method.head || m == Method.put || m == Method.delete || m == Method.options || m == Method.trace

fun followsRedirect(m: Method, s: Status): bool =
  (m == Method.get || m == Method.head) && (s.code == 301 || s.code == 302 || s.code == 303 || s.code == 307 || s.code == 308)

fun retryableStatus(s: Status): bool = s.code == 502 || s.code == 503 || s.code == 504

fun retryable(e: FetchError): bool =
  e.kind == FetchKind.Connect || e.kind == FetchKind.Closed || e.kind == FetchKind.Io || e.kind == FetchKind.Timeout

// 100 ms, 200 ms, 400 ms … up to five seconds
fun backoff(attempt: i64): Duration {
  val millis = 100 * (1 << (attempt - 1).min(6))
  Duration.millis(millis.min(5000))
}

fun merged(base: Map<string, string>, extra: Map<string, string>): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  loop ((k, v) in base.entries()) {
    out.set(k.toLower(), v)
  }
  loop ((k, v) in extra.entries()) {
    out.set(k.toLower(), v)
  }
  out.toMap()
}

const credentialHeaders = ["authorization", "cookie", "proxy-authorization"]

fun withoutCredentials(headers: Map<string, string>): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  loop ((k, v) in headers.entries()) {
    if (!credentialHeaders.contains(k)) out.set(k, v)
  }
  out.toMap()
}

// ---------------------------------------------------------------------------
// the wire

fun open(url: Url, deadline: time.Deadline): net.Conn suspends throws FetchError {
  when (withTimeout(deadline.remaining(), () => try net.connect(url.host, url.port))) {
    is Ok(c)  => c
    is Err(e) => when (e) {
      is Timeout => throw FetchError(url: url.text(), detail: "timed out connecting", kind: FetchKind.Timeout)
      is IoError => throw FetchError(url: url.text(), detail: "could not connect (${e.detail})", kind: FetchKind.Connect)
    }
  }
}

const defaultAgent = "veles-http/1"

// The framing is the client's: a header of the caller's that contradicted it
// would let the request be read two ways.
const clientHeaders = ["host", "content-length", "transfer-encoding", "connection", "expect"]

fun requestBytes(url: Url, method: Method, headers: Map<string, string>, payload: Payload?): List<u8> throws FetchError {
  val shownUrl = url.text()
  if (!isToken(method.name)) throw invalid(shownUrl, "malformed method '${shown(method.name)}'")
  val head = StringBuilder()
  head.append("${method.name} ${url.target} HTTP/1.1\r\n")
  head.append("host: ${url.authority()}\r\n")
  var hasType = false
  var hasAgent = false
  loop ((name, value) in headers.entries()) {
    val lower = name.toLower()
    if (!isToken(name)) throw invalid(shownUrl, "malformed header name '${shown(name)}'")
    if (clientHeaders.contains(lower)) throw invalid(shownUrl, "the client sets '$lower' itself")
    // a line break in a value would let it write the rest of the request
    if (value.contains("\r") || value.contains("\n") || value.contains("\u{0}")) {
      throw invalid(shownUrl, "the header '$lower' holds a line break")
    }
    if (lower == "content-type") hasType = true
    if (lower == "user-agent") hasAgent = true
    head.append("$lower: $value\r\n")
  }
  if (!hasAgent) head.append("user-agent: $defaultAgent\r\n")
  var body: List<u8> = []
  if (val p = payload) {
    body = p.data
    val kind = p.contentType
    if (!hasType && kind != null) head.append("content-type: $kind\r\n")
  }
  if (payload != null || method == Method.post || method == Method.put || method == Method.patch) {
    head.append("content-length: ${body.len()}\r\n")
  }
  head.append("\r\n")
  head.toString().bytes().concat(body)
}

fun sendAll(c: net.Conn, bytes: List<u8>, deadline: time.Deadline, url: string) suspends throws FetchError {
  when (withTimeout(deadline.remaining(), () => try c.write(bytes))) {
    is Ok(_)  => { }
    is Err(e) => when (e) {
      is Timeout => throw timedOut(url)
      is IoError => throw ioFailure(url, e)
    }
  }
}

// One line of the head, within the request's time
fun responseLine(c: net.Conn, url: string, deadline: time.Deadline, max: i64): string? suspends throws FetchError {
  when (withTimeout(deadline.remaining(), () => try c.readLine(max: max))) {
    is Ok(l)  => l
    is Err(e) => when (e) {
      is Timeout    => throw timedOut(url)
      is io.TooLong => throw protocolError(url, "a line of the response head is too long")
      is IoError    => throw ioFailure(url, e)
    }
  }
}

struct Head {
  status:  Status
  headers: Map<string, string>
  cookies: List<string>
  http10:  bool
}

// The head of the answer, interim `1xx` ones skipped. Read as strictly as the
// server reads a request: whatever a proxy might have read another way is
// refused.
fun readHead(c: net.Conn, url: string, deadline: time.Deadline): Head suspends throws FetchError {
  var interim: i64 = 0
  loop {
    val head = try readOneHead(c, url, deadline)
    if (head.status == Status.switchingProtocols) {
      throw FetchError(url, detail: "the server asked to switch protocols", kind: FetchKind.Unsupported)
    }
    if (head.status.isInformational()) {
      interim += 1
      if (interim > 8) throw protocolError(url, "too many interim responses")
      continue
    }
    return head
  }
}

fun readOneHead(c: net.Conn, url: string, deadline: time.Deadline): Head suspends throws FetchError {
  val first = try responseLine(c, url, deadline, 8192) ?: throw closedWithoutAnswer(url)
  val (version, afterVersion) = first.splitOnce(" ") else throw protocolError(url, "malformed status line")
  if (!version.startsWith("HTTP/1.")) throw protocolError(url, "the answer is not HTTP/1.x")
  val (codeText, _) = afterVersion.splitOnce(" ") ?: (afterVersion, "")
  if (codeText.len() != 3 || !isDigits(codeText)) throw protocolError(url, "malformed status code")
  val code = try codeText.toInt() ?! protocolError(url, "malformed status code")
  if (code < 100) throw protocolError(url, "malformed status code")
  val headers: MutableMap<string, string> = [:]
  val cookies: MutableList<string> = []
  var lines: i64 = 0
  var bytes: i64 = 0
  loop {
    val line = try responseLine(c, url, deadline, 8192) ?: throw protocolError(url, "the connection closed inside the response headers")
    if (line.isEmpty()) break
    lines += 1
    bytes += line.len()
    if (lines > 100 || bytes > 65536) throw protocolError(url, "the response headers are too large")
    if (line.startsWith(" ") || line.startsWith("\t")) throw protocolError(url, "folded header line")
    val (rawName, rawValue) = line.splitOnce(":") else throw protocolError(url, "malformed header line")
    if (!isToken(rawName)) throw protocolError(url, "malformed header name")
    if (rawValue.contains("\r") || rawValue.contains("\u{0}")) throw protocolError(url, "CR or NUL in a header value")
    val name = rawName.toLower()
    val value = rawValue.trim()
    if (name == "set-cookie") {
      cookies.push(value)
      continue
    }
    val earlier = headers.get(name)
    if (earlier == null) {
      headers.set(name, value)
    } else if (name == "content-length") {
      if (earlier != value) throw protocolError(url, "conflicting content-length")
    } else {
      headers.set(name, "$earlier, $value")
    }
  }
  Head(status: Status(code), headers: headers.toMap(), cookies: cookies.toList(), http10: version == "HTTP/1.0")
}

// Whether the connection may carry another request (RFC 9112 §9.3).
fun keepsAlive(head: Head): bool {
  val tokens = (head.headers.get("connection") ?: "").toLower().split(",").map(t => t.trim())
  if (tokens.contains("close")) return false
  if (head.http10) tokens.contains("keep-alive") else true
}

// Send the request on `c` and read as far as the head of the answer; the
// body is read later, by whoever holds the response.
fun exchange(c: net.Conn, url: Url, method: Method, request: List<u8>, deadline: time.Deadline, keep: Keep): ClientResponse suspends throws FetchError {
  val text = url.text()
  try sendAll(c, request, deadline, text)
  val head = try readHead(c, text, deadline)
  var reusable = keepsAlive(head)
  val coding = head.headers.get("transfer-encoding")
  val declared = head.headers.get("content-length")
  val noBody = method == Method.head || head.status.code == 204 || head.status.code == 304
  val body = if (noBody) {
    ClientBody.none(c, keep, reusable, text, deadline)
  } else if (coding != null) {
    // the framing is the transfer coding's; a length beside it is a sign of
    // a confused sender, and the connection is not trusted past this answer
    if (coding.toLower() != "chunked") throw protocolError(text, "unsupported transfer coding")
    if (declared != null) reusable = false
    ClientBody.wire(c, keep, framing: 2, length: 0, reusable: reusable, url: text, deadline: deadline)
  } else if (declared != null) {
    if (!isDigits(declared)) throw protocolError(text, "malformed content-length")
    val length = try declared.toInt() ?! protocolError(text, "malformed content-length")
    if (length == 0) {
      ClientBody.none(c, keep, reusable, text, deadline)
    } else {
      ClientBody.wire(c, keep, framing: 1, length: length, reusable: reusable, url: text, deadline: deadline)
    }
  } else {
    // no length and no chunks: the body is whatever comes before the close
    ClientBody.wire(c, keep, framing: 3, length: 0, reusable: false, url: text, deadline: deadline)
  }
  ClientResponse(status: head.status, headers: head.headers, setCookies: head.cookies, url: text, ok: head.status.isSuccess(), body)
}

// ---------------------------------------------------------------------------
// the functions

/// The client the top-level functions use: default settings, one pool for the
/// whole program.
val defaultClient = Client()

/// One request on the shared client; see `Client.fetch`.
///
/// ```veles
/// val res = try http.fetch("http://example.com/notes", method: http.Method.post, body: try http.Payload.json(note))
/// ```
public fun fetch(
  url: string,
  method: Method = Method.get,
  headers: Map<string, string> = [:],
  body: Payload? = null,
  timeout: Duration? = null,
  redirect: bool = true,
  retry: i64 = 0,
): ClientResponse suspends throws FetchError =
  try defaultClient.fetch(url, method, headers, body, timeout, redirect, retry)

/// `GET` on the shared client.
///
/// ```veles
/// val page = try http.get("http://example.com").text()
/// ```
public fun get(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
  try defaultClient.get(url, headers, timeout, redirect, retry)

/// `HEAD` on the shared client.
public fun head(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
  try defaultClient.head(url, headers, timeout, redirect, retry)

/// `POST` on the shared client.
public fun post(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true): ClientResponse suspends throws FetchError =
  try defaultClient.post(url, body, headers, timeout, redirect)

/// `PUT` on the shared client.
public fun put(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
  try defaultClient.put(url, body, headers, timeout, redirect, retry)

/// `PATCH` on the shared client.
public fun patch(url: string, body: Payload? = null, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true): ClientResponse suspends throws FetchError =
  try defaultClient.patch(url, body, headers, timeout, redirect)

/// `DELETE` on the shared client.
public fun delete(url: string, headers: Map<string, string> = [:], timeout: Duration? = null, redirect: bool = true, retry: i64 = 0): ClientResponse suspends throws FetchError =
  try defaultClient.delete(url, headers, timeout, redirect, retry)
