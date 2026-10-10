/// OpenTelemetry (D126): traces, metrics and logs, exported as OTLP.
///
/// Three kinds of evidence about a running service, in the format every
/// collector reads. *Traces* show where one request spent its time, as
/// spans that nest; *metrics* show how the service is doing over time;
/// *logs* are the `std/log` lines, each with the id of the trace it was
/// written in, so a slow request, its log lines and its failure are one click
/// apart.
///
/// ```veles
/// with tel = try otel.start(service: "notes", exporter: http.otlp(endpoint: "http://collector:4318"))
/// val created = otel.meter("notes").counter("notes.created", unit: "1")
///
/// fun load(id: i64): Note throws db.Error {
///   with span = otel.span("load note", attrs: [otel.attr("note.id", id)])
///   ...
/// }
/// // at the end of main:
/// try tel.shutdown()
/// ```
///
/// Before `start` (and in a test) every span, instrument and log record is a
/// no-op that costs one load and a branch, so a library can be instrumented
/// without asking whether anyone is listening. After it, finished spans and
/// log records queue (up to `maxQueue` of each — then they are dropped and
/// counted, never allowed to grow), and every `interval` a background task
/// sends what has gathered, with the metrics' current totals.
///
/// `std/http` carries a trace across services by itself: `http.serve` opens a
/// span per request from an incoming `traceparent` header, and `http.fetch`
/// opens one per call and sends the header on.
use io { eprintln }
use time

/// Which of the three signals an export carries.
public enum Signal {
  Traces
  Metrics
  Logs
}

/// The path an OTLP/HTTP collector serves `signal` on.
public fun signalPath(signal: Signal): string => when (signal) {
  Signal.Traces  => "/v1/traces"
  Signal.Metrics => "/v1/metrics"
  Signal.Logs    => "/v1/logs"
}

/// An export that did not go through.
public error ExportError {
  public message: string
  /// Whether sending the same data again may work (a collector that is busy
  /// or unreachable), as against data the collector refuses.
  public retryable: bool = true
}

/// Where telemetry goes: one OTLP request body (Protocol Buffers) at a time.
/// `http.otlp(endpoint:)` is the HTTP exporter; a test writes its own to see
/// what would have been sent.
public trait Exporter : Sendable {
  fun export(signal: Signal, body: List<u8>) suspends throws ExportError
}

/// `start` was called while another pipeline was running.
public error StartError {
  public message: string
}

// ---------------------------------------------------------------------------
// logs

val logQueue: Mutex<MutableList<LogData>> = newQueue<LogData>()

/// What `std/log` hands over for each line it writes; `severity` is
/// OpenTelemetry's number (5 debug, 9 info, 13 warn, 17 error). Nothing
/// happens before `start`.
public fun logRecord(severity: i64, severityText: string, body: string, attrs: List<Attr>) {
  if (!enabled.load()) return
  val s = currentSpan.get()
  val record = LogData(
    timeNanos: unixNanos(),
    severity,
    severityText,
    body,
    attrs,
    traceId: s?.context?.traceId ?: [],
    spanId: s?.context?.spanId ?: [],
  )
  val limit = settings.withLock(c => c.maxQueue)
  val kept = logQueue.withLock(q => {
    if (q.len() >= limit) {
      false
    } else {
      q.push(record)
      true
    }
  })
  if (!kept) {
    lost.add(1)
  }
}

// ---------------------------------------------------------------------------
// the pipeline

// whether a pipeline is running: one per process
val running = Atomic(value: false)

// whether the last export failed, so that a down collector is reported once
// when it goes down and once when it is back, not at every interval
val failing = Atomic(value: false)

/// A running pipeline: the background task that exports, and what it needs to
/// finish the job. Received with `with`: the end of the block stops the task
/// and sends what is still queued, waiting for the exporter (D147).
/// `shutdown()` does the same earlier, and says whether it went through.
public struct Telemetry {
  exporter:   Exporter
  resource:   List<Attr>
  startNanos: i64
  runner:     Task<()>

  /// Sends everything that has gathered now, without waiting for the
  /// interval; the spans and logs that could not be sent are dropped and
  /// counted. Returns whether every part went through.
  public fun flush(): bool suspends => flushAll(this.exporter, this.resource)

  /// Sends what is left, stops recording, and lets the next `start` happen.
  /// Call it once, at the end of the program or of a graceful stop; the
  /// instruments become no-ops again.
  public fun shutdown(): bool suspends {
    val ok = flushAll(this.exporter, this.resource)
    enabled.store(false)
    ok
  }

  implement Closeable {
    fun close() suspends {
      // the last interval's data, unless shutdown() sent it already; the
      // exporter's own timeouts bound the wait
      if (enabled.load()) {
        val _ = flushAll(this.exporter, this.resource)
      }
      enabled.store(false)
      running.store(false)
    }
  }
}

