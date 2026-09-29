// Cross-origin resource sharing (D99). A browser asks before it lets a page
// read a response from another origin: with an `Origin` header on the request,
// and for anything beyond a plain form post or GET first with a "preflight"
// `OPTIONS`. `cors` is the middleware that answers both, and the one that
// refuses the combinations the browsers would refuse anyway.

/// Lets browser pages from other origins call this server. Nothing is
/// allowed by default: `http.cors()` alone changes no response.
///
/// ```veles
/// app.wrap(http.cors(
///   origins: ["https://app.example.com", "https://*.example.org"],
///   headers: ["authorization", "content-type"],
///   credentials: true,
///   maxAge: Duration.hours(1),
/// ))
/// ```
///
/// `origins` are exact (`"https://app.example.com"`, scheme and port
/// included), `"*"` for every origin, or `"https://*.example.com"` for any
/// subdomain — matched on a dot, so `https://evilexample.com` is not one.
/// A request without `Origin` is not a cross-origin request and is passed on
/// untouched; so is one from an origin that is not listed, which the browser
/// then refuses to show to the page.
///
/// A preflight from an allowed origin is answered here (204) and does not
/// reach the router. Its `Access-Control-Allow-Headers` is `headers` when
/// given, otherwise the headers the browser asked for. The responses that
/// depend on the origin say `Vary: Origin`, so a cache keeps them apart.
///
/// `"*"` together with `credentials: true` is a panic at the caller's line —
/// browsers refuse a wildcard for a request with cookies — and so is a
/// pattern that is not one of the three forms above.
@caller_location
public fun cors(
  origins: List<string> = [],
  methods: List<Method> = [Method.get, Method.head, Method.post, Method.put, Method.patch, Method.delete],
  headers: List<string> = [],
  expose: List<string> = [],
  credentials: bool = false,
  maxAge: Duration? = null,
): Middleware {
  checkCors(origins, credentials, maxAge)
  val config = Cors(
    origins,
    any: origins.contains("*"),
    methods: methods.map(m => m.name).join(", "),
    headers: headers.map(h => h.toLower()).join(", "),
    expose: expose.join(", "),
    credentials,
    maxAge: maxAge?.toSeconds() ?: -1,
  )
  next => req => corsAnswer(config, next, req)
}

struct Cors {
  origins:     List<string>
  any:         bool
  methods:     string
  headers:     string
  expose:      string
  credentials: bool
  maxAge:      i64  // seconds; -1 when not said
}

@caller_location
fun checkCors(origins: List<string>, credentials: bool, maxAge: Duration?) {
  if (origins.contains("*") && origins.len() > 1) panic("cors: '*' already allows every origin; list nothing next to it")
  if (origins.contains("*") && credentials) panic("cors: '*' cannot be combined with credentials: true, browsers refuse it; list the origins")
  if (val age = maxAge && age.isNegative()) panic("cors: maxAge cannot be negative")
  loop (o in origins) {
    if (o.isEmpty()) panic("cors: an origin cannot be empty")
    if (o != "*" && o.contains("*") && wildcardParts(o) == null) {
      panic("cors: '$o' is not an origin pattern; a wildcard is written 'https://*.example.com'")
    }
  }
}

// "https://*.example.com" → ("https://", ".example.com"), or null when the
// star is anywhere else
fun wildcardParts(pattern: string): (string, string)? {
  val (before, after) = pattern.splitOnce("*") ?: return null
  if (!before.endsWith("://") || !after.startsWith(".") || after.len() < 2 || after.contains("*")) return null
  (before, after)
}

fun originAllowed(c: Cors, origin: string): bool {
  if (c.any) return true
  loop (o in c.origins) {
    if (o == origin) return true
    val (before, after) = wildcardParts(o) ?: continue
    if (origin.len() <= before.len() + after.len() || !origin.startsWith(before) || !origin.endsWith(after)) continue
    // what the star stands for: labels, nothing that could end the host early
    val middle = origin.substring(before.len(), origin.len() - after.len()) ?: continue
    if (middle.bytes().all(b => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '.' || b == '-')) return true
  }
  false
}

// `names` added to the response's `Vary`, once each
fun withVary(resp: Response, names: List<string>): Response {
  val have = (resp.headers.get("vary") ?: "").split(",").map(s => s.trim().toLower()).filter(s => !s.isEmpty())
  val all: MutableList<string> = have.toMutable()
  loop (n in names) {
    if (!all.contains(n)) all.push(n)
  }
  resp.withHeader("vary", all.toList().join(", "))
}

fun corsAnswer(c: Cors, next: Handler, req: Request): Response {
  val origin = req.header(Header.origin) ?: return next(req)
  if (!originAllowed(c, origin)) return withVary(next(req), ["origin"])
  // a wildcard is sent as one, unless credentials need the very origin
  val shown = if (c.any && !c.credentials) "*" else origin
  val varies = shown != "*"
  if (req.method == Method.options && req.header("access-control-request-method") != null) {
    var r = Response.empty(Status.noContent)
      .withHeader("access-control-allow-origin", shown)
      .withHeader("access-control-allow-methods", c.methods)
    val asked = req.header("access-control-request-headers") ?: ""
    val reflected = c.headers.isEmpty()
    val allowHeaders = if (reflected) asked else c.headers
    if (!allowHeaders.isEmpty()) r = r.withHeader("access-control-allow-headers", allowHeaders)
    if (c.maxAge >= 0) r = r.withHeader("access-control-max-age", "${c.maxAge}")
    if (c.credentials) r = r.withHeader("access-control-allow-credentials", "true")
    val names: MutableList<string> = []
    if (varies) names.push("origin")
    if (reflected) names.push("access-control-request-headers")
    return withVary(r, names.toList())
  }
  var r = next(req).withHeader("access-control-allow-origin", shown)
  if (c.credentials) r = r.withHeader("access-control-allow-credentials", "true")
  if (!c.expose.isEmpty()) r = r.withHeader("access-control-expose-headers", c.expose)
  if (varies) r = withVary(r, ["origin"])
  r
}
