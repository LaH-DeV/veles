// httpd: a small web server — static files from a directory and a JSON
// notes API kept in memory. Point a browser or curl at it:
//
//   httpd public --port 8080          then open http://127.0.0.1:8080/
//   curl -d "buy milk" http://127.0.0.1:8080/api/notes
//   curl http://127.0.0.1:8080/api/notes
//
// `--check` starts the same server on a free port, plays a scripted client
// against it in the same process and prints the exchange; it is how the
// test suite runs this program.
//
//   httpd [<dir>] [--host H] [--port N] [--check]
use fs, http, io, json, net, os

error UsageError {
  message: string
}

struct Options {
  var dir:   string = "public"
  var host:  string = "127.0.0.1"
  var port:  i64 = 8080
  var check: bool = false

  static fun parse(args: List<string>): Options throws UsageError {
    var opts = Options()
    var dirGiven = false
    var i = 0
    loop (i < args.len()) {
      val arg = args.at(i)
      when (arg) {
        "--port"  => {
          opts.port = args.at(i + 1)?.toInt() ?: throw UsageError(message: "--port needs a number")
          i += 1
        }
        "--host"  => {
          opts.host = args.at(i + 1) ?: throw UsageError(message: "--host needs an address")
          i += 1
        }
        "--check" => opts.check = true
        else      => {
          if (arg.startsWith("-")) throw UsageError(message: "unknown option '$arg'")
          if (dirGiven) throw UsageError(message: "one directory at a time")
          opts.dir = arg
          dirGiven = true
        }
      }
      i += 1
    }
    opts
  }
}

// ---------------------------------------------------------------------------
// the notes API

struct Note {
  id:   i64
  text: string
  implement Codable  // the wire form, written by the compiler (D58)
}

/// What a client sends to change a note: `{"text": "..."}`.
struct NotePatch {
  text: string
  implement Codable
}

/// The store behind the API. Handlers run in connection tasks, so it lives
/// in a Mutex (D35); its fields are private, so the only way in is through
/// the methods, which keep `next` and `items` consistent.
struct Notes {
  private var next: i64 = 1
  private items:    MutableList<Note> = []

  fun all(): List<Note> = this.items.toList()

  fun find(id: i64): Note? = this.items.find(x => x.id == id)

  /// Stores a new note with the next id.
  fun add(text: string): Note {
    val created = Note(id: this.next, text)
    this.next += 1
    this.items.push(created)
    created
  }

  /// Removes the note with that id; false when there is none. (`items` is a
  /// MutableList — a handle — so this needs no `mut`: only `this`'s own
  /// fields are guarded by it, D22/D25.)
  fun remove(id: i64): bool {
    val at = this.items.indexOfFirst(x => x.id == id)
    if (at >= 0) this.items.removeAt(at)
    at >= 0
  }

  /// Replaces the text of a note; false when there is none.
  fun update(id: i64, text: string): bool {
    val at = this.items.indexOfFirst(x => x.id == id)
    if (at >= 0) this.items.set(at, Note(id, text))
    at >= 0
  }
}

