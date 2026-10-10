// Tests of the pipeline (D126): spans, metrics and logs from the call to the
// request body an exporter is given, through a recording exporter. The switch
// and the queues are the process's, so every test that starts the pipeline
// holds the gate and the tests take turns.

val gate = Semaphore(permits: 1)

struct Sent {
  signal: Signal
  body:   List<u8>
}

// an exporter that keeps what it is given, and can be made to fail first
struct Recorder {
  seen: Mutex<MutableList<Sent>> = newQueue<Sent>()
  // how many exports to refuse before one works; -1 refuses for ever
  failures:  Atomic<i64> = Atomic(value: 0)
  retryable: bool = true
  calls:     Atomic<i64> = Atomic(value: 0)

  implement Exporter {
    fun export(signal: Signal, body: List<u8>) suspends throws ExportError {
      this.calls.add(1)
      val left = this.failures.load()
      if (left != 0) {
        if (left > 0) {
          this.failures.sub(1)
        }
        throw ExportError(message: "collector down", retryable: this.retryable)
      }
      this.seen.withLock(q => {
        q.push(Sent(signal, body))
      })
    }
  }

  fun bodies(signal: Signal): List<List<u8>> =>
    this.seen.withLock(q => q.filter(s => s.signal == signal).map(s => s.body))
}

// every span of every trace export, as its fields, oldest first
test fun spansOf(rec: Recorder): List<List<Fld>> {
  val out: MutableList<List<Fld>> = []
  loop (body in rec.bodies(Signal.Traces)) {
    val scopeSpans = child(child(fields(body), 1), 2)
    loop (s in all(scopeSpans, 2)) {
      out.push(fields(s.data))
    }
  }
  out.toList()
}

// every metric of every metrics export
test fun metricsOf(rec: Recorder): List<List<Fld>> {
  val out: MutableList<List<Fld>> = []
  loop (body in rec.bodies(Signal.Metrics)) {
    loop (scopeMetrics in all(child(fields(body), 1), 2)) {
      loop (m in all(fields(scopeMetrics.data), 2)) {
        out.push(fields(m.data))
      }
    }
  }
  out.toList()
}

test fun metricNamed(rec: Recorder, name: string): List<Fld>? =>
  metricsOf(rec).filter(m => text(m, 1) == name).last()

test fun logsOf(rec: Recorder): List<List<Fld>> {
  val out: MutableList<List<Fld>> = []
  loop (body in rec.bodies(Signal.Logs)) {
    val scopeLogs = child(child(fields(body), 1), 2)
    loop (r in all(scopeLogs, 2)) {
      out.push(fields(r.data))
    }
  }
  out.toList()
}

test fun spanNamed(rec: Recorder, name: string): List<Fld> {
  val found = spansOf(rec).filter(s => text(s, 5) == name)
  expect(found.len() == 1)
  found.at(0) ?: []
}

test fun bytesOf(fs: List<Fld>, num: i64): List<u8> => (all(fs, num).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data

// the registry is the process's: a test that made thousands of series drops them
// so that the tests after it do not carry them in every export
test fun forgetInstruments() {
  registry.withLock(all => {
    all.clear()
  })
}

test fun quiet(): Duration => Duration.seconds(3600)

// ---- before start ----

test "before start everything is a no-op" {
  with permit = gate.acquire()
  with s = span("nothing")
  expect(!s.recording())
  expect(current() == null)
  expect(traceparent() == null)
  expect(traceId() == null)
  val hits = meter("t.noop").counter("t.noop.hits")
  hits.add(5)
  expect(collectMetrics().filter(m => m.name == "t.noop.hits").isEmpty())
  logRecord(9, "INFO", "ignored", [])
  expect(logQueue.withLock(q => q.len()) == 0)
}

// ---- spans ----

test fun nested(): List<List<u8>> {
  with outer = span("outer", attrs: [attr("route", "/x")], kind: SpanKind.Server)
  outer.event("started")
  with inner = span("inner")
  inner.set(attr("n", 3))
  inner.failWith("bad thing")
  [outer.context().spanId, inner.context().spanId, outer.context().traceId, inner.context().traceId]
}

test "spans nest under the current span and are exported when they end" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "notes", exporter: rec, interval: quiet())
  val ids = nested()
  expect(tel.flush())
  val outer = spanNamed(rec, "outer")
  val inner = spanNamed(rec, "inner")
  expect(bytesOf(outer, 2) == (ids.at(0) ?: []))
  expect(bytesOf(inner, 2) == (ids.at(1) ?: []))
  // one trace, the inner span's parent is the outer one, the outer has none
  expect(bytesOf(outer, 1) == bytesOf(inner, 1))
  expect(bytesOf(inner, 4) == bytesOf(outer, 2))
  expect(all(outer, 4).isEmpty())
  expect(number(outer, 6) == 2)
  expect(number(inner, 6) == 1)
  expect(number(inner, 8) >= number(inner, 7))
  expect(number(outer, 8) >= number(inner, 8))
  expect(text(child(outer, 9), 1) == "route")
  expect(text(child(outer, 11), 2) == "started")
  // the inner span failed: status error with the message, and an exception event
  val status = child(inner, 15)
  expect(number(status, 3) == 2)
  expect(text(status, 2) == "bad thing")
  expect(text(child(inner, 11), 2) == "exception")
  expect(all(outer, 15).isEmpty())
}

