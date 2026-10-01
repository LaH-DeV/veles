// Tests of streamed responses (D97): chunk framing, a declared length,
// HTTP/1.0 and HEAD, a producer that fails, pieces that arrive as they are
// written, and files that go out from disk.

use net, time

fun pieces(): Handler = handler(req => Response.stream(MediaType.text, out => {
  try out.writeText("abc")
  try out.writeText("")
  try out.write([100, 101])
}))

fun counted(calls: Atomic<i64>): Handler = handler(req => Response.stream(MediaType.text, out => {
  val _ = calls.update(n => n + 1)
  try out.writeText("body")
}, length: 4))

// one request over a fresh connection, everything read until the server closes it
fun ask(port: i64, request: string): string throws IoError {
  with conn = try net.connect("127.0.0.1", port)
  try conn.writeText(request)
  try conn.shutdownWrite()
  var out = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    out = out + (chunk.decodeUtf8() ?: "<binary>")
  }
  out
}

test "call collects a streamed body" {
  val r = call(pieces(), Method.get, "/")
  expect(r.status == Status.ok)
  expect(text(r) == "abcde")
  expect(header(r, "content-type") == "text/plain; charset=utf-8")
  expect(call(pieces(), Method.head, "/").body.isEmpty())
}

test "a streamed body shorter or longer than its declared length is not passed off as complete" {
  val short = handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("ab")
  }, length: 5))
  expect(call(short, Method.get, "/").status == Status.internalServerError)
  val long = handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("abcdef")
  }, length: 5))
  expect(call(long, Method.get, "/").status == Status.internalServerError)
}

test "a streamed response is chunked, and an empty write is not the end" {
  try withServer(pieces(), Limits(), port => {
    val r = try ask(port, "GET / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(r.startsWith("HTTP/1.1 200 OK"))
    expect(r.contains("transfer-encoding: chunked"))
    expect(!r.contains("content-length"))
    expect(r.endsWith("\r\n\r\n3\r\nabc\r\n2\r\nde\r\n0\r\n\r\n"))
  })
}

test "a declared length is sent as content-length and the bytes are not framed" {
  try withServer(counted(Atomic(value: 0)), Limits(), port => {
    val r = try ask(port, "GET / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(r.contains("content-length: 4"))
    expect(!r.contains("transfer-encoding"))
    expect(r.endsWith("\r\n\r\nbody"))
  })
}

test "an HTTP/1.0 client gets the bytes and the end of the connection" {
  try withServer(pieces(), Limits(), port => {
    val r = try ask(port, "GET / HTTP/1.0\r\n\r\n")
    expect(r.startsWith("HTTP/1.1 200 OK"))
    expect(!r.contains("transfer-encoding"))
    expect(r.contains("connection: close"))
    expect(r.endsWith("\r\n\r\nabcde"))
  })
}

test "HEAD gets the headers of a GET and the producer is not run" {
  val calls: Atomic<i64> = Atomic(value: 0)
  try withServer(counted(calls), Limits(), port => {
    val r = try ask(port, "HEAD / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(r.contains("content-length: 4"))
    expect(r.endsWith("\r\n\r\n"))
    expect(calls.load() == 0)
    val g = try ask(port, "GET / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(g.endsWith("body"))
    expect(calls.load() == 1)
  })
}

test "a producer that panics ends the connection without the last chunk" {
  val bad = handler(req => Response.stream(MediaType.text, out => {
    try out.writeText("partial")
    panic("the producer gave up")
  }))
  try withServer(bad, Limits(), port => {
    val r = try ask(port, "GET / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(r.startsWith("HTTP/1.1 200 OK"))
    expect(r.contains("7\r\npartial\r\n"))
    // no `0\r\n\r\n`: the client sees a body that stops short
    expect(!r.endsWith("0\r\n\r\n"))
  })
}

test "pieces arrive as they are written, not when the response is done" {
  val slow = handler(req => Response.stream(MediaType.eventStream, out => {
    try out.writeText("data: one\n\n")
    await sleep(Duration.millis(600))
    try out.writeText("data: two\n\n")
  }))
  try withServer(slow, Limits(), port => {
    with conn = try net.connect("127.0.0.1", port)
    try conn.writeText("GET / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    val sw = time.Stopwatch.start()
    var seen = ""
    loop (!seen.contains("one")) {
      val chunk = try conn.read()
      if (chunk.isEmpty()) break
      seen = seen + (chunk.decodeUtf8() ?: "<binary>")
    }
    expect(seen.contains("data: one"))
    // the second piece is still half a second away
    expect(!seen.contains("two"))
    expect(sw.elapsed().toMillis() < 450)
  })
}

test "a file goes out in pieces with its length, and a range of it starts anywhere" {
  val h = served(files(tree()))
  val whole = fetch(h, "/static/big.txt")
  expect(whole.status == Status.ok)
  expect(whole.body.len() == 300000)
  expect(header(whole, "content-length") == "<none>")
  // a range across the boundary of the 64 KiB pieces the file is read in
  val part = fetch(h, "/static/big.txt", ["Range": "bytes=65530-65545"])
  expect(part.status == Status.partialContent)
  expect(text(part) == "0123456789012345")
  expect(header(part, "content-range") == "bytes 65530-65545/300000")
  expect(text(fetch(h, "/static/big.txt", ["Range": "bytes=-5"])) == "56789")
}

test "over a socket a served file announces its length" {
  val h = served(files(tree()))
  try withServer(h, Limits(), port => {
    val r = try ask(port, "GET /static/big.txt HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(r.contains("content-length: 300000"))
    expect(!r.contains("transfer-encoding"))
    expect(r.len() > 300000)
    val ranged = try ask(port, "GET /static/big.txt HTTP/1.1\r\nHost: t\r\nRange: bytes=0-9\r\nConnection: close\r\n\r\n")
    expect(ranged.startsWith("HTTP/1.1 206"))
    expect(ranged.contains("content-length: 10"))
    expect(ranged.endsWith("\r\n\r\n0123456789"))
    // HEAD does not open the file at all
    val head = try ask(port, "HEAD /static/big.txt HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
    expect(head.contains("content-length: 300000"))
    expect(head.endsWith("\r\n\r\n"))
  })
}
