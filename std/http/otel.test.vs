// Tests of http's part in OpenTelemetry (D126): a server span per request
// that continues the caller's trace, a client span and a traceparent on every
// call, probes left out, and the OTLP exporter over a real HTTP collector.
// The pipeline is the process's, so the tests that start it take turns.

use compress as gz
use otel

val otelGate = Semaphore(permits: 1)

// ---- a protobuf reader, for looking inside what was exported ----

test fun hexString(bytes: List<u8>): string => bytes.map(b => (if (b < 16) "0" else "") + b.toI64().toString(radix: 16)).join("")

struct PbField {
  num:   i64
  wire:  i64
  value: i64
  data:  List<u8>
}

test fun pbVarint(bytes: List<u8>, at: i64): (i64, i64) {
  var result: u64 = 0
  var shift: i64 = 0
  var i = at
  loop {
    val b = bytes.at(i) ?: break
    i += 1
    result = result | ((b & 127).toU64() << shift)
    if (b < 128) break
    shift += 7
  }
  (result.wrapI64(), i)
}

test fun pbFields(bytes: List<u8>): List<PbField> {
  val out: MutableList<PbField> = []
  var i: i64 = 0
  loop (i < bytes.len()) {
    val (key, afterKey) = pbVarint(bytes, i)
    i = afterKey
    val wire = key % 8
    if (wire == 0) {
      val (v, next) = pbVarint(bytes, i)
      i = next
      out.push(PbField(num: key / 8, wire, value: v, data: []))
    } else if (wire == 2) {
      val (len, next) = pbVarint(bytes, i)
      out.push(PbField(num: key / 8, wire, value: len, data: bytes.slice(next, next + len)))
      i = next + len
    } else if (wire == 1) {
      out.push(PbField(num: key / 8, wire, value: 0, data: bytes.slice(i, i + 8)))
      i += 8
    } else {
      fail("unexpected wire type $wire")
    }
  }
  out.toList()
}

test fun pbAll(fs: List<PbField>, num: i64): List<PbField> => fs.filter(f => f.num == num)

test fun pbOne(fs: List<PbField>, num: i64): PbField => pbAll(fs, num).at(0) ?: PbField(num: 0, wire: 0, value: 0, data: [])

test fun pbText(fs: List<PbField>, num: i64): string => pbOne(fs, num).data.decodeUtf8() ?: "<binary>"

// the attribute `key` of a span's field 9, as its AnyValue fields; null when absent
test fun pbAttr(span: List<PbField>, key: string): List<PbField>? {
  loop (kv in pbAll(span, 9)) {
    val f = pbFields(kv.data)
    if (pbText(f, 1) == key) return pbFields(pbOne(f, 2).data)
  }
  null
}

// ---- a recording exporter ----

struct Capture {
  traces: Mutex<MutableList<List<u8>>> = newCaptured()

  implement otel.Exporter {
    fun export(signal: otel.Signal, body: List<u8>) suspends throws otel.ExportError {
      if (signal == otel.Signal.Traces) {
        this.traces.withLock(q => {
          q.push(body)
        })
      }
    }
  }

  // every span exported so far, as its fields
  fun spans(): List<List<PbField>> {
    val bodies = this.traces.withLock(q => q.toList())
    val out: MutableList<List<PbField>> = []
    loop (body in bodies) {
      val scopeSpans = pbFields(pbOne(pbFields(pbOne(pbFields(body), 1).data), 2).data)
      loop (s in pbAll(scopeSpans, 2)) {
        out.push(pbFields(s.data))
      }
    }
    out.toList()
  }

  fun named(name: string): List<List<PbField>> => this.spans().filter(s => pbText(s, 5) == name)

  // The pipeline is the process's, and the other tests of this package run
  // requests while it is on (they take no turn at the gate), so a capture
  // also holds their spans. The spans of one server are the ones a test is
  // about: its own server spans carry the Host it was called by, and the
  // client spans the port they called.
  fun on(port: i64): List<List<PbField>> => this.spans().filter(s => touches(s, port))

  fun namedOn(name: string, port: i64): List<List<PbField>> => this.on(port).filter(s => pbText(s, 5) == name)
}

test fun touches(span: List<PbField>, port: i64): bool {
  val address = pbText(pbAttr(span, "server.address") ?: [], 1)
  if (address.endsWith(":$port")) return true
  pbOne(pbAttr(span, "server.port") ?: [], 3).value == port
}

test fun newCaptured(): Mutex<MutableList<List<u8>>> {
  val empty: MutableList<List<u8>> = []
  Mutex(value: empty)
}

