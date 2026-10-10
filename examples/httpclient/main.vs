// httpclient: the other side of `httpd` — a program that calls a web API.
// It starts a small notes service on a free port of this machine, then uses
// `http.Client` against it the way a real client would: send and read JSON,
// treat a 404 as an answer and not a crash, follow a redirect, survive a
// service that is busy for a moment, give up on one that is too slow, and ask
// for several pages at once.
use http
use io { println }
use json
use time

struct Note {
  id:    i64
  title: string

  implement Codable
}

struct NewNote {
  title: string

  implement Codable
}

fun service(): http.Handler {
  val notes: Mutex<MutableList<Note>> = Mutex(value: [Note(id: 1, title: "water the plants")])
  // how many times /busy has been asked, to be busy for the first two
  val busy: Atomic<i64> = Atomic(value: 0)
  val app = http.Router()

  app.get("/notes", req => http.Response.json(try json.encode(notes.withLock(n => n.toList()))))

  app.post("/notes", req => {
    val given = when (json.decode<NewNote>(try req.text())) {
      is Ok(n)  => n
      is Err(e) => throw http.badRequest(e.message())
    }
    val note = notes.withLock(n => {
      val created = Note(id: n.len() + 1, title: given.title)
      n.push(created)
      created
    })
    http.Response.json(try json.encode(note), status: http.Status.created)
  })

  app.get("/notes/{id}", req => {
    val id = try req.param("id").toInt() ?! http.badRequest("the id must be a number")
    val note = try notes.withLock(n => n.find(x => x.id == id)) ?! http.notFound("no note $id")
    http.Response.json(try json.encode(note))
  })

  // the old address of the list
  app.get("/all-notes", req => http.Response.redirect("/notes", status: http.Status.movedPermanently))

  app.get("/busy", req => {
    val n = busy.add(1)
    if (n <= 2) throw http.Fail(status: http.Status.serviceUnavailable, text: "try again")
    http.Response.text("served on try $n")
  })

  app.get("/slow", req => {
    await sleep(Duration.seconds(30))
    http.Response.text("finally")
  })

  app.get("/page/{n}", req => {
    // later pages answer sooner, so the order they finish in is not the order asked
    await sleep(Duration.millis(60 - 20 * (req.param("n").toInt() ?: 0)))
    http.Response.text("page ${req.param("n")}")
  })

  app.handler()
}

fun pageText(client: http.Client, url: string): string suspends throws http.FetchError =>
  try client.get(url).text()

fun threePages(client: http.Client, base: string) suspends throws http.FetchError {
  scope {
    val first = async pageText(client, "$base/page/1")
    val second = async pageText(client, "$base/page/2")
    val third = async pageText(client, "$base/page/3")
    println(await first)
    println(await second)
    println(await third)
  }
}

fun main() throws IoError {
  with srv = try http.testServer(service())
  with client = http.Client(timeout: Duration.seconds(5), headers: ["accept": "application/json"])
  val base = srv.url

  do {
    // read a list of things
    val notes = try client.get("$base/notes").json<List<Note>>()
    println("${notes.len()} note(s); first: ${notes.at(0)?.title ?: "-"}")

    // send one
    val made = try client.post("$base/notes", body: try http.Payload.json(NewNote(title: "call mum")))
    val note = try made.json<Note>()
    println("created ${made.status}: #${note.id} ${note.title}")

    // a 404 is an answer: look at it, or demand success
    with missing = try client.get("$base/notes/99")
    println("note 99: ${missing.status}, ok=${missing.ok}, said: ${try missing.text()}")
    with again = try client.get("$base/notes/99")
    when (again.ensureSuccess()) {
      is Ok(_)  => println("unexpected")
      is Err(e) => println("ensureSuccess: ${e.message().replace(base, "")}")
    }

    // a bad body is the server's 400, with its reasons
    with bad = try client.post("$base/notes", body: http.Payload.text("{\"title\": 5}", contentType: http.MediaType.json))
    println("bad body: ${bad.status}")

    // a redirect is followed; the answer says where it ended
    val moved = try client.get("$base/all-notes")
    println("moved: ${moved.url.replace(base, "")} (${moved.status})")
    with kept = try client.get("$base/all-notes", redirect: false)
    println("not followed: ${kept.status}")

    // busy twice, then fine — with retries it is one call
    with once = try client.get("$base/busy")
    println("no retry: ${once.status}")
    val patient = try client.get("$base/busy", retry: 3)
    println("retry: ${try patient.text()}")

    // too slow for the time we have
    val started = time.Stopwatch.start()
    val slow = client.get("$base/slow", timeout: Duration.millis(150))
    when (slow) {
      is Ok(_)  => println("slow: unexpected answer")
      is Err(e) => println("slow: ${e.kind}")
    }
    println("gave up in time: ${started.elapsed() < Duration.seconds(3)}")

    // three pages at once, collected in the order asked
    try threePages(client, base)
  } catch (e) {
    println("failed: ${e.message()}")
  }
}
