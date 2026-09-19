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

fun request(port: i64, target: string): string throws IoError {
  with (conn = try net.connect("127.0.0.1", port)) {
    try conn.writeText("GET $target HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
    val status = try conn.readLine() ?: ""
    loop {
      val line = try conn.readLine() ?: break   // skip the headers
      if (line.isEmpty()) break
    }
    val body = try conn.read()
    "$status | ${body.decodeUtf8() ?: "?"}"
  }
}

fun main() throws IoError {
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
  A path that matches a pattern with another method answers 405, no match
  404.
- `app.handler()` freezes the routes into one `Handler`; `http.serve`
  accepts connections forever, one task per connection, until its task is
  cancelled — which is how the program above ends. A real server calls
  `serve` from `main` and runs until killed.
- `Request` has `method`, `path`, `query`, `headers` (lower-case names),
  `body: List<u8>` and `text()`; `Response` is built with `text`, `html`,
  `json`, `bytes`, `empty(status)`, `redirect`, and adjusted with
  `withHeader`.

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

## Connections and time

`serve` keeps a connection open for further requests (HTTP/1.1
keep-alive) until the client closes it, sends `Connection: close`, or says
nothing for `idleTimeout` milliseconds (15 s by default) — a `withTimeout`
around each read, so a silent client costs one parked task and nothing
else. Requests are logged to standard error as
`peer METHOD path status ms` unless `log: false`.

Not in the module yet: TLS, chunked request bodies (answered with 501),
a client, WebSockets, and graceful shutdown on a signal. Cancelling the
`serve` task closes the listener and unwinds every connection task —
that is the shutdown, once a signal can request it.

Next: back to the [index](index.md).
