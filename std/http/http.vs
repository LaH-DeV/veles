/// An HTTP/1.1 server on top of `net`: a `Router` maps methods and paths
/// to handlers, `serve` runs the accept loop with one task per connection,
/// keep-alive and an idle timeout. A handler is an ordinary function from
/// `Request` to `Response`; it may throw — `Fail(status, text)` answers
/// with that status, any other error becomes a 500 and a log line — and
/// a panic in a handler answers 500 as well, without taking the server
/// down (D56).
///
/// ```veles
/// val app = http.router()
/// app.get("/users/{id}", req => http.Response.json(try loadUser(req.param("id")) ?! http.notFound()))
/// app.get("/static/*", http.files("./public"))
/// with (listener = try net.listen(host: "", port: 8080)) {
///   http.serve(listener, app.handler())
/// }
/// ```
use fs, io, net, path, time

// ---------------------------------------------------------------------------
// failing a request

/// "Answer this request with `status`": thrown from a handler.
public error Fail {
  public status: i64
  public text:   string
  fun message(): string = "${self.status} ${self.text}"
}

public fun notFound(text: string = "not found"): Fail = Fail(status: 404, text)
public fun badRequest(text: string = "bad request"): Fail = Fail(status: 400, text)
public fun forbidden(text: string = "forbidden"): Fail = Fail(status: 403, text)

// ---------------------------------------------------------------------------
// messages

/// One request. Header names are lower-case; `params` holds the route's
/// `{name}` captures and `*` the rest matched by a trailing wildcard.
public struct Request {
  public method:  string
  public path:    string
  public query:   Map<string, string>
  public headers: Map<string, string>
  public body:    List<u8>
  public peer:    string
  public params:  Map<string, string> = [:]

  /// A header by name, case-insensitively.
  public fun header(name: string): string? = self.headers.get(name.toLower())

  /// A route parameter (`{id}` in the pattern); empty when the route has none.
  public fun param(name: string): string = self.params.get(name) ?: ""

  /// The body as text; a body that is not UTF-8 is a 400.
  public fun text(): string throws Fail = try self.body.decodeUtf8() ?! badRequest("body is not valid UTF-8")

  /// The same request with route parameters filled in.
  fun withParams(params: Map<string, string>): Request =
    Request(method: self.method, path: self.path, query: self.query, headers: self.headers, body: self.body, peer: self.peer, params)
}

/// One response. Build it with the statics, adjust with `withHeader`.
public struct Response {
  public status:  i64 = 200
  public headers: Map<string, string> = [:]
  public body:    List<u8> = []

  /// Plain text.
  public static fun text(body: string, status: i64 = 200): Response =
    Response(status, headers: ["content-type": "text/plain; charset=utf-8"], body: body.bytes())

  /// HTML.
  public static fun html(body: string, status: i64 = 200): Response =
    Response(status, headers: ["content-type": "text/html; charset=utf-8"], body: body.bytes())

  /// JSON text the caller already produced.
  public static fun json(body: string, status: i64 = 200): Response =
    Response(status, headers: ["content-type": "application/json"], body: body.bytes())

  /// Raw bytes with a content type.
  public static fun bytes(body: List<u8>, contentType: string, status: i64 = 200): Response =
    Response(status, headers: ["content-type": contentType], body)

  /// A status and nothing else (`204`, `404`, ...).
  public static fun empty(status: i64): Response = Response(status)

  /// A redirect to `location`.
  public static fun redirect(location: string, status: i64 = 302): Response =
    Response(status, headers: ["location": location])

  /// The same response with a header set (names are lower-cased).
  public fun withHeader(name: string, value: string): Response {
    val h = self.headers.toMutable()
    h.set(name.toLower(), value)
    Response(status: self.status, headers: h.toMap(), body: self.body)
  }
}

// ---------------------------------------------------------------------------
// handlers

/// What the server calls for a request. It cannot throw: `handler` turns
/// a throwing function into one, deciding the status for each error.
public type Handler = sendable fun(Request): Response suspends

/// Adapts a handler that may throw: a `Fail` answers with its status, any
/// other error answers 500 and is logged. This is what the router applies
/// to every handler it is given, so `try` is free inside a handler and an
/// error nobody mapped is never silent.
public fun handler<E>(h: sendable fun(Request): Response suspends throws E | Fail): Handler =
  req => when (h(req)) {
    is Ok(resp) => resp
    is Err(e)   => when (e) {
      is Fail => Response.text(e.text, status: e.status)
      else    => {
        io.eprintln("http: ${req.method} ${req.path}: ${e.message()}")
        Response.text("internal server error", status: 500)
      }
    }
  }