/// The whole application as one handler: the routes below plus the files
/// under `dir`.
fun app(dir: string): http.Handler {
  val notes = Mutex(value: Notes())
  val router = http.Router()

  // Middleware, outermost first: every answer carries the request id the
  // client sent (or a fresh one), and no handler may run for more than a
  // second. Both wrap the 404s and 405s too.
  router.wrap(http.requestId())
  router.wrap(http.timeout(Duration.seconds(1)))

  router.get("/", req => http.Response.redirect("/static/"))
  router.get("/static/*", http.files(dir))

  router.get("/api/echo", req => http.Response.text(req.query.get("msg") ?: "(no msg)"))

  router.get("/api/notes", req => http.Response.json(try json.encode(notes.withLock(n => n.all()))))

  router.post("/api/notes", req => {
    val text = (try req.text()).trim()
    if (text.isEmpty()) throw http.badRequest("a note needs some text")
    val note = notes.withLock(n => n.add(text))
    http.Response.json(try json.encode(note), status: http.Status.created).withHeader(http.Header.location, "/api/notes/${note.id}")
  })

  router.get("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val note = try notes.withLock(n => n.find(id)) ?! http.notFound("no note $id")
    http.Response.json(try json.encode(note))
  })

  router.put("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    // a bad body answers 400 with every problem found, each with its path
    val patch = when (json.decode<NotePatch>(try req.text())) {
      is Ok(p)  => p
      is Err(e) => throw http.badRequest(e.message())
    }
    if (patch.text.trim().isEmpty()) throw http.badRequest("a note needs some text")
    if (!notes.withLock(n => n.update(id, patch.text.trim()))) throw http.notFound("no note $id")
    http.Response.json(try json.encode(Note(id, text: patch.text.trim())))
  })

  router.delete("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val removed = notes.withLock(n => n.remove(id))
    if (!removed) throw http.notFound("no note $id")
    http.Response.empty(http.Status.noContent)
  })

  router.handler()
}

// ---------------------------------------------------------------------------
// --check: a scripted client in the same process

// The client bounds its reads too: a server is a stranger from here, and
// `readLine` has no unbounded form by design.
const maxResponseLine: i64 = 8192

/// One raw HTTP/1.1 exchange over a fresh connection: the status line, the
/// content type and the body, as the test output.
fun exchange(port: i64, method: string, target: string, body: string, extra: string = ""): string throws IoError | net.TooLong {
  with (conn = try net.connect("127.0.0.1", port)) {
    val head = StringBuilder()
    head.append("$method $target HTTP/1.1\r\nHost: check\r\nConnection: close\r\nX-Request-Id: check\r\n")
    if (!body.isEmpty()) head.append("Content-Length: ${body.len()}\r\n")
    head.append(extra)
    head.append("\r\n")
    try conn.writeText(head.toString() + body)
    val status = try conn.readLine(max: maxResponseLine) ?: "(no response)"
    var contentType = "-"
    var id = "-"
    // shown only when the server sends them: the methods a 405 or an
    // OPTIONS names, and the length a HEAD promises without a body
    var more = ""
    loop {
      val line = try conn.readLine(max: maxResponseLine) ?: break
      if (line.isEmpty()) break
      val (rawName, rawValue) = line.splitOnce(":") ?: continue
      val value = rawValue.trim()
      when (rawName.toLower()) {
        "content-type"   => contentType = value
        "x-request-id"   => id = value
        "allow"          => more = more + " allow=$value"
        "content-length" => if (method == "HEAD") more = more + " length=$value"
        else             => { }
      }
    }
    var text = ""
    loop {
      val chunk = try conn.read()
      if (chunk.isEmpty()) break
      text = text + (chunk.decodeUtf8() ?: "<binary>")
    }
    "< $status [$contentType] id=$id$more" + (if (text.isEmpty()) "" else "\n< $text")
  }
}

// What one request may cost this server. A note is a line of text and the
// API takes no uploads, so the body ceiling is small on purpose: the
// defaults are for a server that does not know what it serves, and this
// one does. Everything not named here keeps its default (see http.Limits).
val limits = http.Limits(bodyBytes: 4096, headerCount: 32)

// A request that is refused for its size is thousands of characters long;
// the transcript wants the answer, not the characters.
fun brief(s: string): string =
  if (s.len() <= 48) s else (s.substring(0, 24) ?: s) + "...(${s.len()} bytes)"

// Header ceilings, as raw header lines: more headers than `headerCount`,
// and one header longer than `headerLineBytes`.
val headerScript = [
  ("40 headers", "X-N: 1\r\n".repeat(40)),
  ("one long header", "X-Long: " + "y".repeat(9000) + "\r\n"),
]

