# 20. Time

Three types carry everything in this chapter, and the first thing to know
is why there are three rather than one.

| type | where it lives | what it is |
|---|---|---|
| `Duration` | the prelude | a **length** of time, to the nanosecond |
| `time.Timestamp` | `std/time` | a point on the **wall** clock, to the microsecond |
| `time.Deadline` | `std/time` | a point on the **monotonic** clock |

The wall clock tells you what time it is. It is what a log line, a JSON
body and a token's `exp` claim carry, it is comparable between machines,
and it can jump — backwards as well as forwards — when the operator or NTP
corrects it. The monotonic clock only ever moves forward and its readings
mean nothing on their own; it is what you *measure* with. Using the wrong
one is the classic bug: a request that appears to take minus four hours
because the clock was corrected while it was being served.

So Veles does not hand out a monotonic reading at all. The only two things
you can make from that clock are a `Stopwatch` and a `Deadline`, which are
the only two things anyone wants from it, and neither can be compared with
a `Timestamp`, serialized into a log, or mistaken by a reader for a time of
day.

## Duration

```veles
use io

fun main() {
  val a = Duration.seconds(90)
  val b = Duration.millis(250)
  io.println("$a and $b")
  io.println("${a + b}  ${a - b}  ${b * 6}  ${a / 3}  ${-b}")
  io.println("${a > b}  ${a.toMillis()}  ${b.asSeconds()}")
  io.println("${Duration.parse("1h30m")}  ${Duration.parse("1.5s")}  ${Duration.parse("5")}")
}
```

Output:
```text
1m30s and 250ms
1m30.25s  1m29.75s  1.5s  30s  -250ms
true  90000  0.25
1h30m  1.5s  null
```

Constructors: `nanos`, `micros`, `millis`, `seconds`, `minutes`, `hours`,
`days`, `zero`, and `ofSeconds` for a fractional count. Conversions
`toNanos` … `toDays` truncate toward zero; `asSeconds` and `asMillis` keep
the fraction.

The arithmetic operators are the prelude's operator traits (D71): a
`Duration` adds and subtracts another, multiplies and divides by an
`i64`, negates, and `+=`/`-=` work on a `var`. Dividing one length by
another is `a.over(b)`, an `i64`. The two sums that actually get
written, "now plus a timeout" and "how much is left", are usually
`Deadline`, below.

`Display` prints the largest units that fit and drops empty ones, and
`Parsable` reads back exactly what `Display` writes — the fraction is
carried in whole nanoseconds, never through a float, so nothing is
rounded and `Duration.parse("$d") == d` for every `d`. A number with no
unit is refused (`0` is the exception, since zero has no unit), and so is
a fraction finer than a nanosecond.

The inside is an `i64` of nanoseconds, so the range is ±292 years.

## Timestamp

```veles
use io, time

fun main() {
  val t = time.Timestamp.ofMicros(1790000000481123)
  io.println("$t")
  io.println("${t.at(time.Offset.of(2) ?: time.Offset.utc)}")
  io.println("${t.toSeconds()}  ${t.subsecondMicros()}")
  io.println("${t + Duration.days(1)}")
  io.println("${t.since(time.Timestamp.epoch).toDays()} days since the epoch")
}
```

Output:
```text
2026-09-21T14:13:20.481123Z
2026-09-21T16:13:20.481123+02:00
1790000000  481123
2026-09-22T14:13:20.481123Z
20717 days since the epoch
```

`time.now()` is the clock. `Timestamp.epoch`, `ofSeconds`, `ofMillis` and
`ofMicros` build one from a number; `toSeconds`, `toMillis` and `toMicros`
go back, rounding **down** rather than toward zero, so they keep naming the
second that contains the instant on both sides of 1970.

The unit is microseconds — enough to round-trip RFC 3339's six fractional
digits and a PostgreSQL `timestamptz` without losing anything, and enough
to order two events in the same millisecond. An `i64` of them spans
±292,000 years.

`Display` is RFC 3339 in UTC, `Parsable` reads it, and `Codable` uses the
same text, so a `Timestamp` field in a derived struct is a string on the
wire and never a number of microseconds:

```veles
use io, json, time

struct Note {
  public id: i64
  public createdAt: time.Timestamp
  implement Codable
}

fun main() throws {
  val n = Note(id: 7, createdAt: time.Timestamp.ofSeconds(1790000000))
  val text = try json.encode(n)
  io.println(text)
  io.println("${try json.decode<Note>(text).createdAt.utc().date()}")
}
```

Output:
```text
{"id":7,"createdAt":"2026-09-21T14:13:20Z"}
2026-09-21
```

## Calendar fields, and offsets

`t.utc()`, `t.local()` and `t.at(offset)` give a `DateTime`: year, month,
day, hour, minute, second, microsecond, and the `Offset` it is told at.
`weekday()` and `yearDay()` (1 for January 1st) are computed from the
date, and `timestamp()` goes back to the instant. A weekday is a
`time.Weekday` — `Monday` … `Sunday`, an enum whose `.value` is the ISO
number (Monday is 1) — so a `when` over it names every day or says `else`:

```veles
// fragment
fun isWeekend(d: time.DateTime): bool = when (d.weekday()) {
  time.Weekday.Saturday, time.Weekday.Sunday => true
  else => false
}
```

An `Offset` is a fixed number of minutes east of UTC — `Offset.utc`,
`Offset.of(2)`, `Offset.of(-5, 30)`, `Offset.ofMinutes(330)` — refused
beyond ±18:00, printed as `Z` or `+02:00`. `Offset.local(at: t)` asks the
host zone for *its* offset at that instant, which is the only way the
question can be asked: the answer changes twice a year in most of the
world.