test "the resource says which service this is" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "notes", exporter: rec, interval: quiet(), resource: [attr("deployment.environment", "test")])
  expect(nested().len() == 4)
  expect(tel.flush())
  val resource = child(child(fields(rec.bodies(Signal.Traces).at(0) ?: []), 1), 1)
  val names = all(resource, 1).map(kv => text(fields(kv.data), 1))
  expect(names.contains("service.name"))
  expect(names.contains("service.instance.id"))
  expect(names.contains("telemetry.sdk.language"))
  expect(names.contains("deployment.environment"))
  val service = all(resource, 1).filter(kv => text(fields(kv.data), 1) == "service.name")
  expect(text(child(fields((service.at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data), 2), 1) == "notes")
}

test fun ended(): Span {
  val s = span("early")
  s.end()
  s
}

test "ending twice is one span; a span can end before its block" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val s = ended()
  s.end()
  s.set(attr("late", true))
  expect(tel.flush())
  expect(spansOf(rec).len() == 1)
  expect(all(spanNamed(rec, "early"), 9).isEmpty())
}

test fun discarded() {
  with s = span("probe")
  s.discard()
}

test "a discarded span is not exported" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  discarded()
  expect(tel.flush())
  expect(spansOf(rec).isEmpty())
}

test fun childTask(): List<List<u8>> {
  with parent = span("parent")
  var seen: List<u8> = []
  var traceSeen: List<u8> = []
  scope {
    val t = async whoIsCurrent()
    val got = await t
    seen = got.at(0) ?: []
    traceSeen = got.at(1) ?: []
  }
  [parent.context().spanId, seen, parent.context().traceId, traceSeen]
}

test fun whoIsCurrent(): List<List<u8>> {
  val h = current()
  if (h == null) return [[], []]
  [h.context.spanId, h.context.traceId]
}

test "a task started inside a span sees it as the current one" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val ids = childTask()
  expect(ids.at(0) == ids.at(1))
  expect(ids.at(2) == ids.at(3))
  expect(current() == null)
}

test fun withTraceparent(): string? {
  with s = span("send")
  traceparent()
}

test "the traceparent to send on is the current span's" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val header = withTraceparent() ?: fail("no traceparent inside a span")
  val ctx = SpanContext.parse(header) ?: fail("the traceparent we write does not parse")
  expect(ctx.sampled)
  expect(traceparent() == null)
}

test fun continued(remote: SpanContext): List<u8> {
  with s = span("served", kind: SpanKind.Server, parent: remote)
  s.context().traceId
}

test "a span continues a trace that came from another service" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet(), sampleRatio: 0.0)
  val remote = SpanContext.parse("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01") ?: fail("refused")
  val trace = continued(remote)
  expect(trace == remote.traceId)
  expect(tel.flush())
  // the ratio is 0, but the caller had decided to record: that decision stands
  val served = spanNamed(rec, "served")
  expect(bytesOf(served, 4) == remote.spanId)
  expect(bytesOf(served, 1) == remote.traceId)
  expect(number(served, 6) == 2)
}

test fun unsampled(): (bool, bool) {
  with a = span("root")
  with b = span("child")
  (a.recording(), b.recording())
}

