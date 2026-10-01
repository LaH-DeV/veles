/// An HTTP/1.1 server on top of `net`: a `Router` maps methods and paths
/// to handlers, `serve` runs the accept loop with one task per connection,
/// keep-alive and an idle timeout. A handler is an ordinary function from
/// `Request` to `Response`; it may throw — `Fail(status, text)` answers
/// with that status, any other error becomes a 500 and a log line — and
/// a panic in a handler answers 500 as well, without taking the server
/// down (D56).
///
/// ```veles
/// val app = http.Router()
/// app.get("/users/{id}", req => http.Response.json(try loadUser(req.param("id")) ?! http.notFound()))
/// app.get("/static/*", http.files("./public"))
/// with listener = try net.listen(host: "", port: 8080)
/// http.serve(listener, app.handler())
/// ```
use codec, fs, log as logs { field }, net, path, random, time

// ---------------------------------------------------------------------------
// failing a request

/// "Answer this request with `status`": thrown from a handler.
public error Fail {
  public status: Status
  public text:   string
  fun message(): string = "${this.status.code} ${this.text}"
}

public fun notFound(text: string = "not found"): Fail = Fail(status: Status.notFound, text)
public fun badRequest(text: string = "bad request"): Fail = Fail(status: Status.badRequest, text)
public fun forbidden(text: string = "forbidden"): Fail = Fail(status: Status.forbidden, text)

// ---------------------------------------------------------------------------
// what a request may cost

/// The ceilings one request may reach before the server refuses it. A
/// listener is open to strangers, so every one of these has a default
/// meant to be safe rather than generous; raise the ones your API needs
/// and leave the rest.
///
/// ```veles
/// http.serve(listener, app, limits: http.Limits(bodyBytes: 8 * 1024 * 1024))
/// ```
///
/// Byte ceilings answer the client and close: 414 for the request line,
/// 431 for the headers, 413 for the body. Time ceilings answer 408 —
/// except `idleTimeout`, which is the ordinary end of a kept-alive
/// connection and closes without a word.
public struct Limits {
  /// `GET /some/path HTTP/1.1` — the target is most of it.
  public requestLineBytes: i64 = 8192
  /// One header line, name and value together.
  public headerLineBytes: i64 = 8192
  /// How many header lines one request may carry.
  public headerCount: i64 = 100
  /// Every header line added up, so many small headers cost as much as
  /// one large one.
  public headerBytes: i64 = 65536
  /// What `req.bytes()`, `req.text()` and `req.form()` collect into memory,
  /// whatever `Content-Length` claims; more is a 413 before its first byte
  /// when the length says so. A handler that takes a larger upload streams it,
  /// with a ceiling of its own: `req.stream(max:)`, `req.multipart(max:)`.
  public bodyBytes: i64 = 1048576
  /// From the request line to the blank line that ends the headers. This
  /// is what a connection holding its headers open half-sent runs into.
  public headerTimeout: Duration = Duration.seconds(10)
  /// The longest silence while the body is read: between two reads, not
  /// the whole upload, so a large body that keeps arriving is never cut off
  /// and one that stalls is.
  public bodyTimeout: Duration = Duration.seconds(30)
  /// How long a kept-alive connection may stay silent before the next
  /// request. Reaching it is not an error: the connection closes.
  public idleTimeout: Duration = Duration.seconds(15)
  /// How many connections are served at once; `0` is no limit. At the limit
  /// the server stops accepting until one closes, so the burst waits in the
  /// operating system's backlog (or is refused there) instead of costing a
  /// task each. A kept-alive connection holds its place until `idleTimeout`.
  public connections: i64 = 10000
}

// ---------------------------------------------------------------------------
// messages

/// One request. Header names are lower-case; `params` holds the route's
/// `{name}` captures and `*` the rest matched by a trailing wildcard.
public struct Request {
  public method:  Method
  public path:    string
  public query:   Map<string, string>
  public headers: Map<string, string>
  body:           Body
  public peer:    string
  public params:  Map<string, string> = [:]
  /// The query string as it came, after the `?` and before decoding: what
  /// `queryFields()` and `query<T>()` read, repeats and order kept.
  public rawQuery: string = ""

  /// A header by name, case-insensitively.
  public fun header(name: string): string? = this.headers.get(name.toLower())

