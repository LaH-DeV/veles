/// Clocks, the calendar and the two text formats a server speaks.
///
/// There are two clocks and they are different kinds of thing, so they are
/// different types. The **wall clock** tells you what time it is: it is what
/// a log line, a JSON body and an `exp` claim carry, it is comparable across
/// machines, and it can jump backwards when NTP corrects it. That is
/// `Timestamp`. The **monotonic clock** only ever goes forward and its
/// readings mean nothing on their own; it is what you measure with. It is
/// never handed out as a value — the only two things anyone builds from it
/// are `Stopwatch` and `Deadline`, so a monotonic reading and a wall-clock
/// reading cannot be confused, added or compared by accident.
///
/// A *length* of time is `Duration`, which is in the prelude because
/// `sleep` and `withTimeout` take one.
///
/// ```veles
/// val t = time.now()                                  // Timestamp
/// io.println("$t")                                    // 2026-09-24T09:15:02.481Z
/// io.println("${t.local()}")                          // 2026-09-24T11:15:02.481+02:00
/// io.println(time.formatHttp(t))                      // Thu, 24 Sep 2026 09:15:02 GMT
///
/// val until = time.Deadline.after(Duration.seconds(30))
/// loop (!until.expired()) { ... }
/// ```
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44). The calendar itself is pure Veles — Howard
/// Hinnant's `days_from_civil` and its inverse over the proleptic Gregorian
/// calendar — so a conversion is deterministic, needs no C round trip and
/// has no `mktime` ambiguity. The host zone is asked for one number only:
/// its offset from UTC at a given instant.

extern "C" {
  fun veles_time_now_us(): i64
  fun veles_time_monotonic_ns(): i64
  fun veles_time_local_offset_minutes(secs: i64): i64
}

/// What time it is, now.
public fun now(): Timestamp = Timestamp.now()

/// A raw monotonic reading in nanoseconds. The value means nothing; only
/// differences do. `Stopwatch` and `Deadline` are what this is for, and
/// what almost every caller should use instead.
public fun monotonicNanos(): i64 = unsafe {
  veles_time_monotonic_ns()
}

/// What the host answers when it will not convert an instant at all. Not
/// zero: zero is the offset of every machine running in UTC, and a correct
/// answer must not be spelled like a failure.
val offsetUnknown: i64 = 100000

/// Two years the host is certain to be able to convert, one of each
/// leap-year parity, for the fallback in `Offset.local`.
val referenceYear: i64 = 2019
val leapReferenceYear: i64 = 2020

fun hostOffsetMinutes(secs: i64): i64 = unsafe {
  veles_time_local_offset_minutes(secs)
}

// ---- arithmetic that rounds toward negative infinity ---------------------
//
// A point in time is a point on a line, so the second (or day) *containing*
// an instant is the one below it, on both sides of 1970. Veles' `/` and `%`
// truncate toward zero, which would put 1969-12-31T23:59:59.5Z in 1970.

fun floorDiv(a: i64, b: i64): i64 {
  val q = a / b
  if (a % b != 0 && ((a < 0) != (b < 0))) q - 1 else q
}

/// The remainder of `floorDiv`, for a positive `b`: always `0..<b`.
///
/// Written as `((a % b) + b) % b` rather than `a - floorDiv(a, b) * b`,
/// because that product overflows for an `a` near the end of the i64 range
/// — which is exactly where a `Timestamp` built from raw microseconds can
/// sit. `a % b` is already smaller than `b`, so nothing here can.
fun floorMod(a: i64, b: i64): i64 = ((a % b) + b) % b

// ---- the civil calendar ---------------------------------------------------

/// Days since 1970-01-01 from a proleptic Gregorian date. Howard Hinnant's
/// `days_from_civil`: exact for every year an i64 holds, no tables, no
/// branches per month. `m` is 1..12 and `d` is 1..31; values outside those
/// ranges are carried, so month 13 is January of the next year.
public fun daysFromCivil(year: i64, month: i64, day: i64): i64 {
  // carry an out-of-range month into the year first, so the shifted-year
  // trick below sees a real month
  val ym = year + floorDiv(month - 1, 12)
  val m = floorMod(month - 1, 12) + 1
  val y = if (m <= 2) ym - 1 else ym
  val era = floorDiv(y, 400)
  val yoe = y - era * 400  // 0..399
  val doy = (153 * (m + (if (m > 2) -3 else 9)) + 2) / 5 + day - 1
  val doe = yoe * 365 + yoe / 4 - yoe / 100 + doy  // 0..146096
  era * 146097 + doe - 719468
}