test "a trace that is sampled out records nothing, and its children follow" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet(), sampleRatio: 0.0)
  val (root, child) = unsampled()
  expect(!root)
  expect(!child)
  expect(tel.flush())
  expect(spansOf(rec).isEmpty())
}

test fun traced(fail: bool): i64 throws Boom {
  if (fail) throw Boom()
  7
}

error Boom { }

test "inSpan marks success ok and a thrown error failed, and lets the error go on" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  expect(inSpan("good", () => try traced(false)) is Ok)
  expect(inSpan("bad", () => try traced(true)) is Err)
  expect(tel.flush())
  expect(number(child(spanNamed(rec, "good"), 15), 3) == 1)
  val status = child(spanNamed(rec, "bad"), 15)
  expect(number(status, 3) == 2)
}

// ---- metrics ----

test "a counter adds per attribute set, whatever the order of the attributes" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val hits = meter("t.counter").counter("t.counter.hits", unit: "1", description: "hits")
  hits.add(1, [attr("a", "x"), attr("b", "y")])
  hits.add(2, [attr("b", "y"), attr("a", "x")])
  hits.add(5, [attr("a", "other")])
  hits.add(4)
  expect(tel.flush())
  val m = metricNamed(rec, "t.counter.hits") ?: fail("the counter was not exported")
  expect(text(m, 2) == "hits")
  val sum = child(m, 7)
  expect(number(sum, 3) == 1)
  val points = all(sum, 1).map(p => fields(p.data))
  expect(points.len() == 3)
  val values = points.map(p => number(p, 6)).sorted()
  expect(values == [3, 4, 5])
}

test "asking for an instrument twice is the same instrument" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  meter("t.same").counter("t.same.n").add(1)
  meter("t.same").counter("t.same.n").add(1)
  expect(tel.flush())
  val sum = child(metricNamed(rec, "t.same.n") ?: [], 7)
  expect(all(sum, 1).len() == 1)
  expect(number(child(sum, 1), 6) == 2)
}

test "a counter never goes down, and a name is one kind of instrument" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val c = meter("t.refuse").counter("t.refuse.n")
  expectPanics(() => c.add(-1))
  expectPanics(() => meter("t.refuse").gauge("t.refuse.n"))
  expectPanics(() => meter("t.refuse").histogram("t.refuse.h", buckets: [1.0, 1.0]))
  expectPanics(() => meter("t.refuse").histogram("t.refuse.h2", buckets: [2.0, 1.0]))
}

test "an up-down counter goes both ways, a gauge keeps the last value" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val open = meter("t.ud").upDownCounter("t.ud.open")
  open.add(3)
  open.add(-5)
  val level = meter("t.ud").gauge("t.ud.level")
  level.set(1.5)
  level.set(2.5)
  expect(tel.flush())
  val sum = child(metricNamed(rec, "t.ud.open") ?: [], 7)
  expect(number(child(sum, 1), 6) == -2)
  expect(all(sum, 3).isEmpty())
  val gauge = child(metricNamed(rec, "t.ud.level") ?: [], 5)
  val point = child(gauge, 1)
  expect(bytesOf(point, 4) == [0, 0, 0, 0, 0, 0, 4, 64])
}

test "a histogram counts values into buckets and keeps sum, least and greatest" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val h = meter("t.hist").histogram("t.hist.latency", unit: "s", buckets: [1.0, 2.0])
  h.record(0.5)
  h.record(1.0)
  h.record(1.5)
  h.record(Duration.seconds(5))
  h.record(3)
  expect(tel.flush())
  val data = child(metricNamed(rec, "t.hist.latency") ?: [], 9)
  val p = child(data, 1)
  expect(number(p, 4) == 5)
  // bounds 1 and 2: [0.5, 1.0] in the first, [1.5] in the second, [5, 3] past them
  val buckets = bytesOf(p, 6)
  expect(buckets.len() == 24)
  expect(buckets.at(0) == 2)
  expect(buckets.at(8) == 1)
  expect(buckets.at(16) == 2)
  expect(bytesOf(p, 7).len() == 16)
}

test "a histogram of seconds takes a Duration as seconds" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val h = meter("t.hist").histogram("t.hist.seconds", unit: "s")
  h.record(Duration.millis(250))
  expect(tel.flush())
  val p = child(child(metricNamed(rec, "t.hist.seconds") ?: [], 9), 1)
  // sum is 0.25 as a double: 0x3FD0000000000000
  expect(bytesOf(p, 5) == [0, 0, 0, 0, 0, 0, 208, 63])
}

