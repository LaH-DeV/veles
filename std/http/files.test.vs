// Tests of `http.files` (D96): validators and conditional requests, byte
// ranges, Cache-Control, directories, dotfiles. Each one serves a small tree
// written under the temp directory and asks it through `call`.

use fs, os, path, time

// a.txt is ten bytes, so the ranges below can be read off the digits
// The tests run in parallel and a rewrite truncates the file for a moment, so
// the tree is written once, before any of them starts.
val fixture: string = build()

fun tree(): string = fixture

fun build(): string {
  val root = path.join(os.tempDir(), "veles-http-files-test")
  when (prepare(root)) {
    is Err(e) => panic("cannot write the test tree: ${e.message()}")
    is Ok(_)  => root
  }
}

fun prepare(root: string) throws IoError {
  if (!fs.isDir(root)) try fs.mkdir(root)
  val docs = path.join(root, "docs")
  if (!fs.isDir(docs)) try fs.mkdir(docs)
  val bare = path.join(root, "bare")
  if (!fs.isDir(bare)) try fs.mkdir(bare)
  val hidden = path.join(root, ".well-known")
  if (!fs.isDir(hidden)) try fs.mkdir(hidden)
  try fs.writeFile(path.join(root, "a.txt"), "0123456789")
  try fs.writeFile(path.join(root, ".env"), "secret")
  try fs.writeFile(path.join(root, "big.txt"), "0123456789".repeat(30000))
  try fs.writeFile(path.join(docs, "index.html"), "<h1>docs</h1>")
  try fs.writeFile(path.join(hidden, "x.txt"), "ok")
}

fun served(server: sendable fun(Request): Response suspends throws Fail | IoError): Handler {
  val router = Router()
  router.get("/static/*", server)
  router.any("/all/*", server)
  router.get("/*", server)
  router.handler()
}

fun fetch(h: Handler, target: string, headers: Map<string, string> = [:]): Response = call(h, Method.get, target, headers: headers)

fun text(r: Response): string = r.body.decodeUtf8() ?: "<not text>"

fun header(r: Response, name: string): string = r.headers.get(name) ?: "<none>"

test "a file comes with its validators and the safe cache header" {
  val h = served(files(tree()))
  val r = fetch(h, "/static/a.txt")
  expect(r.status == Status.ok)
  expect(text(r) == "0123456789")
  expect(header(r, "content-type") == "text/plain; charset=utf-8")
  expect(header(r, "cache-control") == "no-cache")
  expect(header(r, "accept-ranges") == "bytes")
  expect(header(r, "etag").startsWith("W/\"10-"))
  expect(header(r, "last-modified").endsWith(" GMT"))
}

test "If-None-Match and If-Modified-Since answer 304 with the validators and no body" {
  val h = served(files(tree()))
  val first = fetch(h, "/static/a.txt")
  val tag = header(first, "etag")
  val date = header(first, "last-modified")

  val same = fetch(h, "/static/a.txt", ["If-None-Match": tag])
  expect(same.status == Status.notModified)
  expect(same.body.isEmpty())
  expect(header(same, "etag") == tag)
  expect(header(same, "cache-control") == "no-cache")

  // weak comparison: a list, and the W/ mark ignored
  expect(fetch(h, "/static/a.txt", ["If-None-Match": "\"x\", $tag"]).status == Status.notModified)
  expect(fetch(h, "/static/a.txt", ["If-None-Match": tag.replace("W/", "")]).status == Status.notModified)
  expect(fetch(h, "/static/a.txt", ["If-None-Match": "*"]).status == Status.notModified)
  expect(fetch(h, "/static/a.txt", ["If-None-Match": "\"other\""]).status == Status.ok)

  expect(fetch(h, "/static/a.txt", ["If-Modified-Since": date]).status == Status.notModified)
  expect(fetch(h, "/static/a.txt", ["If-Modified-Since": "Thu, 01 Jan 1970 00:00:00 GMT"]).status == Status.ok)
  // a date that does not parse is ignored, as RFC 9110 says
  expect(fetch(h, "/static/a.txt", ["If-Modified-Since": "yesterday"]).status == Status.ok)
  // If-None-Match wins when both are sent: the tag does not match, so the date is not asked
  expect(fetch(h, "/static/a.txt", ["If-None-Match": "\"other\"", "If-Modified-Since": date]).status == Status.ok)
}