test fun otelApp(health: Health): Handler {
  val router = Router()
  router.get("/users/{id}", req => Response.text("user ${req.param("id")}"))
  router.get("/boom", req => throw Boom())
  router.get("/echo", req => Response.text(req.header("traceparent") ?: "none"))
  router.wrap(health.endpoints())
  router.handler()
}

error Boom { }

val parentTrace = "4bf92f3577b34da6a3ce929d0e0e4736"
val parentSpan = "00f067aa0ba902b7"

// ---- the server ----

test "a request is a server span, named by its route, in the caller's trace" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  with res = try Client.untraced(Duration.seconds(5)).get("${srv.url}/users/7?token=secret", headers: ["traceparent": "00-$parentTrace-$parentSpan-01"])
  expect(res.status == Status.ok)
  expect(tel.flush())
  val found = capture.named("GET /users/{id}")
  expect(found.len() == 1)
  val span = found.at(0) ?: []
  expect(pbOne(span, 6).value == 2)
  expect(hexString(pbOne(span, 1).data) == parentTrace)
  expect(hexString(pbOne(span, 4).data) == parentSpan)
  expect(pbText(pbAttr(span, "http.route") ?: [], 1) == "/users/{id}")
  expect(pbText(pbAttr(span, "url.path") ?: [], 1) == "/users/7")
  expect(pbOne(pbAttr(span, "http.response.status_code") ?: [], 3).value == 200)
  // the query string, where tokens go, is not recorded
  expect(pbAttr(span, "url.query") == null)
  expect(pbAll(span, 15).isEmpty())
}

test "a request with no traceparent starts a trace of its own" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  with res = try Client.untraced(Duration.seconds(5)).get("${srv.url}/users/1")
  expect(res.ok)
  expect(tel.flush())
  val span = capture.named("GET /users/{id}").at(0) ?: []
  expect(pbOne(span, 1).data.len() == 16)
  expect(pbAll(span, 4).isEmpty())
}

test "a handler that fails marks its span failed; an unknown path is named by its method" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  with client = Client.untraced(Duration.seconds(5))
  with boom = try client.get("${srv.url}/boom")
  expect(boom.status == Status.internalServerError)
  with nowhere = try client.get("${srv.url}/nowhere")
  expect(nowhere.status == Status.notFound)
  expect(tel.flush())
  val failed = capture.named("GET /boom").at(0) ?: []
  expect(pbOne(pbFields(pbOne(failed, 15).data), 3).value == 2)
  expect(pbText(pbAttr(failed, "error.type") ?: [], 1) == "500")
  // a 404 is the client's mistake, not the server's failure
  val missing = capture.namedOn("GET", srv.port()).at(0) ?: []
  expect(pbAll(missing, 15).isEmpty())
}

test "health probes are not traced" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  with client = Client.untraced(Duration.seconds(5))
  with live = try client.get("${srv.url}/healthz")
  with ready = try client.get("${srv.url}/readyz")
  with real = try client.get("${srv.url}/users/2")
  expect(tel.flush())
  expect(capture.on(srv.port()).len() == 1)
}

// ---- the client ----

test fun callEcho(url: string): string throws FetchError {
  with span = otel.span("caller")
  try get(url).text()
}

test "a call is a client span, and the service called continues the trace" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  val seen = try callEcho("${srv.url}/echo?key=hidden")
  expect(tel.flush())
  val sent = otel.SpanContext.parse(seen) ?: fail("the server saw no valid traceparent: $seen")
  val client = capture.namedOn("GET", srv.port()).filter(s => pbOne(s, 6).value == 3).at(0) ?: []
  val caller = capture.named("caller").at(0) ?: []
  val server = capture.named("GET /echo").at(0) ?: []
  // caller → client call → server span, one trace
  expect(pbOne(client, 4).data == pbOne(caller, 2).data)
  expect(hexString(pbOne(client, 2).data) == sent.spanIdHex())
  expect(pbOne(server, 4).data == pbOne(client, 2).data)
  expect(pbOne(server, 1).data == pbOne(caller, 1).data)
  expect(pbText(pbAttr(client, "url.full") ?: [], 1) == "${srv.url}/echo")
  expect(pbOne(pbAttr(client, "http.response.status_code") ?: [], 3).value == 200)
  expect(pbOne(pbAttr(client, "server.port") ?: [], 3).value == srv.port())
}

test "a traceparent the caller set is left alone" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with srv = try testServer(otelApp(Health()))
  val own = "00-$parentTrace-$parentSpan-00"
  val seen = try get("${srv.url}/echo", headers: ["Traceparent": own]).text()
  expect(seen == own)
}