/// The inverse of `daysFromCivil`: `(year, month, day)` for a count of days
/// since 1970-01-01.
public fun civilFromDays(days: i64): (i64, i64, i64) {
  val z = days + 719468
  val era = floorDiv(z, 146097)
  val doe = z - era * 146097  // 0..146096
  val yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365  // 0..399
  val y = yoe + era * 400
  val doy = doe - (365 * yoe + yoe / 4 - yoe / 100)  // 0..365
  val mp = (5 * doy + 2) / 153                       // 0..11
  val d = doy - (153 * mp + 2) / 5 + 1               // 1..31
  val m = mp + (if (mp < 10) 3 else -9)              // 1..12
  (if (m <= 2) y + 1 else y, m, d)
}

// The edges of what a `Timestamp` holds. Every conversion from calendar
// fields goes through `instantOf`, which checks against these *before* it
// multiplies — a year read out of a JSON body or an `If-Modified-Since`
// header is a number someone else chose, and `+999999-01-01T00:00:00Z` is
// 3.2e19 microseconds, which is not a panic a server may have (D21 checks
// the overflow in a debug build and wraps in release; neither is an answer).
val maxSeconds: i64 = 9223372036854  // i64 microseconds, floored
val maxDays: i64 = 106751991         // maxSeconds / 86400

/// The instant a set of calendar fields names, or `null` when it is outside
/// the ±292,277 years a `Timestamp` holds. Out-of-range months and days
/// carry, as `daysFromCivil` defines; it is only the *magnitude* that is
/// refused.
fun instantOf(d: DateTime): Timestamp? {
  // first, so that `daysFromCivil`'s own `era * 146097` cannot overflow
  if (d.year > 300000 || d.year < -300000) return null
  val days = daysFromCivil(d.year, d.month, d.day)
  if (days > maxDays || days < 0 - maxDays) return null
  val secs = days * 86400 + d.hour * 3600 + d.minute * 60 + d.second -
    d.offset.totalMinutes() * 60
  // a second of headroom, so the microseconds added below still fit
  if (secs >= maxSeconds || secs <= 0 - maxSeconds) return null
  Timestamp(us: secs * 1000000 + d.micros)
}

/// True for a proleptic Gregorian leap year.
public fun isLeapYear(year: i64): bool =
  year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)

/// How many days the month has, 1..12; 0 for a month outside that range.
public fun daysInMonth(year: i64, month: i64): i64 = when (month) {
  1    => 31
  2    => if (isLeapYear(year)) 29 else 28
  3    => 31
  4    => 30
  5    => 31
  6    => 30
  7    => 31
  8    => 31
  9    => 30
  10   => 31
  11   => 30
  12   => 31
  else => 0
}

// ---- Timestamp ------------------------------------------------------------

/// A point on the wall clock: microseconds since 1970-01-01T00:00:00Z.
///
/// Microseconds, not milliseconds, so that RFC 3339's six fractional digits
/// and a PostgreSQL `timestamptz` survive a round trip, and two events in
/// the same millisecond still order. An i64 of them spans ±292,000 years.
///
/// `Display` and `Parsable` are RFC 3339, and so is `Codable`: a `Timestamp`
/// field in a derived struct is an RFC 3339 string on the wire, never a
/// number of microseconds.
public struct Timestamp {
  private us: i64

  /// 1970-01-01T00:00:00Z.
  public static val epoch: Timestamp = Timestamp(us: 0)

  /// What time it is, from the host's wall clock. It can go backwards
  /// between two calls; measure with `Stopwatch`, not with two of these.
  public static fun now(): Timestamp = Timestamp(us: unsafe {
    veles_time_now_us()
  })

