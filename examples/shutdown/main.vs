// Graceful shutdown (D68): a server that stops when the process is asked
// to, and what happens to the connections it has at that moment — one idle
// between requests, one in the middle of a request, and one whose handler
// runs past the grace period.
//
// `os.raiseSignal` plays the part of Ctrl+C or `kill`, so the run is
// deterministic; a real program writes the same `stop:` and is stopped
// from outside.
use http, io, net, os

// What a client sees of one response: the status line, whether the server
// said it will close, and the body.
fun response(c: net.Conn): string throws IoError | net.TooLong {
  val status = try c.readLine(max: 8192) ?: return "(closed without a response)"
  var length: i64 = 0
  var closing = false
  loop {
    val line = try c.readLine(max: 8192) ?: break
    if (line.isEmpty()) break
    val (name, value) = line.splitOnce(":") ?: continue
    when (name.toLower()) {
      "content-length" => length = value.trim().toInt() ?: 0
      "connection"     => closing = value.trim() == "close"
      else             => { }
    }
  }
  val body = (try c.readExact(length)).decodeUtf8() ?: "?"
  "$status | $body" + (if (closing) " | connection: close" else "")
}

fun ask(c: net.Conn, path: string): string throws IoError | net.TooLong {
  try c.writeText("GET $path HTTP/1.1\r\nHost: x\r\n\r\n")
  try response(c)
}

// Connected, one request answered, then quiet: the kind of connection a
// browser keeps open. A stopping server closes it at once.
fun idleClient(port: i64) throws IoError | net.TooLong {
  with (c = try net.connect("127.0.0.1", port)) {
    io.println("idle:  ${try ask(c, "/fast")}")
    val rest = try c.read()
    io.println("idle:  closed by the server (${rest.len()} more bytes)")
  }
}

// Mid-request when the stop comes: the answer still arrives, marked as
// the connection's last.
fun busyClient(port: i64) throws IoError | net.TooLong {
  with (c = try net.connect("127.0.0.1", port)) {
    io.println("busy:  ${try ask(c, "/slow")}")
  }
}

// A handler that outlives `grace`: it is cancelled, its `with` still
// closes, and the client gets no response.
fun stuckClient(port: i64) throws IoError | net.TooLong {
  with (c = try net.connect("127.0.0.1", port)) {
    io.println("stuck: ${try ask(c, "/stuck")}")
  }
}

struct Resource {
  implement Closeable {
    fun close() {
      io.println("stuck: the handler's resource was closed")
    }
  }
}

fun app(): http.Handler {
  val router = http.router()
  router.get("/fast", _ => http.Response.text("fast"))
  router.get("/slow", _ => {
    await sleep(Duration.millis(600))
    http.Response.text("slow, but finished")
  })
  router.get("/stuck", _ => {
    with (held = Resource()) {
      await sleep(Duration.seconds(30))
    }
    http.Response.text("never sent")
  })
  router.handler()
}

fun stopLater(after: Duration) {
  await sleep(after)
  io.println("--- the process is asked to stop")
  os.raiseSignal(os.Signal.Terminate)
}

// What a real server writes: serve until the process is asked to stop.
fun untilSignal() {
  scope {
    async stopLater(Duration.millis(100))
    val sig = os.shutdownSignal()
    io.println("server: $sig, draining")
  }
}

fun start(clients: sendable fun(i64) suspends throws IoError | net.TooLong, port: i64) throws IoError | net.TooLong = try clients(port)

// Serves until `stop` returns while `clients` run against the server;
// returns once both are done.
fun run(
  clients: sendable fun(i64) suspends throws IoError | net.TooLong,
  stop: sendable fun() suspends,
  grace: Duration,
) throws IoError | net.TooLong {
  with (listener = try net.listen()) {
    val port = listener.port()
    scope {
      async start(clients, port)
      http.serve(listener, app(), log: false, stop: stop, grace: grace)
    }
    io.println("server: serve returned")
  }
}

fun main() throws IoError | net.TooLong {
  // an idle and a busy connection, both finished inside the grace period
  try run(port => {
    scope {
      async idleClient(port)
      await sleep(Duration.millis(20))
      async busyClient(port)
    }
  }, stop: () => untilSignal(), grace: Duration.seconds(5))
  io.println("")
  // a handler that does not finish in time; the stop condition is any
  // code at all — here a channel
  val stopNow = Channel<bool>(capacity: 1)
  try run(port => {
    scope {
      async stuckClient(port)
      await sleep(Duration.millis(100))
      stopNow.send(true)
    }
  }, stop: () => {
    val _ = await stopNow.recv()
    io.println("server: told to stop, 100ms of grace")
  }, grace: Duration.millis(100))
}
