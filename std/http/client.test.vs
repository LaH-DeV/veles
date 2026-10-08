// Tests of the HTTP client (D127): against a real `testServer` for what a
// well-behaved server does, and against canned bytes on a raw socket for what
// a server must not do.

use io
use net

struct Note {
  title: string
  stars: i64 = 0

  implement Codable
}

// the kind of a request that failed, null for one that did not
test fun failKind(r: Result<ClientResponse, FetchError>): FetchKind? => when (r) {
  is Ok(_)  => null
  is Err(e) => e.kind
}

// ---- a server that says exactly what it is told to ----

struct Canned {
  listener: net.Listener
  server:   Task<()>
  url:      string

  implement Closeable {
    fun close() {
      this.listener.close()
    }
  }
}

// reads one request head, writes `reply`, closes — for every connection
test fun canned(reply: string): Canned throws IoError {
  val listener = try net.listen()
  Canned(listener, server: async cannedLoop(listener, reply), url: "http://127.0.0.1:${listener.port()}")
}

test fun cannedLoop(listener: net.Listener, reply: string) {
  loop {
    val conn = when (listener.accept()) {
      is Ok(c)  => c
      is Err(_) => break
    }
    cannedAnswer(conn, reply)
  }
}

test fun cannedAnswer(conn: net.Conn, reply: string) {
  with c = conn
  loop {
    val line = when (c.readLine(max: 8192)) {
      is Ok(l)  => l ?: break
      is Err(_) => break
    }
    if (line.isEmpty()) break
  }
  val _ = c.writeText(reply)
}

test fun echo(): Handler => handler(req => Response.text("${req.method} ${req.path}"))

// ---- the basics ----

test "get reads a body as text" {
  with srv = try testServer(echo())
  val res = try get(srv.url + "/hello")
  expect(res.status == Status.ok)
  expect(res.ok)
  expect(res.url == srv.url + "/hello")
  expect(res.header("Content-Type") == "text/plain; charset=utf-8")
  expect(try res.text() == "GET /hello")
}

test "a body is sent with its length and type, and the server reads it" {
  with srv = try testServer(handler(req => Response.text("${req.method} ${req.header("content-type") ?: "-"} ${req.header("content-length") ?: "-"} ${try req.text()}")))
  val text = try post(srv.url, body: Payload.text("hello")).text()
  expect(text == "POST text/plain; charset=utf-8 5 hello")
  val bytes = try put(srv.url, body: Payload.bytes([104, 105], contentType: MediaType.octetStream)).text()
  expect(bytes == "PUT application/octet-stream 2 hi")
  val empty = try post(srv.url).text()
  expect(empty == "POST - 0 ")
  val patched = try patch(srv.url, body: Payload.text("x")).text()
  expect(patched.startsWith("PATCH "))
  val removed = try delete(srv.url).text()
  expect(removed == "DELETE - - ")
}

test "json goes out and comes back" {
  with srv = try testServer(handler(req => Response.json(try req.text())))
  val note = Note(title: "buy milk", stars: 3)
  val res = try post(srv.url, body: try Payload.json(note))
  val back = try res.json<Note>()
  expect(back.title == "buy milk")
  expect(back.stars == 3)
}

test "json that does not fit is a decode error, not a panic" {
  with srv = try testServer(handler(req => Response.json("{\"title\": 5}")))
  val res = try get(srv.url)
  expect(res.json<Note>() is Err)
}

test "a form goes out urlencoded, repeats kept" {
  with srv = try testServer(handler(req => Response.text("${req.header("content-type") ?: "-"} ${try req.text()}")))
  val out = try post(srv.url, body: Payload.form([("name", "ann lee"), ("tag", "a&b"), ("tag", "c")])).text()
  expect(out == "application/x-www-form-urlencoded name=ann%20lee&tag=a%26b&tag=c")
}

test "headers: the client's, the request's, and a default agent" {
  with srv = try testServer(handler(req => Response.text("${req.header("user-agent") ?: "-"} ${req.header("x-a") ?: "-"} ${req.header("x-b") ?: "-"} ${req.header("host") ?: "-"}")))
  with client = Client(headers: ["X-A": "client", "x-b": "client"])
  val out = try client.get(srv.url, headers: ["X-B": "request"]).text()
  expect(out == "veles-http/1 client request 127.0.0.1:${srv.port()}")
  with plain = Client(headers: ["User-Agent": "notes/1.0"])
  expect((try plain.get(srv.url).text()).startsWith("notes/1.0 "))
}