  public static fun ofMicros(us: i64): Timestamp = Timestamp(us)
  public static fun ofMillis(ms: i64): Timestamp = Timestamp(us: ms * 1000)
  public static fun ofSeconds(s: i64): Timestamp = Timestamp(us: s * 1000000)

  /// Rounded **down**, not toward zero: these name the second, millisecond
  /// or microsecond that contains the instant, which is what a calendar
  /// conversion and a `Last-Modified` header both need, and which keeps
  /// working before 1970.
  public fun toMicros(): i64 = this.us
  public fun toMillis(): i64 = floorDiv(this.us, 1000)
  public fun toSeconds(): i64 = floorDiv(this.us, 1000000)

  /// The microsecond within the second, always 0..999999.
  public fun subsecondMicros(): i64 = floorMod(this.us, 1000000)

  // `t + d` and `t - d` (D71). The difference of two instants is a
  // `Duration` and has its own names, `since` and `until`, because the
  // operator can only mean one thing per type.
  implement Addable {
    fun plus(other: Duration): Timestamp = Timestamp(us: this.us + other.toMicros())
  }
  implement Subtractable {
    fun minus(other: Duration): Timestamp = Timestamp(us: this.us - other.toMicros())
  }

  /// How long after `earlier` this instant is; negative when it is before.
  public fun since(earlier: Timestamp): Duration =
    Duration.micros(this.us - earlier.us)

  /// How long until `later`; negative when it has passed.
  public fun until(later: Timestamp): Duration =
    Duration.micros(later.us - this.us)

  /// The calendar fields in UTC.
  public fun utc(): DateTime = this.at(Offset.utc)

  /// The calendar fields in the host's time zone, at its offset for *this*
  /// instant — so a summer timestamp reads in summer time.
  public fun local(): DateTime = this.at(Offset.local(at: this))

  /// The calendar fields at a fixed offset from UTC.
  public fun at(offset: Offset): DateTime {
    // the offset is applied to the day and the microsecond within it, not to
    // the microsecond count: `us + 18 hours` overflows for an instant near
    // the end of the i64 range, and every Timestamp has to be printable
    val whole = floorDiv(this.us, 86400000000)
    val within = floorMod(this.us, 86400000000) + offset.totalMinutes() * 60000000
    val days = whole + floorDiv(within, 86400000000)
    val rest = floorMod(within, 86400000000)
    val (y, mo, d) = civilFromDays(days)
    DateTime(
      year: y, month: mo, day: d,
      hour: rest / 3600000000,
      minute: (rest / 60000000) % 60,
      second: (rest / 1000000) % 60,
      micros: rest % 1000000,
      offset,
    )
  }

  implement Comparable {
    fun compareTo(other: Timestamp): Ordering = this.us.compareTo(other.us)
  }

  implement Display {
    /// RFC 3339 in UTC: `2026-09-24T09:15:02.481Z`. The fraction is written
    /// only when there is one, to three digits when the microseconds are a
    /// whole millisecond and to six otherwise.
    fun toString(): string = this.utc().toString()
  }

  implement Parsable {
    /// RFC 3339; see `time.parseRfc3339` for exactly what is accepted.
    static fun parse(s: string): Timestamp? = parseRfc3339(s)
  }

  implement Encodable {
    fun encode(to: Encoder) throws EncodeError {
      try to.writeString(this.toString())
    }
  }

  implement Decodable {
    static fun decode(from: Decoder): Timestamp throws DecodeError {
      val text = try from.readString()
      val t = parseRfc3339(text)
      if (t != null) return t
      from.problem("not an RFC 3339 timestamp: '$text'")
      Timestamp.epoch
    }
  }
}

// ---- Offset ---------------------------------------------------------------

/// A fixed offset from UTC, in whole minutes east of it.
///
/// This is the whole of the zone model for now: enough to read and write
/// RFC 3339, enough to print local time correctly at a given instant, and
/// not enough to answer "what is 09:30 local on the morning the clocks go
/// forward" — a question that needs the IANA database, which is a later
/// item. Build a `DateTime` with an explicit offset, or stay in UTC.
public struct Offset {
  private mins: i64

