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
  with (conn = try net.connect("127.0.0.1", port)) {
    try conn.writeText("GET $target HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
    val status = try conn.readLine(max: 8192) ?: ""
    loop {
      val line = try conn.readLine(max: 8192) ?: break   // skip the headers
      if (line.isEmpty()) break
    }
    val body = try conn.read()
    "$status | ${body.decodeUtf8() ?: "?"}"
  }
}

fun main() throws IoError | net.TooLong {
  val app = http.router()
  app.get("/", req => http.Response.text("hello ${req.query.get("name") ?: "world"}"))
  app.get("/users/{id}", req => http.Response.json("{\"id\": \"${req.param("id")}\"}"))
  app.get("/secret", req => throw http.Fail(status: 403, text: "not for you"))
  with (listener = try net.listen()) {
    scope {
      val server = async http.serve(listener, app.handler(), log: false)
      loop (t in ["/?name=veles", "/users/42", "/secret", "/nothing"]) {
        io.println(try request(listener.port(), t))
      }
      server.cancel()
    }
  }
}
```

Output:
```text
HTTP/1.1 200 OK | hello veles
HTTP/1.1 200 OK | {"id": "42"}
HTTP/1.1 403 Forbidden | not for you
HTTP/1.1 404 Not Found | not found
```

- `http.router()` collects routes; `get`/`post`/`put`/`delete`/`any` take
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
  cancelled — which is how the program above ends. A real server calls
  `serve` from `main` and runs until killed.
- `Request` has `method`, `path`, `query`, `headers` (lower-case names),
  `body: List<u8>` and `text()`; `Response` is built with `text`, `html`,
  `json`, `bytes`, `empty(status)`, `redirect`, and adjusted with
  `withHeader`.

## Testing a handler

A `Handler` is a function, so a test does not need a socket.
`http.call(handler, method, target, body:, headers:)` builds the request
the way `serve` would — the target split into `path` and `query` and
percent-decoded, header names in any case — runs the handler behind the
same panic boundary, and returns what the client would receive:

```veles
use http, io

fun main() {
  val app = http.router()
  app.get("/notes/{id}", req => http.Response.text("note ${req.param("id")}"))
  app.post("/notes", req => http.Response.text("saved ${try req.text()}", status: 201))
  app.get("/crash", _ => panic("a bug"))
  val h = app.handler()
  loop ((method, target) in [("GET", "/notes/7"), ("HEAD", "/notes/7"), ("DELETE", "/notes/7"), ("GET", "/crash")]) {
    show("$method $target", http.call(h, method, target))
  }
  show("POST /notes", http.call(h, "POST", "/notes", body: "buy milk", headers: ["Content-Type": "text/plain"]))
}

fun show(what: string, resp: http.Response) {
  val allow = resp.headers.get("allow")
  val text = resp.body.decodeUtf8() ?: "?"
  io.println("$what → ${resp.status}" + (if (allow == null) "" else " [allow: $allow]") + (if (text.isEmpty()) "" else " $text"))
}
```

Output (the panic's line goes to standard error):
```text
GET /notes/7 → 200 note 7
HEAD /notes/7 → 200
DELETE /notes/7 → 405 [allow: GET, HEAD, OPTIONS] method not allowed
GET /crash → 500 internal server error
POST /notes → 201 saved buy milk
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
val hits = mutex(0)
app.get("/hits", req => http.Response.text("${hits.withLock(n => { *n += 1; *n })}"))
```

The compiler refuses a captured `var` or a bare `MutableMap` at the
registration — the handler would not be a sendable function
([chapter 12](12-concurrency.md)).

## Static files

`http.files(dir)` serves the files under a directory for a route ending in
`*`; a directory serves its `index.html`, `..` is refused, and the content
type follows the extension:

```veles
// fragment
app.get("/static/*", http.files("./public"))
app.get("/", req => http.Response.redirect("/static/"))
```

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
  with (conn = try net.connect("127.0.0.1", port)) {
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
}

fun main() throws IoError | net.TooLong {
  val app = http.router()
  app.wrap(http.requestId())                                        // outermost
  app.wrap(http.timeout(Duration.millis(50)))
  app.wrap(next => req => next(req).withHeader("x-served-by", "veles"))
  app.get("/fast", req => http.Response.text("quick"))
  app.get("/slow", req => {
    await sleep(Duration.millis(500))
    http.Response.text("eventually")
  })
  with (listener = try net.listen()) {
    scope {
      val server = async http.serve(listener, app.handler(), log: false)
      io.println(try request(listener.port(), "/fast", "X-Request-Id: abc123\r\n"))
      io.println(try request(listener.port(), "/slow", "X-Request-Id: def456\r\n"))
      server.cancel()
    }
  }
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
| `http.logging()` | one line per request on standard error: peer, method, path, status, duration, request id when there is one |
| `http.requestId()` | the client's `X-Request-Id`, or a fresh 16 hex digits, on the request for the handlers and on the response for the client |
| `http.timeout(d)` | 503 when the handler takes longer; it runs in its own task and is cancelled, so its `with`s close |

`serve` logs a plainer version of `logging()`'s line itself, so pass
`log: false` when you wrap it. A **recovery** middleware is not among
these because it would have nothing to do: a panic in a handler is
already caught at the request boundary and answered with a 500.

A middleware runs in a task, so it is a `sendable` function: it may
capture `val`s of Sendable types and nothing mutable. There are no
task-local values yet, so the way to hand something to the handlers
behind you is `req.withHeader(...)`, as `requestId` does.

## Connections and time

`serve` keeps a connection open for further requests (HTTP/1.1
keep-alive) until the client closes it, sends `Connection: close`, or says
nothing for `limits.idleTimeout` (15 s by default) — a
`withTimeout` around each read, so a silent client costs one parked task
and nothing else. Requests are logged to standard error as
`peer METHOD path status ms` unless `log: false`.

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
| `bodyBytes` | 1048576 | the body, whatever `Content-Length` claims |
| `headerTimeout` | `Duration.seconds(10)` | request line to the blank line, on one clock |
| `bodyTimeout` | `Duration.seconds(30)` | the body |
| `idleTimeout` | `Duration.seconds(15)` | silence between requests on a kept-alive connection |

A request that reaches a byte ceiling is answered and the connection
closed: `414` for the request line, `431` for the headers, `413` for the
body. A time ceiling answers `408` — except `idleTimeout`, which is the
ordinary end of a kept-alive connection and closes without a word. None
of this reaches a handler: the refusal happens while the request is being
read, which is the point — a body is never assembled in order to discover
it was too big, and `headerCount` counts *lines*, so repeating one header
name a thousand times costs what it should.

Two ceilings are worth setting for your own server rather than taking the
default. `bodyBytes` should be as small as the largest thing you accept —
a JSON API that takes a line of text wants kilobytes, not a megabyte. And
if a handler streams uploads, raise it there rather than everywhere.

Below `http`, `net.Conn.readLine(max:)` is where the byte ceiling is
actually enforced; it has no unbounded form, so a program that reads
lines from a socket has to say how long a line it will hold
([chapter 16](16-networking.md)).

Not in the module yet: TLS, chunked request bodies (answered with 501),
a client, WebSockets, and graceful shutdown on a signal. Cancelling the
`serve` task closes the listener and unwinds every connection task —
that is the shutdown, once a signal can request it.

Next: back to the [index](index.md).