test "a 404 is an answer; ensureSuccess makes it an error" {
  with srv = try testServer(handler(req => if (req.path == "/missing") Response.text("nope", status: Status.notFound) else Response.text("fine")))
  with res = try get(srv.url + "/missing")
  expect(res.status == Status.notFound)
  expect(!res.ok)
  expect(try res.text() == "nope")
  with again = try get(srv.url + "/missing")
  when (again.ensureSuccess()) {
    is Ok(_)  => fail("a 404 passed ensureSuccess")
    is Err(e) => {
      expect(e.status == Status.notFound)
      expect(e.message() == "${srv.url}/missing answered 404 Not Found")
    }
  }
  with good = try get(srv.url)
  expect(good.ensureSuccess() is Ok)
}

test "a response with no body: 204, HEAD, and a declared zero" {
  with srv = try testServer(handler(req => if (req.path == "/none") Response.empty(Status.noContent) else Response.text("twelve bytes")))
  with client = Client()
  val none = try client.get(srv.url + "/none")
  expect(none.status == Status.noContent)
  expect(try none.bytes().isEmpty())
  val head = try client.head(srv.url + "/x")
  expect(head.status == Status.ok)
  expect(head.header("content-length") == "12")
  expect(try head.bytes().isEmpty())
  // both went back to the pool, so a third request is on the same connection
  val third = try client.get(srv.url + "/x")
  expect(try third.text() == "twelve bytes")
}

// ---- connections ----

test "a connection is reused when its body was read" {
  with srv = try testServer(handler(req => Response.text(req.peer)))
  with client = Client()
  val first = try client.get(srv.url).text()
  val second = try client.get(srv.url).text()
  expect(first == second)
  // a body left unread is not reusable
  val third = try client.get(srv.url)
  third.close()
  val fourth = try client.get(srv.url).text()
  expect(fourth != first)
}

test "a connection the server closed while idle is replaced for a repeatable request" {
  with srv = try testServer(handler(req => Response.text(req.peer)), limits: Limits(idleTimeout: Duration.millis(100)))
  with client = Client()
  val first = try client.get(srv.url).text()
  await sleep(Duration.millis(500))
  val second = try client.get(srv.url).text()
  expect(first != second)
}

test "idle connections beyond the limit are closed, and close() drops the rest" {
  with srv = try testServer(handler(req => Response.text(req.peer)))
  with client = Client(maxIdlePerHost: 0)
  val first = try client.get(srv.url).text()
  val second = try client.get(srv.url).text()
  expect(first != second)
}

test "the timeout covers the whole request" {
  with srv = try testServer(handler(req => {
    await sleep(Duration.seconds(5))
    Response.text("late")
  }))
  expect(failKind(get(srv.url, timeout: Duration.millis(200))) == FetchKind.Timeout)
}

test "the timeout covers reading the body too" {
  with srv = try testServer(handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("first")
    await sleep(Duration.seconds(5))
    try out.writeText("second")
  })))
  val res = try get(srv.url, timeout: Duration.millis(300))
  val outcome = res.text()
  when (outcome) {
    is Ok(_)  => fail("the slow body was read in time")
    is Err(e) => expect(e.kind == FetchKind.Timeout)
  }
}

test "a refused connection is a Connect error" {
  var port = 0
  do {
    with listener = try net.listen()
    port = listener.port()
  } catch (e) {
    fail("could not listen")
  }
  expect(failKind(get("http://127.0.0.1:$port/")) == FetchKind.Connect)
}

// ---- redirects ----

test fun redirector(): Handler => handler(req => when (req.path) {
  "/start"  => Response.redirect("/middle")
  "/middle" => Response.redirect("end", status: Status.movedPermanently)
  "/end"    => Response.text("arrived")
  "/other"  => Response.redirect("/dir/../end")
  "/loop"   => Response.redirect("/loop")
  "/dir/x"  => Response.redirect("../end")
  "/q"      => if (req.rawQuery.isEmpty()) Response.redirect("?a=1") else Response.text("path ${req.path} query ${req.rawQuery}")
  else      => Response.text("path ${req.path} query ${req.rawQuery}")
})

test "redirects are followed, relative locations resolved" {
  with srv = try testServer(redirector())
  with client = Client()
  val res = try client.get(srv.url + "/start")
  expect(res.status == Status.ok)
  expect(res.url == srv.url + "/end")
  expect(try res.text() == "arrived")
  expect(try client.get(srv.url + "/other").text() == "arrived")
  expect(try client.get(srv.url + "/dir/x").text() == "arrived")
  expect(try client.get(srv.url + "/q").text() == "path /q query a=1")
}

test "redirect: false returns the redirect itself" {
  with srv = try testServer(redirector())
  with res = try get(srv.url + "/start", redirect: false)
  expect(res.status == Status.found)
  expect(res.header("location") == "/middle")
}

test "a redirect loop stops at maxRedirects" {
  with srv = try testServer(redirector())
  with client = Client(maxRedirects: 3)
  expect(failKind(client.get(srv.url + "/loop")) == FetchKind.TooManyRedirects)
}

