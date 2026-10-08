// The OTLP messages, written in Protocol Buffers' wire format (D126). This is
// a small encoder for exactly the messages OpenTelemetry's collectors read —
// no schema compiler and no general protobuf module. The field numbers are
// those of opentelemetry-proto (trace.proto, metrics.proto, logs.proto,
// common.proto, resource.proto).

// wire types
const wireVarint: i64 = 0
const wireFixed64: i64 = 1
const wireBytes: i64 = 2

// One message being written. `u64` throughout: a protobuf varint is unsigned
// on the wire, and a negative `int64` is its two's complement, ten bytes long.
struct Pb {
  out: MutableList<u8> = []

  fun varint(n: u64) {
    var rest = n
    loop (rest >= 128) {
      this.out.push(((rest & 127) | 128).wrapU8())
      rest = rest >> 7
    }
    this.out.push(rest.wrapU8())
  }

  fun tag(field: i64, wire: i64) {
    this.varint((field * 8 + wire).wrapU64())
  }

  // proto3 leaves a zero out; so do these
  fun uint(field: i64, n: i64) {
    if (n == 0) return
    this.tag(field, wireVarint)
    this.varint(n.wrapU64())
  }

  fun int(field: i64, n: i64) {
    if (n == 0) return
    this.tag(field, wireVarint)
    this.varint(n.wrapU64())
  }

  fun bool(field: i64, b: bool) {
    if (!b) return
    this.tag(field, wireVarint)
    this.varint(1)
  }

  // a value that must be there even when it is zero (a oneof member)
  fun intAlways(field: i64, n: i64) {
    this.tag(field, wireVarint)
    this.varint(n.wrapU64())
  }

  fun fixed64(field: i64, n: i64) {
    this.tag(field, wireFixed64)
    this.out.pushI64Le(n)
  }

  fun fixed32(field: i64, n: i64) {
    if (n == 0) return
    this.tag(field, 5)
    this.out.pushU32Le(n.wrapU32())
  }

  fun double(field: i64, x: f64) {
    this.tag(field, wireFixed64)
    this.out.pushU64Le(x.toBits())
  }

  fun bytes(field: i64, data: List<u8>) {
    if (data.isEmpty()) return
    this.tag(field, wireBytes)
    this.varint(data.len().wrapU64())
    this.out.addAll(data)
  }

  fun string(field: i64, s: string) {
    this.bytes(field, s.bytes())
  }

  // a nested message, written even when empty
  fun message(field: i64, sub: List<u8>) {
    this.tag(field, wireBytes)
    this.varint(sub.len().wrapU64())
    this.out.addAll(sub)
  }

  // proto3 packs a repeated number: one length-delimited run
  fun packedFixed64(field: i64, xs: List<i64>) {
    if (xs.isEmpty()) return
    val run = Pb()
    loop (x in xs) {
      run.out.pushI64Le(x)
    }
    this.message(field, run.done())
  }

  fun packedDouble(field: i64, xs: List<f64>) {
    if (xs.isEmpty()) return
    val run = Pb()
    loop (x in xs) {
      run.out.pushU64Le(x.toBits())
    }
    this.message(field, run.done())
  }

  fun done(): List<u8> => this.out.toList()
}

// ---------------------------------------------------------------------------
// common.proto, resource.proto

fun anyValue(v: AnyValue): List<u8> {
  val pb = Pb()
  when (v) {
    is StringValue  => {
      // a oneof member is written even when empty
      pb.tag(1, wireBytes)
      pb.varint(v.value.len().wrapU64())
      pb.out.addAll(v.value.bytes())
    }
    is BoolValue    => {
      pb.tag(2, wireVarint)
      pb.varint(if (v.value) 1 else 0)
    }
    is IntValue     => pb.intAlways(3, v.value)
    is DoubleValue  => pb.double(4, v.value)
    is StringsValue => {
      val array = Pb()
      loop (s in v.values) {
        array.message(1, anyValue(StringValue(value: s)))
      }
      pb.message(5, array.done())
    }
  }
  pb.done()
}

fun keyValue(a: Attr): List<u8> {
  val pb = Pb()
  pb.string(1, a.key)
  pb.message(2, anyValue(a.value))
  pb.done()
}

// `field` repeated once per attribute
fun attributes(pb: Pb, field: i64, attrs: List<Attr>) {
  loop (a in attrs) {
    pb.message(field, keyValue(a))
  }
}

fun resource(attrs: List<Attr>): List<u8> {
  val pb = Pb()
  attributes(pb, 1, attrs)
  pb.done()
}

fun instrumentation(name: string, version: string): List<u8> {
  val pb = Pb()
  pb.string(1, name)
  pb.string(2, version)
  pb.done()
}

// ---------------------------------------------------------------------------
// trace.proto

fun spanBytes(s: SpanData): List<u8> {
  val pb = Pb()
  pb.bytes(1, s.traceId)
  pb.bytes(2, s.spanId)
  pb.bytes(4, s.parentId)
  pb.string(5, s.name)
  pb.int(6, s.kind.value)
  pb.fixed64(7, s.startNanos)
  pb.fixed64(8, s.endNanos)
  attributes(pb, 9, s.attrs)
  loop (e in s.events) {
    val ev = Pb()
    ev.fixed64(1, e.timeNanos)
    ev.string(2, e.name)
    attributes(ev, 3, e.attrs)
    pb.message(11, ev.done())
  }
  if (s.status != SpanStatus.Unset) {
    val st = Pb()
    st.string(2, s.statusMessage)
    st.int(3, s.status.value)
    pb.message(15, st.done())
  }
  pb.done()
}

