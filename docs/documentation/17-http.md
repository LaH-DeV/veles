# 17. An HTTP server

`http` is an HTTP/1.1 server written in Veles on top of `net`
([chapter 16](16-networking.md)): a router, request and response values,
static files, keep-alive, timeouts. A handler is a plain function from
`Request` to `Response` that may throw, and the module decides what a
thrown error means on the wire. `examples/httpd` is a complete program
built on it; this chapter is the parts.

## Routes and handlers

```veles
use http, io, net

fun request(port: i64, target: string): string throws IoError | net.TooLong {
  with conn = try net.connect("127.0.0.1", port)
  try conn.writeText("GET $target HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
  val status = try conn.readLine(max: 8192) ?: ""
  loop {
    val line = try conn.readLine(max: 8192) ?: break   // skip the headers
    if (line.isEmpty()) break
  }
  val body = try conn.read()
  "$status | ${body.decodeUtf8() ?: "?"}"
}

fun main() throws IoError | net.TooLong {
  val app = http.Router()
  app.get("/", req => http.Response.text("hello ${req.query.get("name") ?: "world"}"))
  app.get("/users/{id}", req => http.Response.json("{\"id\": \"${req.param("id")}\"}"))
  app.get("/secret", req => throw http.Fail(status: http.Status.forbidden, text: "not for you"))
  with listener = try net.listen()
  with server = async http.serve(listener, app.handler(), log: false)
  loop (t in ["/?name=veles", "/users/42", "/secret", "/nothing"]) {
    io.println(try request(listener.port(), t))
  }
}
```

Output:
```text
HTTP/1.1 200 OK | hello veles
HTTP/1.1 200 OK | {"id": "42"}
HTTP/1.1 403 Forbidden | not for you
HTTP/1.1 404 Not Found | Not Found
```