  /// A route parameter (`{id}` in the pattern); empty when the route has none.
  public fun param(name: string): string = this.params.get(name) ?: ""

  /// The whole body, read now: at most `max` bytes (`Limits.bodyBytes` when
  /// not said), and a longer one is a 413 — before its first byte when its
  /// `Content-Length` already says so. A slow sender is a 408, a body that
  /// ends short or is framed wrongly a 400. What was read is kept, so
  /// `bytes()`, `text()`, `form()` and the form readers can each be called
  /// again and see the same body; `stream` reads from the wire and does not
  /// share it, so after `bytes()` a stream is at its end.
  public fun bytes(max: i64? = null): List<u8> suspends throws Fail | IoError {
    val limit = max ?: this.body.defaultMax
    if (val kept = this.body.cached()) {
      if (kept.len() > limit) throw tooLarge()
      return kept
    }
    val all = try this.body.withLimit(limit).readAll()
    this.body.remember(all)
    all
  }

  /// The body as text; a body that is not UTF-8 is a 400. Reads like `bytes`.
  public fun text(max: i64? = null): string suspends throws Fail | IoError =
    try (try this.bytes(max)).decodeUtf8() ?! badRequest("body is not valid UTF-8")

  /// The body as it arrives, a piece at a time, for one too large to hold or
  /// worth handling early: `req.stream(max: 1024 * 1024 * 1024)` allows a gigabyte. `max`
  /// has no default on purpose — the sender decides how much comes, and a
  /// handler that never says would take whatever a stranger sends.
  ///
  /// ```veles
  /// with f = try fs.open(target, fs.FileMode.Write)
  /// val body = req.stream(max: 100 * 1024 * 1024)
  /// loop {
  ///   val chunk = try body.read()
  ///   if (chunk.isEmpty()) break
  ///   try f.write(chunk)
  /// }
  /// ```
  public fun stream(max: i64): Body = this.body.withLimit(max)

  /// The parts of a `multipart/form-data` body, read one at a time as the
  /// upload arrives; `max` is the ceiling for the whole body, as in `stream`,
  /// and `maxParts` how many parts are allowed. Another content type is a
  /// 415, a missing boundary a 400. See `Multipart`.
  public fun multipart(max: i64, maxParts: i64 = 100): Multipart throws Fail {
    val (kind, params) = parseParams(this.header(Header.contentType) ?: "")
    if (kind.toLower() != "multipart/form-data") {
      throw Fail(status: Status.unsupportedMediaType, text: "expected multipart/form-data, got '${if (kind.isEmpty()) "nothing" else kind.toLower()}'")
    }
    val boundary = params.get("boundary") ?: throw badRequest("multipart/form-data without a boundary")
    if (boundary.isEmpty() || boundary.len() > 70) throw badRequest("malformed multipart boundary")
    Multipart(source: this.body.withLimit(max), delimiter: "\r\n--$boundary".bytes(), maxParts)
  }

  /// The same request with route parameters filled in.
  fun withParams(params: Map<string, string>): Request =
    Request(method: this.method, path: this.path, query: this.query, headers: this.headers, body: this.body, peer: this.peer, params, rawQuery: this.rawQuery)

  /// The same request with a header set (names are lower-cased). This is
  /// how a middleware hands something to the handlers behind it — there
  /// are no task-local values yet.
  public fun withHeader(name: string, value: string): Request {
    val h = this.headers.toMutable()
    h.set(name.toLower(), value)
    Request(method: this.method, path: this.path, query: this.query, headers: h.toMap(), body: this.body, peer: this.peer, params: this.params, rawQuery: this.rawQuery)
  }

  /// Every cookie the browser sent, by name. A value is percent-decoded (see
  /// `Cookie`), and when a name is sent twice the first counts.
  public fun cookies(): Map<string, string> = parseCookies(this.header(Header.cookie) ?: "")

  /// One cookie by name, or null.
  public fun cookie(name: string): string? = this.cookies().get(name)

  /// The query string's fields, repeats kept (`req.query` keeps the last of
  /// each name).
  public fun queryFields(): Fields = Fields.parse(this.rawQuery)

