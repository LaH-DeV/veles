// Tests of multipart/form-data (D97): fields and files read part by part,
// what a boundary-shaped byte sequence inside data does not do, and what a
// malformed or oversized form is answered with.

use fs, os, path

val uploads: string = makeUploads()

test fun makeUploads(): string {
  val dir = path.join(os.tempDir(), "veles-http-multipart-test")
  when (fs.mkdir(dir)) {
    is Err(e) => panic("cannot make the upload directory: ${e.message()}")
    is Ok(_)  => dir
  }
}

val formType: Map<string, string> = ["Content-Type": "multipart/form-data; boundary=XyZ"]

test fun mpField(name: string, content: string): string =
  "--XyZ\r\nContent-Disposition: form-data; name=\"$name\"\r\n\r\n$content\r\n"

test fun mpFile(name: string, filename: string, content: string): string =
  "--XyZ\r\nContent-Disposition: form-data; name=\"$name\"; filename=\"$filename\"\r\nContent-Type: text/plain\r\n\r\n$content\r\n"

val mpEnd = "--XyZ--\r\n"

// every part, one line each, files saved under a name of ours
test fun uploader(max: i64 = 10000000, partMax: i64 = 1000000, maxParts: i64 = 100): Handler = handler(req => {
  val form = try req.multipart(max: max, maxParts: maxParts)
  val out = StringBuilder()
  loop {
    val part = try form.next() ?: break
    if (val original = part.filename) {
      val n = try part.saveTo(path.join(uploads, "up-${part.name}"), max: partMax)
      out.append("file ${part.name}=$original ${part.contentType ?: "-"} $n\n")
    } else {
      out.append("field ${part.name}=${try part.text(max: partMax)}\n")
    }
  }
  Response.text(out.toString())
})

test fun mpSend(h: Handler, body: string, headers: Map<string, string> = formType): Response = call(h, Method.post, "/", body: body, headers: headers)

test "fields and a file are read part by part" {
  val body = mpField("title", "Hello") + mpFile("doc", "a.txt", "file body") + mpField("empty", "") + mpEnd
  val r = mpSend(uploader(), body)
  expect(r.status == Status.ok)
  expect(text(r) == "field title=Hello\nfile doc=a.txt text/plain 9\nfield empty=\n")
  expect(try fs.readFile(path.join(uploads, "up-doc")) == "file body")
}

test "a preamble, an epilogue, transport padding and a quoted boundary are all tolerated" {
  val body = "ignored preamble\r\n--XyZ  \r\nContent-Disposition: form-data; name=\"a\"\r\n\r\n1\r\n--XyZ--\r\nignored epilogue"
  expect(text(mpSend(uploader(), body)) == "field a=1\n")
  val quoted = mpSend(uploader(), mpField("a", "2") + mpEnd, ["Content-Type": "multipart/form-data; charset=utf-8; boundary=\"XyZ\""])
  expect(text(quoted) == "field a=2\n")
}

test "data that looks like the start of a boundary is data" {
  val tricky = "line\r\n--Xy not the end\r\n--XyQ\r\n--\r\nlast"
  val r = mpSend(uploader(), mpFile("f", "t.txt", tricky) + mpEnd)
  expect(text(r) == "file f=t.txt text/plain ${tricky.len()}\n")
  expect(try fs.readFile(path.join(uploads, "up-f")) == tricky)
}

test "a part left unread is skipped by the next call" {
  val h = handler(req => {
    val form = try req.multipart(max: 1000000)
    val first = try form.next() ?: return Response.text("none")
    // the first part is never read
    val second = try form.next() ?: return Response.text("only ${first.name}")
    Response.text("${first.name} then ${second.name}=${try second.text(max: 100)}")
  })
  expect(text(mpSend(h, mpField("a", "x".repeat(200000)) + mpField("b", "y") + mpEnd)) == "a then b=y")
}

test "the parts' own headers and the file name are what the client said" {
  val h = handler(req => {
    val form = try req.multipart(max: 100000)
    val part = try form.next() ?: return Response.text("none")
    Response.text("${part.name}|${part.filename ?: "-"}|${part.contentType ?: "-"}|${part.headers.get("x-extra") ?: "-"}")
  })
  val body = "--XyZ\r\nContent-Disposition: form-data; name=\"up\"; filename=\"C:\\dir\\a;b.png\"\r\nContent-Type: image/png\r\nX-Extra: 1\r\n\r\nabc\r\n" + mpEnd
  expect(text(mpSend(h, body)) == "up|C:\\dir\\a;b.png|image/png|1")
}