/// The named function a task needs: calls the handler.
fun invoke(h: Handler, req: Request): Response = h(req)

/// Runs the handler in a task of its own, so a panic inside it answers 500
/// (and is logged) instead of failing the connection's task (D52).
fun dispatch(h: Handler, req: Request): Response {
  val outcome = (gather {
    async invoke(h, req)
  }).0
  when (outcome) {
    is Ok(resp) => resp
    is Err(p)   => {
      io.eprintln("http: ${req.method} ${req.path}: panic: ${p.message()}")
      Response.text("internal server error", status: 500)
    }
  }
}

// ---------------------------------------------------------------------------
// routing

struct Route {
  method:   string        // "*" for any
  segments: List<string>  // "users", "{id}", "*"
  handler:  Handler
}

/// Maps `METHOD /pattern` to handlers. A pattern segment `{name}` captures
/// one path segment into `req.params`; a final `*` captures the rest under
/// `"*"`. Register routes, then hand `handler()` to `serve`.
public struct Router {
  routes: MutableList<Route> = []

  public fun get<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    self.add("GET", pattern, handler(h))
  }
  public fun post<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    self.add("POST", pattern, handler(h))
  }
  public fun put<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    self.add("PUT", pattern, handler(h))
  }
  public fun delete<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    self.add("DELETE", pattern, handler(h))
  }
  public fun any<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    self.add("*", pattern, handler(h))
  }

  /// Registers an already adapted handler for `method` (`"*"` for any).
  public fun add(method: string, pattern: string, h: Handler) {
    self.routes.push(Route(method, segments: segmentsOf(pattern), handler: h))
  }

  /// The routes as one handler: first match wins, 405 when only the method
  /// differs, 404 otherwise.
  public fun handler(): Handler {
    val routes = self.routes.toList()
    req => route(routes, req)
  }
}

/// A new, empty router.
public fun router(): Router = Router()

fun route(routes: List<Route>, req: Request): Response {
  val segments = segmentsOf(req.path)
  var methodMismatch = false
  loop (r in routes) {
    val params = matchRoute(r.segments, segments) ?: continue
    if (r.method != "*" && r.method != req.method) {
      methodMismatch = true
      continue
    }
    return r.handler(req.withParams(params))
  }
  if (methodMismatch) Response.text("method not allowed", status: 405) else Response.text("not found", status: 404)
}

// the non-empty segments of a path or pattern: "/users/42/" → [users, 42]
fun segmentsOf(p: string): List<string> = p.split("/").filter(s => !s.isEmpty())

// the captures when the pattern matches the path, or null
fun matchRoute(pattern: List<string>, segments: List<string>): Map<string, string>? {
  val params: MutableMap<string, string> = [:]
  loop (i in 0..<pattern.len()) {
    val p = pattern.atOrPanic(i)
    if (p == "*") {
      params.set("*", segments.drop(i).join("/"))
      return params.toMap()
    }
    if (i >= segments.len()) return null
    val s = segments.atOrPanic(i)
    if (p.startsWith("{") && p.endsWith("}")) {
      params.set(p.substring(1, p.len() - 1) ?: p, s)
    } else if (p != s) {
      return null
    }
  }
  if (segments.len() != pattern.len()) return null
  params.toMap()
}

// ---------------------------------------------------------------------------
// static files

/// A handler serving files under `dir` for a route ending in `*`:
/// `app.get("/static/*", http.files("./public"))`. A directory serves its
/// `index.html`; `..` is refused; the content type follows the extension.
/// It throws like a handler you would write (a missing file is a 404, a
/// read error a 500), so the router adapts it; `http.handler(http.files(d))`
/// is the form `serve` takes directly.
public fun files(dir: string): sendable fun(Request): Response suspends throws Fail | IoError = req => {
  val rel = req.param("*")
  loop (seg in rel.split("/")) {
    if (seg == "..") throw forbidden()
  }
  var p = if (rel.isEmpty()) dir else path.join(dir, rel)
  if (fs.isDir(p)) p = path.join(p, "index.html")
  if (!fs.isFile(p)) throw notFound("no such file: /$rel")
  Response.bytes(try fs.readBytes(p), contentType: contentTypeOf(p))
}