  /// The query string read into a `T`, as `form<T>()` reads a body: a value
  /// that does not fit, or a missing required field, is a 400 naming the fields.
  ///
  /// ```veles
  /// struct Search { q: string, page: i64 = 1, tags: List<string> = [] }
  /// val s = try req.query<Search>()   // /find?q=veles&tags=a&tags=b
  /// ```
  public fun query<T: Decodable>(keys: codec.KeyStyle = codec.KeyStyle.AsWritten): T throws Fail =
    when (decodeFields<T>(this.queryFields(), keys)) {
      is Ok(v)  => v
      is Err(e) => throw badRequest("invalid query:\n" + e.message())
    }

  /// The fields of an `application/x-www-form-urlencoded` body. Another
  /// content type is a 415, a body that is not UTF-8 a 400.
  public fun formFields(): Fields suspends throws Fail | IoError {
    val media = (this.header(Header.contentType) ?: "").split(";").at(0)?.trim()?.toLower() ?: ""
    if (media != "application/x-www-form-urlencoded") {
      throw Fail(status: Status.unsupportedMediaType, text: "expected application/x-www-form-urlencoded, got '${if (media.isEmpty()) "nothing" else media}'")
    }
    Fields.parse(try this.text())
  }

  /// The first value of a form field, or null.
  public fun formValue(name: string): string? suspends throws Fail | IoError = (try this.formFields()).get(name)

  /// Every value of a form field (a checkbox group, a multiple select).
  public fun formValues(name: string): List<string> suspends throws Fail | IoError = (try this.formFields()).all(name)

  /// The form read into a `T`: numbers and booleans are parsed from the
  /// text, a `List` field takes every value of its name, an optional field
  /// is null when empty, and every problem is reported together as a 400 —
  /// one line per field, at its name. Nested structs are not forms.
  ///
  /// ```veles
  /// struct Signup { name: string, age: i64, newsletter: bool = false }
  /// val s = try req.form<Signup>()
  /// ```
  public fun form<T: Decodable>(keys: codec.KeyStyle = codec.KeyStyle.AsWritten): T suspends throws Fail | IoError =
    when (decodeFields<T>(try this.formFields(), keys)) {
      is Ok(v)  => v
      is Err(e) => throw badRequest("invalid form:\n" + e.message())
    }
}

/// One response. Build it with the statics, adjust with `withHeader`.
public struct Response {
  public status:  Status = Status.ok
  public headers: Map<string, string> = [:]
  public body:    List<u8> = []
  /// The cookies to set, one `Set-Cookie` line each (a header map holds a
  /// name once, and a response may set many).
  public cookies: List<Cookie> = []
  /// A body produced while it is sent, instead of `body`: see `Response.stream`.
  public stream: (sendable fun(BodyWriter) suspends throws IoError)? = null
  /// The length of a streamed body, when known (see `Response.stream`).
  public streamLength: i64? = null

  /// A body written as it is produced, for what is too large to hold, is not
  /// ready yet, or never ends: a big download, a live feed, server-sent
  /// events. The server sends the head, then calls `producer` with a
  /// `BodyWriter`; each `write` goes out at once, framed as a chunk
  /// (`transfer-encoding: chunked`; an HTTP/1.0 client gets the bytes
  /// followed by the end of the connection). The response is complete when
  /// `producer` returns.
  ///
  /// A failed `write` means the client went away; the producer stops with
  /// that error. A panic in the producer, or an error it lets escape, can no
  /// longer become a 500 — the head has gone — so the connection is closed
  /// without the final chunk and the client sees a truncated body, never one
  /// that looks complete. A `HEAD` request gets the head and no call.
  ///
  /// The producer runs after the handler has returned, so middleware that
  /// waits for the handler (`timeout`) does not bound it, and it must be a
  /// sendable function: capture values and handles, not a `var`.
  ///
  /// ```veles
  /// http.Response.stream(http.MediaType.eventStream, out => {
  ///   loop (i in 0..<3) {
  ///     try out.writeText("data: tick $i\n\n")
  ///     await sleep(Duration.seconds(1))
  ///   }
  /// })
  /// ```
  public static fun stream<T: AsMediaType>(contentType: T, producer: sendable fun(BodyWriter) suspends throws IoError, length: i64? = null, status: Status = Status.ok): Response =
    Response(status, headers: ["content-type": contentType.mediaType().name], stream: producer, streamLength: length)