  /// No offset. What `Display` prints as `Z`.
  public static val utc: Offset = Offset(mins: 0)

  /// `null` beyond ±18:00, the widest range the tz database allows.
  public static fun ofMinutes(m: i64): Offset? =
    if (m < -1080 || m > 1080) null else Offset(mins: m)

  /// `Offset.of(2)` is `+02:00`, `Offset.of(-5, 30)` is `-05:30`: the sign
  /// of `hours` is the sign of the whole offset, and `minutes` is never
  /// negative. For `-00:30`, whose hour has no sign to carry, use
  /// `ofMinutes(-30)`.
  public static fun of(hours: i64, minutes: i64 = 0): Offset? {
    if (minutes < 0 || minutes > 59) return null
    Offset.ofMinutes(if (hours < 0) hours * 60 - minutes else hours * 60 + minutes)
  }

  /// The host time zone's offset at a given instant — which is the only way
  /// to ask: the answer changes twice a year in most of the world.
  ///
  /// Not every instant can be asked about. The Microsoft CRT refuses a
  /// negative `time_t` and anything past the year 3000, where glibc is
  /// happy, so a date of birth or a far-future lease would get no answer on
  /// Windows and a silent `Z` — wrong for every zone but one. Instead the
  /// same month, day and time of day is asked for in a year the host *can*
  /// convert, keeping the leap-year parity so the 29th of February survives:
  /// the answer is then the zone's rule for that date, which is right unless
  /// the rules themselves changed. That is the only approximation available
  /// without a tz database (§5.6), and it is the assumption a fixed-offset
  /// zone model makes anyway.
  public static fun local(at: Timestamp): Offset {
    val direct = hostOffsetMinutes(at.toSeconds())
    if (direct != offsetUnknown) return Offset.ofMinutes(direct) ?: Offset.utc
    val d = at.utc()
    val year = if (isLeapYear(d.year)) leapReferenceYear else referenceYear
    val shifted = DateTime(
      year, month: d.month, day: d.day, hour: d.hour,
      minute: d.minute, second: d.second,
    ).timestamp()
    val nearby = hostOffsetMinutes(shifted.toSeconds())
    if (nearby == offsetUnknown) return Offset.utc
    Offset.ofMinutes(nearby) ?: Offset.utc
  }

  /// Minutes east of UTC; negative west of it.
  public fun totalMinutes(): i64 = this.mins

  /// The same offset as a length of time.
  public fun duration(): Duration = Duration.minutes(this.mins)

  implement Comparable {
    fun compareTo(other: Offset): Ordering = this.mins.compareTo(other.mins)
  }

  implement Display {
    /// `Z`, `+02:00`, `-05:30`.
    fun toString(): string {
      if (this.mins == 0) return "Z"
      val sign = if (this.mins < 0) "-" else "+"
      val m = if (this.mins < 0) 0 - this.mins else this.mins
      "$sign${pad(m / 60, 2)}:${pad(m % 60, 2)}"
    }
  }

  implement Parsable {
    /// `Z`, `z`, `+02:00`, `-05:30` — the RFC 3339 spellings and no others.
    static fun parse(s: string): Offset? = parseOffset(s)
  }
}

fun parseOffset(s: string): Offset? {
  if (s == "Z" || s == "z") return Offset.utc
  if (s.len() != 6) return null
  val sign = s.byteAt(0)
  if (sign != '+' && sign != '-') return null
  if (s.byteAt(3) != ':') return null
  val h = twoDigits(s, 1) ?: return null
  val m = twoDigits(s, 4) ?: return null
  if (h > 23 || m > 59) return null
  val total = h * 60 + m
  Offset.ofMinutes(if (sign == '-') 0 - total else total)
}

// ---- DateTime -------------------------------------------------------------

