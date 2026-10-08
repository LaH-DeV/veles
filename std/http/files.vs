// Static files (D96): `http.files(dir)` and what a browser or a cache expects
// of a file it may keep — validators (`ETag`, `Last-Modified`) and the
// conditional requests that use them, one byte range, `Cache-Control`, and
// what a directory means. The decisions are taken from `fs.stat` alone, so a
// request that ends in a 304 or a 412 never reads the file.
use fs
use path
use time

/// A handler serving files under `dir` for a route ending in `*`:
/// `app.get("/static/*", http.files("./public"))`. It throws like a handler
/// you would write (a missing file is a 404, a read error a 500), so the
/// router adapts it; `http.handler(http.files(d))` is the form `serve` takes
/// directly.
///
/// What it does, and the option that changes it:
/// - A response carries `last-modified` and a weak `etag` made of the size
///   and the write time, and answers `If-None-Match`, `If-Modified-Since`
///   with 304 and `If-Match`, `If-Unmodified-Since` with 412 (`etag: false`,
///   `lastModified: false` turn either off).
/// - One `Range: bytes=…` is answered with 206 (`accept-ranges: bytes`); an
///   unsatisfiable one with 416, several at once with the whole file, which
///   RFC 9110 allows.
/// - `cache-control` is `no-cache` — keep a copy, ask before using it, which
///   the 304 makes cheap. `maxAge` lets a client use its copy for that long
///   without asking; `immutable` adds that the file never changes under this
///   name (fingerprinted assets) and needs a `maxAge`.
/// - A directory serves the first of `index` that exists there, and 404
///   otherwise (there is no listing). `/docs`, with no slash, is a 308 to
///   `/docs/` so relative links in its page resolve; `redirect: false` serves
///   the index at `/docs` itself.
/// - A path with a segment starting with `.` (`.env`, `.git/config`) is a
///   404 unless `dotfiles: true`.
/// - Where `name.gz` lies beside `name`, a client that accepts gzip gets the
///   `.gz` as it is (`content-encoding: gzip`, its own validators, the type
///   of `name`) unless it asks for a range; both answers say
///   `vary: accept-encoding`.
/// - Only GET and HEAD; anything else is a 405.
///
/// A negative `maxAge`, or `immutable` without one, panics at this call.
///
/// ```veles
/// app.get("/assets/*", http.files("./public", maxAge: Duration.days(365), immutable: true))
/// ```
@caller_location
public fun files(
  dir: string,
  maxAge: Duration? = null,
  immutable: bool = false,
  index: List<string> = ["index.html"],
  dotfiles: bool = false,
  redirect: bool = true,
  etag: bool = true,
  lastModified: bool = true,
): sendable fun(Request): Response suspends throws Fail | IoError {
  if (immutable && maxAge == null) panic("http.files: immutable needs a maxAge; without one the header would say both 'no-cache' and 'never changes'")
  if (val age = maxAge && age.isNegative()) panic("http.files: maxAge must not be negative")
  val server = FileServer(dir, cacheControl: cacheControlOf(maxAge, immutable), index, dotfiles, redirect, etag, lastModified)
  req => try server.serve(req)
}

// "no-cache", or "max-age=N" with ", immutable"
fun cacheControlOf(maxAge: Duration?, immutable: bool): string {
  val age = maxAge ?: return "no-cache"
  "max-age=${age.toSeconds()}" + (if (immutable) ", immutable" else "")
}

struct FileServer {
  dir:          string
  cacheControl: string
  index:        List<string>
  dotfiles:     bool
  redirect:     bool
  etag:         bool
  lastModified: bool