  /// Plain text.
  public static fun text(body: string, status: Status = Status.ok): Response =
    Response(status, headers: ["content-type": "text/plain; charset=utf-8"], body: body.bytes())

  /// HTML.
  public static fun html(body: string, status: Status = Status.ok): Response =
    Response(status, headers: ["content-type": "text/html; charset=utf-8"], body: body.bytes())

  /// JSON text the caller already produced.
  public static fun json(body: string, status: Status = Status.ok): Response =
    Response(status, headers: ["content-type": "application/json"], body: body.bytes())

  /// Raw bytes with a content type.
  public static fun bytes<T: AsMediaType>(body: List<u8>, contentType: T, status: Status = Status.ok): Response =
    Response(status, headers: ["content-type": contentType.mediaType().name], body)

  /// A status and nothing else (`Status.noContent`, `Status.notFound`, ...).
  public static fun empty(status: Status): Response = Response(status)

  /// A redirect to `location`.
  public static fun redirect(location: string, status: Status = Status.found): Response =
    Response(status, headers: [Header.location: location])

  /// The same response with a header set (names are lower-cased).
  public fun withHeader(name: string, value: string): Response {
    val h = this.headers.toMutable()
    h.set(name.toLower(), value)
    Response(status: this.status, headers: h.toMap(), body: this.body, cookies: this.cookies, stream: this.stream, streamLength: this.streamLength)
  }

  /// The same response setting `cookie` as well. The value is percent-encoded;
  /// a cookie the browsers would refuse or that cannot be written safely — a
  /// bad name, path or domain, `SameSite.None` without `secure`, a `__Host-`
  /// name that breaks its rules — panics at this call.
  ///
  /// ```veles
  /// resp.withCookie(http.Cookie(name: "sid", value: id, maxAge: Duration.days(7)))
  /// ```
  @caller_location
  public fun withCookie(cookie: Cookie): Response {
    checkCookie(cookie)
    Response(status: this.status, headers: this.headers, body: this.body, cookies: this.cookies.concat([cookie]), stream: this.stream, streamLength: this.streamLength)
  }

  /// The same response telling the browser to forget a cookie: it must be
  /// named with the `path` and `domain` it was set with (`secure` for a
  /// `__Host-` or `__Secure-` name).
  @caller_location
  public fun withoutCookie(name: string, path: string = "/", domain: string? = null, secure: bool = false): Response =
    this.withCookie(Cookie(name, value: "", path, domain, maxAge: Duration.seconds(0), secure, httpOnly: false, sameSite: null))
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
        logs.error("handler failed", field("method", "${req.method}"), field("path", req.path), field("error", e.message()))
        val status = Status.internalServerError
        Response.text(body: status.reason(), status: status)
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
      logs.error("handler panicked", field("method", "${req.method}"), field("path", req.path), field("panic", p.message()))
      val status = Status.internalServerError
      Response.text(body: status.reason(), status: status)
    }
  }
}

/// Runs `handler` on a request built in memory — no socket, no port — and
/// returns what a client would receive. The target is split and decoded
/// as `serve` does it (`"/notes/1?full=yes"`), the handler runs behind the
/// same panic boundary (a panic answers 500), and a `HEAD` answer or a
/// 1xx/204/304 comes back without a body. Header names may be written in
/// any case. The peer is `"test"`.
///
/// ```veles
/// val resp = http.call(app.handler(), http.Method.post, "/notes", body: "{\"text\":\"hi\"}")
/// io.println("${resp.status} ${resp.body.decodeUtf8() ?: ""}")   // 201 Created ...
/// ```
public fun call(handler: Handler, method: Method, target: string, body: string = "", headers: Map<string, string> = [:]): Response {
  val lower: MutableMap<string, string> = [:]
  loop ((name, value) in headers.entries()) {
    lower.set(name.toLower(), value)
  }
  val (rawPath, rawQuery) = target.splitOnce("?") ?: (target, "")
  val req = Request(
    method,
    path: percentDecode(rawPath, plusIsSpace: false),
    query: parseQuery(rawQuery),
    headers: lower.toMap(),
    body: Body.hold(body.bytes(), Limits()),
    peer: "test",
    rawQuery,
  )
  val resp = collect(dispatch(handler, req))
  if (method == Method.head || hasNoBody(resp.status)) Response(status: resp.status, headers: resp.headers, cookies: resp.cookies) else resp
}