fun check(handler: http.Handler) throws IoError | EncodeError | net.TooLong {
  val script = [
    ("GET", "/api/echo?msg=hello+world", ""),
    ("GET", "/api/notes", ""),
    ("POST", "/api/notes", "buy milk"),
    ("POST", "/api/notes", "call \"mum\""),
    ("POST", "/api/notes", "   "),
    ("GET", "/api/notes", ""),
    ("GET", "/api/notes/2", ""),
    ("GET", "/api/notes/9", ""),
    ("GET", "/api/notes/x", ""),
    ("DELETE", "/api/notes/1", ""),
    ("DELETE", "/api/notes/1", ""),
    ("PUT", "/api/notes/2", "{\"text\": \"call mum back\"}"),
    ("PUT", "/api/notes/2", "{\"text\": 5, \"extra\": true}"),
    ("PUT", "/api/notes/2", "{\"text\": \"x\""),
    ("PUT", "/api/notes/9", "{\"text\": \"nope\"}"),
    ("GET", "/api/notes", ""),
    ("PUT", "/api/notes", "nope"),
    // HEAD is answered by the GET route without the body; OPTIONS lists
    // what the path accepts
    ("HEAD", "/api/notes/2", ""),
    ("OPTIONS", "/api/notes/2", ""),
    ("GET", "/static/", ""),
    ("GET", "/static/style.css", ""),
    ("GET", "/static/../main.vs", ""),
    ("GET", "/static/..\\main.vs", ""),
    ("GET", "/static/%2e%2e/main.vs", ""),
    ("GET", "/static/missing.txt", ""),
    ("GET", "/", ""),
    ("GET", "/nowhere", ""),
    // the ceilings, refused before a handler ever sees the request
    ("POST", "/api/notes", "x".repeat(5000)),
    ("GET", "/api/notes/" + "9".repeat(9000), ""),
  ]
  with (listener = try net.listen()) {
    val port = listener.port()
    scope {
      val server = async http.serve(listener, handler, limits, log: false)
      loop ((method, target, body) in script) {
        io.println("> $method ${brief(target)}" + (if (body.isEmpty()) "" else " ${try json.encode(brief(body))}"))
        io.println(try exchange(port, method, target, body))
      }
      // the header ceilings need raw header lines, which the script above
      // does not carry
      loop ((what, extra) in headerScript) {
        io.println("> GET /api/notes ($what)")
        io.println(try exchange(port, "GET", "/api/notes", "", extra))
      }
      server.cancel()
    }
  }
  // the same handler in memory — no socket, the same routing and panic
  // boundary — which is how a handler is unit-tested
  loop ((method, target) in [(http.Method.get, "/api/notes/2"), (http.Method.delete, "/api/echo")]) {
    val resp = http.call(handler, method, target)
    io.println("> $method $target (in memory)")
    io.println("< ${resp.status} allow=${resp.headers.get(http.Header.allow) ?: "-"} ${resp.body.decodeUtf8() ?: "<binary>"}")
  }
}

// ---------------------------------------------------------------------------

fun run(args: List<string>) throws UsageError | IoError | EncodeError | net.TooLong {
  val opts = try Options.parse(args)
  if (!fs.isDir(opts.dir)) throw UsageError(message: "'${opts.dir}' is not a directory (the files to serve)")
  val handler = app(opts.dir)
  if (opts.check) {
    return try check(handler)
  }
  with (listener = try net.listen(host: opts.host, port: opts.port)) {
    io.println("serving ${opts.dir} on http://${opts.host}:${listener.port()}/ — Ctrl+C stops it gracefully")
    // Ctrl+C or a SIGTERM: stop accepting, finish what is in flight (up to
    // ten seconds), then return and close the listener
    http.serve(listener, handler, limits, stop: () => os.shutdownSignal())
    io.println("stopped")
  }
}

fun main() {
  when (val result = run(os.args())) {
    is Err => {
      io.println("httpd: ${result.message()}")
      os.exit(if (result is UsageError) 2 else 1)
    }
    is Ok  => { }
  }
}