/// An ExportTraceServiceRequest for `spans` from `res`.
fun traceRequest(res: List<Attr>, spans: List<SpanData>): List<u8> {
  val scopeSpans = Pb()
  scopeSpans.message(1, instrumentation(sdkName, sdkVersion))
  loop (s in spans) {
    scopeSpans.message(2, spanBytes(s))
  }
  val resourceSpans = Pb()
  resourceSpans.message(1, resource(res))
  resourceSpans.message(2, scopeSpans.done())
  val request = Pb()
  request.message(1, resourceSpans.done())
  request.done()
}

// ---------------------------------------------------------------------------
// metrics.proto

// aggregation temporality: cumulative
const cumulative: i64 = 2

fun numberPoint(p: PointData, startNanos: i64, nowNanos: i64, asInt: bool): List<u8> {
  val pb = Pb()
  attributes(pb, 7, p.attrs)
  pb.fixed64(2, startNanos)
  pb.fixed64(3, nowNanos)
  if (asInt) pb.fixed64(6, p.intValue) else pb.double(4, p.realValue)
  pb.done()
}

fun histogramPoint(p: PointData, bounds: List<f64>, startNanos: i64, nowNanos: i64): List<u8> {
  val pb = Pb()
  attributes(pb, 9, p.attrs)
  pb.fixed64(2, startNanos)
  pb.fixed64(3, nowNanos)
  pb.fixed64(4, p.count)
  pb.double(5, p.realValue)
  pb.packedFixed64(6, p.buckets)
  pb.packedDouble(7, bounds)
  if (p.count > 0) {
    pb.double(11, p.min)
    pb.double(12, p.max)
  }
  pb.done()
}

fun metricBytes(m: MetricData, startNanos: i64, nowNanos: i64): List<u8> {
  val pb = Pb()
  pb.string(1, m.name)
  pb.string(2, m.description)
  pb.string(3, m.unit)
  when (m.kind) {
    MetricKind.Counter, MetricKind.UpDownCounter => {
      val sum = Pb()
      loop (p in m.points) {
        sum.message(1, numberPoint(p, startNanos, nowNanos, m.integer))
      }
      sum.int(2, cumulative)
      sum.bool(3, m.kind == MetricKind.Counter)
      pb.message(7, sum.done())
    }
    MetricKind.Gauge => {
      val gauge = Pb()
      loop (p in m.points) {
        gauge.message(1, numberPoint(p, startNanos, nowNanos, false))
      }
      pb.message(5, gauge.done())
    }
    MetricKind.Histogram => {
      val h = Pb()
      loop (p in m.points) {
        h.message(1, histogramPoint(p, m.bounds, startNanos, nowNanos))
      }
      h.int(2, cumulative)
      pb.message(9, h.done())
    }
  }
  pb.done()
}

/// An ExportMetricsServiceRequest for `metrics` from `res`, one scope per
/// meter name, in the order the meters first appear.
fun metricsRequest(res: List<Attr>, metrics: List<MetricData>, nowNanos: i64): List<u8> {
  val resourceMetrics = Pb()
  resourceMetrics.message(1, resource(res))
  val scopes: MutableList<string> = []
  loop (m in metrics) {
    if (!scopes.contains(m.meter)) scopes.push(m.meter)
  }
  loop (name in scopes) {
    val scopeMetrics = Pb()
    scopeMetrics.message(1, instrumentation(name, sdkVersion))
    loop (m in metrics) {
      if (m.meter == name) scopeMetrics.message(2, metricBytes(m, m.startNanos, nowNanos))
    }
    resourceMetrics.message(2, scopeMetrics.done())
  }
  val request = Pb()
  request.message(1, resourceMetrics.done())
  request.done()
}

// ---------------------------------------------------------------------------
// logs.proto

fun logBytes(r: LogData): List<u8> {
  val pb = Pb()
  pb.fixed64(1, r.timeNanos)
  pb.int(2, r.severity)
  pb.string(3, r.severityText)
  pb.message(5, anyValue(StringValue(value: r.body)))
  attributes(pb, 6, r.attrs)
  pb.bytes(9, r.traceId)
  pb.bytes(10, r.spanId)
  pb.fixed64(11, r.timeNanos)
  pb.done()
}

/// An ExportLogsServiceRequest for `records` from `res`.
fun logsRequest(res: List<Attr>, records: List<LogData>): List<u8> {
  val scopeLogs = Pb()
  scopeLogs.message(1, instrumentation(sdkName, sdkVersion))
  loop (r in records) {
    scopeLogs.message(2, logBytes(r))
  }
  val resourceLogs = Pb()
  resourceLogs.message(1, resource(res))
  resourceLogs.message(2, scopeLogs.done())
  val request = Pb()
  request.message(1, resourceLogs.done())
  request.done()
}

const sdkName = "veles"
const sdkVersion = "1"