test "a failed call fails its span with the reason" {
  with permit = otelGate.acquire()
  val capture = Capture()
  with tel = try otel.start(service: "t", exporter: capture, interval: Duration.seconds(3600))
  with silent = try canned("")
  expect(failKind(get("${silent.url}/x", timeout: Duration.seconds(2))) == FetchKind.Closed)
  expect(tel.flush())
  val span = capture.namedOn("GET", silent.listener.port()).at(0) ?: []
  expect(pbOne(pbFields(pbOne(span, 15).data), 3).value == 2)
  expect(pbText(pbAttr(span, "error.type") ?: [], 1) == "Closed")
}

test "nothing is added when otel is not running" {
  with permit = otelGate.acquire()
  with srv = try testServer(otelApp(Health()))
  val seen = try get("${srv.url}/echo").text()
  expect(seen == "none")
}

// ---- the exporter ----

struct Collector {
  // path, content-type, content-encoding, authorization, body
  posts: Mutex<MutableList<(string, string, string, string, List<u8>)>> = newPosts()
  // the status to answer with, and how many more times
  status: Atomic<i64> = Atomic(value: 200)
}

test fun newPosts(): Mutex<MutableList<(string, string, string, string, List<u8>)>> {
  val empty: MutableList<(string, string, string, string, List<u8>)> = []
  Mutex(value: empty)
}

test fun collecting(c: Collector): Handler => handler(req => {
  val body = try req.bytes()
  c.posts.withLock(q => {
    q.push((req.path, req.header("content-type") ?: "", req.header("content-encoding") ?: "", req.header("authorization") ?: "", body))
  })
  Response.empty(Status(code: c.status.load()))
})

test "exporter: posts gzipped protobuf to the signal's path, with its headers" {
  with permit = otelGate.acquire()
  val collector = Collector()
  with srv = try testServer(collecting(collector))
  val exporter = otlp(endpoint: "${srv.url}/", headers: ["Authorization": Secret.of("Bearer k-123")])
  with tel = try otel.start(service: "exported", exporter: exporter, interval: Duration.seconds(3600))
  otelSpan()
  expect(tel.flush())
  val posts = collector.posts.withLock(q => q.toList())
  val traces = posts.filter(p => p.0 == "/v1/traces")
  expect(traces.len() == 1)
  val (_, kind, coding, auth, body) = traces.at(0) ?: ("", "", "", "", [])
  expect(kind == "application/x-protobuf")
  expect(coding == "gzip")
  expect(auth == "Bearer k-123")
  // the body is a gzip of an ExportTraceServiceRequest: resource_spans, field 1
  val plain = try gz.gunzip(body)
  expect(pbAll(pbFields(plain), 1).len() == 1)
  expect(posts.filter(p => p.0 == "/v1/metrics").len() <= 1)
  // its own requests are not spans: only the one the test made
  expect(pbAll(pbFields(pbOne(pbFields(pbOne(pbFields(plain), 1).data), 2).data), 2).len() == 1)
}

test fun otelSpan() {
  with s = otel.span("work")
  s.ok()
}

test "exporter: a busy collector is retried, a refusing one is not" {
  with permit = otelGate.acquire()
  val collector = Collector(status: Atomic(value: 503))
  with srv = try testServer(collecting(collector))
  with tel = try otel.start(service: "t", exporter: otlp(endpoint: srv.url), interval: Duration.seconds(3600))
  otelSpan()
  expect(!tel.flush())
  // four tries for the spans (the first and three retries), at 200, 400 and 800 ms
  expect(collector.posts.withLock(q => q.filter(p => p.0 == "/v1/traces").len()) == 4)
}

test "exporter: a collector that refuses the data is asked once" {
  with permit = otelGate.acquire()
  val collector = Collector(status: Atomic(value: 400))
  with srv = try testServer(collecting(collector))
  with tel = try otel.start(service: "t", exporter: otlp(endpoint: srv.url), interval: Duration.seconds(3600))
  otelSpan()
  expect(!tel.flush())
  expect(collector.posts.withLock(q => q.filter(p => p.0 == "/v1/traces").len()) == 1)
}

test "exporter: an unreachable collector is a retryable failure, not a crash" {
  with permit = otelGate.acquire()
  // a collector that takes the connection and drops it
  with silent = try canned("")
  with tel = try otel.start(service: "t", exporter: otlp(endpoint: silent.url, timeout: Duration.seconds(2)), interval: Duration.seconds(3600))
  otelSpan()
  expect(!tel.flush())
  expect(otel.dropped() >= 1)
}