test "only GET and HEAD follow a redirect" {
  with srv = try testServer(redirector())
  with res = try post(srv.url + "/start", body: Payload.text("x"))
  expect(res.status == Status.found)
}

test "credentials do not follow a redirect to another host" {
  with target = try testServer(handler(req => Response.text("auth=${req.header("authorization") ?: "none"} other=${req.header("x-other") ?: "none"}")))
  with source = try testServer(handler(req => Response.redirect("${target.url}/landing")))
  with client = Client()
  val out = try client.get(source.url, headers: ["Authorization": "Bearer secret", "X-Other": "kept"]).text()
  expect(out == "auth=none other=kept")
}

// ---- chunked and streamed answers ----

test "a chunked body is read whole, and a piece at a time" {
  with srv = try testServer(handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("alpha ")
    try out.writeText("beta ")
    try out.writeText("gamma")
  })))
  with client = Client()
  val res = try client.get(srv.url)
  expect(res.header("transfer-encoding") == "chunked")
  expect(try res.text() == "alpha beta gamma")
  with streamed = try client.get(srv.url)
  val body = streamed.stream()
  var seen = ""
  loop {
    val piece = try body.read(max: 4)
    if (piece.isEmpty()) break
    expect(piece.len() <= 4)
    seen = seen + (piece.decodeUtf8() ?: "?")
  }
  expect(seen == "alpha beta gamma")
  // the chunked body ended cleanly, so the connection is back in the pool
  expect(try client.get(srv.url).text() == "alpha beta gamma")
}

test "a body over the ceiling is refused, before it is read when its length says so" {
  with srv = try testServer(handler(req => Response.text("x".repeat(1000))))
  val res = try get(srv.url)
  expect(res.length() == 1000)
  val refused = res.text(max: 10)
  when (refused) {
    is Ok(_)  => fail("a 1000-byte body passed a 10-byte ceiling")
    is Err(e) => expect(e.kind == FetchKind.TooLarge)
  }
}

test "a chunked body over the ceiling is refused as it comes" {
  with srv = try testServer(handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("x".repeat(600))
    try out.writeText("x".repeat(600))
  })))
  val res = try get(srv.url)
  val refused = res.bytes(max: 1000)
  when (refused) {
    is Ok(_)  => fail("a 1200-byte body passed a 1000-byte ceiling")
    is Err(e) => expect(e.kind == FetchKind.TooLarge)
  }
}

test "retry: a repeatable request is tried again after a 503" {
  val calls: Atomic<i64> = Atomic(value: 0)
  with srv = try testServer(handler(req => {
    val n = calls.update(c => c + 1)
    if (n < 3) Response.text("busy", status: Status.serviceUnavailable) else Response.text("ok $n")
  }))
  expect(try get(srv.url).status == Status.serviceUnavailable)
  val before = calls.load()
  with res = try get(srv.url, retry: 5)
  expect(res.status == Status.ok)
  expect(before == 1)
  expect(calls.load() == 3)
}

test "retry: a POST is never repeated" {
  val calls: Atomic<i64> = Atomic(value: 0)
  with srv = try testServer(handler(req => {
    val _ = calls.update(c => c + 1)
    Response.text("busy", status: Status.serviceUnavailable)
  }))
  with res = try post(srv.url, body: Payload.text("x"))
  expect(res.status == Status.serviceUnavailable)
  expect(calls.load() == 1)
}

// ---- what a client must refuse ----

test "a URL that is not one is refused before anything is sent" {
  val bad = [
    "example.com",
    "ftp://example.com/",
    "http://",
    "http:///path",
    "http://a b/",
    "http://host/pa\r\nth",
    "http://user@host/",
    "http://host:99999/",
    "http://host:0/",
    "http://host:x/",
    "http://[::1/",
    "http://[zz]/",
    "http://exämple.com/",
  ]
  loop (url in bad) {
    expect(failKind(get(url)) == FetchKind.InvalidRequest)
  }
}

test "an https URL that nobody serves is a connect failure, not an unsupported scheme" {
  expect(failKind(get("https://127.0.0.1:1/")) == FetchKind.Connect)
}

test "a header that could split the request is refused" {
  expect(failKind(get("http://127.0.0.1:1/", headers: ["X-A": "b\r\nX-Evil: 1"])) == FetchKind.InvalidRequest)
  expect(failKind(get("http://127.0.0.1:1/", headers: ["X A": "b"])) == FetchKind.InvalidRequest)
  expect(failKind(get("http://127.0.0.1:1/", headers: ["Content-Length": "0"])) == FetchKind.InvalidRequest)
  expect(failKind(get("http://127.0.0.1:1/", headers: ["Host": "evil.example"])) == FetchKind.InvalidRequest)
}

