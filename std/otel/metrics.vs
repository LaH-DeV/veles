// Metrics: counters, gauges and histograms, aggregated in place and read out
// by the exporter once per interval. Aggregation is cumulative: a point is
// the total since the instrument was made, so a lost export loses nothing.
use time

/// What a histogram records: a number, a count or a duration (in seconds).
public trait Measure {
  fun measure(): f64
}

implement Measure for f64 {
  fun measure(): f64 => this
}

implement Measure for i64 {
  fun measure(): f64 => this.toF64()
}

implement Measure for Duration {
  fun measure(): f64 => this.toNanos().toF64() / 1000000000.0
}

// an instrument's data: one aggregate per attribute set
struct Series {
  attrs:     List<Attr>
  intValue:  i64
  realValue: f64
  count:     i64
  buckets:   List<i64>
  min:       f64
  max:       f64
}

// at most this many attribute sets per instrument; the rest are folded into
// one overflow series, so a label taken from user input cannot grow memory
const maxSeries: i64 = 2000

struct Core {
  meter:       string
  name:        string
  description: string
  unit:        string
  kind:        MetricKind
  integer:     bool
  bounds:      List<f64>
  startNanos:  i64
  series:      Mutex<MutableMap<string, Series>>

  fun update(attrs: List<Attr>, change: fun(Series): Series) {
    if (!enabled.load()) return
    val key = seriesKey(attrs)
    this.series.withLock(m => {
      var at = key
      var shown = attrs
      if (m.get(key) == null && m.len() >= maxSeries) {
        at = "\u{1f}overflow"
        shown = [attr("otel.metric.overflow", true)]
      }
      val before = m.get(at) ?: Series(attrs: shown, intValue: 0, realValue: 0.0, count: 0, buckets: this.zeroBuckets(), min: 0.0, max: 0.0)
      m.set(at, change(before))
    })
  }

  fun zeroBuckets(): List<i64> =>
    if (this.kind == MetricKind.Histogram) zeros(this.bounds.len() + 1) else []

  // a snapshot for the exporter; null when nothing was recorded
  fun snapshot(): MetricData? {
    val points = this.series.withLock(m => m.values().map(s => PointData(
      attrs: s.attrs,
      intValue: s.intValue,
      realValue: s.realValue,
      count: s.count,
      buckets: s.buckets,
      min: s.min,
      max: s.max,
    )))
    if (points.isEmpty()) return null
    MetricData(
      meter: this.meter,
      name: this.name,
      description: this.description,
      unit: this.unit,
      kind: this.kind,
      integer: this.integer,
      bounds: this.bounds,
      startNanos: this.startNanos,
      points,
    )
  }
}

fun zeros(n: i64): List<i64> {
  val out: MutableList<i64> = []
  loop (_ in 0..<n) {
    out.push(0)
  }
  out.toList()
}

val registry: Mutex<MutableList<Core>> = newQueue<Core>()

// the instrument already made under this name, or a new one: asking twice
// for "requests" is one counter, not two
fun instrument(meter: string, name: string, description: string, unit: string, kind: MetricKind, integer: bool, bounds: List<f64>): Core {
  registry.withLock(all => {
    val existing = all.find(c => c.meter == meter && c.name == name)
    if (existing != null) {
      if (existing.kind != kind) panic("otel: '$name' is already a ${kindName(existing.kind)}; it cannot also be a ${kindName(kind)}")
      existing
    } else {
      val empty: MutableMap<string, Series> = [:]
      val core = Core(meter, name, description, unit, kind, integer, bounds, startNanos: unixNanos(), series: Mutex(value: empty))
      all.push(core)
      core
    }
  })
}

fun kindName(k: MetricKind): string => when (k) {
  MetricKind.Counter       => "counter"
  MetricKind.UpDownCounter => "up-down counter"
  MetricKind.Gauge         => "gauge"
  MetricKind.Histogram     => "histogram"
}

// every instrument that has recorded something
fun collectMetrics(): List<MetricData> {
  val cores = registry.withLock(all => all.toList())
  val out: MutableList<MetricData> = []
  loop (c in cores) {
    val m = c.snapshot()
    if (m != null) out.push(m)
  }
  out.toList()
}

/// The explicit boundaries a histogram uses when none are given: seconds,
/// from 5 ms to 10 s — what request latencies need.
public val defaultBuckets: List<f64> = [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]