/// The media type for a file name, by extension; `application/octet-stream`
/// when unknown.
public fun contentTypeOf(name: string): string = when (path.ext(name).toLower()) {
  ".html", ".htm" => "text/html; charset=utf-8"
  ".css"          => "text/css; charset=utf-8"
  ".js", ".mjs"   => "text/javascript; charset=utf-8"
  ".json"         => "application/json"
  ".txt", ".md"   => "text/plain; charset=utf-8"
  ".svg"          => "image/svg+xml"
  ".png"          => "image/png"
  ".jpg", ".jpeg" => "image/jpeg"
  ".gif"          => "image/gif"
  ".ico"          => "image/x-icon"
  ".wasm"         => "application/wasm"
  ".pdf"          => "application/pdf"
  else            => "application/octet-stream"
}

// ---------------------------------------------------------------------------
// the server

/// Accepts connections forever, serving each in a task of its own, and
/// returns only when its task is cancelled. A connection is kept open for
/// further requests until the client closes it, asks for `Connection:
/// close`, or stays silent for `idleTimeout` milliseconds. Every request
/// is logged to standard error unless `log` is false.
public fun serve(listener: net.Listener, handler: Handler, idleTimeout: i64 = 15000, log: bool = true) {
  scope {
    loop {
      when (listener.accept()) {
        is Ok(conn) => {
          async connection(conn, handler, idleTimeout, log)
        }
        is Err(e)   => {
          // out of descriptors, a reset before accept: report and go on
          io.eprintln("http: accept: ${e.message()}")
          await sleep(100)
        }
      }
    }
  }
}

fun connection(conn: net.Conn, handler: Handler, idleTimeout: i64, log: bool) {
  with (c = conn) {
    loop {
      val req = when (readRequest(c, idleTimeout)) {
        is Ok(r)  => r ?: break
        is Err(e) => {
          when (e) {
            is Fail => {
              val _ = writeResponse(c, Response.text(e.text, status: e.status), close: true)
              if (log) io.eprintln("${c.peer()} - ${e.status} ${e.text}")
            }
            else    => { }  // the peer went away or stayed silent
          }
          break
        }
      }
      val started = time.monotonic()
      val keepAlive = wantsKeepAlive(req)
      val resp = dispatch(handler, req)
      val sent = writeResponse(c, resp, close: !keepAlive)
      if (log) io.eprintln("${req.peer} ${req.method} ${req.path} ${resp.status} ${time.monotonic() - started}ms")
      if (sent is Err || !keepAlive) break
    }
  }
}

fun wantsKeepAlive(req: Request): bool = when (req.header("connection")?.toLower()) {
  "close"      => false
  "keep-alive" => true
  else         => true  // HTTP/1.1 default
}

// ---------------------------------------------------------------------------
// wire format

// One request from the connection, or null when the peer closed between
// requests; a malformed request is a Fail (400/411/501/505).
fun readRequest(c: net.Conn, idleTimeout: i64): Request? throws Fail | IoError | Timeout {
  val first = try withTimeout(idleTimeout, () => try c.readLine()) ?: return null
  val parts = first.split(" ")
  if (parts.len() != 3) throw badRequest("malformed request line")
  val method = parts.atOrPanic(0)
  val target = parts.atOrPanic(1)
  if (!parts.atOrPanic(2).startsWith("HTTP/1.")) throw Fail(status: 505, text: "HTTP version not supported")
  val headers: MutableMap<string, string> = [:]
  loop {
    val line = try withTimeout(idleTimeout, () => try c.readLine()) ?: throw badRequest("connection closed inside the headers")
    if (line.isEmpty()) break
    val colon = line.indexOf(":")
    if (colon <= 0) throw badRequest("malformed header line")
    val name = (line.substring(0, colon) ?: "").trim().toLower()
    val value = (line.substring(colon + 1, line.len()) ?: "").trim()
    headers.set(name, value)
  }
  if (headers.get("transfer-encoding") != null) throw Fail(status: 501, text: "chunked requests are not supported")
  var body: List<u8> = []
  val declared = headers.get("content-length")
  if (declared != null) {
    val length = try declared.toInt() ?! badRequest("malformed content-length")
    if (length < 0) throw badRequest("malformed content-length")
    if (length > 0) {
      body = try withTimeout(idleTimeout, () => try c.readExact(length))
      if (body.len() < length) throw badRequest("body shorter than content-length")
    }
  }
  val q = target.indexOf("?")
  val rawPath = if (q < 0) target else target.substring(0, q) ?: target
  val rawQuery = if (q < 0) "" else target.substring(q + 1, target.len()) ?: ""
  Request(method, path: percentDecode(rawPath, plusIsSpace: false), query: parseQuery(rawQuery), headers: headers.toMap(), body, peer: c.peer())
}