test "an answer that is not HTTP is a Protocol error" {
  val replies = [
    "garbage\r\n\r\n",
    "HTTP/2.0 200 OK\r\n\r\n",
    "HTTP/1.1 20 OK\r\n\r\n",
    "HTTP/1.1 200 OK\r\nbroken header\r\n\r\n",
    "HTTP/1.1 200 OK\r\nX : spaced\r\n\r\n",
    "HTTP/1.1 200 OK\r\nContent-Length: 1\r\nContent-Length: 2\r\n\r\nx",
    "HTTP/1.1 200 OK\r\nContent-Length: +1\r\n\r\nx",
    "HTTP/1.1 200 OK\r\nTransfer-Encoding: gzip\r\n\r\nx",
    "HTTP/1.1 200 OK\r\n folded: x\r\n\r\n",
  ]
  loop (reply in replies) {
    with srv = try canned(reply)
    expect(failKind(get(srv.url)) == FetchKind.Protocol)
  }
}

test "a server that closes without answering is a Closed error" {
  with srv = try canned("")
  expect(failKind(get(srv.url)) == FetchKind.Closed)
}

test "a body cut short is an error, never a short body" {
  with srv = try canned("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\nabc")
  val res = try get(srv.url)
  expect(res.bytes() is Err)
  with cut = try canned("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n3\r\nab")
  val chunked = try get(cut.url)
  expect(chunked.bytes() is Err)
}

test "a body with no length ends where the connection does" {
  with srv = try canned("HTTP/1.0 200 OK\r\n\r\nuntil the end")
  val res = try get(srv.url)
  expect(try res.text() == "until the end")
}

test "interim answers are skipped, trailers and chunk extensions ignored" {
  with srv = try canned("HTTP/1.1 100 Continue\r\n\r\nHTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nSet-Cookie: a=1\r\nSet-Cookie: b=2\r\nX-Dup: 1\r\nx-dup: 2\r\n\r\n5;ext=1\r\nhello\r\n6\r\n world\r\n0\r\nX-Trailer: t\r\n\r\n")
  val res = try get(srv.url)
  expect(res.status == Status.ok)
  expect(res.setCookies == ["a=1", "b=2"])
  expect(res.header("X-Dup") == "1, 2")
  expect(try res.text() == "hello world")
}

test "a head of endless headers is refused" {
  with srv = try canned("HTTP/1.1 200 OK\r\n" + "X-Pad: 0123456789012345678901234567890123456789\r\n".repeat(2000) + "\r\n")
  expect(failKind(get(srv.url)) == FetchKind.Protocol)
}

// ---- the pieces ----

test "URLs parse into where to connect and what to ask for" {
  val u = try parseUrl("HTTP://Example.COM:8080/a/b?x=1#frag")
  expect(!u.secure)
  expect(u.host == "example.com")
  expect(u.port == 8080)
  expect(u.target == "/a/b?x=1")
  expect(u.authority() == "example.com:8080")
  val plain = try parseUrl("http://example.com")
  expect(plain.target == "/")
  expect(plain.port == 80)
  expect(plain.authority() == "example.com")
  expect(try parseUrl("http://example.com?q=1").target == "/?q=1")
  val v6 = try parseUrl("http://[::1]:9000/x")
  expect(v6.host == "::1")
  expect(v6.authority() == "[::1]:9000")
  expect(try parseUrl("https://example.com/").port == 443)
  expect(try parseUrl("http://example.com/caf\u{e9}").target == "/caf%C3%A9")
}

test "locations resolve against the URL that sent them" {
  val base = try parseUrl("http://h.example/a/b/c?q=1")
  expect(try resolve(base, "/x").text() == "http://h.example/x")
  expect(try resolve(base, "d").text() == "http://h.example/a/b/d")
  expect(try resolve(base, "../d").text() == "http://h.example/a/d")
  expect(try resolve(base, "../../../../d").text() == "http://h.example/d")
  expect(try resolve(base, "./").text() == "http://h.example/a/b/")
  expect(try resolve(base, "?z=2").text() == "http://h.example/a/b/c?z=2")
  expect(try resolve(base, "//other.example/p").text() == "http://other.example/p")
  expect(try resolve(base, "http://third.example:81/").text() == "http://third.example:81/")
  expect(try resolve(base, "#frag").text() == "http://h.example/a/b/c?q=1")
}

test "backoff doubles from 100 ms and stops at five seconds" {
  expect(backoff(1) == Duration.millis(100))
  expect(backoff(2) == Duration.millis(200))
  expect(backoff(4) == Duration.millis(800))
  expect(backoff(30) == Duration.millis(5000))
}