  fun serve(req: Request): Response throws Fail | IoError {
    if (req.method != Method.get && req.method != Method.head) {
      return Response.text("method not allowed", status: Status.methodNotAllowed).withHeader(Header.allow, "GET, HEAD")
    }
    val rel = req.param("*")
    val segments = rel.replace("\\", "/").split("/")
    var p = if (rel.isEmpty()) this.dir else path.join(this.dir, rel)
    // Two checks, either enough on its own today. `within` is the one that
    // holds by construction: whatever the request spells, the file must be
    // under `dir` once `.`, `..` and both separators are resolved — a
    // `..\secret` is a traversal on Windows even though it has no `/`. A `..`
    // segment is refused outright as well, since a browser never sends one.
    if (!path.within(this.dir, p) || segments.contains("..")) throw forbidden()
    // a 404 and not a 403: it does not say the file is there
    if (!this.dotfiles && segments.any(s => s.startsWith(".") && s != ".")) throw notFound("no such file: /$rel")
    if (fs.isDir(p)) {
      if (this.redirect && !req.path.endsWith("/")) return this.slashRedirect(req)
      var found: string? = null
      loop (name in this.index) {
        val candidate = path.join(p, name)
        if (fs.isFile(candidate)) {
          found = candidate
          break
        }
      }
      p = found ?: throw notFound("no such file: /$rel")
    }
    if (!fs.isFile(p)) throw notFound("no such file: /$rel")

    // a "name.gz" beside "name" is the same file, compressed beforehand: sent
    // as it lies to a client that takes gzip, unless it asks for a range (of the
    // uncompressed bytes) — and its validators are its own
    val hasGzip = fs.isFile(p + ".gz")
    val gzipped = hasGzip && acceptsGzip(req.header(Header.acceptEncoding)) && req.header("range") == null
    val sent = if (gzipped) p + ".gz" else p
    val st = try fs.stat(sent)
    val tag: string? = if (this.etag) "W/\"${if (gzipped) "gz-" else ""}${st.size}-${st.modified.toMicros()}\"" else null
    val modified: time.Timestamp? = if (this.lastModified) st.modified else null
    val headers: MutableMap<string, string> = ["cache-control": this.cacheControl]
    if (!gzipped) headers.set("accept-ranges", "bytes")
    if (hasGzip) headers.set("vary", "accept-encoding")
    if (gzipped) headers.set(Header.contentEncoding, "gzip")
    if (val t = tag) headers.set("etag", t)
    if (val m = modified) headers.set("last-modified", time.formatHttp(m))

    if (val status = precondition(req, tag, modified)) {
      return if (status == Status.notModified) Response(status, headers: headers.toMap()) else Response.empty(status)
    }
    val size = st.size
    val media = MediaType.ofExtension(path.ext(p))
    if (val header = req.header("range") && val r = parseRange(header, size) && ifRangeHolds(req, modified)) {
      if (!r.satisfiable) {
        return Response(status: Status.rangeNotSatisfiable, headers: ["content-range": "bytes */$size", "accept-ranges": "bytes"])
      }
      headers.set("content-range", "bytes ${r.from}-${r.to}/$size")
      return this.send(media, headers, p, from: r.from, count: r.to - r.from + 1, status: Status.partialContent)
    }
    this.send(media, headers, sent, from: 0, count: size, status: Status.ok)
  }

  // The file goes out as it is read, a piece at a time, so a large one is
  // never held whole and a request that only wants its head (HEAD) never
  // opens it. The length is announced from the stat; a file that shrinks
  // meanwhile ends the response short and the connection is closed, which
  // the client sees as a truncated download.
  fun send(media: MediaType, headers: MutableMap<string, string>, p: string, from: i64, count: i64, status: Status): Response {
    var resp = Response.stream(media, sendFile(p, from, count), length: count, status: status)
    loop ((name, value) in headers.entries()) {
      resp = resp.withHeader(name, value)
    }
    resp
  }

  // `/docs` → `/docs/`, keeping the query. The target is built from the
  // decoded path re-encoded, so a `\` or a run of slashes can never make it
  // a protocol-relative `//host` (which would send a visitor off the site).
  fun slashRedirect(req: Request): Response {
    val target = "/" + segmentsOf(req.path).map(s => percentEncode(s)).join("/") + "/"
    Response.redirect(if (req.rawQuery.isEmpty()) target else target + "?" + req.rawQuery, Status.permanentRedirect)
  }
}