// ---------------------------------------------------------------------------
// routing

struct Route {
  method:   Method?       // null for any
  segments: List<string>  // "users", "{id}", "*"
  handler:  Handler
}

/// Maps `METHOD /pattern` to handlers. A pattern segment `{name}` captures
/// one path segment into `req.params`; a final `*` captures the rest under
/// `"*"`. `http.Router()` makes an empty one; register routes, then hand
/// `handler()` to `serve`.
public struct Router {
  private routes:     MutableList<Route> = []
  private middleware: MutableList<Middleware> = []

  public fun get<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.add(Method.get, pattern, handler(h))
  }
  public fun post<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.add(Method.post, pattern, handler(h))
  }
  public fun put<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.add(Method.put, pattern, handler(h))
  }
  public fun delete<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.add(Method.delete, pattern, handler(h))
  }
  public fun patch<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.add(Method.patch, pattern, handler(h))
  }
  public fun any<E>(pattern: string, h: sendable fun(Request): Response suspends throws E | Fail) {
    this.routes.push(Route(method: null, segments: segmentsOf(pattern), handler: handler(h)))
  }

  /// Registers an already adapted handler for `method`.
  public fun add(method: Method, pattern: string, h: Handler) {
    this.routes.push(Route(method, segments: segmentsOf(pattern), handler: h))
  }

  /// Puts `m` around everything this router answers — its 404s and 405s
  /// included, which is what an access log or a request id wants. The
  /// first `wrap` ends up outermost: it sees the request first and the
  /// response last.
  ///
  /// (`use` is the keyword that imports a module, so the verb here is
  /// `wrap`.)
  public fun wrap(m: Middleware) {
    this.middleware.push(m)
  }

  /// The routes as one handler: first match wins, 405 when only the method
  /// differs, 404 otherwise — then the middleware, outermost first.
  public fun handler(): Handler {
    val routes = this.routes.toList()
    var h: Handler = req => route(routes, req)
    // applied last to first, so the first one wrapped ends up outside
    loop (m in this.middleware.toList().reversed()) {
      h = m(h)
    }
    h
  }
}

// ---------------------------------------------------------------------------
// middleware

/// A wrapper around a handler: it takes the handler that comes after it
/// and returns one that does something before, after, or instead.
///
/// ```veles
/// app.wrap(next => req => next(req).withHeader("x-served-by", "veles"))
/// ```
public type Middleware = sendable fun(Handler): Handler

/// One log line per request (std/log: text on a terminal, JSON elsewhere):
/// peer, method, path, status and how long it took — a `Duration`, so the
/// unit is in the line — plus the request id when `requestId()` wraps this
/// one from the outside. `serve` logs the same line itself, so pass
/// `log: false` when you wrap this one.
public fun logging(): Middleware = next => req => {
  val sw = time.Stopwatch.start()
  val resp = next(req)
  logs.info("request", field("peer", "${req.peer}"), field("method", "${req.method}"), field("path", req.path), field("status", resp.status.code), field("took", "${sw.elapsed()}"))
  resp
}

/// Gives every request an id — the client's `X-Request-Id` if it sent one,
/// a fresh one otherwise — and puts it on the request for the handlers
/// behind it and on the response for the client. It is also bound as the
/// log field `id` (`log.withFields`) for the whole request, so every line a
/// handler logs — and `logging()`'s, wrapped inside — carries it.
public fun requestId(): Middleware = next => req => {
  val id = req.header(Header.requestId) ?: newRequestId()
  logs.withFields([field("id", id)], () => next(req.withHeader(Header.requestId, id))).withHeader(Header.requestId, id)
}

// 16 hex digits: enough to tell a day's requests apart in a log, and not
// a claim to be unguessable — this is for tracing, not for security.
fun newRequestId(): string {
  val digits = "0123456789abcdef"
  val out = StringBuilder()
  var bits = random.nextU64()
  loop (_ in 0..<16) {
    val d = (bits % 16).wrapI64()
    out.append(digits.substring(d, d + 1) ?: "0")
    bits = bits / 16
  }
  out.toString()
}

