// Traces: spans, the current span, and the W3C trace context that carries a
// trace from one service to the next.
use crypto
use time

// ---------------------------------------------------------------------------
// the switch and the settings

// Until `start`, every instrument is a no-op that costs this one load.
val enabled = Atomic(value: false)

struct Settings {
  var sampleRatio: f64 = 1.0
  var maxQueue:    i64 = 2048
}

val settings = Mutex(value: Settings())

val spanQueue: Mutex<MutableList<SpanData>> = newQueue<SpanData>()

// how many items were dropped because a queue was full or the collector
// could not be reached
val lost = Atomic(value: 0)

fun newQueue<T>(): Mutex<MutableList<T>> {
  val empty: MutableList<T> = []
  Mutex(value: empty)
}

fun unixNanos(): i64 => time.now().toMicros() * 1000

// ---------------------------------------------------------------------------
// identifiers and the trace context

// from the operating system's generator: ids that two processes started in the
// same microsecond must not share (D126), which a clock-seeded one cannot promise
fun randomBytes(n: i64): List<u8> {
  loop {
    val bytes = crypto.randomBytes(n)
    // all zeros is not a valid id
    if (bytes.any(b => b != 0)) return bytes
  }
}

fun hexOf(bytes: List<u8>): string {
  val digits = "0123456789abcdef".bytes()
  val out: MutableList<u8> = []
  loop (b in bytes) {
    out.push(digits.at(b.toI64() / 16) ?: '0')
    out.push(digits.at(b.toI64() % 16) ?: '0')
  }
  out.toList().decodeUtf8() ?: ""
}

// lower-case hexadecimal of exactly `digits` characters, as bytes; null for anything else
fun bytesOfHex(text: string, digits: i64): List<u8>? {
  if (text.len() != digits) return null
  val out: MutableList<u8> = []
  val chars = text.bytes()
  var i: i64 = 0
  loop (i < chars.len()) {
    val hi = hexNibble(chars.at(i))
    val lo = hexNibble(chars.at(i + 1) ?: return null)
    if (hi < 0 || lo < 0) return null
    out.push((hi * 16 + lo).wrapU8())
    i += 2
  }
  out.toList()
}

// only lower-case digits are trace context (W3C §3.2)
fun hexNibble(b: u8): i64 {
  if (b >= '0' && b <= '9') return (b - '0').toI64()
  if (b >= 'a' && b <= 'f') return (b - 'a').toI64() + 10
  -1
}

/// Where a span sits in a trace: enough to continue it in another span, in
/// another task or in another service (`traceparent`).
public struct SpanContext {
  public traceId: List<u8>
  public spanId:  List<u8>
  /// Whether this trace is being recorded; an unsampled trace is still
  /// passed on, so every service in it makes the same choice.
  public sampled: bool

  /// The trace id as 32 hex digits.
  public fun traceIdHex(): string => hexOf(this.traceId)

  /// The span id as 16 hex digits.
  public fun spanIdHex(): string => hexOf(this.spanId)

  /// The W3C `traceparent` header value: `00-<trace id>-<span id>-<flags>`.
  public fun traceparent(): string =>
    "00-${this.traceIdHex()}-${this.spanIdHex()}-${if (this.sampled) "01" else "00"}"

  /// Reads a `traceparent` header. Anything that is not exactly the W3C form —
  /// wrong lengths, upper-case digits, an all-zero id, version `ff` — is
  /// `null`, and the trace starts afresh rather than half-continuing.
  public static fun parse(text: string): SpanContext? {
    val parts = text.split("-")
    if (parts.len() < 4) return null
    val version = bytesOfHex(parts.at(0), 2) ?: return null
    if (version.at(0) == 255) return null
    // version 00 is exactly four fields; a later version may add more
    if (version.at(0) == 0 && parts.len() != 4) return null
    val traceId = bytesOfHex(parts.at(1), 32) ?: return null
    val spanId = bytesOfHex(parts.at(2), 16) ?: return null
    val flags = bytesOfHex(parts.at(3), 2) ?: return null
    if (traceId.all(b => b == 0) || spanId.all(b => b == 0)) return null
    SpanContext(traceId, spanId, sampled: ((flags.at(0) ?: 0) & 1) == 1)
  }
}

// parent-based, with a ratio for a trace that has no parent: the same trace
// id always gets the same answer, so every service that sees it agrees
fun sampleTrace(traceId: List<u8>, ratio: f64): bool {
  if (ratio >= 1.0) return true
  if (ratio <= 0.0) return false
  var bits: u64 = 0
  loop (i in 0..<8) {
    bits = (bits << 8) | (traceId.at(8 + i) ?: 0).toU64()
  }
  ((bits >> 11).toF64() / 9007199254740992.0) < ratio
}