test "If-Match and If-Unmodified-Since answer 412" {
  val h = served(files(tree()))
  val first = fetch(h, "/static/a.txt")
  val tag = header(first, "etag")
  expect(fetch(h, "/static/a.txt", ["If-Match": "*"]).status == Status.ok)
  // a strong comparison, and this tag is weak
  expect(fetch(h, "/static/a.txt", ["If-Match": tag]).status == Status.preconditionFailed)
  expect(fetch(h, "/static/a.txt", ["If-Unmodified-Since": "Thu, 01 Jan 1970 00:00:00 GMT"]).status == Status.preconditionFailed)
  expect(fetch(h, "/static/a.txt", ["If-Unmodified-Since": header(first, "last-modified")]).status == Status.ok)
}

test "etag: false and lastModified: false leave those headers and their conditions out" {
  val h = served(files(tree(), etag: false, lastModified: false))
  val r = fetch(h, "/static/a.txt")
  expect(r.status == Status.ok)
  expect(!r.headers.containsKey("etag"))
  expect(!r.headers.containsKey("last-modified"))
  expect(fetch(h, "/static/a.txt", ["If-Modified-Since": "Fri, 01 Jan 2100 00:00:00 GMT"]).status == Status.ok)
  expect(fetch(h, "/static/a.txt", ["If-None-Match": "\"x\""]).status == Status.ok)
}

test "one byte range is a 206" {
  val h = served(files(tree()))
  val r = fetch(h, "/static/a.txt", ["Range": "bytes=2-4"])
  expect(r.status == Status.partialContent)
  expect(text(r) == "234")
  expect(header(r, "content-range") == "bytes 2-4/10")
  expect(header(r, "content-type") == "text/plain; charset=utf-8")

  expect(text(fetch(h, "/static/a.txt", ["Range": "bytes=7-"])) == "789")
  expect(text(fetch(h, "/static/a.txt", ["Range": "bytes=-3"])) == "789")
  expect(text(fetch(h, "/static/a.txt", ["Range": "bytes=0-0"])) == "0")
  // past the end is cut to the end; a suffix longer than the file is the file
  val cut = fetch(h, "/static/a.txt", ["Range": "bytes=8-99"])
  expect(cut.status == Status.partialContent && text(cut) == "89" && header(cut, "content-range") == "bytes 8-9/10")
  val whole = fetch(h, "/static/a.txt", ["Range": "bytes=-50"])
  expect(whole.status == Status.partialContent && text(whole) == "0123456789" && header(whole, "content-range") == "bytes 0-9/10")
}

test "a range that starts past the end is a 416 naming the size" {
  val h = served(files(tree()))
  val r = fetch(h, "/static/a.txt", ["Range": "bytes=10-"])
  expect(r.status == Status.rangeNotSatisfiable)
  expect(header(r, "content-range") == "bytes */10")
  expect(r.body.isEmpty())
  expect(fetch(h, "/static/a.txt", ["Range": "bytes=-0"]).status == Status.rangeNotSatisfiable)
}

test "a Range header that cannot be honoured is ignored" {
  val h = served(files(tree()))
  loop (bad in ["bytes=5-2", "bytes=0-1,4-5", "items=0-1", "bytes=", "bytes=a-b", "bytes=--3", "0-4"]) {
    val r = fetch(h, "/static/a.txt", ["Range": bad])
    expect(r.status == Status.ok)
    expect(text(r) == "0123456789")
  }
}

test "If-Range keeps the range only for the same date" {
  val h = served(files(tree()))
  val first = fetch(h, "/static/a.txt")
  val date = header(first, "last-modified")
  expect(fetch(h, "/static/a.txt", ["Range": "bytes=0-1", "If-Range": date]).status == Status.partialContent)
  expect(fetch(h, "/static/a.txt", ["Range": "bytes=0-1", "If-Range": "Thu, 01 Jan 1970 00:00:00 GMT"]).status == Status.ok)
  // an entity tag needs a strong match, and ours is weak
  expect(fetch(h, "/static/a.txt", ["Range": "bytes=0-1", "If-Range": header(first, "etag")]).status == Status.ok)
}