/// Answers 503 when the handler behind it takes longer than `ms`. The
/// handler runs in a task of its own and is cancelled on the way out, so
/// whatever it held in a `with` is closed.
public fun timeout(limit: Duration): Middleware = next => req => {
  when (withTimeout(limit, () => next(req))) {
    is Ok(resp) => resp
    is Err      => {
      val status = Status.serviceUnavailable
      Response.text(status.reason(), status: status)
    }
  }
}

// First match wins. When the path matches but the method does not, the
// answer names the methods that would have worked (RFC 9110 §15.5.6: a
// 405 must carry `Allow`); `HEAD` falls back to the `GET` route (the
// writer drops the body, §9.3.2) and `OPTIONS` without a route of its own
// answers 204 with the same `Allow`.
fun route(routes: List<Route>, req: Request): Response {
  val segments = segmentsOf(req.path)
  val allowed: MutableList<Method> = []
  loop (r in routes) {
    val params = matchRoute(r.segments, segments) ?: continue
    val m = r.method ?: return r.handler(req.withParams(params))
    if (m == req.method) return r.handler(req.withParams(params))
    if (!allowed.contains(m)) allowed.push(m)
  }
  if (req.method == Method.head && allowed.contains(Method.get)) {
    loop (r in routes) {
      if (r.method != Method.get) continue
      val params = matchRoute(r.segments, segments) ?: continue
      return r.handler(req.withParams(params))
    }
  }
  if (allowed.isEmpty()) {
    val status = Status.notFound
    return Response.text(status.reason(), status: status)
  }
  if (allowed.contains(Method.get) && !allowed.contains(Method.head)) allowed.push(Method.head)
  if (!allowed.contains(Method.options)) allowed.push(Method.options)
  val allow = allowed.map(m => m.name).join(", ")
  if (req.method == Method.options) return Response.empty(Status.noContent).withHeader(Header.allow, allow)
  val status = Status.methodNotAllowed
  Response.text(status.reason(), status: status).withHeader(Header.allow, allow)
}

// the non-empty segments of a path or pattern: "/users/42/" → [users, 42]
fun segmentsOf(p: string): List<string> = p.split("/").filter(s => !s.isEmpty())