// the producer of a file's bytes `from`.. for `count` of them
fun sendFile(p: string, from: i64, count: i64): sendable fun(BodyWriter) suspends throws IoError => (out => {
  with f = try fs.open(p)
  var at = from
  var left = count
  loop (left > 0) {
    val chunk = try f.readAt(at, if (left < 65536) left else 65536)
    // the file is shorter than it was: what was announced cannot be sent
    if (chunk.isEmpty()) break
    try out.write(chunk)
    at += chunk.len()
    left -= chunk.len()
  }
})

// RFC 9110 §13.2.2, in its order: the status a request that must not reach
// the body gets — 304 or 412 — or null. `If-Match` compares strongly and
// this validator is weak, so only `*` satisfies it. A date that does not
// parse is ignored, as the RFC says.
fun precondition(req: Request, tag: string?, modified: time.Timestamp?): Status? {
  if (val m = req.header("if-match")) {
    if (tag != null && m.trim() != "*") return Status.preconditionFailed
  } else if (val since = req.header("if-unmodified-since")) {
    if (val mod = modified && val t = time.parseHttp(since) && mod.toSeconds() > t.toSeconds()) return Status.preconditionFailed
  }
  if (val none = req.header("if-none-match")) {
    if (none.trim() == "*" || (tag != null && etagListHas(none, tag))) return Status.notModified
  } else if (val since = req.header("if-modified-since")) {
    if (val mod = modified && val t = time.parseHttp(since) && mod.toSeconds() <= t.toSeconds()) return Status.notModified
  }
  null
}

// `"a", W/"b"` against one tag, weakly: the `W/` marks are ignored
fun etagListHas(list: string, tag: string): bool {
  val want = opaque(tag)
  list.split(",").any(t => opaque(t.trim()) == want)
}

fun opaque(tag: string): string => if (tag.startsWith("W/")) tag.substring(2, tag.len()) ?: tag else tag

// A range of bytes: `from..to` inclusive; not `satisfiable` when it starts
// beyond the end
struct ByteRange {
  from:        i64
  to:          i64
  satisfiable: bool
}

// What `Range` asks of a body of `size` bytes; null means ignore the header
// and send everything — for another unit, for several ranges, for text that
// is not a range (RFC 9110 §14.2). A suffix longer than the body is the
// whole body.
fun parseRange(header: string, size: i64): ByteRange? {
  val h = header.trim()
  if (!h.startsWith("bytes=")) return null
  val spec = (h.substring(6, h.len()) ?: return null).trim()
  if (spec.contains(",")) return null
  val (first, last) = spec.splitOnce("-") ?: return null
  val none = ByteRange(from: 0, to: -1, satisfiable: false)
  if (first.isEmpty()) {
    val n = last.toInt() ?: return null
    if (n < 0) return null
    if (n == 0 || size == 0) return none
    return ByteRange(from: if (n > size) 0 else size - n, to: size - 1, satisfiable: true)
  }
  val a = first.toInt() ?: return null
  if (a < 0) return null
  if (last.isEmpty()) {
    if (a >= size) return none
    return ByteRange(from: a, to: size - 1, satisfiable: true)
  }
  val b = last.toInt() ?: return null
  if (b < a) return null
  if (a >= size) return none
  ByteRange(from: a, to: if (b >= size) size - 1 else b, satisfiable: true)
}

// `If-Range` makes a range conditional on the file being the one the client
// has. An entity tag needs a strong match, which a weak tag never gives, so
// it always means "send everything"; a date must be this file's, to the second.
fun ifRangeHolds(req: Request, modified: time.Timestamp?): bool {
  val value = (req.header("if-range") ?: return true).trim()
  if (value.startsWith("\"") || value.startsWith("W/")) return false
  val mod = modified ?: return false
  val date = time.parseHttp(value) ?: return false
  mod.toSeconds() == date.toSeconds()
}
