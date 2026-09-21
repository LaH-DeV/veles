/// Clocks and calendar time. Times are `i64` milliseconds since the Unix
/// epoch (UTC), the unit `sleep` and timers use; `monotonic()` is for
/// measuring, `now()` for telling.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44).

extern "C" {
  fun veles_time_now_ms(): i64
  fun veles_time_monotonic_ns(): i64
  fun veles_time_civil(ms: i64, local: bool, out: *raw string)
}

/// Milliseconds since 1970-01-01T00:00:00Z.
public fun now(): i64 = unsafe {
  veles_time_now_ms()
}

/// A monotonic reading in milliseconds; differences between readings are
/// elapsed time, the value itself means nothing.
public fun monotonic(): i64 = unsafe {
  veles_time_monotonic_ns()
} / 1000000

/// A monotonic reading in nanoseconds.
public fun monotonicNanos(): i64 = unsafe {
  veles_time_monotonic_ns()
}

/// Measures elapsed time: `val sw = time.Stopwatch.start()`, then
/// `sw.elapsedMillis()`.
public struct Stopwatch {
  var started: i64

  /// A stopwatch running from now.
  public static fun start(): Stopwatch = Stopwatch(started: monotonicNanos())

  public fun elapsedNanos(): i64 = monotonicNanos() - self.started
  public fun elapsedMillis(): i64 = self.elapsedNanos() / 1000000
  public fun elapsedSeconds(): f64 = (self.elapsedNanos() as f64) / 1000000000.0

  /// Starts over.
  public fun reset() {
    self.started = monotonicNanos()
  }
}

/// A calendar date and time of day; `weekday` is 0 for Sunday, `yearDay`
/// 1 for January 1st.
public struct DateTime {
  public year:    i64
  public month:   i64
  public day:     i64
  public hour:    i64
  public minute:  i64
  public second:  i64
  public millis:  i64
  public weekday: i64
  public yearDay: i64

  implement Display {
    /// ISO 8601 without a zone: `2026-09-17T12:34:56.789`.
    fun toString(): string = "${self.date()}T${self.time()}"
  }

  /// `2026-09-17`
  public fun date(): string = "${pad(self.year, 4)}-${pad(self.month, 2)}-${pad(self.day, 2)}"

  /// `12:34:56.789`
  public fun time(): string = "${pad(self.hour, 2)}:${pad(self.minute, 2)}:${pad(self.second, 2)}.${pad(self.millis, 3)}"
}

fun pad(n: i64, width: i64): string = n.toString().padStart(width, "0")

fun civil(ms: i64, local: bool): DateTime {
  var text = ""
  unsafe {
    veles_time_civil(ms, local, &text)
  }
  val f = text.split(" ").map(x => x.toInt() ?: 0)
  var millis = ms % 1000
  if (millis < 0) millis += 1000
  DateTime(year: f.atOrPanic(0), month: f.atOrPanic(1), day: f.atOrPanic(2), hour: f.atOrPanic(3), minute: f.atOrPanic(4), second: f.atOrPanic(5), millis, weekday: f.atOrPanic(6), yearDay: f.atOrPanic(7))
}

/// The calendar fields of `ms` in UTC.
public fun utc(ms: i64): DateTime = civil(ms, false)

/// The calendar fields of `ms` in the local time zone.
public fun local(ms: i64): DateTime = civil(ms, true)
