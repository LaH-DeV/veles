// Tests of response and request compression (D124): which responses
// `http.compress()` touches and which it leaves alone, a streamed body
// compressed as it goes, a `.gz` served from beside the file, and request
// bodies opened by `http.decompressRequests()` within their ceiling.

use compress as gz, fs, net, os, path

val gzipHeaders: Map<string, string> = ["accept-encoding": "gzip, deflate"]

test fun article(): string = "The quick brown fox jumps over the lazy dog. ".repeat(100)

test fun page(contentType: string = "text/html; charset=utf-8", body: string = article()): Handler =
  compress()(handler(req => Response(headers: ["content-type": contentType], body: body.bytes())))

test fun unpacked(r: Response): string = (gz.gunzip(r.body) ?? []).decodeUtf8() ?: "<not text>"

test "a large text response is gzipped for a client that accepts it" {
  val r = call(page(), Method.get, "/", headers: gzipHeaders)
  expect(r.status == Status.ok)
  expect(header(r, "content-encoding") == "gzip")
  expect(header(r, "vary") == "accept-encoding")
  expect(header(r, "content-type") == "text/html; charset=utf-8")
  expect(r.body.len() < 400)
  expect(unpacked(r) == article())
}

test "nothing changes for a client that did not ask" {
  val plain = call(page(), Method.get, "/")
  expect(plain.headers.get("content-encoding") == null)
  expect(plain.body == article().bytes())
  val refused = call(page(), Method.get, "/", headers: ["accept-encoding": "gzip;q=0, identity"])
  expect(refused.headers.get("content-encoding") == null)
  val other = call(page(), Method.get, "/", headers: ["accept-encoding": "br, deflate"])
  expect(other.headers.get("content-encoding") == null)
}

test "gzip is chosen by name or by a star, and a quality of zero refuses it" {
  expect(call(page(), Method.get, "/", headers: ["accept-encoding": "*"]).headers.get("content-encoding") == "gzip")
  expect(call(page(), Method.get, "/", headers: ["accept-encoding": "deflate, gzip;q=0.5"]).headers.get("content-encoding") == "gzip")
  expect(call(page(), Method.get, "/", headers: ["accept-encoding": "*;q=0"]).headers.get("content-encoding") == null)
  expect(call(page(), Method.get, "/", headers: ["accept-encoding": "GZIP"]).headers.get("content-encoding") == "gzip")
}