test "a label with endless values is folded into one series" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  val c = meter("t.card").counter("t.card.n")
  loop (i in 0..<2100) {
    c.add(1, [attr("user", i)])
  }
  expect(tel.flush())
  val sum = child(metricNamed(rec, "t.card.n") ?: [], 7)
  // 2000 series and the overflow one
  expect(all(sum, 1).len() == 2001)
  forgetInstruments()
}

// ---- logs ----

test fun logged() {
  with s = span("handling")
  logRecord(17, "ERROR", "went wrong", [attr("code", 5)])
}

test "a log record carries the trace of the span it was written in" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  logged()
  logRecord(9, "INFO", "outside", [])
  expect(tel.flush())
  val records = logsOf(rec)
  expect(records.len() == 2)
  val inside = records.at(0) ?: []
  val outside = records.at(1) ?: []
  expect(text(child(inside, 5), 1) == "went wrong")
  expect(number(inside, 2) == 17)
  expect(bytesOf(inside, 9).len() == 16)
  expect(bytesOf(inside, 10).len() == 8)
  expect(bytesOf(outside, 9).isEmpty())
  expect(bytesOf(inside, 9) == bytesOf(spanNamed(rec, "handling"), 1))
}

// ---- export ----

test "an export that fails is tried again with a pause, and then goes through" {
  with permit = gate.acquire()
  val rec = Recorder(failures: Atomic(value: 2))
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  expect(nested().len() == 4)
  expect(tel.flush())
  expect(spansOf(rec).len() == 2)
  // two refusals and the try that worked, for the spans; the metrics had their own
  expect(rec.calls.load() >= 3)
}

test "a refusal is not retried, and the data is dropped and counted" {
  with permit = gate.acquire()
  val rec = Recorder(failures: Atomic(value: -1), retryable: false)
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  expect(nested().len() == 4)
  expect(!tel.flush())
  // one try for the spans and one for the metrics: a retry would make it four or more
  expect(rec.calls.load() <= 2)
  expect(lost.load() == 2)
  expect(spansOf(rec).isEmpty())
}

test fun manySpans(n: i64) {
  loop (i in 0..<n) {
    with s = span("s")
    s.set(attr("i", i))
  }
}

test "a full queue drops what does not fit, and says how much" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet(), maxQueue: 3)
  manySpans(10)
  expect(tel.flush())
  expect(spansOf(rec).len() == 3)
  expect(lost.load() == 7)
  val m = metricNamed(rec, "otel.dropped") ?: fail("the drop count was not exported")
  expect(number(child(child(m, 7), 1), 6) == 7)
}

test "the interval exports without being asked" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: Duration.millis(100))
  expect(nested().len() == 4)
  await sleep(Duration.millis(700))
  expect(spansOf(rec).len() == 2)
}

test fun startAgain(rec: Recorder): bool throws StartError {
  with second = try start(service: "again", exporter: rec)
  true
}

test "one pipeline at a time; shutdown flushes and ends it" {
  with permit = gate.acquire()
  val rec = Recorder()
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  expect(startAgain(rec) is Err)
  expect(nested().len() == 4)
  expect(tel.shutdown())
  expect(spansOf(rec).len() == 2)
  // after shutdown the instruments are no-ops again
  expect(!span("late").recording())
}

test fun recordAndLeave(rec: Recorder) throws StartError {
  with tel = try start(service: "t", exporter: rec, interval: quiet())
  expect(nested().len() == 4)
}

test "the end of the with sends what is still queued, waiting for the exporter" {
  with permit = gate.acquire()
  val rec = Recorder()
  try recordAndLeave(rec)
  // the close flushed (D147), and the next start may run
  expect(spansOf(rec).len() == 2)
  expect(!running.load())
}

test "start refuses settings that cannot work" {
  with permit = gate.acquire()
  expectPanics(() => checkStart(1.5, 10, Duration.seconds(1)))
  expectPanics(() => checkStart(-0.1, 10, Duration.seconds(1)))
  expectPanics(() => checkStart(1.0, 0, Duration.seconds(1)))
  expectPanics(() => checkStart(1.0, 10, Duration.millis(1)))
  expect(!running.load())
}