/// A calendar date, a time of day and the offset they are told at.
///
/// The fields are data, not an invariant: nothing stops `month: 13` or
/// `day: 40`, and `timestamp()` carries them the way a human does — month 13
/// is January of the next year, and `day + 40` is a date forty days later.
/// That is what makes calendar arithmetic writable without a second API.
/// Text, on the other hand, is strict: `parseRfc3339` refuses month 13.
public struct DateTime {
  public year:   i64
  public month:  i64
  public day:    i64
  public hour:   i64 = 0
  public minute: i64 = 0
  public second: i64 = 0
  public micros: i64 = 0
  public offset: Offset = Offset.utc

  /// The instant these fields name.
  ///
  /// Panics when they name one outside the ±292,277 years a `Timestamp`
  /// holds — a `DateTime` is a set of fields and nothing stops a caller
  /// writing `year: 999999999`, but the answer then does not exist. Every
  /// parser here checks before it builds, so text from outside reaches a
  /// `null`, never this.
  public fun timestamp(): Timestamp = instantOf(this)
    ?: panic("${this.year}-${this.month}-${this.day} is outside the range a Timestamp holds")

  /// The same instant with every field brought back into range.
  public fun normalized(): DateTime = this.timestamp().at(this.offset)

  /// 0 for Sunday, 6 for Saturday.
  public fun weekday(): i64 =
    floorMod(daysFromCivil(this.year, this.month, this.day) + 4, 7)

  /// 1 for the first of January.
  public fun yearDay(): i64 =
    daysFromCivil(this.year, this.month, this.day) - daysFromCivil(this.year, 1, 1) + 1

  /// The same wall-clock instant told at another offset.
  public fun at(offset: Offset): DateTime = this.timestamp().at(offset)

  /// `2026-09-24`
  public fun date(): string = "${year4(this.year)}-${pad(this.month, 2)}-${pad(this.day, 2)}"

  /// `09:15:02`, without the fraction.
  public fun time(): string = "${pad(this.hour, 2)}:${pad(this.minute, 2)}:${pad(this.second, 2)}"

  implement Comparable {
    /// By the instant, so two `DateTime`s at different offsets order by
    /// when they happened, not by how they read.
    fun compareTo(other: DateTime): Ordering =
      this.timestamp().compareTo(other.timestamp())
  }

  implement Display {
    /// RFC 3339: `2026-09-24T11:15:02.481+02:00`. The fraction appears only
    /// when it is not zero — three digits for a whole millisecond, six
    /// otherwise — and a zero offset is written `Z`.
    fun toString(): string = "${this.date()}T${this.time()}${fraction(this.micros)}${this.offset}"
  }

  implement Parsable {
    static fun parse(s: string): DateTime? = parseRfc3339Fields(s)
  }
}

/// The fractional-second part of an RFC 3339 time: empty, `.481` or
/// `.481123`.
fun fraction(micros: i64): string {
  if (micros == 0) return ""
  if (micros % 1000 == 0) return ".${pad(micros / 1000, 3)}"
  ".${pad(micros, 6)}"
}

fun pad(n: i64, width: i64): string = n.toString().padStart(width, "0")

/// Four digits for a year RFC 3339 can spell, and ISO 8601's expanded form
/// (`+271821`, `-000001`) for one it cannot — so that `parse(t.toString())`
/// reads back every `Timestamp`, not only the ones between the year 0 and
/// the year 9999. The exception is the last second at each end of the
/// microsecond range, which prints but does not parse: the parser stops
/// where `secs * 1000000 + micros` would leave the i64, and buying those
/// two seconds back would cost arithmetic nobody could check.
fun year4(y: i64): string {
  if (y >= 0 && y <= 9999) return pad(y, 4)
  val sign = if (y < 0) "-" else "+"
  val a = if (y < 0) 0 - y else y
  "$sign${pad(a, 6)}"
}

// ---- RFC 3339 -------------------------------------------------------------

fun isDigit(b: u8): bool = b >= '0' && b <= '9'

/// The number spelled by `count` digits at `i`, or `null` when they are not
/// all digits or run past the end.
fun digitsAt(s: string, i: i64, count: i64): i64? {
  if (i < 0 || i + count > s.len()) return null
  var n: i64 = 0
  loop (k in 0..<count) {
    val b = s.byteAt(i + k)
    if (!isDigit(b)) return null
    n = n * 10 + ((b - '0') as i64)
  }
  n
}