// ---------------------------------------------------------------------------
// spans

// the part of a span that changes while it runs
struct OpenSpan {
  var name:          string
  var attrs:         List<Attr> = []
  var events:        List<SpanEvent> = []
  var status:        SpanStatus = SpanStatus.Unset
  var statusMessage: string = ""
  var ended:         bool = false
  var discarded:     bool = false
}

val currentSpan = TaskLocal<SpanHandle?>(fallback: null)

/// A span that can be shared: the one running in this task (`otel.current()`),
/// or the one `span.handle()` gives a child task. It records into the same
/// span as its owner, and cannot end it.
public struct SpanHandle {
  /// Where this span sits in its trace.
  public context: SpanContext
  parentId:       List<u8>
  kind:           SpanKind
  startNanos:     i64
  // null when nothing is recorded: not started, or sampled out
  data: Mutex<OpenSpan>?

  /// Whether this span records anything. A sampled-out span does not, but
  /// still carries the trace along.
  public fun recording(): bool => this.data != null

  /// Adds an attribute (`otel.attr("note.id", id)`); setting a name again
  /// adds another value, the last one wins at the collector.
  public fun set(a: Attr) {
    val d = this.data ?: return
    d.withLock(s => {
      if (!s.ended) s.attrs = s.attrs.concat([a])
    })
  }

  /// Adds something that happened at a moment inside the span.
  public fun event(name: string, attrs: List<Attr> = []) {
    val d = this.data ?: return
    val at = unixNanos()
    d.withLock(s => {
      if (!s.ended) s.events = s.events.concat([SpanEvent(timeNanos: at, name, attrs)])
    })
  }

  /// Marks the span failed, with `error`'s message: status Error, and an
  /// `exception` event carrying the message.
  public fun fail(error: Error) {
    this.failWith(error.message())
  }

  /// Marks the span failed with `message`.
  public fun failWith(message: string) {
    val d = this.data ?: return
    val at = unixNanos()
    d.withLock(s => {
      if (s.ended) return
      s.status = SpanStatus.Error
      s.statusMessage = message
      s.events = s.events.concat([SpanEvent(timeNanos: at, name: "exception", attrs: [attr("exception.message", message)])])
    })
  }

  /// Marks the span as having succeeded; a span never marked has no status,
  /// which backends show as unset.
  public fun ok() {
    val d = this.data ?: return
    d.withLock(s => {
      if (!s.ended && s.status != SpanStatus.Error) s.status = SpanStatus.Ok
    })
  }

  /// Changes the span's name — for one that is named before what it is
  /// about is known, such as a request before its route is.
  public fun rename(name: string) {
    val d = this.data ?: return
    d.withLock(s => {
      if (!s.ended) s.name = name
    })
  }

  // ends the span and hands it to the pipeline; once only
  fun finish() {
    val d = this.data ?: return
    val at = unixNanos()
    val closed = d.withLock(s => {
      if (s.ended) {
        null
      } else {
        s.ended = true
        if (s.discarded) null else SpanData(
          traceId: this.context.traceId,
          spanId: this.context.spanId,
          parentId: this.parentId,
          name: s.name,
          kind: this.kind,
          startNanos: this.startNanos,
          endNanos: at,
          attrs: s.attrs,
          events: s.events,
          status: s.status,
          statusMessage: s.statusMessage,
        )
      }
    })
    val done = closed ?: return
    queueSpan(done)
  }

  // the span will not be exported (see `Span.discard`)
  fun discard() {
    val d = this.data ?: return
    d.withLock(s => {
      s.discarded = true
    })
  }
}

fun queueSpan(s: SpanData) {
  val limit = settings.withLock(c => c.maxQueue)
  val kept = spanQueue.withLock(q => {
    if (q.len() >= limit) {
      false
    } else {
      q.push(s)
      true
    }
  })
  if (!kept) {
    val _ = lost.update(n => n + 1)
  }
}

/// A span being run: made by `otel.span`, held with `with`, which ends it
/// when the block does — by a return, a throw, a panic or a cancel — and
/// puts the previous span back as the current one.
///
/// ```veles
/// with span = otel.span("load note", attrs: [otel.attr("note.id", id)])
/// span.event("cache miss")
/// ```
public struct Span {
  /// The shareable form: what a child task is given, and what carries the
  /// context.
  public handle: SpanHandle
  binding:       LocalBinding?

  /// Where this span sits in its trace.
  public fun context(): SpanContext => this.handle.context

