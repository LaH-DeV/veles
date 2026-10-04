// Tests of request bodies (D97): read when asked, under a ceiling, framed by
// Content-Length or chunks, held back by `Expect: 100-continue`, and what
// happens to what the handler never read. The first tests use `call`, no
// socket; the rest run a server on a loopback port.

use net

test fun reader(): Handler = handler(req => Response.text("${try req.bytes().len()}:${try req.text()}"))

test fun ignorer(): Handler = handler(req => Response.text("ignored"))

test "the body is read when asked, and kept for the next reader" {
  val h = handler(req => {
    val first = try req.text()
    val second = try req.text()
    val n = try req.bytes().len()
    Response.text("$first/$second/$n")
  })
  expect(text(call(h, Method.post, "/", body: "hello")) == "hello/hello/5")
  expect(text(call(h, Method.post, "/")) == "//0")
}

test "a body counts in bytes and reads as text in UTF-8" {
  val h = handler(req => Response.text("${try req.bytes().len()}:${try req.text()}"))
  expect(text(call(h, Method.post, "/", body: "abc")) == "3:abc")
  // two bytes, one character
  expect(text(call(h, Method.post, "/", body: "\u{e9}")) == "2:\u{e9}")
}

test "a body that is not UTF-8 is a 400 as text and fine as bytes" {
  with srv = try testServer(reader())
  val port = srv.port()
  with conn = try net.connect("127.0.0.1", port)
  try conn.write("POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\nContent-Length: 2\r\n\r\n".bytes().concat([0xFF, 0xFE]))
  try conn.shutdownWrite()
  var all = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    all = all + (chunk.decodeUtf8() ?: "<binary>")
  }
  // reader() asks for bytes, then for text
  expect(all.startsWith("HTTP/1.1 400"))
  expect(all.endsWith("body is not valid UTF-8"))
}

test "a body over the ceiling is a 413, and max: moves the ceiling" {
  val big = "x".repeat(1048577)
  val h = handler(req => Response.text("${try req.bytes().len()}"))
  expect(call(h, Method.post, "/", body: big).status == Status.contentTooLarge)
  expect(call(h, Method.post, "/", body: "x".repeat(1048576)).status == Status.ok)
  val small = handler(req => Response.text("${try req.bytes(max: 4).len()}"))
  expect(text(call(small, Method.post, "/", body: "abcd")) == "4")
  expect(call(small, Method.post, "/", body: "abcde").status == Status.contentTooLarge)
  val wide = handler(req => Response.text("${try req.bytes(max: 2000000).len()}"))
  expect(text(call(wide, Method.post, "/", body: big)) == "1048577")
}

test "a stream hands out pieces, knows its length, and has a ceiling of its own" {
  val h = handler(req => {
    val body = req.stream(max: 100)
    val out = StringBuilder()
    out.append("length=${body.length() ?: -1}")
    loop {
      val piece = try body.read(4)
      if (piece.isEmpty()) break
      out.append(" ${piece.len()}")
    }
    Response.text(out.toString())
  })
  expect(text(call(h, Method.post, "/", body: "0123456789")) == "length=10 4 4 2")
  expect(text(call(h, Method.post, "/")) == "length=0")
  expect(call(h, Method.post, "/", body: "x".repeat(101)).status == Status.contentTooLarge)
}

test "the untyped and typed form readers can share one body" {
  val h = handler(req => Response.text("${(try req.formValue("a")) ?: "-"}/${try req.formValues("a")}/${try req.text().len()}"))
  val r = call(h, Method.post, "/", body: "a=x&a=y", headers: ["Content-Type": "application/x-www-form-urlencoded"])
  expect(text(r) == "x/[x, y]/7")
}

// what the server sends for `parts`, written one after the other with a pause
// between, until it closes the connection
test fun talk(port: i64, parts: List<string>, finish: bool = true): string throws IoError {
  with conn = try net.connect("127.0.0.1", port)
  loop (p in parts) {
    try conn.writeText(p)
    await sleep(Duration.millis(30))
  }
  // the client is done writing: a server waiting for more sees the end
  if (finish) try conn.shutdownWrite()
  var out = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    out = out + (chunk.decodeUtf8() ?: "<binary>")
  }
  out
}

test fun head(extra: string): string = "POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n$extra\r\n"

test "a chunked body is decoded, whatever the pieces and extensions" {
  with srv = try testServer(reader())
  val port = srv.port()
  val whole = try talk(port, [head("Transfer-Encoding: chunked\r\n"), "5\r\nhello\r\n", "6;ext=1\r\n world\r\n", "0\r\nX-Trailer: 1\r\n\r\n"])
  expect(whole.startsWith("HTTP/1.1 200 OK"))
  expect(whole.endsWith("11:hello world"))
  // one write, then a chunk split across two
  val split = try talk(port, [head("Transfer-Encoding: chunked\r\n") + "A\r\n0123", "45678", "9\r\n0\r\n\r\n"])
  expect(split.endsWith("10:0123456789"))
  val empty = try talk(port, [head("Transfer-Encoding: chunked\r\n") + "0\r\n\r\n"])
  expect(empty.endsWith("0:"))
}

