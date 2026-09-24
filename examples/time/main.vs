// std/time and the prelude's Duration, against the things the two text
// formats are actually judged by: the RFC 3339 grammar and its three
// leniencies, the leap second, the calendar across 1970 and the year 2000,
// the three HTTP-date forms RFC 9110 §5.6.7 says a recipient must accept,
// and the round trip `Duration.parse(d.toString()) == d`.
//
// Nothing here reads the host clock: every value is fixed, so the output is
// the same on any machine, in any zone, on any day. The clock itself is
// exercised at the end, by the only assertions that hold for every reading
// of it.

use io, time

fun show(label: string, value: string) {
  io.println("  ${label.padEnd(34)} $value")
}

fun ok(label: string, cond: bool) {
  io.println("  ${label.padEnd(34)} ${if (cond) "ok" else "FAILED"}")
}

// ---------------------------------------------------------------------------

fun durations() {
  io.println("-- Duration: what it prints --")
  loop (d in [
    Duration.zero, Duration.nanos(1), Duration.nanos(1500),
    Duration.micros(250), Duration.millis(1), Duration.millis(1500),
    Duration.seconds(1), Duration.seconds(90), Duration.minutes(90),
    Duration.hours(25), Duration.days(3), Duration.millis(-90),
  ]) {
    show("${d.toNanos()} ns", "$d")
  }

  io.println("-- Duration: what it reads --")
  loop (text in ["0", "90s", "1h30m", "250ms", "1.5s", "-2m30s", "1d1h", "1.000001ms"]) {
    show("\"$text\"", "${Duration.parse(text)}")
  }

  io.println("-- Duration: what it refuses --")
  loop (text in ["", "5", "5 s", "s", "1h30", "1x", "1.s", "1.0000000001s", "-"]) {
    val d = Duration.parse(text)
    show("\"$text\"", if (d == null) "rejected" else "$d")
  }

  io.println("-- Duration: the round trip --")
  var roundTrips = true
  loop (d in [
    Duration.zero, Duration.nanos(1), Duration.nanos(999999999),
    Duration.micros(1), Duration.millis(1), Duration.seconds(1),
    Duration.seconds(90), Duration.hours(25), Duration.days(400),
    Duration.nanos(-1), Duration.millis(-1500),
  ]) {
    if (Duration.parse("$d") != d) {
      io.println("  BROKEN: $d -> ${Duration.parse("$d")}")
      roundTrips = false
    }
  }
  ok("parse(d.toString()) == d", roundTrips)

  io.println("-- Duration: arithmetic and order --")
  show("1s + 250ms", "${Duration.seconds(1).plus(Duration.millis(250))}")
  show("1s - 250ms", "${Duration.seconds(1).minus(Duration.millis(250))}")
  show("250ms * 6", "${Duration.millis(250).times(6)}")
  show("1s / 4", "${Duration.seconds(1).dividedBy(4)}")
  show("1h / 250ms", "${Duration.hours(1).over(Duration.millis(250))}")
  show("-90ms abs", "${Duration.millis(-90).abs()}")
  show("sorted", "${[Duration.seconds(1), Duration.zero, Duration.millis(-5)].sorted()}")
  show("min, max", "${Duration.seconds(1).min(Duration.millis(10))} ${Duration.seconds(1).max(Duration.millis(10))}")
  show("0.25 seconds", "${Duration.ofSeconds(0.25)}")
  show("90s as seconds, millis", "${Duration.seconds(90).toSeconds()} ${Duration.seconds(90).toMillis()}")
  show("-1500ns truncates toward zero", "${Duration.nanos(-1500).toMicros()}")
}

// ---------------------------------------------------------------------------