  /// See `SpanHandle.recording`.
  public fun recording(): bool => this.handle.recording()

  /// See `SpanHandle.set`.
  public fun set(a: Attr) {
    this.handle.set(a)
  }

  /// See `SpanHandle.event`.
  public fun event(name: string, attrs: List<Attr> = []) {
    this.handle.event(name, attrs)
  }

  /// See `SpanHandle.fail`.
  public fun fail(error: Error) {
    this.handle.fail(error)
  }

  /// See `SpanHandle.failWith`.
  public fun failWith(message: string) {
    this.handle.failWith(message)
  }

  /// See `SpanHandle.ok`.
  public fun ok() {
    this.handle.ok()
  }

  /// See `SpanHandle.rename`.
  public fun rename(name: string) {
    this.handle.rename(name)
  }

  /// Ends the span now, instead of at the end of its block (the end of the
  /// block does nothing more). Its duration stops here.
  public fun end() {
    this.handle.finish()
  }

  /// Ends the span and leaves it out of what is exported: a health probe's
  /// request, say, which nobody wants to see a thousand times a day.
  public fun discard() {
    this.handle.discard()
    this.handle.finish()
  }

  implement Closeable {
    fun close() {
      this.handle.finish()
      val b = this.binding
      if (b != null) b.close()
    }
  }
}

fun noSpan(): Span {
  val ctx = SpanContext(traceId: [], spanId: [], sampled: false)
  Span(handle: SpanHandle(context: ctx, parentId: [], kind: SpanKind.Internal, startNanos: 0, data: null), binding: null)
}

/// Starts a span and makes it the current one until the `with` that holds it
/// ends. Spans started meanwhile — here, or in tasks started here — are its
/// children, and log lines written meanwhile carry its trace id.
///
/// `parent` continues a trace that came from elsewhere (see
/// `SpanContext.parse`); without one, the current span is the parent, and
/// with neither a new trace begins. Before `otel.start` this makes nothing
/// and costs one load.
///
/// ```veles
/// with span = otel.span("charge card", attrs: [otel.attr("amount", 1999)])
/// ```
public fun span(name: string, attrs: List<Attr> = [], kind: SpanKind = SpanKind.Internal, parent: SpanContext? = null): Span {
  if (!enabled.load()) return noSpan()
  val inherited = currentSpan.get()
  val fromTask = inherited?.context
  val parentContext: SpanContext? = if (parent != null) parent else fromTask
  val traceId = parentContext?.traceId ?: randomBytes(16)
  val ratio = settings.withLock(c => c.sampleRatio)
  val sampled = if (parentContext != null) parentContext.sampled else sampleTrace(traceId, ratio)
  val context = SpanContext(traceId, spanId: randomBytes(8), sampled)
  val data: Mutex<OpenSpan>? = if (sampled) Mutex(value: OpenSpan(name, attrs)) else null
  val handle = SpanHandle(context, parentId: parentContext?.spanId ?: [], kind, startNanos: unixNanos(), data)
  Span(handle, binding: currentSpan.bind(handle))
}

/// How many spans and log records have been dropped since `start`, because
/// a queue was full or the collector could not be reached.
public fun dropped(): i64 => lost.load()

/// Whether a pipeline is running (`start` was called and `shutdown` was not):
/// what a caller asks before it builds something only a pipeline would use.
public fun active(): bool => enabled.load()

/// The span running in this task (or the task that started it), as a handle
/// that can be passed on; `null` outside any span.
public fun current(): SpanHandle? => currentSpan.get()

/// The `traceparent` value that continues the current trace, for a request
/// this task is about to send; `null` outside any span or before `start`.
public fun traceparent(): string? {
  val s = currentSpan.get() ?: return null
  s.context.traceparent()
}

/// The current trace's id as 32 hex digits, for a log line or an error report;
/// `null` outside any span.
public fun traceId(): string? {
  val s = currentSpan.get() ?: return null
  s.context.traceIdHex()
}

/// Runs `f` inside a span and returns what it returns. A thrown error marks
/// the span failed (and goes on its way); returning marks it ok. This is
/// the form for work that can fail — a `with` block cannot see an error
/// that is on its way out.
///
/// ```veles
/// val note = try otel.inSpan("load note", () => try db.find(id))
/// ```
public fun inSpan<R, E: Error>(name: string, f: fun(): R suspends throws E, attrs: List<Attr> = [], kind: SpanKind = SpanKind.Internal): R throws E {
  with s = span(name, attrs, kind)
  do {
    val v = try f()
    s.ok()
    return v
  } catch (e) {
    s.failWith(e.message())
    throw e
  }
}
