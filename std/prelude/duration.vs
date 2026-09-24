// Prelude — a length of time. In scope in every file (D24).
//
// A timeout, a retry delay, an elapsed measurement and a cache lifetime are
// all the same thing, and before this type they were all a bare `i64` whose
// unit lived in a parameter name: `withTimeout(5000, ...)` meant
// milliseconds, `Stopwatch.elapsedNanos()` meant nanoseconds, and
// `withTimeout(5, ...)` — five seconds, surely — compiled and gave up after
// five thousandths of one. The unit belongs in the type, for the same reason
// a MAC is a `Digest` and not a `List<u8>` (D59): the mistake that matters
// is the one that compiles.
//
// `Duration` lives in the prelude, not in `std/time`, because the things
// that take one are here: `sleep`, `withTimeout`, `Timeout`. A *point* in
// time is a different question and lives in `std/time` — `Timestamp` on the
// wall clock, `Deadline` on the monotonic one.

/// The largest `i64`. Used to recognise the most negative one without
/// writing a literal that would have to be negated to be written.
val nanosMax: i64 = 9223372036854775807

/// A length of time, to the nanosecond.
///
/// ```veles
/// try withTimeout(Duration.seconds(5), () => try conn.readLine(max: 8192))
/// val limits = http.Limits(headerTimeout: Duration.seconds(10))
/// if (sw.elapsed() > Duration.millis(250)) log.warn("slow")
/// ```
///
/// The unit inside is nanoseconds, so the range is ±292 years and a
/// microbenchmark and a cache lifetime are the same type. A constructor
/// handed more than the range holds overflows, which is checked in a debug
/// build and wraps in release (D21).
///
/// Arithmetic is methods — `a.plus(b)`, not `a + b` — because Veles has no
/// arithmetic operator traits today. The two sums that get written most,
/// "now plus a timeout" and "how much is left", are not written here at all:
/// they are `time.Deadline`. Should the language gain operator overloading,
/// `plus`, `minus`, `times` and `dividedBy` are named to be adopted by it
/// without a second spelling appearing.
public struct Duration {
  private ns: i64

  /// No time at all. `Duration.zero` is the identity of `plus` and the value
  /// `isZero` reports.
  public static val zero: Duration = Duration(ns: 0)

  public static fun nanos(n: i64): Duration = Duration(ns: n)
  public static fun micros(n: i64): Duration = Duration(ns: n * 1000)
  public static fun millis(n: i64): Duration = Duration(ns: n * 1000000)
  public static fun seconds(n: i64): Duration = Duration(ns: n * 1000000000)
  public static fun minutes(n: i64): Duration = Duration(ns: n * 60000000000)
  public static fun hours(n: i64): Duration = Duration(ns: n * 3600000000000)
  public static fun days(n: i64): Duration = Duration(ns: n * 86400000000000)

  /// A fractional number of seconds — `Duration.ofSeconds(0.25)`. The
  /// product is rounded to the nearest nanosecond; `seconds` is the exact
  /// one, and what a literal count of seconds should use.
  public static fun ofSeconds(v: f64): Duration {
    val scaled = v * 1000000000.0
    val rounded: f64 = if (scaled < 0.0) scaled - 0.5 else scaled + 0.5
    Duration(ns: rounded as i64)
  }

  /// The whole nanoseconds. Every other accessor is derived from this one.
  public fun toNanos(): i64 = self.ns

  /// Truncated toward zero, so `Duration.nanos(-1500).toMicros()` is -1.
  public fun toMicros(): i64 = self.ns / 1000
  public fun toMillis(): i64 = self.ns / 1000000
  public fun toSeconds(): i64 = self.ns / 1000000000
  public fun toMinutes(): i64 = self.ns / 60000000000
  public fun toHours(): i64 = self.ns / 3600000000000
  public fun toDays(): i64 = self.ns / 86400000000000

  /// Seconds with the fraction kept — for a rate, a ratio or a report.
  /// `toSeconds()` is the truncating one.
  public fun asSeconds(): f64 = (self.ns as f64) / 1000000000.0

  /// Milliseconds with the fraction kept.
  public fun asMillis(): f64 = (self.ns as f64) / 1000000.0

  public fun plus(other: Duration): Duration = Duration(ns: self.ns + other.ns)
  public fun minus(other: Duration): Duration = Duration(ns: self.ns - other.ns)
  public fun times(n: i64): Duration = Duration(ns: self.ns * n)

  /// Truncated toward zero, as integer division is; `dividedBy(0)` panics
  /// for the same reason `1 / 0` does.
  public fun dividedBy(n: i64): Duration = Duration(ns: self.ns / n)

  /// How many times `other` fits in this one, truncated. `Duration.zero`
  /// divides nothing and panics.
  public fun over(other: Duration): i64 = self.ns / other.ns

  public fun negated(): Duration = Duration(ns: 0 - self.ns)

  /// The length without its sign. The single most negative `Duration` has no
  /// positive counterpart, so it saturates at the largest one rather than
  /// overflowing — the same answer Go gives.
  public fun abs(): Duration =
    if (self.ns >= 0) self
    else if (self.ns + 1 == 0 - nanosMax) Duration(ns: nanosMax)
    else Duration(ns: 0 - self.ns)

  public fun isZero(): bool = self.ns == 0
  public fun isNegative(): bool = self.ns < 0

  /// The shorter of the two — what a caller writes when a deadline and a
  /// configured limit both apply.
  public fun min(other: Duration): Duration = if (self.ns <= other.ns) self else other

  /// The longer of the two.
  public fun max(other: Duration): Duration = if (self.ns >= other.ns) self else other

  implement Comparable {
    fun compareTo(other: Duration): Ordering = self.ns.compareTo(other.ns)
  }