// the captures when the pattern matches the path, or null
fun matchRoute(pattern: List<string>, segments: List<string>): Map<string, string>? {
  val params: MutableMap<string, string> = [:]
  loop (i in 0..<pattern.len()) {
    val p = pattern.at(i)
    if (p == "*") {
      params.set("*", segments.drop(i).join("/"))
      return params.toMap()
    }
    if (i >= segments.len()) return null
    val s = segments.at(i)
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
// the server

/// Accepts connections, serving each in a task of its own. A connection
/// is kept open for further requests until the client closes it, asks for
/// `Connection: close`, or stays silent for `limits.idleTimeout`. What one
/// request may cost is `limits` (see `Limits`); every request is logged to
/// standard error unless `log` is false.
///
/// Without `stop` it serves until its task is cancelled. With it, it
/// serves until `stop` returns and then shuts down gracefully (D68): it
/// stops accepting, closes kept-alive connections that are waiting for a
/// next request, lets requests in flight finish — their responses say
/// `connection: close` — for up to `grace`, cancels whatever is still
/// running after that (its `with` blocks run), and returns.
///
/// ```veles
/// http.serve(listener, app.handler(), stop: () => os.shutdownSignal())
/// ```
public fun serve(
  listener: net.Listener,
  handler: Handler,
  limits: Limits = Limits(),
  log: bool = true,
  stop: (sendable fun() suspends)? = null,
  grace: Duration = Duration.seconds(10),
) {
  val drain = Drain(stopping: Atomic(value: false), wake: Channel<bool>(capacity: 1))
  if (stop == null) return acceptAndServe(listener, handler, limits, log, drain)
  scope {
    val serving = async acceptAndServe(listener, handler, limits, log, drain)
    stop()
    if (log) logs.info("stopping", field("grace", "$grace"))
    drain.begin()
    race {
      val _ = await serving => { }
      sleep(grace)          => serving.cancel()
    }
  }
}

// How a stopping server reaches its connections: `stopping` is read after
// each response, and closing `wake` ends every wait for a next request and
// the wait for the next connection.
struct Drain {
  stopping: Atomic<bool>
  wake:     Channel<bool>

  fun begin() {
    this.stopping.store(true)
    this.wake.close()
  }
}

fun acceptAndServe(listener: net.Listener, handler: Handler, limits: Limits, log: bool, drain: Drain) {
  val accepted = Channel<net.Conn>(capacity: 16)
  // one item per connection being served: `accept` waits while the channel
  // is full, so the limit costs nothing for what it holds back
  val permits: Channel<bool>? = if (limits.connections > 0) Channel<bool>(capacity: limits.connections) else null
  scope {
    // accepts until the loop below stops taking connections
    with acceptor = async acceptLoop(listener, accepted, permits)
    loop {
      val conn = race {
        val c = accepted.recv()   => c
        val _ = drain.wake.recv() => null
      } ?: break
      async serveConnection(conn, handler, limits, log, drain, permits)
    }
  }
  // accepted but never served: close them instead of leaving the
  // descriptors to the process's exit
  loop {
    val c = accepted.tryRecv() ?: break
    c.close()
  }
}

fun acceptLoop(listener: net.Listener, out: Channel<net.Conn>, permits: Channel<bool>?) {
  loop {
    // a place first, then the connection: at the limit this is where the
    // server waits, and cancelling it (a stop) ends the wait
    if (permits != null) permits.send(true)
    when (listener.accept()) {
      is Ok(conn) => out.send(conn)
      is Err(e)   => {
        if (permits != null) release(permits)
        // out of descriptors, a reset before accept: report and go on
        logs.warn("accept failed", field("error", e.message()))
        await sleep(Duration.millis(100))
      }
    }
  }
}

// gives one place back
fun release(permits: Channel<bool>) {
  val _ = permits.tryRecv()
}

fun serveConnection(conn: net.Conn, handler: Handler, limits: Limits, log: bool, drain: Drain, permits: Channel<bool>?) {
  connection(conn, handler, limits, log, drain)
  if (permits != null) release(permits)
}

fun connection(conn: net.Conn, handler: Handler, limits: Limits, log: bool, drain: Drain) {
  with c = conn
  loop {
    val (req, http10) = when (readRequest(c, limits, drain)) {
      is Ok(r)  => r ?: break
      is Err(e) => {
        if (e is Fail) {
          val _ = writeResponse(c, Response.text(e.text, status: e.status), close: true)
          if (log) logs.warn("bad request", field("peer", "${c.peer()}"), field("status", e.status.code), field("error", e.text))
        }
        break
      }
    }
    val sw = time.Stopwatch.start()
    val resp = dispatch(handler, req)
    // read after the handler: a stop that began while it ran still
    // closes this connection
    val close = !wantsKeepAlive(req, http10) || drain.stopping.load() || resp.headers.get(Header.connection)?.toLower() == "close" || !req.body.reusable(drainCap) || (resp.stream != null && http10 && resp.streamLength == null)
    val sent = writeResponse(c, resp, close, headOnly: req.method == Method.head, http10: http10)
    if (log) logs.info("request", field("peer", "${req.peer}"), field("method", "${req.method}"), field("path", req.path), field("status", resp.status.code), field("took", "${sw.elapsed()}"))
    if (sent is Err || close) break
    // what the handler left unread is read and dropped, after the answer
    // has gone out, so that the next request starts where it should
    if (!req.body.drain(drainCap)) break
  }
}

// HTTP/1.1 keeps a connection unless told otherwise; HTTP/1.0 closes it
// unless told otherwise (RFC 9112 §9.3), and a 1.0 client that reads to
// the end of the connection would otherwise wait out the idle timeout.
fun wantsKeepAlive(req: Request, http10: bool): bool = when (req.header("connection")?.toLower()) {
  "close"      => false
  "keep-alive" => true
  else         => !http10
}

// ---------------------------------------------------------------------------
// wire format

// One line of the head, refusing a ceiling breach with the status the
// client should see rather than letting the read error escape.
fun headLine(c: net.Conn, max: i64, limit: Duration, tooLong: Fail): string? suspends throws Fail | IoError | Timeout {
  when (withTimeout(limit, () => try c.readLine(max: max))) {
    is Ok(line) => line
    is Err(e)   => when (e) {
      is net.TooLong => throw tooLong
      is Timeout     => throw e
      is IoError     => try rethrow(e)
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
fun requestLine(c: net.Conn, limits: Limits, drain: Drain): string? suspends throws Fail | IoError | Timeout {
  if (drain.stopping.load()) return null
  scope {
    val line = async readLineOf(c, limits.requestLineBytes)
    race {
      val r = await line        => return try r
      sleep(limits.idleTimeout) => throw Timeout(limit: limits.idleTimeout)
      val _ = drain.wake.recv() => return null
    }
  }
}

// the line, with a line past `max` already the 414 it answers
fun readLineOf(c: net.Conn, max: i64): string? suspends throws Fail | IoError {
  when (c.readLine(max: max)) {
    is Ok(line) => line
    is Err(e)   => when (e) {
      is net.TooLong => throw Fail(status: Status.uriTooLong, text: "URI too long")
      is IoError     => try rethrow(e)
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
fun readRequest(c: net.Conn, limits: Limits, drain: Drain): (Request, bool)? throws Fail | IoError | Timeout {
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
    headers.set(name, value)
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
  val req = Request(method: Method(name: method), path: percentDecode(rawPath, plusIsSpace: false), query: parseQuery(rawQuery), headers: headers.toMap(), body, peer: c.peer(), rawQuery)
  (req, http10)
}

// The characters a method or a header name may hold (RFC 9110 §5.6.2).
fun isToken(s: string): bool =
  !s.isEmpty() && s.bytes().all(b => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || tokenMarks.contains(b))

val tokenMarks: List<u8> = "!#$%&'*+-.^_`|~".bytes()

fun isDigits(s: string): bool = !s.isEmpty() && s.bytes().all(b => b >= '0' && b <= '9')

fun parseQuery(text: string): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  if (text.isEmpty()) return out.toMap()
  loop (pair in text.split("&")) {
    if (pair.isEmpty()) continue
    // a key without `=` has the empty value
    val (k, v) = pair.splitOnce("=") ?: (pair, "")
    out.set(percentDecode(k, plusIsSpace: true), percentDecode(v, plusIsSpace: true))
  }
  out.toMap()
}

fun hexValue(b: u8): i64 {
  if (b >= '0' && b <= '9') return (b - '0').toI64()
  if (b >= 'a' && b <= 'f') return (b - 'a').toI64() + 10
  if (b >= 'A' && b <= 'F') return (b - 'A').toI64() + 10
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
    val b = bytes.at(i)
    if (b == '%' && i + 2 < bytes.len()) {
      val hi = hexValue(bytes.at(i + 1) ?: panic("percentDecode: i + 2 < len was checked"))
      val lo = hexValue(bytes.at(i + 2) ?: panic("percentDecode: i + 2 < len was checked"))
      if (hi >= 0 && lo >= 0) {
        out.push((hi * 16 + lo).wrapU8())
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
fun writeResponse(c: net.Conn, resp: Response, close: bool, headOnly: bool = false, http10: bool = false) suspends throws IoError {
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
    if (!complete) throw IoError(path: c.peer(), code: 0, detail: "the response body could not be completed", kind: IoKind.Other)
    if (chunked) try c.write("0\r\n\r\n".bytes())
    return
  }
  try c.write(if (headOnly || bodyless) bytes else bytes.concat(resp.body))
}

val framing = ["content-length", "transfer-encoding", "connection"]

// 1xx, 204 and 304 never carry a body, and a 1xx or 204 may not even say
// `content-length` (RFC 9110 §6.4.1, §8.6); a 304's length would be the
// 200's, which a handler returning `empty(304)` does not know.
fun hasNoBody(status: Status): bool = status.isInformational() || status == Status.noContent || status == Status.notModified

/// A time in the format HTTP dates use: `Sun, 06 Nov 1994 08:49:37 GMT`.
///
/// The format itself lives in `std/time`, with the parser for the two
/// obsolete forms a recipient must also accept (`time.parseHttp`) —
/// `Last-Modified` and `If-Modified-Since` are two ends of one conversation
/// and belong in one place.
public fun httpDate(t: time.Timestamp): string = time.formatHttp(t)