That is the whole zone model for now. It is enough to read and write
RFC 3339 and to print local time correctly, and not enough to answer "what
is 09:30 local on the morning the clocks go forward" — that needs the IANA
database, which is a later item.

A `DateTime`'s fields are data, not an invariant. Nothing stops `month: 13`
or `day: 40`, and `timestamp()` carries them the way a person means them,
which is what makes date arithmetic writable without a second API:

```veles
use io, time

fun main() {
  val d = time.DateTime(year: 2026, month: 1, day: 31)
  io.println("${d.normalized().date()}")
  io.println("${time.DateTime(year: 2026, month: 1, day: 31 + 40).normalized().date()}")
  io.println("${time.DateTime(year: 2026, month: 13, day: 1).normalized().date()}")
}
```

Output:
```text
2026-01-31
2026-03-12
2027-01-01
```

Text is the strict half: `parseRfc3339` refuses month 13.

## RFC 3339

`time.parseRfc3339(s)` gives a `Timestamp?`, `parseRfc3339Fields(s)` a
`DateTime?` that keeps the offset the text was written at, and
`time.formatRfc3339(t)` — which is what `Display` calls — writes UTC.

The parser is strict, with three leniencies it documents:

- lower-case `t` and `z`, which the RFC's own NOTE allows;
- a space where the `T` goes, allowed by §5.6 "by mutual agreement" and
  what PostgreSQL prints;
- ISO 8601's expanded year (`+271821-04-20T…`), so the text `toString()`
  produces reads back, even outside the years 0000–9999 — every instant
  but the last second at each end of the microsecond range, which prints
  and does not parse. A date beyond the ±292,277 years a `Timestamp` holds
  is refused, not overflowed.

It refuses a missing offset, `24:00:00`, a bare date, a decimal point with
no digits after it, and any field out of range. `-00:00` — RFC 3339's
"offset unknown" — parses as UTC and is never written back. A fraction
longer than six digits is truncated rather than rounded, so the order of
two texts stays the order of their instants.

`23:59:60`, the leap second, is legal RFC 3339 and has no instant in POSIX
time. It is accepted and read as the last microsecond of that minute, which
is the instant a POSIX clock reports while it is happening. Rejecting it
would mean a conforming producer's timestamp failing to parse.

## HTTP dates

`time.formatHttp(t)` writes the IMF-fixdate of RFC 9110 §5.6.7 —
`Sun, 06 Nov 1994 08:49:37 GMT` — which is the only form a sender may use.
`time.parseHttp(s)` reads that and the two obsolete forms a recipient
**must** also accept, because those are exactly what turns up in an old
client's `If-Modified-Since`:

```veles
use io, time

fun main() {
  loop (text in ["Sun, 06 Nov 1994 08:49:37 GMT",
                 "Sunday, 06-Nov-94 08:49:37 GMT",
                 "Sun Nov  6 08:49:37 1994",
                 "Sun, 31 Nov 1994 08:49:37 GMT"]) {
    io.println("${time.parseHttp(text)}")
  }
}
```

Output:
```text
1994-11-06T08:49:37Z
1994-11-06T08:49:37Z
1994-11-06T08:49:37Z
null
```

The weekday in the text is ignored — only the date decides — and an
RFC 850 two-digit year is read into the century that puts it no more than
fifty years ahead of today, as the RFC prescribes. `http.httpDate(t)` is
the same function under the server's own name.

## Measuring, and deadlines

```veles
use io, time

fun main() {
  val sw = time.Stopwatch.start()
  var n: i64 = 0
  loop (i in 0..<200000) {
    n += i
  }
  io.println("${n > 0 && sw.elapsed() < Duration.seconds(10)}")

  val until = time.Deadline.after(Duration.seconds(30))
  io.println("${until.expired()}  ${until.remaining() <= Duration.seconds(30)}")
  io.println("${time.Deadline.after(Duration.zero).remaining()}")
}
```

Output:
```text
true
false  true
0s
```

`Stopwatch.start()` then `sw.elapsed()` is the whole of measuring.
`Deadline.after(d)` is the whole of a timeout: `remaining()` (never
negative), `expired()`, `extend(d)` and `earlier(other)` for when a request
deadline and a configured one both apply. This is what replaced
`monotonic() + limit` arithmetic in the HTTP server's header loop:

```veles
// fragment
val head = time.Deadline.after(limits.headerTimeout)
loop {
  if (head.expired()) throw Fail(status: Status.requestTimeout, text: "request header timeout")
  val line = try headLine(c, limits.headerLineBytes, head.remaining(), tooLong)
  ...
}
```

## Where durations turn up

`sleep` and `withTimeout` take a `Duration`, not a number:

```veles
use io

fun slow(): i64 {
  await sleep(Duration.millis(500))
  42
}

fun main() {
  when (withTimeout(Duration.millis(20), () => slow())) {
    is Ok(v)  => io.println("got $v")
    is Err(e) => io.println("failed: ${e.message()}")
  }
}
```

Output:
```text
failed: timed out after 20ms
```

`sleep(20)` is a compile error that names the fix. The executor's timers
are in milliseconds, so a duration is rounded *up* to one — a sleep is
never shorter than it was asked for, and `sleep(Duration.zero)` still
yields.

`http.Limits` says its three clocks the same way
(`Limits(headerTimeout: Duration.seconds(10))`), and `Timeout` carries the
limit it reached as a `Duration`, which is why its message reads `20ms`
rather than `20`.

## A worked example

`examples/time` runs the vectors: the RFC 3339 grammar and its leniencies,
the leap second, the calendar across 1970 and across the year 2000, the
three HTTP-date forms, the refusals, and the round trips for both
`Duration` and `Timestamp`. Nothing in it reads the host clock except the
last section, which asserts only what holds for every reading of one.

Next: back to the [index](index.md).