fun calendar() {
  io.println("-- the calendar, across the awkward dates --")
  loop (t in [
    time.Timestamp.epoch,
    time.Timestamp.ofSeconds(-1),
    time.Timestamp.ofSeconds(951782400),   // 2000-02-29, a leap year
    time.Timestamp.ofSeconds(4107542400),  // 2100-03-01, not one
    time.Timestamp.ofSeconds(1000000000),
    time.Timestamp.ofSeconds(-2208988800),
  ]) {  // 1900-01-01
    val d = t.utc()
    show("${t.toSeconds()}", "$d weekday ${d.weekday()} day ${d.yearDay()}")
  }

  io.println("-- civil conversion is its own inverse --")
  var exact = true
  var day: i64 = -800000
  loop (day < 800000) {
    val (y, m, d) = time.civilFromDays(day)
    if (time.daysFromCivil(y, m, d) != day) {
      io.println("  BROKEN at day $day")
      exact = false
      day = 800000
    }
    day += 997  // a prime stride over ±2190 years
  }
  ok("civilFromDays round trip", exact)
  ok("2000 is a leap year", time.isLeapYear(2000))
  ok("1900 is not", !time.isLeapYear(1900))
  ok("February 2024 has 29 days", time.daysInMonth(2024, 2) == 29)

  io.println("-- out-of-range fields carry, as a human means them --")
  show("2026-13-01", "${time.DateTime(year: 2026, month: 13, day: 1).normalized().date()}")
  show("2026-01-32", "${time.DateTime(year: 2026, month: 1, day: 32).normalized().date()}")
  show("2026-03-00", "${time.DateTime(year: 2026, month: 3, day: 0).normalized().date()}")
}

// ---------------------------------------------------------------------------

fun rfc3339() {
  io.println("-- RFC 3339: what it reads --")
  loop (text in [
    "2026-09-24T09:15:02Z",
    "2026-09-24T09:15:02.481Z",
    "2026-09-24T09:15:02.481123Z",
    "2026-09-24T09:15:02.4811239999Z",  // truncated to six digits
    "2026-09-24t09:15:02z",             // the RFC's own NOTE
    "2026-09-24 09:15:02Z",             // §5.6, what PostgreSQL prints
    "2026-09-24T11:15:02+02:00",
    "2026-09-24T04:15:02-05:00",
    "2026-09-24T09:15:02-00:00",  // "offset unknown" is UTC
    "1969-12-31T23:59:59Z",
    "2016-12-31T23:59:60Z",
  ]) {  // the leap second
    show(text, "${time.parseRfc3339(text)}")
  }

  io.println("-- RFC 3339: what it refuses --")
  loop (text in [
    "2026-09-24T09:15:02",   // no offset
    "2026-09-24",            // a date is not a timestamp
    "2026-09-24T24:00:00Z",  // ISO 8601's end of day
    "2026-13-01T00:00:00Z",
    "2026-02-30T00:00:00Z",
    "2026-09-24T09:15:02.Z",  // a point with no digits
    "2026-09-24T09:15:02+2:00",
    "2026-09-24T09:15:02 Z",
    "not a timestamp",
    // a year no Timestamp holds is refused, not overflowed: the number came
    // from whoever wrote the document
    "+999999-01-01T00:00:00Z",
    "+300000-01-01T00:00:00Z",
    "-999999-01-01T00:00:00Z",
  ]) {
    val t = time.parseRfc3339(text)
    show(text, if (t == null) "rejected" else "$t")
  }
  // the same field, in a header, reaches a different parser
  loop (text in [
    "Sun, 06 Nov 99999999999999 08:49:37 GMT",
    "Sun Nov  6 08:49:37 999999999",
  ]) {
    val t = time.parseHttp(text)
    show(text, if (t == null) "rejected" else "$t")
  }
  // and every instant a Timestamp holds still has a text, both ends included
  loop (us in [9223372036854775807, -9223372036854775807, 0]) {
    val t = time.Timestamp.ofMicros(us)
    show("ofMicros($us)", "$t")
  }

  io.println("-- RFC 3339: the offset is kept when the fields are --")
  val fields = time.parseRfc3339Fields("2026-09-24T11:15:02.481+02:00")
  show("parseRfc3339Fields", "$fields")
  show("  same instant in UTC", "${fields?.timestamp()}")
  show("  offset", "${fields?.offset}")

  io.println("-- an offset is a value of its own --")
  loop (o in [
    time.Offset.utc, time.Offset.of(2) ?: time.Offset.utc,
    time.Offset.of(-5, 30) ?: time.Offset.utc,
    time.Offset.ofMinutes(330) ?: time.Offset.utc,
  ]) {
    show("${o.totalMinutes()} minutes", "$o")
  }
  show("beyond 18 hours", "${time.Offset.ofMinutes(1100)}")
  show("parse \"+05:45\"", "${time.Offset.parse("+05:45")}")

  io.println("-- the round trip, at both ends of the range --")
  var roundTrips = true
  loop (t in [
    time.Timestamp.epoch,
    time.Timestamp.ofSeconds(-2208988800),
    time.Timestamp.ofMicros(1790000000481123),
    time.Timestamp.ofSeconds(253402300799),  // 9999-12-31T23:59:59Z
    time.Timestamp.ofSeconds(300000000000),
  ]) {  // an expanded year
    if (time.parseRfc3339("$t") != t) {
      io.println("  BROKEN: $t")
      roundTrips = false
    }
  }
  ok("parse(t.toString()) == t", roundTrips)
  show("an expanded year", "${time.Timestamp.ofSeconds(300000000000)}")
}

