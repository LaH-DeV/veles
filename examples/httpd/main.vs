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
use fs, http, io, net, os

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
      val arg = args.atOrPanic(i)
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
}

/// The store behind the API. Handlers run in connection tasks, so it lives
/// in a Mutex (D35); its fields are private, so the only way in is through
/// the methods, which keep `next` and `items` consistent.
struct Notes {
  private var next: i64 = 1
  private items:    MutableList<Note> = []

  fun all(): List<Note> = self.items.toList()

  fun find(id: i64): Note? = self.items.find(x => x.id == id)

  /// Stores a new note with the next id.
  fun add(text: string): Note {
    val created = Note(id: self.next, text)
    self.next += 1
    self.items.push(created)
    created
  }

  /// Removes the note with that id; false when there is none. (`items` is a
  /// MutableList — a handle — so this needs no `mut`: only `self`'s own
  /// fields are guarded by it, D22/D25.)
  fun remove(id: i64): bool {
    val at = self.items.indexOfFirst(x => x.id == id)
    if (at >= 0) self.items.removeAt(at)
    at >= 0
  }
}

fun jsonString(s: string): string =
  "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t") + "\""

fun noteJson(n: Note): string = "{\"id\": ${n.id}, \"text\": ${jsonString(n.text)}}"

fun notesJson(notes: List<Note>): string = "[" + notes.map(n => noteJson(n)).join(", ") + "]"

/// The whole application as one handler: the routes below plus the files
/// under `dir`.
fun app(dir: string): http.Handler {
  val notes = mutex(Notes())
  val router = http.router()

  router.get("/", req => http.Response.redirect("/static/"))
  router.get("/static/*", http.files(dir))

  router.get("/api/echo", req => http.Response.text(req.query.get("msg") ?: "(no msg)"))

  router.get("/api/notes", req => http.Response.json(notes.withLock(n => notesJson(n.all()))))

  router.post("/api/notes", req => {
    val text = (try req.text()).trim()
    if (text.isEmpty()) throw http.badRequest("a note needs some text")
    val note = notes.withLock(n => n.add(text))
    http.Response.json(noteJson(note), status: 201).withHeader("location", "/api/notes/${note.id}")
  })

  router.get("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val note = try notes.withLock(n => n.find(id)) ?! http.notFound("no note $id")
    http.Response.json(noteJson(note))
  })

  router.delete("/api/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val removed = notes.withLock(n => n.remove(id))
    if (!removed) throw http.notFound("no note $id")
    http.Response.empty(204)
  })

  router.handler()
}

// ---------------------------------------------------------------------------
// --check: a scripted client in the same process

/// One raw HTTP/1.1 exchange over a fresh connection: the status line, the
/// content type and the body, as the test output.
fun exchange(port: i64, method: string, target: string, body: string): string throws IoError {
  with (conn = try net.connect("127.0.0.1", port)) {
    val head = stringBuilder()
    head.append("$method $target HTTP/1.1\r\nHost: check\r\nConnection: close\r\n")
    if (!body.isEmpty()) head.append("Content-Length: ${body.len()}\r\n")
    head.append("\r\n")
    try conn.writeText(head.toString() + body)
    val status = try conn.readLine() ?: "(no response)"
    var contentType = "-"
    loop {
      val line = try conn.readLine() ?: break
      if (line.isEmpty()) break
      if (line.toLower().startsWith("content-type:")) contentType = (line.substring(13, line.len()) ?: "").trim()
    }
    var text = ""
    loop {
      val chunk = try conn.read()
      if (chunk.isEmpty()) break
      text = text + (chunk.decodeUtf8() ?: "<binary>")
    }
    "< $status [$contentType]" + (if (text.isEmpty()) "" else "\n< $text")
  }
}

fun check(handler: http.Handler) throws IoError {
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
    ("GET", "/api/notes", ""),
    ("PUT", "/api/notes", "nope"),
    ("GET", "/static/", ""),
    ("GET", "/static/style.css", ""),
    ("GET", "/static/../main.vs", ""),
    ("GET", "/static/missing.txt", ""),
    ("GET", "/", ""),
    ("GET", "/nowhere", ""),
  ]
  with (listener = try net.listen()) {
    val port = listener.port()
    scope {
      val server = async http.serve(listener, handler, log: false)
      loop ((method, target, body) in script) {
        io.println("> $method $target" + (if (body.isEmpty()) "" else " ${jsonString(body)}"))
        io.println(try exchange(port, method, target, body))
      }
      server.cancel()
    }
  }
}

// ---------------------------------------------------------------------------

fun run(args: List<string>) throws UsageError | IoError {
  val opts = try Options.parse(args)
  if (!fs.isDir(opts.dir)) throw UsageError(message: "'${opts.dir}' is not a directory (the files to serve)")
  val handler = app(opts.dir)
  if (opts.check) {
    try check(handler)
    return
  }
  with (listener = try net.listen(host: opts.host, port: opts.port)) {
    io.println("serving ${opts.dir} on http://${opts.host}:${listener.port()}/ — Ctrl+C stops")
    http.serve(listener, handler)
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
