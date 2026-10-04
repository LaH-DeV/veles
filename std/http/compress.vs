// Response compression (D124): `http.compress()` gzips what a client says it
// can read, `http.files` serves a `.gz` that lies beside a file, and
// `http.decompressRequests()` opens a request body a client compressed.
use compress as gz

/// Compresses responses with gzip when the client's `Accept-Encoding` allows
/// it. Only a response worth it is touched: its type is text (`text/*`,
/// JSON, JavaScript, XML, SVG — not an event stream), its body is at least
/// `minBytes`, it is a plain 200-class answer without a `Content-Range`, it
/// sets no `Content-Encoding` and no `Cache-Control: no-transform`. The
/// response gains `Content-Encoding: gzip` and `Vary: Accept-Encoding`, and a
/// strong `ETag` becomes weak (the bytes differ from the uncompressed ones).
///
/// A streamed body (`Response.stream`) is compressed as it streams, its
/// length no longer known, so it goes out chunked.
///
/// ```veles
/// app.wrap(http.compress())                       // 1 KiB and up
/// app.wrap(http.compress(minBytes: 256, level: 4)) // smaller bodies, faster
/// ```
///
/// `level` is 0–9 as in `compress.gzip` (6 by default); another value panics
/// at this call, as does a negative `minBytes`.
@caller_location
public fun compress(minBytes: i64 = 1024, level: i64 = 6): Middleware {
  if (minBytes < 0) panic("http.compress: minBytes is $minBytes; it cannot be negative")
  if (level < 0 || level > 9) panic("http.compress: level is $level; it is 0 (stored) to 9 (smallest)")
  next => req => {
    val resp = next(req)
    if (acceptsGzip(req.header(Header.acceptEncoding)) && worthCompressing(resp, minBytes)) gzipped(resp, level) else resp
  }
}

/// Whether an `Accept-Encoding` value lets the server send gzip: `gzip` or
/// `*` with a quality above zero, and not `gzip;q=0`. An absent header means
/// only the identity encoding.
fun acceptsGzip(header: string?): bool {
  val value = header ?: return false
  var gzipQ = -1.0
  var starQ = -1.0
  loop (part in value.split(",")) {
    val (name, params) = part.splitOnce(";") ?: (part, "")
    var q = 1.0
    loop (param in params.split(";")) {
      val (key, number) = param.splitOnce("=") ?: ("", "")
      if (key.trim().toLower() == "q") q = number.trim().toF64() ?: 0.0
    }
    val coding = name.trim().toLower()
    if (coding == "gzip" || coding == "x-gzip") gzipQ = q
    if (coding == "*") starQ = q
  }
  if (gzipQ >= 0.0) gzipQ > 0.0 else starQ > 0.0
}

// the media type's essence: `text/html; charset=utf-8` is `text/html`
fun essenceOf(contentType: string): string = (contentType.split(";").at(0) ?: "").trim().toLower()

fun isTextual(contentType: string): bool {
  val essence = essenceOf(contentType)
  if (essence == "text/event-stream") return false  // a live feed is not buffered into blocks
  essence.startsWith("text/") ||
    essence == "application/json" || essence == "application/javascript" || essence == "application/xml" ||
    essence == "image/svg+xml" || essence.endsWith("+json") || essence.endsWith("+xml")
}

fun worthCompressing(resp: Response, minBytes: i64): bool {
  if (resp.status.code < 200 || resp.status.code == 204 || resp.status.code == 206 || resp.status.code == 304) return false
  if (resp.headers.get(Header.contentEncoding) != null || resp.headers.get("content-range") != null) return false
  if ((resp.headers.get(Header.cacheControl) ?: "").toLower().contains("no-transform")) return false
  if (!isTextual(resp.headers.get(Header.contentType) ?: "")) return false
  if (resp.stream != null) return true
  resp.body.len() >= minBytes
}

// `Vary` with Accept-Encoding in it, keeping what the handler said
fun varyAcceptEncoding(headers: Map<string, string>): string {
  val existing = headers.get("vary") ?: return "accept-encoding"
  val has = existing.split(",").any(v => v.trim().toLower() == "accept-encoding" || v.trim() == "*")
  if (has) existing else existing + ", accept-encoding"
}

fun gzipped(resp: Response, level: i64): Response {
  val headers = resp.headers.toMutable()
  headers.set(Header.contentEncoding, "gzip")
  headers.set("vary", varyAcceptEncoding(resp.headers))
  if (val tag = resp.headers.get(Header.etag)) {
    if (!tag.startsWith("W/")) headers.set(Header.etag, "W/" + tag)
  }
  headers.remove(Header.contentLength)
  if (val inner = resp.stream) {
    val producer: sendable fun(BodyWriter) suspends throws IoError = out => {
      val packed = out.gzipped(level)
      try inner(packed)
      try packed.finishGzip()
    }
    return Response(status: resp.status, headers: headers.toMap(), cookies: resp.cookies, stream: producer)
  }
  Response(status: resp.status, headers: headers.toMap(), body: gz.gzip(resp.body, level), cookies: resp.cookies)
}

/// Opens request bodies sent with `Content-Encoding: gzip`, which a server
/// otherwise passes to the handler as they are. The decompressed body is
/// what `req.bytes()`, `req.text()` and the form readers see; its size is
/// bounded by `max` (a larger one is a 413 — a few kilobytes can claim
/// gigabytes), the compressed body by the usual `Limits.bodyBytes`. A body
/// that is not valid gzip is a 400, another `Content-Encoding` a 415, and a
/// request without one passes untouched. The request loses its
/// `Content-Encoding` header, which no longer describes it.
///
/// ```veles
/// app.wrap(http.decompressRequests(max: 8 * 1024 * 1024))
/// ```
///
/// A negative or zero `max` panics at this call.
@caller_location
public fun decompressRequests(max: i64): Middleware {
  if (max <= 0) panic("http.decompressRequests: max is $max; it must be positive")
  next => req => {
    val coding = (req.header(Header.contentEncoding) ?: "").trim().toLower()
    if (coding.isEmpty() || coding == "identity") return next(req)
    if (coding != "gzip" && coding != "x-gzip") {
      return Response.text("unsupported content-encoding '$coding'", status: Status.unsupportedMediaType)
    }
    when (inflateRequest(req, max)) {
      is Ok(opened) => next(opened)
      is Err(e)     => when (e) {
        is Fail    => Response.text(e.text, status: e.status)
        is IoError => Response.text("the request body could not be read", status: Status.badRequest)
      }
    }
  }
}

// `req` with its gzip body opened; null of the body is not touched
fun inflateRequest(req: Request, max: i64): Request suspends throws Fail | IoError {
  val packed = try req.bytes()
  val plain = when (gz.gunzip(packed, max)) {
    is Ok(bytes) => bytes
    is Err(e)    => throw (if (e.kind == gz.CompressKind.TooLarge) tooLarge() else badRequest("the request body is not valid gzip: ${e.message}"))
  }
  val headers = req.headers.toMutable()
  headers.remove(Header.contentEncoding)
  headers.set(Header.contentLength, "${plain.len()}")
  Request(method: req.method, path: req.path, query: req.query, headers: headers.toMap(), body: Body.hold(plain, Limits(bodyBytes: max)), peer: req.peer, params: req.params, rawQuery: req.rawQuery)
}
