// Tests of std/http's cookies (D94): what is written, what is read back, and
// what is refused.

test "a value is percent-encoded but for the unreserved bytes" {
  expect(encodeCookieValue("plain-Text_1.~") == "plain-Text_1.~")
  expect(encodeCookieValue("a b;c,d\"e\\f") == "a%20b%3Bc%2Cd%22e%5Cf")
  expect(encodeCookieValue("é") == "%C3%A9")
  expect(encodeCookieValue("") == "")
  expect(encodeCookieValue("a=b\r\nSet-Cookie: x") == "a%3Db%0D%0ASet-Cookie%3A%20x")
}

test "the defaults are the safe ones" {
  expect(setCookieLine(Cookie(name: "sid", value: "x")) == "sid=x; Path=/; HttpOnly; SameSite=Lax")
}

test "every attribute is written, in one order" {
  val c = Cookie(name: "sid", value: "v 1", path: "/app", domain: "example.com", maxAge: Duration.minutes(5), secure: true, httpOnly: true, sameSite: SameSite.Strict)
  expect(setCookieLine(c) == "sid=v%201; Max-Age=300; Domain=example.com; Path=/app; Secure; HttpOnly; SameSite=Strict")
}

test "an attribute switched off is left out" {
  val c = Cookie(name: "t", value: "1", httpOnly: false, sameSite: null)
  expect(setCookieLine(c) == "t=1; Path=/")
}

test "a zero maxAge is written, and forgets the cookie" {
  val c = Cookie(name: "sid", value: "", maxAge: Duration.seconds(0), httpOnly: false, sameSite: null)
  expect(setCookieLine(c) == "sid=; Max-Age=0; Path=/")
}

test "a cookie header is read into names and decoded values" {
  val got = parseCookies("a=1; b=two%20words;c=\"quoted\";  d=; e")
  expect(got.get("a") == "1")
  expect(got.get("b") == "two words")
  expect(got.get("c") == "quoted")
  expect(got.get("d") == "")
  expect(got.get("e") == null)
  expect(got.len() == 4)
}

test "the first of a repeated cookie name counts, and names are case-sensitive" {
  val got = parseCookies("sid=first; SID=upper; sid=second")
  expect(got.get("sid") == "first")
  expect(got.get("SID") == "upper")
}

test "a value another system wrote is kept as it came when it is not percent-encoded text" {
  val got = parseCookies("a=100%; b=%zz; c=%E9")
  expect(got.get("a") == "100%")
  expect(got.get("b") == "%zz")
  expect(got.get("c") == "%E9")
}

test "any text survives writing and reading" {
  val texts = ["héllo wörld; a=b, \"q\"", "", "line\r\nbreak", "100%", "日本語", "+plus+"]
  loop (text in texts) {
    val line = setCookieLine(Cookie(name: "k", value: text))
    val sent = line.splitOnce(";")?.0 ?: ""
    expect(parseCookies(sent).get("k") == text)
  }
}

test "a bad name is refused" {
  expectPanics(() => checkCookie(Cookie(name: "no spaces", value: "x")))
  expectPanics(() => checkCookie(Cookie(name: "", value: "x")))
  expectPanics(() => checkCookie(Cookie(name: "a;b", value: "x")))
  checkCookie(Cookie(name: "ok-name_1", value: "x"))
}

test "a bad path or domain is refused" {
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", path: "app")))
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", path: "/a;b")))
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", domain: "a b")))
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", domain: "")))
  checkCookie(Cookie(name: "a", value: "x", path: "/a/b", domain: ".example.com"))
}

test "SameSite=None needs Secure, and a negative maxAge is refused" {
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", sameSite: SameSite.None)))
  checkCookie(Cookie(name: "a", value: "x", secure: true, sameSite: SameSite.None))
  expectPanics(() => checkCookie(Cookie(name: "a", value: "x", maxAge: Duration.seconds(-1))))
}

test "the __Host- and __Secure- prefixes are enforced" {
  expectPanics(() => checkCookie(Cookie(name: "__Host-id", value: "x")))
  expectPanics(() => checkCookie(Cookie(name: "__Host-id", value: "x", secure: true, path: "/app")))
  expectPanics(() => checkCookie(Cookie(name: "__Host-id", value: "x", secure: true, domain: "example.com")))
  checkCookie(Cookie(name: "__Host-id", value: "x", secure: true))
  expectPanics(() => checkCookie(Cookie(name: "__Secure-id", value: "x")))
  checkCookie(Cookie(name: "__Secure-id", value: "x", secure: true, domain: "example.com"))
}

test "a response keeps its cookies through withHeader and a head-only copy" {
  val resp = Response.text("hi").withCookie(Cookie(name: "a", value: "1")).withCookie(Cookie(name: "b", value: "2")).withHeader("x-thing", "y")
  expect(resp.cookies.len() == 2)
  val answered = call(req => resp, Method.head, "/")
  expect(answered.cookies.len() == 2)
  expect(answered.body.isEmpty())
}

test "withoutCookie is a Max-Age=0 cookie with the same path and domain" {
  val resp = Response.empty(Status.noContent).withoutCookie("sid", path: "/app", domain: "example.com")
  expect(resp.cookies.len() == 1)
  expect(setCookieLine(resp.cookies.at(0) ?: panic("no cookie")) == "sid=; Max-Age=0; Domain=example.com; Path=/app")
}

test "a Request reads its cookies" {
  val resp = call(req => Response.text("${req.cookie("sid") ?: "none"}/${req.cookies().len()}"), Method.get, "/", headers: ["Cookie": "sid=a%20b; theme=dark"])
  expect((resp.body.decodeUtf8() ?: "") == "a b/2")
}