// ---------------------------------------------------------------------------

fun httpDates() {
  io.println("-- HTTP-date: the three forms of one instant --")
  loop (text in [
    "Sun, 06 Nov 1994 08:49:37 GMT",   // IMF-fixdate
    "Sunday, 06-Nov-94 08:49:37 GMT",  // RFC 850
    "Sun Nov  6 08:49:37 1994",
  ]) {  // asctime
    show(text, "${time.parseHttp(text)}")
  }

  io.println("-- HTTP-date: what a sender writes --")
  loop (t in [
    time.Timestamp.ofSeconds(784111777),
    time.Timestamp.epoch,
    time.Timestamp.ofMicros(1790000000481123),
  ]) {
    show("${t.toSeconds()}", time.formatHttp(t))
  }
  ok(
    "format then parse is the second",
    time.parseHttp(time.formatHttp(time.Timestamp.ofMicros(1790000000481123)))
      == time.Timestamp.ofSeconds(1790000000),
  )

  io.println("-- HTTP-date: what it refuses --")
  loop (text in [
    "Sun, 06 Nov 1994 08:49:37",  // no zone
    "Sun, 06 Nov 1994 08:49:37 UTC",
    "Sun, 6 Nov 1994 08:49:37 GMT",  // the day is two digits
    "Sun, 06 Foo 1994 08:49:37 GMT",
    "Sun, 31 Nov 1994 08:49:37 GMT",  // November has 30 days
    "",
  ]) {
    val t = time.parseHttp(text)
    show("\"$text\"", if (t == null) "rejected" else "$t")
  }
}

// ---------------------------------------------------------------------------

fun clocks() {
  io.println("-- the clocks: what holds for every reading --")
  val a = time.now()
  val sw = time.Stopwatch.start()
  val deadline = time.Deadline.after(Duration.seconds(30))
  var spin: i64 = 0
  loop (_ in 0..<200000) {
    spin += 1
  }
  val b = time.now()
  if (spin != 200000) io.println("  the spin loop did not run")

  ok("now() is after the epoch", a > time.Timestamp.epoch)
  ok("now() does not run backwards", b >= a)
  ok("since() is what minus is", b.since(a) == Duration.micros(b.toMicros() - a.toMicros()))
  ok("a stopwatch runs forward", !sw.elapsed().isNegative())
  ok("a 30 s deadline has not passed", !deadline.expired())
  ok("its remainder is under 30 s", deadline.remaining() <= Duration.seconds(30))
  ok(
    "a passed deadline has no remainder",
    time.Deadline.after(Duration.zero).remaining() == Duration.zero,
  )
  ok(
    "the local offset is a real one",
    time.Offset.local(at: a).totalMinutes() >= -1080 &&
      time.Offset.local(at: a).totalMinutes() <= 1080,
  )
  ok("local and UTC are the same instant", a.local().timestamp() == a.utc().timestamp())
  ok("plus and minus undo each other", a.plus(Duration.hours(1)).minus(Duration.hours(1)) == a)
}

fun main() {
  durations()
  calendar()
  rfc3339()
  httpDates()
  clocks()
}