fun twoDigits(s: string, i: i64): i64? = digitsAt(s, i, 2)

/// An RFC 3339 timestamp as its calendar fields, or `null`.
///
/// Strict, with three documented leniencies:
///
///  * `t` and `z` in lower case — the RFC's own NOTE allows them;
///  * a space where the `T` goes — RFC 3339 §5.6 allows it "by mutual
///    agreement", and it is what PostgreSQL prints;
///  * ISO 8601's expanded year (`+271821-04-20T...`), so that the text
///    `Timestamp.toString()` produces reads back — every instant but the
///    last second at each end of the microsecond range, which prints and
///    does not parse.
///
/// A date outside the ±292,277 years a `Timestamp` holds is refused rather
/// than overflowed: the year in `+999999-01-01T00:00:00Z` is a number
/// whoever sent the document chose.
///
/// Refused: a missing offset, `24:00:00`, a date with no time, a decimal
/// point with no digits after it, and any field out of range. `-00:00`,
/// which RFC 3339 defines as "offset unknown", parses as UTC and is never
/// written back. A fraction longer than six digits is **truncated**, not
/// rounded, so the order of two texts is the order of their timestamps.
///
/// `23:59:60`, the leap second, is legal RFC 3339 and has no instant in
/// POSIX time; it is accepted and read as the last microsecond of the
/// minute, which is the instant a POSIX clock reports while it happens.
public fun parseRfc3339Fields(s: string): DateTime? {
  if (s.len() < 20) return null
  // the year: four digits, or a signed six-digit expanded year
  var i: i64 = 0
  var year: i64 = 0
  if (s.byteAt(0) == '+' || s.byteAt(0) == '-') {
    val y = digitsAt(s, 1, 6) ?: return null
    year = if (s.byteAt(0) == '-') 0 - y else y
    i = 7
  } else {
    year = digitsAt(s, 0, 4) ?: return null
    i = 4
  }
  if (i + 15 > s.len()) return null
  if (s.byteAt(i) != '-' || s.byteAt(i + 3) != '-') return null
  val month = twoDigits(s, i + 1) ?: return null
  val day = twoDigits(s, i + 4) ?: return null
  val sep = s.byteAt(i + 6)
  if (sep != 'T' && sep != 't' && sep != ' ') return null
  val hour = twoDigits(s, i + 7) ?: return null
  if (s.byteAt(i + 9) != ':') return null
  val minute = twoDigits(s, i + 10) ?: return null
  if (s.byteAt(i + 12) != ':') return null
  val second = twoDigits(s, i + 13) ?: return null
  i += 15
  // the fraction: any number of digits, of which six are kept
  var micros: i64 = 0
  if (i < s.len() && s.byteAt(i) == '.') {
    i += 1
    if (i >= s.len() || !isDigit(s.byteAt(i))) return null
    var kept: i64 = 0
    loop (i < s.len() && isDigit(s.byteAt(i))) {
      if (kept < 6) {
        micros = micros * 10 + ((s.byteAt(i) - '0') as i64)
        kept += 1
      }
      i += 1
    }
    loop (kept < 6) {
      micros *= 10
      kept += 1
    }
  }
  val offset = parseOffset(s.substring(i, s.len()) ?: return null) ?: return null
  // the ranges RFC 3339 gives, and no month 13
  if (month < 1 || month > 12) return null
  if (day < 1 || day > daysInMonth(year, month)) return null
  if (hour > 23 || minute > 59 || second > 60) return null
  // a leap second is the last microsecond of the minute it belongs to
  val sec = if (second == 60) 59 else second
  val us = if (second == 60) 999999 else micros
  val d = DateTime(
    year, month, day, hour, minute,
    second: sec, micros: us, offset,
  )
  // an expanded year can name a date no Timestamp holds; refuse it here so
  // that the `DateTime` this hands back always converts
  val _ = instantOf(d) ?: return null
  d
}

/// An RFC 3339 timestamp as an instant, or `null`. See
/// `parseRfc3339Fields` for what is accepted; this discards the offset the
/// text was written at, which is what a `Timestamp` is.
public fun parseRfc3339(s: string): Timestamp? =
  (parseRfc3339Fields(s) ?: return null).timestamp()

