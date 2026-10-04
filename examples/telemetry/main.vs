// telemetry: what a service tells the people running it. A small notes
// service and a client in one program, with OpenTelemetry started: every
// request becomes a span, the trace continues across the call from client to
// service, a counter and a histogram measure the work, and each log line
// carries the id of the trace it belongs to.
//
// The exporter here only counts what it is given, so the program runs without
// a collector. To send it to a real one, replace it with
// `http.otlp(endpoint: "http://localhost:4318")`.
use http, io { println }, log, otel, time

// What an exporter receives is Protocol Buffers; this one keeps a tally.
struct Tally {
  traces:  Atomic<i64> = Atomic(value: 0)
  metrics: Atomic<i64> = Atomic(value: 0)
  logs:    Atomic<i64> = Atomic(value: 0)

  implement otel.Exporter {
    fun export(signal: otel.Signal, body: List<u8>) suspends throws otel.ExportError {
      val which = when (signal) {
        otel.Signal.Traces  => this.traces
        otel.Signal.Metrics => this.metrics
        otel.Signal.Logs    => this.logs
      }
      val _ = which.update(n => n + (if (body.isEmpty()) 0 else 1))
    }
  }
}

fun service(): http.Handler {
  val notes = otel.meter("notes")
  val reads = notes.counter("notes.read", unit: "1", description: "notes read")
  val latency = notes.histogram("notes.read.duration", unit: "s")
  val router = http.Router()
  router.get("/notes/{id}", req => {
    val started = time.Stopwatch.start()
    val id = req.param("id")
    with span = otel.span("load note", attrs: [otel.attr("note.id", id)])
    await sleep(Duration.millis(5))
    log.info("note read", log.field("id", id))
    reads.add(1)
    latency.record(started.elapsed())
    // tell the caller which trace this was, so the program can show it
    http.Response.text("note $id").withHeader("x-trace-id", otel.traceId() ?: "none")
  })
  router.handler()
}

fun main() throws IoError | otel.StartError | http.FetchError {
  val tally = Tally()
  with tel = try otel.start(service: "notes", exporter: tally, interval: Duration.seconds(3600))
  with srv = try http.testServer(service())

  with page = otel.span("page view", kind: otel.SpanKind.Internal)
  val mine = otel.traceId() ?: "none"
  // an ordinary call: the client span is a child of "page view", and the
  // request carries the trace on to the service
  with res = try http.get("${srv.url}/notes/7")
  val theirs = res.header("x-trace-id") ?: "none"
  println("answer: ${try res.text()}")
  println("the service handled it in the same trace: ${mine == theirs}")
  println("the trace id is 32 hex digits: ${mine.len() == 32}")

  page.end()
  val sent = tel.shutdown()
  println("exported: $sent")
  println("trace exports: ${tally.traces.load()}, metric exports: ${tally.metrics.load()}, log exports: ${tally.logs.load()}")
}
