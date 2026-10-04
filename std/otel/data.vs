// What the three signals are made of, as the pipeline holds them between the
// moment something happens and the moment it is exported.

/// How a span relates to the work around it, as the OpenTelemetry
/// specification defines it. The value is the number on the wire.
public enum SpanKind {
  /// Work inside the process (the default).
  Internal = 1
  /// A request this process is answering.
  Server = 2
  /// A request this process is making.
  Client = 3
  /// A message put on a queue.
  Producer = 4
  /// A message taken from a queue.
  Consumer = 5
}

// a span's outcome; the value is the number on the wire
enum SpanStatus {
  Unset = 0
  Ok    = 1
  Error = 2
}

struct SpanEvent {
  timeNanos: i64
  name:      string
  attrs:     List<Attr>
}

// a finished span, as exported
struct SpanData {
  traceId:       List<u8>
  spanId:        List<u8>
  parentId:      List<u8>
  name:          string
  kind:          SpanKind
  startNanos:    i64
  endNanos:      i64
  attrs:         List<Attr>
  events:        List<SpanEvent>
  status:        SpanStatus
  statusMessage: string
}

enum MetricKind {
  Counter
  UpDownCounter
  Gauge
  Histogram
}

// one attribute set's aggregate, cumulative since the instrument was made
struct PointData {
  attrs:     List<Attr>
  intValue:  i64
  realValue: f64
  count:     i64
  buckets:   List<i64>
  min:       f64
  max:       f64
}

// an instrument's points, as exported
struct MetricData {
  meter:       string
  name:        string
  description: string
  unit:        string
  kind:        MetricKind
  // a counter of whole numbers is sent as integers
  integer:    bool
  bounds:     List<f64>
  startNanos: i64
  points:     List<PointData>
}

// a log record, with the trace it was written in
struct LogData {
  timeNanos:    i64
  severity:     i64
  severityText: string
  body:         string
  attrs:        List<Attr>
  traceId:      List<u8>
  spanId:       List<u8>
}
