// Tests of http.cors (D99): who is let in, what a preflight is answered, what
// a listed origin's response carries, and the settings that are refused.

test fun corsApp(m: Middleware): Handler {
  val r = Router()
  r.get("/data", req => Response.text("data"))
  r.wrap(m)
  r.handler()
}

test fun corsAllowed(app: Handler, origin: string): bool =>
  call(app, Method.get, "/data", headers: from(origin)).headers.get("access-control-allow-origin") == origin

test fun from(origin: string): Map<string, string> => ["Origin": origin]

test fun preflight(origin: string, asks: string = ""): Map<string, string> {
  val h: MutableMap<string, string> = ["Origin": origin, "Access-Control-Request-Method": "PUT"]
  if (!asks.isEmpty()) h.set("Access-Control-Request-Headers", asks)
  h.toMap()
}

test "no Origin, no change" {
  val app = corsApp(cors(origins: ["https://a.example"]))
  val r = call(app, Method.get, "/data")
  expect(r.headers.get("access-control-allow-origin") == null)
  expect(r.headers.get("vary") == null)
}

test "a listed origin is echoed, and the answer varies by origin" {
  val app = corsApp(cors(origins: ["https://a.example", "https://b.example"]))
  val r = call(app, Method.get, "/data", headers: from("https://b.example"))
  expect(r.headers.get("access-control-allow-origin") == "https://b.example")
  expect(r.headers.get("vary") == "origin")
  expect(r.headers.get("access-control-allow-credentials") == null)
}

test "an origin that is not listed gets no permission, and the answer still varies" {
  val app = corsApp(cors(origins: ["https://a.example"]))
  val r = call(app, Method.get, "/data", headers: from("https://evil.example"))
  expect(r.status == Status.ok)
  expect(r.headers.get("access-control-allow-origin") == null)
  expect(r.headers.get("vary") == "origin")
  // the default allows nobody
  val none = call(corsApp(cors()), Method.get, "/data", headers: from("https://a.example"))
  expect(none.headers.get("access-control-allow-origin") == null)
}

test "a wildcard answers * and does not vary" {
  val app = corsApp(cors(origins: ["*"]))
  val r = call(app, Method.get, "/data", headers: from("https://anyone.example"))
  expect(r.headers.get("access-control-allow-origin") == "*")
  expect(r.headers.get("vary") == null)
}

test "a subdomain pattern matches subdomains on a dot, and nothing else" {
  val app = corsApp(cors(origins: ["https://*.example.com"]))
  expect(corsAllowed(app, "https://app.example.com"))
  expect(corsAllowed(app, "https://a.b.example.com"))
  expect(!corsAllowed(app, "https://example.com"))
  expect(!corsAllowed(app, "https://evilexample.com"))
  expect(!corsAllowed(app, "https://.example.com"))
  expect(!corsAllowed(app, "http://app.example.com"))
  expect(!corsAllowed(app, "https://app.example.com.evil.example"))
  expect(!corsAllowed(app, "https://evil.example/.example.com"))
  expect(!corsAllowed(app, "https://user@evil.example.example.com"))
  expect(!corsAllowed(app, "https://app.example.com:8443"))
}

test "credentials echo the origin, expose lists the headers" {
  val app = corsApp(cors(origins: ["https://a.example"], credentials: true, expose: ["x-request-id", "etag"]))
  val r = call(app, Method.get, "/data", headers: from("https://a.example"))
  expect(r.headers.get("access-control-allow-credentials") == "true")
  expect(r.headers.get("access-control-expose-headers") == "x-request-id, etag")
}

test "a preflight is answered by the middleware, before the router" {
  val app = corsApp(cors(origins: ["https://a.example"], headers: ["Authorization", "Content-Type"], maxAge: Duration.hours(1)))
  val r = call(app, Method.options, "/nowhere", headers: preflight("https://a.example"))
  expect(r.status == Status.noContent)
  expect(r.headers.get("access-control-allow-origin") == "https://a.example")
  expect(r.headers.get("access-control-allow-methods") == "GET, HEAD, POST, PUT, PATCH, DELETE")
  expect(r.headers.get("access-control-allow-headers") == "authorization, content-type")
  expect(r.headers.get("access-control-max-age") == "3600")
  expect(r.headers.get("vary") == "origin")
}

test "without headers listed, a preflight allows the ones asked for" {
  val app = corsApp(cors(origins: ["https://a.example"]))
  val r = call(app, Method.options, "/data", headers: preflight("https://a.example", asks: "x-token, content-type"))
  expect(r.headers.get("access-control-allow-headers") == "x-token, content-type")
  expect(r.headers.get("vary") == "origin, access-control-request-headers")
  expect(r.headers.get("access-control-max-age") == null)
}

test "a preflight from an origin that is not listed goes on to the router" {
  val app = corsApp(cors(origins: ["https://a.example"]))
  val r = call(app, Method.options, "/data", headers: preflight("https://evil.example"))
  expect(r.headers.get("access-control-allow-origin") == null)
  expect(r.headers.get("allow") != null)
}

test "an OPTIONS without a requested method is not a preflight" {
  val app = corsApp(cors(origins: ["https://a.example"]))
  val r = call(app, Method.options, "/data", headers: from("https://a.example"))
  expect(r.headers.get("allow") != null)
  expect(r.headers.get("access-control-allow-origin") == "https://a.example")
}

test "vary keeps what the handler already said" {
  val app = handler(req => Response.text("x").withHeader("vary", "Accept-Encoding"))
  val r = call(cors(origins: ["https://a.example"])(app), Method.get, "/", headers: from("https://a.example"))
  expect(r.headers.get("vary") == "accept-encoding, origin")
}

test "settings the browsers would refuse are a panic" {
  expectPanics(() => checkCors(["*"], true, null))
  expectPanics(() => checkCors(["*", "https://a.example"], false, null))
  expectPanics(() => checkCors([""], false, null))
  expectPanics(() => checkCors(["https://a.example"], false, Duration.seconds(-1)))
  expectPanics(() => checkCors(["*.example.com"], false, null))
  expectPanics(() => checkCors(["https://*"], false, null))
  expectPanics(() => checkCors(["https://a.*.example.com"], false, null))
  expectPanics(() => checkCors(["https://*.*.example.com"], false, null))
  checkCors(["https://*.example.com", "https://a.example"], true, Duration.seconds(0))
}