  implement Display {
    /// The largest units that fit, smallest first omitted: `0s`, `250ms`,
    /// `1.5s`, `2m30s`, `1d1h`, `-90ms`. Below a second one unit is used
    /// with its exact fraction (`1.000001ms`); from a minute up the whole
    /// units are spelled out and zero components dropped.
    ///
    /// Every form this prints is one `Duration.parse` reads back to the same
    /// value: the fraction is never rounded, because nanoseconds divide each
    /// unit exactly.
    fun toString(): string {
      if (self.ns == 0) return "0s"
      val sign = if (self.ns < 0) "-" else ""
      val n = self.abs().toNanos()
      if (n < 1000) return "$sign${n}ns"
      if (n < 1000000) return "$sign${decimal(n / 1000, n % 1000, 3)}µs"
      if (n < 1000000000) return "$sign${decimal(n / 1000000, n % 1000000, 6)}ms"
      if (n < 60000000000) return "$sign${decimal(n / 1000000000, n % 1000000000, 9)}s"
      var rest = n
      var out = sign
      val days = rest / 86400000000000
      if (days > 0) {
        out += "${days}d"
        rest -= days * 86400000000000
      }
      val hours = rest / 3600000000000
      if (hours > 0) {
        out += "${hours}h"
        rest -= hours * 3600000000000
      }
      val mins = rest / 60000000000
      if (mins > 0) {
        out += "${mins}m"
        rest -= mins * 60000000000
      }
      if (rest > 0) out += "${decimal(rest / 1000000000, rest % 1000000000, 9)}s"
      out
    }
  }

  implement Parsable {
    /// A run of `<number><unit>` parts, largest first: `90s`, `1h30m`,
    /// `250ms`, `1.5s`, `-2m30s`, `0`. The units are `ns`, `us` (`µs`),
    /// `ms`, `s`, `m`, `h` and `d`; a fraction is read in whole nanoseconds,
    /// never through a float, so no digit is lost and nothing is rounded.
    ///
    /// `null` for empty text, a number with no unit, a unit that is not one
    /// of those, a fraction finer than a nanosecond, or a total that does
    /// not fit. A bare `0` is the one number allowed without a unit, because
    /// zero has no unit.
    static fun parse(s: string): Duration? = parseDuration(s)
  }
}

/// `whole.frac` with `digits` decimal places and trailing zeros removed; the
/// point goes too when the fraction is zero.
fun decimal(whole: i64, frac: i64, digits: i64): string {
  if (frac == 0) return whole.toString()
  var f = frac.toString().padStart(digits, "0")
  loop (f.endsWith("0")) {
    f = f.substring(0, f.len() - 1) ?: f
  }
  "$whole.$f"
}

/// Nanoseconds in one of the units `Duration.parse` accepts, or -1.
fun unitNanos(u: string): i64 = when (u) {
  "ns" => 1
  "us" => 1000
  "µs" => 1000
  "ms" => 1000000
  "s"  => 1000000000
  "m"  => 60000000000
  "h"  => 3600000000000
  "d"  => 86400000000000
  else => -1
}

fun isAsciiDigit(b: u8): bool = b >= '0' && b <= '9'

/// The body of `Duration.parse`; see its documentation for the grammar.
fun parseDuration(s: string): Duration? {
  if (s.isEmpty()) return null
  var i: i64 = 0
  var neg = false
  if (s.byteAt(0) == '-' || s.byteAt(0) == '+') {
    neg = s.byteAt(0) == '-'
    i = 1
  }
  if (i >= s.len()) return null
  if (s.substring(i, s.len()) == "0") return Duration.zero
  var total: i64 = 0
  var parts: i64 = 0
  loop (i < s.len()) {
    // the whole part, refused as soon as it passes what an i64 of
    // nanoseconds can hold rather than wrapping there (D21)
    val startDigits = i
    var whole: i64 = 0
    loop (i < s.len() && isAsciiDigit(s.byteAt(i))) {
      if (whole > nanosMax / 10) return null
      whole = whole * 10 + ((s.byteAt(i) - '0') as i64)
      i += 1
    }
    // the fraction, in whole decimal places, so no float is involved and
    // `frac` is always smaller than `den`
    var fracDigits: i64 = 0
    var frac: i64 = 0
    var den: i64 = 1
    if (i < s.len() && s.byteAt(i) == '.') {
      i += 1
      if (i >= s.len() || !isAsciiDigit(s.byteAt(i))) return null  // a point with no digits
      loop (i < s.len() && isAsciiDigit(s.byteAt(i))) {
        if (fracDigits < 9) {
          frac = frac * 10 + ((s.byteAt(i) - '0') as i64)
          den *= 10
          fracDigits += 1
        } else if (s.byteAt(i) != '0') {
          return null  // finer than a nanosecond
        }
        i += 1
      }
    }
    if (i == startDigits) return null  // no digits at all
    // the unit
    val startUnit = i
    loop (i < s.len() && !isAsciiDigit(s.byteAt(i)) && s.byteAt(i) != '.' &&
      s.byteAt(i) != '-' && s.byteAt(i) != '+') {
        i += 1
      }
    if (i == startUnit) return null  // a number with no unit
    val scale = unitNanos(s.substring(startUnit, i) ?: return null)
    if (scale < 0) return null
    if (whole > nanosMax / scale) return null
    var add = whole * scale
    if (fracDigits > 0) {
      // `frac < den`, so both products below stay inside an i64
      val exact = if (scale % den == 0) frac * (scale / den) else frac * scale / den
      if (scale % den != 0 && frac * scale % den != 0) return null
      add += exact
    }
    if (total > nanosMax - add) return null
    total += add
    parts += 1
  }
  if (parts == 0) return null
  if (neg) Duration(ns: 0 - total) else Duration(ns: total)
}