test "small bodies, other types and encoded or partial responses are left alone" {
  expect(call(page(body: "short"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == null)
  expect(call(page("image/png"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == null)
  expect(call(page("text/event-stream"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == null)
  expect(call(page("application/json"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == "gzip")
  expect(call(page("application/vnd.api+json"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == "gzip")
  expect(call(page("image/svg+xml"), Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == "gzip")
  val encoded = compress()(handler(req => Response(headers: ["content-type": "text/plain", "content-encoding": "br"], body: article().bytes())))
  expect(call(encoded, Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == "br")
  val partial = compress()(handler(req => Response(status: Status.partialContent, headers: ["content-type": "text/plain"], body: article().bytes())))
  expect(call(partial, Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == null)
  val untouchable = compress()(handler(req => Response(headers: ["content-type": "text/plain", "cache-control": "no-transform"], body: article().bytes())))
  expect(call(untouchable, Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == null)
}

test "minBytes moves the threshold" {
  val small = compress(minBytes: 10)(handler(req => Response(headers: ["content-type": "text/plain"], body: "hello hello hello".bytes())))
  expect(call(small, Method.get, "/", headers: gzipHeaders).headers.get("content-encoding") == "gzip")
}

test "Vary keeps what the handler said, and a strong ETag turns weak" {
  val tagged = compress()(handler(req => Response(headers: ["content-type": "text/plain", "vary": "origin", "etag": "\"abc\""], body: article().bytes())))
  val r = call(tagged, Method.get, "/", headers: gzipHeaders)
  expect(header(r, "vary") == "origin, accept-encoding")
  expect(header(r, "etag") == "W/\"abc\"")
  val already = compress()(handler(req => Response(headers: ["content-type": "text/plain", "vary": "Accept-Encoding", "etag": "W/\"abc\""], body: article().bytes())))
  val again = call(already, Method.get, "/", headers: gzipHeaders)
  expect(header(again, "vary") == "Accept-Encoding")
  expect(header(again, "etag") == "W/\"abc\"")
}

test "a streamed body is compressed as it is written" {
  val streaming = compress()(handler(req => Response.stream(MediaType.text, out => {
    loop (i in 0..<200) {
      try out.writeText("line $i of a streamed body that repeats itself\n")
    }
  })))
  val r = call(streaming, Method.get, "/", headers: gzipHeaders)
  expect(header(r, "content-encoding") == "gzip")
  expect(r.headers.get("content-length") == null)
  val text = unpacked(r)
  expect(text.startsWith("line 0 of a streamed body"))
  expect(text.endsWith("line 199 of a streamed body that repeats itself\n"))
  expect(r.body.len() < text.len() / 3)
  // and for a client that did not ask, it passes untouched
  expect(call(streaming, Method.get, "/").headers.get("content-encoding") == null)
}

// every byte the server sends until it closes
test fun askBytes(port: i64, request: List<u8>): List<u8> throws IoError {
  with conn = try net.connect("127.0.0.1", port)
  try conn.write(request)
  try conn.shutdownWrite()
  val out: MutableList<u8> = []
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    out.addAll(chunk)
  }
  out.toList()
}

// where the blank line after the headers ends
test fun headEnd(response: List<u8>): i64 {
  var at = 0
  loop (at + 3 < response.len() && !(response.at(at) == 13 && response.at(at + 1) == 10 && response.at(at + 2) == 13 && response.at(at + 3) == 10)) {
    at += 1
  }
  at + 4
}

test fun headOf(response: List<u8>): string = response.take(headEnd(response)).decodeUtf8() ?: ""

test fun bodyOf(response: List<u8>): List<u8> = response.drop(headEnd(response))

test fun unchunk(framed: List<u8>): List<u8> {
  val out: MutableList<u8> = []
  var at = 0
  loop {
    var size = 0
    loop (at < framed.len() && framed.at(at) != 13) {
      val digit = (framed.at(at) ?: 48).toI64()
      size = size * 16 + (if (digit >= 97) digit - 87 else digit - 48)
      at += 1
    }
    at += 2
    if (size == 0) break
    out.addAll(framed.slice(at, at + size))
    at += size + 2
  }
  out.toList()
}

test "over a socket the compressed stream is chunked and decodes" {
  val app = compress()(handler(req => Response.stream(MediaType.text, out => {
    loop (i in 0..<100) {
      try out.writeText("row $i, the same words again and again\n")
    }
  })))
  with srv = try testServer(app)
  val wire = try askBytes(srv.port(), "GET / HTTP/1.1\r\nHost: t\r\nAccept-Encoding: gzip\r\nConnection: close\r\n\r\n".bytes())
  val head = headOf(wire)
  expect(head.contains("content-encoding: gzip"))
  expect(head.contains("transfer-encoding: chunked"))
  val packed = unchunk(bodyOf(wire))
  val text = (gz.gunzip(packed) ?? []).decodeUtf8() ?: ""
  expect(text.startsWith("row 0, the same words"))
  expect(text.endsWith("row 99, the same words again and again\n"))
}

test fun precompressed(): string {
  val root = path.join(os.tempDir(), "veles-http-gzip-test")
  when (writeTree(root)) {
    is Err(e) => panic("cannot write the test tree: ${e.message()}")
    is Ok(_)  => root
  }
}

test fun writeTree(root: string) throws IoError {
  if (!fs.isDir(root)) try fs.mkdir(root)
  val text = article()
  try fs.writeFile(path.join(root, "page.html"), text)
  try fs.writeBytes(path.join(root, "page.html.gz"), gz.gzip(text.bytes(), 9))
  try fs.writeFile(path.join(root, "alone.txt"), "no compressed sibling")
}

val gzipTree: string = precompressed()

test fun filesApp(): Handler {
  val router = Router()
  router.get("/*", files(gzipTree))
  router.handler()
}

test "a .gz beside a file is served to a client that takes gzip" {
  val r = call(filesApp(), Method.get, "/page.html", headers: gzipHeaders)
  expect(r.status == Status.ok)
  expect(header(r, "content-encoding") == "gzip")
  expect(header(r, "content-type") == "text/html; charset=utf-8")
  expect(header(r, "vary") == "accept-encoding")
  expect(header(r, "etag").startsWith("W/\"gz-"))
  expect(unpacked(r) == article())
}

test "the plain file is served to everyone else, and to a range" {
  val plain = call(filesApp(), Method.get, "/page.html")
  expect(plain.headers.get("content-encoding") == null)
  expect(header(plain, "vary") == "accept-encoding")
  expect(plain.body == article().bytes())
  val ranged = call(filesApp(), Method.get, "/page.html", headers: ["accept-encoding": "gzip", "range": "bytes=0-9"])
  expect(ranged.status == Status.partialContent)
  expect(ranged.headers.get("content-encoding") == null)
  expect(ranged.body == "The quick ".bytes())
  val lone = call(filesApp(), Method.get, "/alone.txt", headers: gzipHeaders)
  expect(lone.headers.get("content-encoding") == null)
  expect(lone.headers.get("vary") == null)
}

test "the compressed variant has validators of its own" {
  val plain = call(filesApp(), Method.get, "/page.html")
  val packed = call(filesApp(), Method.get, "/page.html", headers: gzipHeaders)
  expect(header(plain, "etag") != header(packed, "etag"))
  val again = call(filesApp(), Method.get, "/page.html", headers: ["accept-encoding": "gzip", "if-none-match": header(packed, "etag")])
  expect(again.status == Status.notModified)
}

test fun echoLength(): Handler = decompressRequests(max: 5000)(handler(req => {
  val text = try req.text()
  Response.text("${text.len()} ${req.header("content-encoding") ?: "plain"}")
}))

test fun postRaw(port: i64, headers: string, body: List<u8>): string throws IoError {
  val head = "POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\n${headers}Content-Length: ${body.len()}\r\n\r\n"
  val wire = try askBytes(port, head.bytes().concat(body))
  wire.decodeUtf8() ?: "<binary>"
}

test "a gzip request body is opened, within its ceiling" {
  with srv = try testServer(echoLength())
  val port = srv.port()
  val ok = try postRaw(port, "Content-Encoding: gzip\r\n", gz.gzip("hello hello hello".bytes()))
  expect(ok.startsWith("HTTP/1.1 200"))
  expect(ok.endsWith("17 plain"))
  // an uncompressed request passes as it is
  expect((try postRaw(port, "", "plain body".bytes())).endsWith("10 plain"))
  // 100 KB of text in a few hundred bytes of gzip: over the ceiling of 5000
  val bomb = gz.gzip("a".repeat(100000).bytes(), 9)
  expect(bomb.len() < 500)
  expect((try postRaw(port, "Content-Encoding: gzip\r\n", bomb)).startsWith("HTTP/1.1 413"))
  expect((try postRaw(port, "Content-Encoding: gzip\r\n", "not gzip at all".bytes())).startsWith("HTTP/1.1 400"))
  expect((try postRaw(port, "Content-Encoding: br\r\n", "whatever".bytes())).startsWith("HTTP/1.1 415"))
}

test "a level or threshold that makes no sense panics at the caller" {
  expectPanics(() => compress(minBytes: -1))
  expectPanics(() => compress(level: 10))
  expectPanics(() => decompressRequests(max: 0))
}