test "framing that could be read two ways is refused" {
  with srv = try testServer(reader())
  val port = srv.port()
  val both = try talk(port, [head("Transfer-Encoding: chunked\r\nContent-Length: 5\r\n") + "0\r\n\r\n"])
  expect(both.startsWith("HTTP/1.1 400"))
  val gzip = try talk(port, [head("Transfer-Encoding: gzip\r\n")])
  expect(gzip.startsWith("HTTP/1.1 501"))
  val size = try talk(port, [head("Transfer-Encoding: chunked\r\n") + "zz\r\nabc\r\n0\r\n\r\n"])
  expect(size.startsWith("HTTP/1.1 400"))
  val cut = try talk(port, [head("Transfer-Encoding: chunked\r\n") + "5\r\nab"])
  expect(cut.startsWith("HTTP/1.1 400"))
  val unknown = try talk(port, [head("Expect: 200-ok\r\nContent-Length: 0\r\n")])
  expect(unknown.startsWith("HTTP/1.1 417"))
}

test "a chunk or a length over the ceiling is refused before its data" {
  with srv = try testServer(reader(), Limits(bodyBytes: 10))
  val port = srv.port()
  val chunk = try talk(port, [head("Transfer-Encoding: chunked\r\n") + "B\r\n"])
  expect(chunk.startsWith("HTTP/1.1 413"))
  expect(chunk.contains("connection: close"))
  val length = try talk(port, [head("Content-Length: 11\r\n")])
  expect(length.startsWith("HTTP/1.1 413"))
  expect(length.contains("connection: close"))
  val fine = try talk(port, [head("Content-Length: 10\r\n") + "0123456789"])
  expect(fine.endsWith("10:0123456789"))
}

test "a stream may take more than the server's default ceiling" {
  val big = handler(req => {
    val body = req.stream(max: 1000)
    Response.text("${try body.readAll().len()}")
  })
  with srv = try testServer(big, Limits(bodyBytes: 10))
  val port = srv.port()
  val r = try talk(port, [head("Content-Length: 500\r\n") + "y".repeat(500)])
  expect(r.endsWith("500"))
  val over = try talk(port, [head("Content-Length: 2000\r\n")])
  expect(over.startsWith("HTTP/1.1 413"))
}

test "Expect: 100-continue is answered when the handler reads, and not before" {
  with srv = try testServer(reader())
  val port = srv.port()
  with conn = try net.connect("127.0.0.1", port)
  try conn.writeText("POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\nExpect: 100-continue\r\nContent-Length: 5\r\n\r\n")
  // the server has nothing to say yet, except to tell us to go on
  val interim = try conn.readLine(max: 200) ?: "(none)"
  expect(interim == "HTTP/1.1 100 Continue")
  val blank = try conn.readLine(max: 200) ?: "(none)"
  expect(blank.isEmpty())
  try conn.writeText("hello")
  var rest = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    rest = rest + (chunk.decodeUtf8() ?: "<binary>")
  }
  expect(rest.startsWith("HTTP/1.1 200 OK"))
  expect(rest.endsWith("5:hello"))
}

test "a handler that answers without reading never invites the body" {
  val refuse = handler(req => Response.text("no", status: Status.unauthorized))
  with srv = try testServer(refuse)
  val port = srv.port()
  val r = try talk(port, ["POST / HTTP/1.1\r\nHost: t\r\nExpect: 100-continue\r\nContent-Length: 5000000\r\n\r\n"])
  expect(r.startsWith("HTTP/1.1 401"))
  expect(!r.contains("100 Continue"))
  expect(r.contains("connection: close"))
}

test "a body the handler left unread is thrown away and the connection goes on" {
  with srv = try testServer(ignorer())
  val port = srv.port()
  with conn = try net.connect("127.0.0.1", port)
  try conn.writeText("POST /a HTTP/1.1\r\nHost: t\r\nContent-Length: 10\r\n\r\n0123456789")
  try conn.writeText("POST /b HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nabc\r\n0\r\n\r\n")
  try conn.writeText("GET /c HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
  var all = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    all = all + (chunk.decodeUtf8() ?: "<binary>")
  }
  // three requests, three answers, none of them confused by a body
  expect(all.split("HTTP/1.1 200 OK").len() == 4)
}

test "an unread body too long to be worth reading closes the connection" {
  with srv = try testServer(ignorer(), Limits(bodyBytes: 100000000))
  val port = srv.port()
  val r = try talk(port, ["POST / HTTP/1.1\r\nHost: t\r\nContent-Length: 1000000\r\n\r\n" + "z".repeat(100)])
  expect(r.startsWith("HTTP/1.1 200 OK"))
  expect(r.contains("connection: close"))
}

test "a sender that stalls in the body is a 408" {
  with srv = try testServer(reader(), Limits(bodyTimeout: Duration.millis(200)))
  val port = srv.port()
  val r = try talk(port, [head("Content-Length: 10\r\n") + "abc", "..."], finish: false)
  expect(r.startsWith("HTTP/1.1 408"))
}