- `http.Router()` collects routes; `get`/`post`/`put`/`delete`/`any` take
  a pattern and a handler. `{id}` captures one path segment into
  `req.param("id")`; a final `*` captures the rest under `req.param("*")`.
  A path that matches a pattern with another method answers 405 with an
  `Allow` header naming the methods that would work; no match is 404.
  `HEAD` is answered by the `GET` route with the body left off the wire
  (the `content-length` is still the `GET`'s), and `OPTIONS` answers 204
  with the same `Allow` — both only where you did not register the method
  yourself. A 1xx, 204 or 304 is sent without a body whatever the handler
  put in it.
- `app.handler()` freezes the routes into one `Handler`; `http.serve`
  accepts connections forever, one task per connection, until its task is
  cancelled — which is how the program above ends: `with server = async
  http.serve(…)` runs the server in the background until `main`'s block
  ends, then cancels it, then closes the listener (D100). A real server calls
  `serve` from `main` and runs until killed.
- `Request` has `method`, `path`, `query`, `headers` (lower-case names),
  `bytes()`, `text()`, `stream(max:)` and `multipart(max:)` for the body, plus cookies and forms (below); `Response`
  is built with `text`, `html`, `json`, `bytes`, `empty(status)`,
  `redirect`, and adjusted with `withHeader` and `withCookie`.
- Statuses, methods and header names are named values, not numbers and
  strings: `http.Status.notFound`, `http.Method.post`,
  `http.Header.cacheControl`. Each is an open set — `http.Status(code: 418)`
  and `http.Method(name: "PROPFIND")` are as good as the constants — and a
  `Status` prints as its status line does, `404 Not Found`, with
  `code`, `reason()` and `isSuccess()`/`isClientError()`/… to ask about it.
  A number written where a `Status` is wanted is an error whose fix names
  the constant: `status: 201` becomes `status: http.Status.created`.

## Testing a handler

A `Handler` is a function, so a test does not need a socket.
`http.call(handler, method, target, body:, headers:)` builds the request
the way `serve` would — the target split into `path` and `query` and
percent-decoded, header names in any case — runs the handler behind the
same panic boundary, and returns what the client would receive:

```veles
use http, io

fun main() {
  val app = http.Router()
  app.get("/notes/{id}", req => http.Response.text("note ${req.param("id")}"))
  app.post("/notes", req => http.Response.text("saved ${try req.text()}", status: http.Status.created))
  app.get("/crash", _ => panic("a bug"))
  val h = app.handler()
  val get = http.Method.get
  loop ((method, target) in [(get, "/notes/7"), (http.Method.head, "/notes/7"), (http.Method.delete, "/notes/7"), (get, "/crash")]) {
    show("$method $target", http.call(h, method, target))
  }
  show("POST /notes", http.call(h, http.Method.post, "/notes", body: "buy milk", headers: ["Content-Type": "text/plain"]))
}

fun show(what: string, resp: http.Response) {
  val allow = resp.headers.get(http.Header.allow)
  val text = resp.body.decodeUtf8() ?: "?"
  io.println("$what → ${resp.status}" + (if (allow == null) "" else " [allow: $allow]") + (if (text.isEmpty()) "" else " $text"))
}
```

Output (the panic's line goes to standard error):
```text
GET /notes/7 → 200 OK note 7
HEAD /notes/7 → 200 OK
DELETE /notes/7 → 405 Method Not Allowed [allow: GET, HEAD, OPTIONS] method not allowed
GET /crash → 500 Internal Server Error internal server error
POST /notes → 201 Created saved buy milk
```

Middleware is part of the handler `app.handler()` returns, so it runs
under `call` too.

## What an error means

A handler's type as the router stores it cannot throw — it is
`sendable fun(Request): Response suspends`, the same for every route — but
the function you *register* may throw anything. The router adapts it with
one rule:

- `http.Fail(status, text)` — "answer with this status". `http.notFound()`,
  `http.badRequest()`, `http.forbidden()` build the common ones.
- any other error is a **500**, and its `message()` goes to the log. It is
  never silent, and it never takes the server down.

So `try` is free inside a handler, and the status is decided where the
error is understood. `?!` ([chapter 7](07-errors.md)) is the usual way to
say which status a failure deserves:

```veles
// fragment
app.get("/users/{id}", req => {
  val id   = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
  val user = try store.find(id) ?! http.notFound("no user $id")
  http.Response.json(user.toJson())
})
```

A **panic** in a handler — an index out of range, a `null` where none was
expected — is caught at the request boundary: the handler runs in a task
of its own (a `gather`), the panic becomes a 500 and a log line, and the
connection and the server carry on. Whatever the handler had open in a
`with` was closed on the way out ([chapter 13](13-memory-and-ffi.md)).

## Shared state

Handlers run in connection tasks, so anything they share must cross a
task boundary: a `Mutex` around the mutable state, captured as a `val`:

```veles
// fragment
val hits = Mutex(value: 0)
app.get("/hits", req => http.Response.text("${hits.withLock(n => { *n += 1; *n })}"))
```

The compiler refuses a captured `var` or a bare `MutableMap` at the
registration — the handler would not be a sendable function
([chapter 12](12-concurrency.md)).

## Static files

`http.files(dir)` serves the files under a directory for a route ending in
`*`. The content type follows the extension and `..` is refused. It also
does what a browser or a cache expects of a file it may keep (D96):

```veles
use fs, http, io, os, path

fun main() throws IoError {
  val root = path.join(os.tempDir(), "veles-doc-static")
  try fs.mkdir(path.join(root, "docs"))
  try fs.writeFile(path.join(root, "hello.txt"), "0123456789")
  try fs.writeFile(path.join(root, "docs", "index.html"), "<h1>docs</h1>")

  val app = http.Router()
  app.get("/static/*", http.files(root))
  val h = app.handler()
  val get = http.Method.get

  val first = http.call(h, get, "/static/hello.txt")
  io.println("${first.status.code} ${first.headers.get("cache-control") ?: ""}")

  // the client keeps the copy and asks whether it is still good
  val tag = first.headers.get("etag") ?: ""
  val again = http.call(h, get, "/static/hello.txt", headers: ["If-None-Match": tag])
  io.println("${again.status.code} with ${again.body.len()} bytes")

  // and can fetch a piece of it
  val part = http.call(h, get, "/static/hello.txt", headers: ["Range": "bytes=2-4"])
  io.println("${part.status.code} ${part.headers.get("content-range") ?: ""} ${part.body.decodeUtf8() ?: ""}")

  val dir = http.call(h, get, "/static/docs")
  io.println("${dir.status.code} ${dir.headers.get("location") ?: ""}")
  val hidden = http.call(h, get, "/static/.env")
  io.println("${hidden.status.code}")
}
```

Output:
```text
200 no-cache
304 with 0 bytes
206 bytes 2-4/10 234
308 /static/docs/
404
```

- **Validators.** Every response carries `last-modified` and a weak `etag`
  made of the file's size and write time. `If-None-Match` and
  `If-Modified-Since` are answered with `304` and no body, and `If-Match` and
  `If-Unmodified-Since` with `412`; all of it is decided from `fs.stat`
  alone, so a `304` never reads the file. `etag: false` or
  `lastModified: false` turns one off, with its conditions.
- **Ranges.** One `Range: bytes=…` is a `206` with `content-range` (video and
  audio seek with it, downloads resume); a range that starts past the end is a
  `416` naming the size; a header it cannot honour — several ranges, another
  unit, nonsense — is ignored and the whole file is sent, as RFC 9110 allows.
  `If-Range` keeps the range only when its date is the file's. The file is
  streamed from disk with its length announced, so a range is a seek and a
  large file is never held in memory.
- **`Cache-Control`.** `no-cache` by default: the client may keep a copy but
  asks before using it, which the `304` makes cheap. To let it use the copy
  unasked, say for how long: `maxAge: Duration.hours(1)`; for a fingerprinted
  name that never changes, `maxAge: Duration.days(365), immutable: true`. A
  negative `maxAge`, or `immutable` without one, panics where `files` is
  called.
- **Directories.** A directory serves the first of `index` (default
  `["index.html"]`) it holds, and is a `404` when it holds none; there is no
  listing. `/docs` without the slash is a `308` to `/docs/` (keeping the
  query), so the links inside the page resolve; `redirect: false` serves the
  index at `/docs` itself instead.
- **Dotfiles.** A path with a segment that starts with `.` (`.env`,
  `.git/config`) is a `404` — not a `403`, which would confirm the file —
  unless `dotfiles: true`; serve `/.well-known/` with that, or by mounting the
  directory on its own.
- **Methods.** Only `GET` and `HEAD`; another method is a `405` with `allow`.

```veles
// fragment
app.get("/assets/*", http.files("./public", maxAge: Duration.days(365), immutable: true))
app.get("/docs/*", http.files("./site", redirect: false, index: ["index.html", "index.htm"]))
app.get("/", req => http.Response.redirect("/assets/"))
```

## Reading a request body

A request's body is not read with its head. The handler asks for it, and it
is read then, off the wire, under a ceiling the handler can see (D97):

```veles
use http, io

fun main() {
  val app = http.Router()
  app.post("/echo", req => http.Response.text(try req.text()))
  app.post("/count", req => {
    val body = req.stream(max: 1000)
    var bytes = 0
    var pieces = 0
    loop {
      val piece = try body.read(4)
      if (piece.isEmpty()) break
      bytes += piece.len()
      pieces += 1
    }
    http.Response.text("$bytes bytes in $pieces pieces")
  })
  val h = app.handler()
  val post = http.Method.post
  io.println(http.call(h, post, "/echo", body: "hello").body.decodeUtf8() ?: "")
  io.println(http.call(h, post, "/count", body: "0123456789").body.decodeUtf8() ?: "")
  io.println("${http.call(h, post, "/count", body: "x".repeat(2000)).status}")
}
```

Output:
```text
hello
10 bytes in 3 pieces
413 Content Too Large
```

- **Into memory.** `req.bytes()`, `req.text()` and the form readers
  (`req.form<T>()`, `formValue`) collect the whole body, at most
  `Limits.bodyBytes` (1 MiB by default) — or the `max:` you pass to `bytes`
  and `text`. A longer body is a `413`, before its first byte when its
  `Content-Length` already says so. What was read is kept, so these can be
  called again, in any mix, and see the same body.
- **As it arrives.** `req.stream(max: n)` gives a `Body` whose `read(max)`
  hands out what has come, a piece at a time, so a body larger than memory
  can go to a file or a parser as it comes. `max` has no default, on purpose:
  the sender decides how much arrives, and a handler that never says would
  take whatever a stranger sends. `body.length()` is the `Content-Length`,
  or `null` when the body is chunked. After `bytes()` a stream is at its end.
- **Chunked.** `Transfer-Encoding: chunked` is decoded for you, whatever the
  pieces and chunk extensions; trailer fields are read and dropped. A request
  that says both `Transfer-Encoding` and `Content-Length`, an unknown chunk
  size, or a body that stops inside a chunk is a `400`; a coding other than
  `chunked` is a `501`; the ceiling applies to the decoded size, and a chunk
  that could not fit under it is refused before its data.
- **`Expect: 100-continue`.** A client that sends it waits for the server
  before sending a body. The `100 Continue` goes out when the handler first
  reads, so a handler that answers `401` or `413` without reading never
  invites the upload — and the connection closes after such an answer, since
  the body was never sent. Any other expectation is a `417`.
- **What the handler did not read.** After the response is sent the rest of
  the body is read and dropped (up to 64 KiB), so the next request on a
  kept-alive connection starts where it should; a body longer than that, or
  one whose read failed, closes the connection instead.
- **Time.** `bodyTimeout` is the longest silence between two reads, not the
  time the whole upload may take: a large body that keeps arriving is never
  cut off, and one that stalls is a `408`.

## Streaming a response

`http.Response.stream(type, producer)` sends a body as the producer writes
it, for what is too large to hold, is not ready yet, or never ends:

```veles
use http, io

fun main() {
  val app = http.Router()
  app.get("/count", req => http.Response.stream(http.MediaType.text, out => {
    loop (i in 1..3) {
      try out.writeText("$i\n")
    }
  }))
  val r = http.call(app.handler(), http.Method.get, "/count")
  io.println("${r.status} ${(r.body.decodeUtf8() ?: "").replace("\n", ",")}")
}
```

Output:
```text
200 OK 1,2,3,
```

The server sends the head, then calls the producer with a `BodyWriter`; each
`write` goes out at once as a chunk (`transfer-encoding: chunked`; an
HTTP/1.0 client, which has no chunks, gets the bare bytes and then the end of
the connection). With `length: n` the body is announced with `content-length`
and sent unframed — what a download needs for a progress bar and for resuming
— and a producer that writes more or fewer than `n` bytes fails.

A failed `write` means the client went away, and the producer stops with that
error. A panic in the producer, or an error it lets escape, can no longer be a
`500` — the head has gone — so the connection is closed without the final
chunk and the client sees a truncated body, never one that looks complete. A
`HEAD` request gets the head and no call. The producer runs after the handler
has returned, so `http.timeout` does not bound it, and it must be a sendable
function: capture values and handles, not a `var`. `http.call` collects a
streamed body whole (a stream that never ends never returns from it), and
`http.files` streams from disk with a declared length, so a range is a seek and
a large file is never in memory.

## Uploads: multipart forms

A form with files is `multipart/form-data`. `req.multipart(max:)` reads it
part by part off the body, so an upload goes to disk as it arrives and only
the part being read is ever held:

```veles
use fs, http, io, os, path

fun main() throws IoError {
  val dir = path.join(os.tempDir(), "veles-doc-upload")
  try fs.mkdir(dir)
  val app = http.Router()
  app.post("/upload", req => {
    val form = try req.multipart(max: 1024 * 1024)
    val out = StringBuilder()
    loop {
      val part = try form.next() ?: break
      if (val original = part.filename) {
        val n = try part.saveTo(path.join(dir, "saved.txt"), max: 64 * 1024)
        out.append("${part.name}: $original, $n bytes\n")
      } else {
        out.append("${part.name} = ${try part.text(max: 256)}\n")
      }
    }
    http.Response.text(out.toString())
  })
  val body = "--b\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nholiday\r\n--b\r\nContent-Disposition: form-data; name=\"photo\"; filename=\"beach.txt\"\r\nContent-Type: text/plain\r\n\r\nsand and sea\r\n--b--\r\n"
  val resp = http.call(app.handler(), http.Method.post, "/upload", body: body, headers: ["Content-Type": "multipart/form-data; boundary=b"])
  io.println((resp.body.decodeUtf8() ?: "").trim())
  io.println(try fs.readFile(path.join(dir, "saved.txt")))
  try fs.remove(path.join(dir, "saved.txt"))
  try fs.remove(dir)
}
```

Output:
```text
title = holiday
photo: beach.txt, 12 bytes
sand and sea
```

- `form.next()` returns the next `Part`, or `null` after the last. A part is
  read from where the last one left off; one left unread is skipped.
- A `Part` has `name`, `filename` (only for a file), `contentType` and
  `headers`, and is read with `read(max)` (a piece), `bytes(max:)`,
  `text(max:)` or `saveTo(path, max:)`. The `max` of the part functions has
  no default, and a longer part is a `413`; `saveTo` removes the file it was
  writing when that happens.
- **The file name is the client's word.** `part.filename` is what the browser
  sent — `..\..\x`, `C:\dir\a.png`, anything — so it never names a path:
  `saveTo` takes the path, built from something you chose.
- `req.multipart(max:)` bounds the whole body like `stream` does, and
  `maxParts:` (100) the number of parts. Another content type is a `415`; a
  missing or overlong boundary, a part without a name, headers that are too
  large, or a body that ends short is a `400`.
- Data that looks like the start of a boundary is data: only the full
  boundary line ends a part.

## Cookies

`req.cookie("name")` reads one cookie (`string?`), `req.cookies()` all of
them; `resp.withCookie(cookie)` sets one, and a response may set as many as
it likes, each as its own `Set-Cookie` line:

```veles
use http, io

fun main() {
  val app = http.Router()
  app.get("/visit", req => {
    val n = (req.cookie("visits") ?: "0").toInt() ?: 0
    http.Response.text("visit ${n + 1}").withCookie(http.Cookie(name: "visits", value: "${n + 1}", maxAge: Duration.days(30)))
  })
  val h = app.handler()
  var jar = ""
  loop (_ in 0..<3) {
    val resp = http.call(h, http.Method.get, "/visit", headers: ["Cookie": jar])
    val c = resp.cookies.at(0) ?: panic("no cookie")
    jar = "${c.name}=${c.value}"
    io.println("${resp.body.decodeUtf8() ?: ""} - cookie ${c.name}=${c.value}, kept ${c.maxAge?.toDays()} days")
  }
}
```

Output:
```text
visit 1 - cookie visits=1, kept 30 days
visit 2 - cookie visits=2, kept 30 days
visit 3 - cookie visits=3, kept 30 days
```

On the wire that is `set-cookie: visits=1; Max-Age=2592000; Path=/;
HttpOnly; SameSite=Lax`. The defaults of `http.Cookie` are the safe ones:
the whole site (`path: "/"`), hidden from scripts (`httpOnly: true`), not
sent cross-site (`sameSite: http.SameSite.Lax`), and a session cookie until
you give it a `maxAge`. `secure` is off, so a program served over plain
http while you write it still works — switch it on for anything deployed.

- **The value is encoded for you.** Every byte but letters, digits and
  `-._~` is written as `%XX` and decoded again by `req.cookie`, so any
  string round-trips and none can end the header early: `"dark mode; yes"`
  goes out as `dark%20mode%3B%20yes`.
- **What cannot be encoded is checked**, and a bad one is a panic at the
  line that wrote it (a handler's panic is a 500 and a log line): a name
  that is not a token (`"no spaces"`), a path that does not start with `/`,
  a domain that is not a host name, `sameSite: http.SameSite.None` without
  `secure: true` (browsers drop such a cookie), a negative `maxAge`, and the
  browsers' `__Host-` and `__Secure-` prefixes used without what they
  require.
- **Forgetting a cookie** is `resp.withoutCookie("sid")`, a `Max-Age=0`
  cookie; give it the `path` and `domain` it was set with.
- **Reading:** a quoted value is unquoted, and when the browser sends a name
  twice the first counts. The server keeps one header of each name, so a
  client that sent two `Cookie` lines has the last read; HTTP/1.1 clients
  send one.

`examples/session` is a login built on these, over a real socket, showing
the lines a browser would see. Signed or encrypted session cookies are not
here yet.

## Forms and query strings

`req.form<T>()` reads an `application/x-www-form-urlencoded` body into a
struct, the way `json.decode` reads a document, and `req.query<T>()` does
the same with the query string. The struct writes `implement Decodable`
(chapter 18) and nothing else:

```veles
use http, io

struct Signup {
  name:       string
  age:        i64
  newsletter: bool = false
  tags:       List<string> = []
  implement Decodable
}

fun main() {
  val app = http.Router()
  app.post("/signup", req => {
    val s = try req.form<Signup>()
    http.Response.text("${s.name} is ${s.age}, newsletter ${s.newsletter}, tags ${s.tags}")
  })
  val h = app.handler()
  val form = ["Content-Type": "application/x-www-form-urlencoded"]
  loop (body in ["name=Ada+L&age=36&newsletter=on&tags=a&tags=b", "name=Ada&age=old"]) {
    val resp = http.call(h, http.Method.post, "/signup", body: body, headers: form)
    io.println("${resp.status} ${resp.body.decodeUtf8() ?: ""}")
  }
}
```

Output:
```text
200 OK Ada L is 36, newsletter true, tags [a, b]
400 Bad Request invalid form:
age: expected an integer, found "old"
```

- Text becomes what the field asks for: integers and numbers are parsed,
  booleans are `true`/`false`, `on`/`off`, `yes`/`no` or `1`/`0` (a checkbox
  sends `on` when ticked and nothing at all when not, so a `bool` with a
  default of `false` is the checkbox's field), an enum is read by name.
- A `List` field takes every value of its name (`tags=a&tags=b`); a single
  value is a list of one, and an empty one an empty list. An optional field
  is `null` when its value is empty.
- A bad value or a missing required field is a **400** that lists every
  problem, one per line, each at its field's name — `invalid form:` for a
  body, `invalid query:` for a query string. Another content type is a
  **415**. The body is bounded by `Limits.bodyBytes` like any other.
- A form is flat: a nested struct, or a list of lists, is a problem.
  `req.form<T>(keys: KeyStyle.SnakeCase)` reads `first_name` into
  `firstName`.
- `req.query` (the field) is still the last value of each name;
  `req.query<T>()` (the method) is the typed reader, and both exist.

The low-level readers are there when a struct is more than you need:
`req.formValue("a")` (first value, `string?`), `req.formValues("a")` (all of
them), and `req.formFields()` / `req.queryFields()`, an `http.Fields` with
`get`, `all` and `names`, repeats and order kept. Uploaded files
(`multipart/form-data`) are not here yet; they need streaming bodies.

## Middleware

A middleware takes the handler that comes after it and returns one that
does something before, after, or instead. `app.wrap(m)` puts it around
everything the router answers, its own 404s and 405s included — which is
what an access log wants. The first `wrap` ends up outermost: it sees the
request first and the response last.

(The verb is `wrap` and not `use` because `use` is the keyword that
imports a module.)

```veles
use http, io, net

fun request(port: i64, target: string, extra: string): string throws IoError | net.TooLong {
  with conn = try net.connect("127.0.0.1", port)
  try conn.writeText("GET $target HTTP/1.1\r\nHost: x\r\nConnection: close\r\n" + extra + "\r\n")
  val status = try conn.readLine(max: 8192) ?: ""
  var id = "-"
  var by = "-"
  loop {
    val line = try conn.readLine(max: 8192) ?: break
    if (line.isEmpty()) break
    val low = line.toLower()
    if (low.startsWith("x-request-id:")) id = (line.substring(13, line.len()) ?: "").trim()
    if (low.startsWith("x-served-by:")) by = (line.substring(12, line.len()) ?: "").trim()
  }
  "$status id=$id by=$by"
}

fun main() throws IoError | net.TooLong {
  val app = http.Router()
  app.wrap(http.requestId())                                        // outermost
  app.wrap(http.timeout(Duration.millis(50)))
  app.wrap(next => req => next(req).withHeader("x-served-by", "veles"))
  app.get("/fast", req => http.Response.text("quick"))
  app.get("/slow", req => {
    await sleep(Duration.millis(500))
    http.Response.text("eventually")
  })
  with listener = try net.listen()
  with server = async http.serve(listener, app.handler(), log: false)
  io.println(try request(listener.port(), "/fast", "X-Request-Id: abc123\r\n"))
  io.println(try request(listener.port(), "/slow", "X-Request-Id: def456\r\n"))
}
```

Output:
```text
HTTP/1.1 200 OK id=abc123 by=veles
HTTP/1.1 503 Service Unavailable id=def456 by=-
```

The second line is the ordering made visible. `timeout` sits outside the
header middleware, so when the slow handler is cancelled the 503 it
answers with never passes through `x-served-by`; `requestId` is outside
`timeout`, so the id is on the response either way. Put a middleware where
you want its effect to survive.

What the module gives you:

| call | what it does |
|---|---|
| `http.logging()` | one `log` line per request ([chapter 21](21-logging.md)): peer, method, path, status, duration, and the request id when `requestId()` wraps it |
| `http.requestId()` | the client's `X-Request-Id`, or a fresh 16 hex digits, on the request for the handlers and on the response for the client; also the log field `id` for every line logged while the request is handled |
| `http.timeout(d)` | 503 when the handler takes longer; it runs in its own task and is cancelled, so its `with`s close |
| `http.cors(origins: [...])` | lets browser pages from other origins in ([below](#other-origins-and-who-may-call)) |
| `http.guard(check)`, `http.basicAuth(realm, verify)`, `http.bearer(verify)` | answer a request before the handlers do ([below](#other-origins-and-who-may-call)) |

`serve` logs the same request line itself (through `log`, so it follows
`VELES_LOG`), so pass `log: false` when you wrap `logging()`. A **recovery** middleware is not among
these because it would have nothing to do: a panic in a handler is
already caught at the request boundary and answered with a 500.

A middleware runs in a task, so it is a `sendable` function: it may
capture `val`s of Sendable types and nothing mutable. There are no
task-local values yet, so the way to hand something to the handlers
behind you is `req.withHeader(...)`, as `requestId` does.

## Other origins, and who may call

A browser lets a page read a response from another origin only when the
server says so. It says so with `Access-Control-*` headers, and for
anything beyond a plain GET or form post the browser first sends a
**preflight**, an `OPTIONS` asking what is allowed. `http.cors` answers both:

```veles
// fragment
app.wrap(http.cors(origins: ["https://app.example.com", "https://*.example.org"], headers: ["authorization", "content-type"], credentials: true, maxAge: Duration.hours(1)))
```

Nothing is allowed by default — `http.cors()` alone changes no response — and
every origin you list is spelled out: exactly (`https://app.example.com`,
scheme and port included), `"*"` for all of them, or `https://*.example.com` for
any subdomain, matched on a dot, so `https://evilexample.com` is not one. A
request without an `Origin` header is not cross-origin and passes untouched;
so does one from an origin you did not list, which the browser then refuses
to show to the page. A preflight from a listed origin is answered by the
middleware (204) and never reaches the router; its
`Access-Control-Allow-Headers` is your `headers`, or the headers the browser
asked about when you gave none. Responses that depend on the origin say
`Vary: Origin`, so a cache keeps them apart. The combinations browsers refuse
are refused earlier: `"*"` with `credentials: true`, a pattern that is not one
of the three forms, and a negative `maxAge` panic at the line that wrote them.

Who may call at all is a `guard`: a function that sees the request first and
returns `null` to let it on or a `Response` to answer instead (a thrown `Fail`
is an answer too):

```veles
// fragment
app.wrap(http.guard(req => if (req.path.startsWith("/admin") && req.header("x-admin") == null) http.Response.text("no", status: http.Status.forbidden) else null))
```

The two schemes nearly every API uses are built on it. `basicAuth(realm,
verify)` wants `Authorization: Basic …` and calls `verify(user, password)`;
`bearer(verify)` wants `Authorization: Bearer <token>` and calls `verify(token)`.
A missing or refused credential is a 401 with the `WWW-Authenticate` challenge;
a request that passes `basicAuth` carries the user in `x-remote-user`
(`http.Header.remoteUser`), replacing whatever the client sent under that
name.

```veles
// fragment
app.wrap(http.basicAuth("admin", (user, pass) => user == "root" && crypto.equalBytes(pass.bytes(), secret.bytes())))
app.wrap(http.bearer(token => crypto.equalBytes(token.bytes(), apiKey.bytes())))
```

Compare a secret with `crypto.equalBytes`, not `==`: the time `==` takes says
how many leading bytes were right. Basic sends the password with every
request, so serve it over TLS. A guard wraps the whole router, its 404s
included; a rule for some routes tests the path itself until route groups
exist. Wrap `cors` outside the auth middleware, so a preflight — which
carries no credentials — is answered before anything asks for them.

## Connections and time

`serve` keeps a connection open for further requests (HTTP/1.1
keep-alive) until the client closes it, sends `Connection: close` (an
HTTP/1.0 client, unless it sends `Connection: keep-alive`), the handler's
response says `connection: close`, or the client says
nothing for `limits.idleTimeout` (15 s by default) — a
`withTimeout` around each read, so a silent client costs one parked task
and nothing else. Requests are logged to standard error as
`peer METHOD path status ms` unless `log: false`.

At most `limits.connections` (10 000 by default; `0` for no limit) are served
at once. A full server stops accepting: the next clients wait in the operating
system's backlog, which is a queue that costs no task and no memory here, and
are served as places free up. A kept-alive connection holds its place until it
closes or reaches `idleTimeout`.

## Stopping gracefully

A server in production is stopped by a signal — Ctrl+C, `kill`,
systemd, Docker — and should not drop the requests it is answering.
Give `serve` a `stop` condition:

```veles
// fragment
fun main() throws IoError {
  with (listener = try net.listen(host: "0.0.0.0", port: 8080)) {
    http.serve(listener, app.handler(), stop: () => os.shutdownSignal(), grace: Duration.seconds(10))
  }
  io.println("bye")
}
```

When `stop` returns, `serve`:

1. stops accepting connections;
2. closes kept-alive connections that are waiting for their next request;
3. lets requests in flight finish, for up to `grace` — their responses
   carry `connection: close`;
4. cancels whatever is still running after that: the handler unwinds at
   its next suspension point and every `with` it is inside closes;
5. returns.

`os.shutdownSignal()` suspends until SIGINT or SIGTERM (on Windows:
Ctrl+C, Ctrl+Break, closing the console, log-off, shutdown) and returns
which one. It intercepts nothing until it is first called, and each call
takes one signal: a second Ctrl+C while the shutdown runs has the
default effect and ends the process, so a shutdown that hangs can still
be stopped. `stop` is ordinary code, so a test stops a server with a
channel (`stop: () => { val _ = await quit.recv() }`), and a program with
two servers waits once and stops both. `examples/shutdown` runs every
case, using `os.raiseSignal` in place of a real Ctrl+C.

## What a request may cost

A listener is open to strangers, and nothing about a request is
trustworthy until it has been measured. `http.Limits` is the measure —
every field a ceiling, every default chosen to be safe rather than
generous — and `serve` takes it:

```veles
// fragment
http.serve(listener, app, limits: http.Limits(bodyBytes: 8 * 1024 * 1024))
```

| field | default | what it bounds |
|---|---|---|
| `requestLineBytes` | 8192 | `GET /some/path HTTP/1.1` — mostly the target |
| `headerLineBytes` | 8192 | one header line |
| `headerCount` | 100 | how many header lines |
| `headerBytes` | 65536 | every header line added up |
| `bodyBytes` | 1048576 | what `bytes()`, `text()` and `form()` collect, whatever `Content-Length` claims |
| `headerTimeout` | `Duration.seconds(10)` | request line to the blank line, on one clock |
| `bodyTimeout` | `Duration.seconds(30)` | the longest silence while the body is read |
| `idleTimeout` | `Duration.seconds(15)` | silence between requests on a kept-alive connection |
| `connections` | 10000 | connections served at once (`0`: no limit); a full server stops accepting |

A request that reaches a byte ceiling is answered and the connection
closed: `414` for the request line, `431` for the headers, `413` for the
body. A time ceiling answers `408` — except `idleTimeout`, which is the
ordinary end of a kept-alive connection and closes without a word. None
of this reaches a handler: the refusal happens while the request is being
read, which is the point — a body is never assembled in order to discover
it was too big, and `headerCount` counts *lines*, so repeating one header
name a thousand times costs what it should.

The head is read strictly, the way RFC 9112 asks of a server, and a
request is refused with `400` wherever two readers could disagree about
where a header or the body ends — which matters as soon as a proxy stands
in front of the server and reads the same bytes first:

- a space between a header name and its colon (`Content-Length : 5`), a
  name that is not a token, or a line that starts with whitespace (the
  obsolete folded form);
- a CR or NUL inside a header value, or a line that is not UTF-8;
- a `Content-Length` that is not plain digits (`+5`, `0x10`), or two that
  disagree;
- an HTTP/1.1 request without exactly one `Host`, and a version that is
  not `HTTP/` digit `.` digit (a major version other than 1 is `505`);
- both `Transfer-Encoding` and `Content-Length`, or a chunked coding in an
  HTTP/1.0 request (a coding other than `chunked` is `501`).

On the way out, the server owns the framing: a handler's
`content-length`, `transfer-encoding` and `connection` headers are not
sent (the last is still obeyed: `connection: close` closes), and a line
break or NUL in a header value is sent as a space. A value often comes
from the request — a redirect's `location`, an echoed path, which can
decode to `\r\n` — and a line break there would let the client write
headers of its own into your response.

`examples/fuzz` throws generated and mutated requests at a server in the
same process and checks every answer against these rules.

Two ceilings are worth setting for your own server rather than taking the
default. `bodyBytes` should be as small as the largest thing you accept —
a JSON API that takes a line of text wants kilobytes, not a megabyte. And
a handler that takes a larger upload streams it — `req.stream(max:)`,
`req.multipart(max:)` — with a ceiling of its own, rather than raising this
for everything.

Below `http`, `net.Conn.readLine(max:)` is where the byte ceiling is
actually enforced; it has no unbounded form, so a program that reads
lines from a socket has to say how long a line it will hold
([chapter 16](16-networking.md)).

Not in the module yet: TLS, a client, WebSockets, and graceful shutdown on a signal. Cancelling the
`serve` task closes the listener and unwinds every connection task —
that is the shutdown, once a signal can request it.

Next: back to the [index](index.md).