test "maxAge and immutable set Cache-Control" {
  val root = tree()
  expect(header(fetch(served(files(root, maxAge: Duration.hours(1))), "/static/a.txt"), "cache-control") == "max-age=3600")
  expect(header(fetch(served(files(root, maxAge: Duration.days(365), immutable: true)), "/static/a.txt"), "cache-control") == "max-age=31536000, immutable")
  expect(header(fetch(served(files(root, maxAge: Duration.seconds(0))), "/static/a.txt"), "cache-control") == "max-age=0")
}

test "a Cache-Control that contradicts itself is refused where it is written" {
  expectPanics(() => files(".", immutable: true))
  expectPanics(() => files(".", maxAge: Duration.seconds(-1)))
}

test "a directory: index, redirect to the slash, and the switch for it" {
  val root = tree()
  val h = served(files(root))
  val slash = fetch(h, "/static/docs/")
  expect(slash.status == Status.ok && text(slash) == "<h1>docs</h1>")
  expect(header(slash, "content-type") == "text/html; charset=utf-8")

  val r = fetch(h, "/static/docs")
  expect(r.status == Status.permanentRedirect)
  expect(header(r, "location") == "/static/docs/")
  expect(header(fetch(h, "/static/docs?v=1&w=%20"), "location") == "/static/docs/?v=1&w=%20")

  // no slash, no redirect: the index is served where it was asked
  val plain = fetch(served(files(root, redirect: false)), "/static/docs")
  expect(plain.status == Status.ok && text(plain) == "<h1>docs</h1>")

  // a directory with none of the index files, and a different index list
  expect(fetch(h, "/static/bare/").status == Status.notFound)
  expect(fetch(served(files(root, index: ["default.htm", "index.html"])), "/static/docs/").status == Status.ok)
  expect(fetch(served(files(root, index: ["default.htm"])), "/static/docs/").status == Status.notFound)
}

test "the redirect never leaves the site, whatever slashes the request has" {
  val h = served(files(tree()))
  // `//docs` names the host `docs` to a browser
  val r = fetch(h, "//docs")
  expect(r.status == Status.permanentRedirect)
  expect(header(r, "location") == "/docs/")
}

test "a path segment starting with a dot is a 404 unless dotfiles: true" {
  val root = tree()
  val h = served(files(root))
  expect(fetch(h, "/static/.env").status == Status.notFound)
  expect(fetch(h, "/static/.well-known/x.txt").status == Status.notFound)
  val open = served(files(root, dotfiles: true))
  val env = fetch(open, "/static/.env")
  expect(env.status == Status.ok)
  expect(env.body.len() == 6)
  expect(text(env) == "secret")
  expect(text(fetch(open, "/static/.well-known/x.txt")) == "ok")
}

test "only GET and HEAD are served, and a traversal is refused" {
  val h = served(files(tree()))
  val post = call(h, Method.post, "/all/a.txt", body: "x")
  expect(post.status == Status.methodNotAllowed)
  expect(header(post, "allow") == "GET, HEAD")
  val head = call(h, Method.head, "/static/a.txt")
  expect(head.status == Status.ok && head.body.isEmpty())
  expect(fetch(h, "/static/../x").status == Status.forbidden)
}

test "fs.stat reports size, kind and a recent write time" {
  val root = tree()
  val file = try fs.stat(path.join(root, "a.txt"))
  expect(file.size == 10 && file.isFile() && !file.isDir)
  val now = time.now().toSeconds()
  expect(file.modified.toSeconds() > now - 3600 && file.modified.toSeconds() < now + 3600)
  expect(try fs.stat(root).isDir)
  expect(when (fs.stat(path.join(root, "nope"))) {
    is Err(_) => true
    is Ok(_)  => false
  })
}
