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
pub fun now(): i64 = unsafe {
  veles_time_now_ms()
}

/// A monotonic reading in milliseconds; differences between readings are
/// elapsed time, the value itself means nothing.
pub fun monotonic(): i64 = unsafe {
  veles_time_monotonic_ns()
} / 1000000

/// A monotonic reading in nanoseconds.
pub fun monotonicNanos(): i64 = unsafe {
  veles_time_monotonic_ns()
}

/// Measures elapsed time: `val sw = time.Stopwatch.start()`, then
/// `sw.elapsedMillis()`.
pub struct Stopwatch {
  started: i64

  /// A stopwatch running from now.
  pub static fun start(): Stopwatch = Stopwatch(started: monotonicNanos())

  pub fun elapsedNanos(): i64 = monotonicNanos() - self.started
  pub fun elapsedMillis(): i64 = self.elapsedNanos() / 1000000
  pub fun elapsedSeconds(): f64 = (self.elapsedNanos() as f64) / 1000000000.0

  /// Starts over.
  pub mut fun reset() {
    self.started = monotonicNanos()
  }
}

/// A calendar date and time of day; `weekday` is 0 for Sunday, `yearDay`
/// 1 for January 1st.
pub struct DateTime {
  pub year:    i64
  pub month:   i64
  pub day:     i64
  pub hour:    i64
  pub minute:  i64
  pub second:  i64
  pub millis:  i64
  pub weekday: i64
  pub yearDay: i64

  impl Display {
    /// ISO 8601 without a zone: `2026-09-17T12:34:56.789`.
    fun toString(): string = "${self.date()}T${self.time()}"
  }

  /// `2026-09-17`
  pub fun date(): string = "${pad(self.year, 4)}-${pad(self.month, 2)}-${pad(self.day, 2)}"

  /// `12:34:56.789`
  pub fun time(): string = "${pad(self.hour, 2)}:${pad(self.minute, 2)}:${pad(self.second, 2)}.${pad(self.millis, 3)}"
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
pub fun utc(ms: i64): DateTime = civil(ms, false)

/// The calendar fields of `ms` in the local time zone.
pub fun local(ms: i64): DateTime = civil(ms, true)