/// `2026-09-24T09:15:02.481Z` — RFC 3339 in UTC.
public fun formatRfc3339(t: Timestamp): string = t.utc().toString()

// ---- HTTP-date ------------------------------------------------------------

val dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
val monthNames = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
]

/// The IMF-fixdate of RFC 9110 §5.6.7: `Thu, 24 Sep 2026 09:15:02 GMT`.
/// Always this form, always GMT — it is the only one a sender may use.
public fun formatHttp(t: Timestamp): string {
  val d = t.utc()
  val day = dayNames.atOrDefault(d.weekday(), "Sun")
  val mon = monthNames.atOrDefault(d.month - 1, "Jan")
  "$day, ${pad(d.day, 2)} $mon ${pad(d.year, 4)} ${d.time()} GMT"
}

/// An HTTP date in any of the three forms RFC 9110 §5.6.7 says a recipient
/// must accept, or `null`:
///
/// ```text
/// Thu, 24 Sep 2026 09:15:02 GMT      IMF-fixdate — what senders must write
/// Thursday, 24-Sep-26 09:15:02 GMT   RFC 850 — obsolete, two-digit year
/// Thu Sep 24 09:15:02 2026           asctime — obsolete, day space-padded
/// ```
///
/// The two obsolete forms are what an old client puts in `If-Modified-Since`,
/// which is the whole reason they are here. The weekday in the text is
/// ignored, as a recipient may: only the date decides. An RFC 850 year is
/// read into the century that puts it no more than fifty years ahead of
/// today, as the RFC prescribes.
public fun parseHttp(s: string): Timestamp? {
  val text = s.trim()
  val (_, afterDay) = text.splitOnce(",") else return parseAsctime(fieldsOf(text))
  val rest = afterDay.trim()
  val f = fieldsOf(rest)
  if (f.len() == 3) return parseRfc850(f)
  parseImf(f)
}

/// The whitespace-separated fields of `s`, with runs of spaces collapsed —
/// asctime pads a one-digit day with one.
fun fieldsOf(s: string): List<string> {
  val out: MutableList<string> = []
  var i: i64 = 0
  loop (i < s.len()) {
    loop (i < s.len() && isHttpSpace(s.byteAt(i))) {
      i += 1
    }
    val start = i
    loop (i < s.len() && !isHttpSpace(s.byteAt(i))) {
      i += 1
    }
    if (i > start) out.push(s.substring(start, i) ?: "")
  }
  out.toList()
}

fun isHttpSpace(b: u8): bool = b == ' ' || b == '\t'

fun monthIndex(name: string): i64 {
  loop (i in 0..<monthNames.len()) {
    if (monthNames.atOrDefault(i, "") == name) return i + 1
  }
  -1
}

/// `HH:MM:SS` as `(hour, minute, second)`.
fun clockOf(s: string): (i64, i64, i64)? {
  if (s.len() != 8 || s.byteAt(2) != ':' || s.byteAt(5) != ':') return null
  val h = twoDigits(s, 0) ?: return null
  val m = twoDigits(s, 3) ?: return null
  val sec = twoDigits(s, 6) ?: return null
  if (h > 23 || m > 59 || sec > 60) return null
  (h, m, if (sec == 60) 59 else sec)
}

fun buildUtc(year: i64, month: i64, day: i64, clock: (i64, i64, i64)): Timestamp? {
  if (month < 1 || month > 12) return null
  if (day < 1 || day > daysInMonth(year, month)) return null
  val (h, mi, sec) = clock
  // the year came out of `toInt()` on a field of the header, so it is a
  // number the client chose; `instantOf` refuses one no Timestamp holds
  instantOf(DateTime(year, month, day, hour: h, minute: mi, second: sec))
}