// the settings `start` refuses, as a caller's bug
fun checkStart(sampleRatio: f64, maxQueue: i64, interval: Duration) {
  if (sampleRatio < 0.0 || sampleRatio > 1.0) panic("otel.start: sampleRatio is a fraction from 0 to 1, got $sampleRatio")
  if (maxQueue < 1) panic("otel.start: maxQueue must be at least 1, got $maxQueue")
  if (interval < Duration.millis(10)) panic("otel.start: interval must be at least 10 ms, got $interval")
}

/// Starts the pipeline: from here on spans, metrics and log records are kept
/// and, every `interval`, sent to `exporter`. Only one pipeline runs at a
/// time; a second `start` throws `StartError`.
///
/// - `service` names this program in every backend (`service.name`).
/// - `resource` adds attributes that describe it (`deployment.environment`,
///   `service.version`).
/// - `sampleRatio` is the fraction of new traces that are recorded; a trace
///   that arrives with a decision keeps it. Panics outside 0 to 1.
/// - `maxQueue` bounds how many finished spans and how many log records wait
///   for the next export.
///
/// ```veles
/// with tel = try otel.start(service: "notes", exporter: http.otlp(endpoint: cfg.otlpEndpoint), sampleRatio: 0.1)
/// ```
public fun start(
  service: string,
  exporter: Exporter,
  interval: Duration = Duration.seconds(10),
  resource: List<Attr> = [],
  sampleRatio: f64 = 1.0,
  maxQueue: i64 = 2048,
): Telemetry throws StartError {
  checkStart(sampleRatio, maxQueue, interval)
  if (running.swap(true)) throw StartError(message: "otel.start: a pipeline is already running; shut it down first")
  settings.withLock(c => {
    c.sampleRatio = sampleRatio
    c.maxQueue = maxQueue
  })
  spanQueue.withLock(q => q.clear())
  logQueue.withLock(q => q.clear())
  lost.store(0)
  val attrs = [attr("service.name", service), attr("service.instance.id", hexOf(randomBytes(8))), attr("telemetry.sdk.name", sdkName), attr("telemetry.sdk.language", "veles"), attr("telemetry.sdk.version", sdkVersion)].concat(resource)
  enabled.store(true)
  Telemetry(exporter, resource: attrs, startNanos: unixNanos(), runner: async exportLoop(exporter, attrs, interval))
}

fun exportLoop(exporter: Exporter, resource: List<Attr>, interval: Duration) {
  loop {
    await sleep(interval)
    val _ = flushAll(exporter, resource)
  }
}

// every signal, now; whether all of it was sent
fun flushAll(exporter: Exporter, resource: List<Attr>): bool suspends {
  var ok = true
  val spans = spanQueue.withLock(q => {
    val all = q.toList()
    q.clear()
    all
  })
  if (!spans.isEmpty() && !sendWithRetry(exporter, Signal.Traces, traceRequest(resource, spans))) {
    ok = false
    val _ = lost.update(n => n + spans.len())
  }
  val logs = logQueue.withLock(q => {
    val all = q.toList()
    q.clear()
    all
  })
  if (!logs.isEmpty() && !sendWithRetry(exporter, Signal.Logs, logsRequest(resource, logs))) {
    ok = false
    val _ = lost.update(n => n + logs.len())
  }
  val metrics = collectMetrics().concat(selfMetrics())
  if (!metrics.isEmpty() && !sendWithRetry(exporter, Signal.Metrics, metricsRequest(resource, metrics, unixNanos()))) {
    ok = false
  }
  noteOutcome(ok)
  ok
}

// how much was dropped, as a metric of its own, so a collector sees it
fun selfMetrics(): List<MetricData> {
  val n = lost.load()
  if (n == 0) return []
  [MetricData(
    meter: "veles.otel",
    name: "otel.dropped",
    description: "spans and log records dropped because a queue was full or the collector was unreachable",
    unit: "1",
    kind: MetricKind.Counter,
    integer: true,
    bounds: [],
    startNanos: 0,
    points: [PointData(attrs: [], intValue: n, realValue: 0.0, count: 0, buckets: [], min: 0.0, max: 0.0)],
  )]
}

// up to four tries, 200 ms, 400 ms, 800 ms apart; a refusal is not retried
fun sendWithRetry(exporter: Exporter, signal: Signal, body: List<u8>): bool suspends {
  var attempt: i64 = 0
  loop {
    when (exporter.export(signal, body)) {
      is Ok(_)  => return true
      is Err(e) => {
        if (!e.retryable || attempt >= 3) {
          reportFailure(e.message)
          return false
        }
      }
    }
    attempt += 1
    await sleep(Duration.millis(100 * (1 << attempt)))
  }
}

fun reportFailure(message: string) {
  if (!failing.swap(true)) eprintln("otel: export failed ($message); data is dropped until the collector is back")
}

fun noteOutcome(ok: bool) {
  if (ok && failing.swap(false)) eprintln("otel: export works again")
}
