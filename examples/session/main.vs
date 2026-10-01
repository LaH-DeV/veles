// session: a login kept in a cookie, a form and a query string read into
// structs (D94). A scripted client talks to the server over a real socket
// in the same process and prints what crosses the wire — status line,
// every Set-Cookie line, the body — which is how the test suite runs it.
use http, io { println }, net

/// What the login form posts. A missing field or a value that does not fit
/// is a 400 naming the field; `remember` is a checkbox, so absent is false.
struct Login {
  user:     string
  password: string
  remember: bool = false
  implement Decodable
}

/// What `/find?q=...&tags=a&tags=b&page=2` asks for.
struct Search {
  q:    string
  page: i64 = 1
  tags: List<string> = []
  implement Decodable
}

/// Who is logged in, by session id. The ids are counted, which is fine for
/// a demonstration and wrong for a real site: use `random` for those.
struct Sessions {
  private var next: i64 = 1
  private users:    MutableMap<string, string> = [:]

  fun start(user: string): string {
    val id = "s${this.next}"
    this.next += 1
    this.users.set(id, user)
    id
  }

  fun user(id: string): string? = this.users.get(id)

  fun end(id: string) {
    this.users.remove(id)
  }
}

fun app(): http.Handler {
  val sessions = Mutex(value: Sessions())
  val router = http.Router()

  router.get("/login", req => http.Response.html("<form method=post action=/login>...</form>"))

  router.post("/login", req => {
    val login = try req.form<Login>()
    if (login.password != "open sesame") throw http.Fail(status: http.Status.unauthorized, text: "wrong password")
    val id = sessions.withLock(s => s.start(login.user))
    // a session cookie, or one that outlives the browser when asked to
    val keep: Duration? = if (login.remember) Duration.days(30) else null
    http.Response.redirect("/me", status: http.Status.seeOther).withCookie(http.Cookie(name: "sid", value: id, maxAge: keep))
  })

  router.get("/me", req => {
    val id = req.cookie("sid") ?: throw http.Fail(status: http.Status.unauthorized, text: "log in first")
    val user = sessions.withLock(s => s.user(id)) ?: throw http.Fail(status: http.Status.unauthorized, text: "unknown session")
    http.Response.text("hello, $user")
  })

  router.post("/logout", req => {
    val id = req.cookie("sid") ?: ""
    sessions.withLock(s => s.end(id))
    http.Response.redirect("/login", status: http.Status.seeOther).withoutCookie("sid")
  })

  // two cookies at once, and a value that would end a header if it were
  // written as it is
  router.get("/prefs", req => http.Response.text("saved")
    .withCookie(http.Cookie(name: "theme", value: "dark mode; yes", maxAge: Duration.days(365), secure: true))
    .withCookie(http.Cookie(name: "lang", value: "en", sameSite: http.SameSite.Strict, httpOnly: false)))

  router.get("/find", req => {
    val s = try req.query<Search>()
    http.Response.text("q=${s.q} page=${s.page} tags=${s.tags}")
  })

  // cookies the browsers would refuse: a panic at the line that wrote them,
  // answered 500 like any handler's panic
  router.get("/bad-name", req => http.Response.text("x").withCookie(http.Cookie(name: "no spaces", value: "x")))
  router.get("/bad-samesite", req => http.Response.text("x").withCookie(http.Cookie(name: "t", value: "x", sameSite: http.SameSite.None)))

  router.handler()
}

// ---------------------------------------------------------------------------
// the scripted client

const maxResponseLine: i64 = 8192

/// One raw exchange over a fresh connection: the status line, then each
/// Set-Cookie or Location header as sent, then the body.
fun exchange(port: i64, method: string, target: string, body: string, extra: string = ""): string throws IoError | net.TooLong {
  with conn = try net.connect("127.0.0.1", port)
  val head = StringBuilder()
  head.append("$method $target HTTP/1.1\r\nHost: check\r\nConnection: close\r\n")
  if (!body.isEmpty()) head.append("Content-Length: ${body.len()}\r\n")
  head.append(extra)
  head.append("\r\n")
  try conn.writeText(head.toString() + body)
  val out = StringBuilder()
  out.append("< ${try conn.readLine(max: maxResponseLine) ?: "(no response)"}")
  loop {
    val line = try conn.readLine(max: maxResponseLine) ?: break
    if (line.isEmpty()) break
    val name = line.splitOnce(":")?.0?.toLower() ?: ""
    if (name == "set-cookie" || name == "location") out.append("\n<   $line")
  }
  var text = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    text = text + (chunk.decodeUtf8() ?: "<binary>")
  }
  if (!text.isEmpty()) out.append("\n< $text")
  out.toString()
}

val form = "Content-Type: application/x-www-form-urlencoded\r\n"

fun check() throws {
  // method, target, body, extra headers
  val script = [
    ("GET", "/login", "", ""),
    ("POST", "/login", "user=ada&password=nope", form),
    ("POST", "/login", "user=ada", form),
    ("POST", "/login", "user=ada&password=open+sesame", form),
    ("POST", "/login", "user=ada&password=open+sesame&remember=on", form),
    ("POST", "/login", "user=ada&password=open+sesame&remember=maybe", form),
    ("POST", "/login", "{\"user\":\"ada\"}", "Content-Type: application/json\r\n"),
    ("GET", "/me", "", "Cookie: sid=s1\r\n"),
    ("GET", "/me", "", "Cookie: theme=dark; sid=s2\r\n"),
    ("GET", "/me", "", ""),
    ("GET", "/me", "", "Cookie: sid=s99\r\n"),
    ("GET", "/prefs", "", ""),
    ("GET", "/find?q=cookie+jar&tags=a&tags=b&page=2", "", ""),
    ("GET", "/find?q=x", "", ""),
    ("GET", "/find?page=two&tags=", "", ""),
    ("POST", "/logout", "", "Cookie: sid=s1\r\n"),
    ("GET", "/me", "", "Cookie: sid=s1\r\n"),
    ("GET", "/bad-name", "", ""),
    ("GET", "/bad-samesite", "", ""),
  ]
  with (listener = try net.listen()) {
    val port = listener.port()
    with server = async http.serve(listener, app(), log: false)
    loop ((method, target, body, extra) in script) {
      println("> $method $target" + (if (body.isEmpty()) "" else " $body"))
      println(try exchange(port, method, target, body, extra))
    }
  }
  // the fields and cookies on their own, without a server
  val fields = http.Fields.parse("a=1&b=x%20y&a=3&flag")
  println("names ${fields.names()} a=${fields.all("a")} first a=${fields.get("a")} b=${fields.get("b")} flag=${fields.get("flag")} nope=${fields.get("nope")}")
}

fun main() {
  when (check()) {
    is Err(e) => println("session: ${e.message()}")
    is Ok     => { }
  }
}