/// A source of instruments, named after what is measured (`"notes"`): its
/// name is the instrumentation scope in the exported data. Asking for the
/// same instrument twice gives the same one.
///
/// ```veles
/// val notes = otel.meter("notes")
/// val created = notes.counter("notes.created", unit: "1")
/// val latency = notes.histogram("notes.save", unit: "s")
/// ```
public struct Meter {
  name: string

  /// A counter of whole numbers that only goes up: requests served, bytes
  /// sent. Panics when the name is already another kind of instrument.
  public fun counter(name: string, unit: string = "", description: string = ""): Counter =>
    Counter(core: instrument(this.name, name, description, unit, MetricKind.Counter, true, []))

  /// A count that goes up and down: connections open, jobs queued.
  public fun upDownCounter(name: string, unit: string = "", description: string = ""): UpDownCounter =>
    UpDownCounter(core: instrument(this.name, name, description, unit, MetricKind.UpDownCounter, true, []))

  /// A value read at a moment: the last one set is what is exported.
  public fun gauge(name: string, unit: string = "", description: string = ""): Gauge =>
    Gauge(core: instrument(this.name, name, description, unit, MetricKind.Gauge, false, []))

  /// A distribution: how many values fell in each bucket, with their sum,
  /// least and greatest. `buckets` are the upper bounds in ascending order
  /// (`defaultBuckets` suits seconds); a value above the last is counted in
  /// one more bucket past it. Panics when the bounds are not ascending.
  public fun histogram(name: string, unit: string = "", description: string = "", buckets: List<f64> = defaultBuckets): Histogram {
    loop (i in 1..<buckets.len()) {
      if (buckets.at(i) <= (buckets.at(i - 1) ?: 0.0)) panic("otel: histogram '$name' needs ascending bucket bounds")
    }
    Histogram(core: instrument(this.name, name, description, unit, MetricKind.Histogram, false, buckets))
  }
}

/// The source of instruments for `name`.
public fun meter(name: string): Meter => Meter(name)

/// A count that only goes up.
public struct Counter {
  core: Core

  /// Adds `n` (not negative — a count that went down is an up-down counter),
  /// for the series `attrs` names. A label from user input makes a series per
  /// value: keep the number of different sets small.
  public fun add(n: i64, attrs: List<Attr> = []) {
    if (n < 0) panic("otel: counter '${this.core.name}' only goes up; add($n) is negative")
    this.core.update(attrs, s => Series(attrs: s.attrs, intValue: s.intValue + n, realValue: s.realValue, count: s.count, buckets: s.buckets, min: s.min, max: s.max))
  }
}

/// A count that goes up and down.
public struct UpDownCounter {
  core: Core

  /// Adds `n`, which may be negative.
  public fun add(n: i64, attrs: List<Attr> = []) {
    this.core.update(attrs, s => Series(attrs: s.attrs, intValue: s.intValue + n, realValue: s.realValue, count: s.count, buckets: s.buckets, min: s.min, max: s.max))
  }
}

/// A value read at a moment.
public struct Gauge {
  core: Core

  /// Sets the value for the series `attrs` names.
  public fun set(value: f64, attrs: List<Attr> = []) {
    this.core.update(attrs, s => Series(attrs: s.attrs, intValue: s.intValue, realValue: value, count: s.count, buckets: s.buckets, min: s.min, max: s.max))
  }
}

/// A distribution of values.
public struct Histogram {
  core: Core

  /// Records one value: a number, a count or a `Duration` (in seconds, the
  /// unit OpenTelemetry's conventions use for time).
  public fun record<T: Measure>(value: T, attrs: List<Attr> = []) {
    val x = value.measure()
    val bounds = this.core.bounds
    this.core.update(attrs, s => {
      // the first bucket whose bound holds the value, else the one past the last
      var at = bounds.len()
      loop (i in 0..<bounds.len()) {
        if (x <= (bounds.at(i) ?: 0.0)) {
          at = i
          break
        }
      }
      val counts: MutableList<i64> = []
      loop (i in 0..<s.buckets.len()) {
        val n = s.buckets.at(i) ?: 0
        counts.push(if (i == at) n + 1 else n)
      }
      Series(
        attrs: s.attrs,
        intValue: s.intValue,
        realValue: s.realValue + x,
        count: s.count + 1,
        buckets: counts.toList(),
        min: if (s.count == 0 || x < s.min) x else s.min,
        max: if (s.count == 0 || x > s.max) x else s.max,
      )
    })
  }
}