fun parseQuery(text: string): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  if (text.isEmpty()) return out.toMap()
  loop (pair in text.split("&")) {
    if (pair.isEmpty()) continue
    val eq = pair.indexOf("=")
    if (eq < 0) {
      out.set(percentDecode(pair, plusIsSpace: true), "")
    } else {
      val k = pair.substring(0, eq) ?: pair
      val v = pair.substring(eq + 1, pair.len()) ?: ""
      out.set(percentDecode(k, plusIsSpace: true), percentDecode(v, plusIsSpace: true))
    }
  }
  out.toMap()
}

fun hexValue(b: u8): i64 {
  if (b >= '0' && b <= '9') return (b - '0') as i64
  if (b >= 'a' && b <= 'f') return (b - 'a') as i64 + 10
  if (b >= 'A' && b <= 'F') return (b - 'A') as i64 + 10
  -1
}

/// Decodes `%XX` escapes (and `+` as a space in query strings); text that
/// does not decode to UTF-8 is returned as it came.
public fun percentDecode(s: string, plusIsSpace: bool): string {
  if (!s.contains("%") && !(plusIsSpace && s.contains("+"))) return s
  val bytes = s.bytes()
  val out: MutableList<u8> = []
  var i: i64 = 0
  loop (i < bytes.len()) {
    val b = bytes.atOrPanic(i)
    if (b == '%' && i + 2 < bytes.len()) {
      val hi = hexValue(bytes.atOrPanic(i + 1))
      val lo = hexValue(bytes.atOrPanic(i + 2))
      if (hi >= 0 && lo >= 0) {
        out.push((hi * 16 + lo) as u8)
        i += 3
        continue
      }
    }
    if (plusIsSpace && b == '+') out.push(' ') else out.push(b)
    i += 1
  }
  out.toList().decodeUtf8() ?: s
}

fun writeResponse(c: net.Conn, resp: Response, close: bool) throws IoError {
  val head = stringBuilder()
  head.append("HTTP/1.1 ${resp.status} ${reasonOf(resp.status)}\r\n")
  var hasType = false
  loop ((name, value) in resp.headers.entries()) {
    if (name == "content-type") hasType = true
    head.append("$name: $value\r\n")
  }
  if (!hasType && !resp.body.isEmpty()) head.append("content-type: application/octet-stream\r\n")
  head.append("content-length: ${resp.body.len()}\r\n")
  head.append("date: ${httpDate(time.now())}\r\n")
  head.append(if (close) "connection: close\r\n" else "connection: keep-alive\r\n")
  head.append("\r\n")
  try c.write(head.toString().bytes().concat(resp.body))
}

/// The standard reason phrase for a status code (empty when unknown).
public fun reasonOf(status: i64): string = when (status) {
  200  => "OK"
  201  => "Created"
  202  => "Accepted"
  204  => "No Content"
  301  => "Moved Permanently"
  302  => "Found"
  303  => "See Other"
  304  => "Not Modified"
  307  => "Temporary Redirect"
  308  => "Permanent Redirect"
  400  => "Bad Request"
  401  => "Unauthorized"
  403  => "Forbidden"
  404  => "Not Found"
  405  => "Method Not Allowed"
  408  => "Request Timeout"
  409  => "Conflict"
  411  => "Length Required"
  413  => "Payload Too Large"
  415  => "Unsupported Media Type"
  422  => "Unprocessable Entity"
  429  => "Too Many Requests"
  500  => "Internal Server Error"
  501  => "Not Implemented"
  503  => "Service Unavailable"
  505  => "HTTP Version Not Supported"
  else => ""
}

val dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
val monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

/// A time in the format HTTP dates use: `Sun, 06 Nov 1994 08:49:37 GMT`.
public fun httpDate(ms: i64): string {
  val t = time.utc(ms)
  val day = dayNames.atOrDefault(t.weekday, "Sun")
  val month = monthNames.atOrDefault(t.month - 1, "Jan")
  "$day, ${pad2(t.day)} $month ${t.year} ${pad2(t.hour)}:${pad2(t.minute)}:${pad2(t.second)} GMT"
}

fun pad2(n: i64): string = if (n < 10) "0$n" else "$n"