test "a form that is not multipart, or has no boundary, is refused" {
  expect(mpSend(uploader(), mpField("a", "1") + mpEnd, ["Content-Type": "text/plain"]).status == Status.unsupportedMediaType)
  expect(mpSend(uploader(), mpField("a", "1") + mpEnd, [:]).status == Status.unsupportedMediaType)
  expect(mpSend(uploader(), mpField("a", "1") + mpEnd, ["Content-Type": "multipart/form-data"]).status == Status.badRequest)
  expect(mpSend(uploader(), mpField("a", "1") + mpEnd, ["Content-Type": "multipart/form-data; boundary="]).status == Status.badRequest)
}

test "a malformed body is a 400" {
  // no closing boundary
  expect(mpSend(uploader(), mpField("a", "1")).status == Status.badRequest)
  // ends inside a part
  expect(mpSend(uploader(), "--XyZ\r\nContent-Disposition: form-data; name=\"a\"\r\n\r\nabc").status == Status.badRequest)
  // ends inside the headers
  expect(mpSend(uploader(), "--XyZ\r\nContent-Disposition: form-data; na").status == Status.badRequest)
  // no boundary at all
  expect(mpSend(uploader(), "just some text").status == Status.badRequest)
  // a part without a name, and one that is not form-data
  expect(mpSend(uploader(), "--XyZ\r\nContent-Disposition: form-data\r\n\r\nx\r\n" + mpEnd).status == Status.badRequest)
  expect(mpSend(uploader(), "--XyZ\r\nContent-Disposition: attachment; name=\"a\"\r\n\r\nx\r\n" + mpEnd).status == Status.badRequest)
  expect(mpSend(uploader(), "--XyZ\r\nX-Only: 1\r\n\r\nx\r\n" + mpEnd).status == Status.badRequest)
  // a header without a colon
  expect(mpSend(uploader(), "--XyZ\r\nnonsense\r\n\r\nx\r\n" + mpEnd).status == Status.badRequest)
  // text that is not UTF-8 in a field is asked for as text
  expect(mpSend(uploader(), mpField("a", "ok") + "--XyZ\r\nContent-Disposition: form-data; name=\"b\"\r\n\r\n").status == Status.badRequest)
}

test "too many parts, a part over its ceiling and a body over the total are 413" {
  val three = mpField("a", "1") + mpField("b", "2") + mpField("c", "3") + mpEnd
  expect(mpSend(uploader(maxParts: 3), three).status == Status.ok)
  expect(mpSend(uploader(maxParts: 2), three).status == Status.contentTooLarge)
  expect(mpSend(uploader(partMax: 4), mpField("a", "12345") + mpEnd).status == Status.contentTooLarge)
  expect(mpSend(uploader(partMax: 5), mpField("a", "12345") + mpEnd).status == Status.ok)
  expect(mpSend(uploader(max: 50), three).status == Status.contentTooLarge)
}

test "a saved part over its ceiling leaves no file behind" {
  val target = path.join(uploads, "up-big")
  expect(mpSend(uploader(partMax: 10), mpFile("big", "b.txt", "x".repeat(100)) + mpEnd).status == Status.contentTooLarge)
  expect(!fs.exists(target))
}

test "a large upload arrives chunk by chunk and lands on disk whole" {
  val size = 300000
  val body = mpFile("big", "big.txt", "0123456789".repeat(size / 10)) + mpField("after", "done") + mpEnd
  val handlerFor = uploader()
  with srv = try testServer(handlerFor)
  val port = srv.port()
  val head = "POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\nContent-Type: multipart/form-data; boundary=XyZ\r\nContent-Length: ${body.len()}\r\n\r\n"
  val r = try talk(port, [head, body.substring(0, 100000) ?: "", body.substring(100000, 200000) ?: "", body.substring(200000, body.len()) ?: ""])
  expect(r.startsWith("HTTP/1.1 200 OK"))
  expect(r.endsWith("file big=big.txt text/plain 300000\nfield after=done\n"))
  // the same form, chunked
  val chunkHead = "POST / HTTP/1.1\r\nHost: t\r\nConnection: close\r\nContent-Type: multipart/form-data; boundary=XyZ\r\nTransfer-Encoding: chunked\r\n\r\n"
  val a = body.substring(0, 150001) ?: ""
  val b = body.substring(150001, body.len()) ?: ""
  val chunked = try talk(port, [chunkHead + "${hexLength(a.len())}\r\n" + a + "\r\n", "${hexLength(b.len())}\r\n" + b + "\r\n0\r\n\r\n"])
  expect(chunked.endsWith("file big=big.txt text/plain 300000\nfield after=done\n"))
  expect((try fs.stat(path.join(uploads, "up-big"))).size == 300000)
}