/// `24 Sep 2026 09:15:02 GMT`
fun parseImf(f: List<string>): Timestamp? {
  if (f.len() != 5 || f.atOrDefault(4, "") != "GMT") return null
  val day = f.atOrDefault(0, "")
  if (day.len() != 2) return null
  buildUtc(
    f.atOrDefault(2, "").toInt() ?: return null,
    monthIndex(f.atOrDefault(1, "")),
    digitsAt(day, 0, 2) ?: return null,
    clockOf(f.atOrDefault(3, "")) ?: return null,
  )
}

/// `24-Sep-26 09:15:02 GMT`
fun parseRfc850(f: List<string>): Timestamp? {
  if (f.atOrDefault(2, "") != "GMT") return null
  val d = f.atOrDefault(0, "")
  if (d.len() != 9 || d.byteAt(2) != '-' || d.byteAt(6) != '-') return null
  val day = digitsAt(d, 0, 2) ?: return null
  val month = monthIndex(d.substring(3, 6) ?: return null)
  val yy = digitsAt(d, 7, 2) ?: return null
  buildUtc(fullYear(yy), month, day, clockOf(f.atOrDefault(1, "")) ?: return null)
}

/// `Thu Sep 24 09:15:02 2026` — the only form whose day name is not
/// followed by a comma, so it still carries it.
fun parseAsctime(f: List<string>): Timestamp? {
  if (f.len() != 5) return null
  buildUtc(
    f.atOrDefault(4, "").toInt() ?: return null,
    monthIndex(f.atOrDefault(1, "")),
    f.atOrDefault(2, "").toInt() ?: return null,
    clockOf(f.atOrDefault(3, "")) ?: return null,
  )
}

/// RFC 9110 §5.6.7: a two-digit year that would fall more than fifty years
/// ahead belongs to the most recent past century with those digits.
fun fullYear(yy: i64): i64 {
  val thisYear = now().utc().year
  val candidate = (thisYear / 100) * 100 + yy
  if (candidate > thisYear + 50) candidate - 100 else candidate
}

// ---- measuring ------------------------------------------------------------

/// Elapsed time, on the monotonic clock: `val sw = time.Stopwatch.start()`,
/// then `sw.elapsed()`. Unaffected by an NTP correction or a clock the
/// operator sets by hand, which is what makes it, and not two `Timestamp`s,
/// the way to measure.
public struct Stopwatch {
  private var startedAt: i64

  /// A stopwatch running from now.
  public static fun start(): Stopwatch = Stopwatch(startedAt: monotonicNanos())

  /// How long it has been running.
  public fun elapsed(): Duration = Duration.nanos(monotonicNanos() - this.startedAt)

  /// Starts over.
  public fun reset() {
    this.startedAt = monotonicNanos()
  }
}

/// A point on the monotonic clock by which something has to be done.
///
/// This is what a timeout is, and writing it as a type rather than as
/// `monotonic() + limit` arithmetic is the reason no monotonic reading is
/// handed out: a `Deadline` cannot be compared with a `Timestamp`, cannot be
/// serialized into a log, and cannot be mistaken for a wall-clock time by a
/// reader either.
///
/// ```veles
/// val head = time.Deadline.after(limits.headerTimeout)
/// loop (!head.expired()) {
///   val line = try withTimeout(head.remaining(), () => try c.readLine(max: 8192))
///   ...
/// }
/// ```
public struct Deadline {
  private atNanos: i64

  /// A deadline `d` from now.
  public static fun after(d: Duration): Deadline =
    Deadline(atNanos: monotonicNanos() + d.toNanos())

  /// How long is left, never negative: `Duration.zero` once it has passed.
  public fun remaining(): Duration {
    val left = this.atNanos - monotonicNanos()
    if (left <= 0) Duration.zero else Duration.nanos(left)
  }

  /// Whether the time is up.
  public fun expired(): bool = monotonicNanos() >= this.atNanos

  /// The same deadline, `d` later.
  public fun extend(d: Duration): Deadline = Deadline(atNanos: this.atNanos + d.toNanos())

  /// Whichever of the two comes first — a request deadline against a
  /// configured one.
  public fun earlier(other: Deadline): Deadline =
    if (this.atNanos <= other.atNanos) this else other

  implement Comparable {
    fun compareTo(other: Deadline): Ordering = this.atNanos.compareTo(other.atNanos)
  }
}
