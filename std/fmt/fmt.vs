/// Numbers, sizes, durations and tables written for people (D171). The
/// format spec in a string — `"${price:>10.2}"` — is the low level: one value,
/// one cell. These functions are the level above it: they pick the unit, the
/// grouping or the layout for you.
///
/// ```veles
/// use fmt
///
/// fmt.thousands(1234567)        // "1,234,567"
/// fmt.bytes(1536)               // "1.5 KiB"
/// fmt.percent(0.256)            // "25.6%"
/// fmt.ordinal(22)               // "22nd"
/// fmt.duration(Duration.seconds(3725))  // "1h 2m 5s"
/// fmt.quote("café\n")          // "\"café\\n\""
/// ```
///
/// Everything here is English and uses `.` for decimals and `,` for groups;
/// locales are a later addition.
use bytes
use unicode
use utf8

/// A Go-style double-quoted representation. Quotes, backslashes, controls
/// and non-printable Unicode are escaped; printable Unicode stays as written.
public fun quote(text: string): string {
  val out = StringBuilder()
  out.append("\"")
  var at: i64 = 0
  loop (at < text.len()) {
    val rune = utf8.decode(text, at) ?: panic("fmt.quote: a string is valid UTF-8 at every character start")
    val code = rune.code
    at += rune.size
    if (code == '"'.toI64() || code == '\\'.toI64()) {
      out.append("\\")
      out.append(utf8.char(code))
      continue
    }
    if (unicode.isPrintable(code)) {
      out.append(utf8.char(code))
      continue
    }
    when (code) {
      0x07 => out.append("\\a")
      0x08 => out.append("\\b")
      0x0C => out.append("\\f")
      0x0A => out.append("\\n")
      0x0D => out.append("\\r")
      0x09 => out.append("\\t")
      0x0B => out.append("\\v")
      else => {
        if (code < 0x20 || code == 0x7F) {
          out.append("\\x${hex(code, 2)}")
        } else if (code < 0x10000) {
          out.append("\\u${hex(code, 4)}")
        } else {
          out.append("\\U${hex(code, 8)}")
        }
      }
    }
  }
  out.append("\"")
  out.toString()
}

fun hex(code: i64, width: i64): string => code.toString(radix: 16).padStart(width, "0")

/// The integer with a comma between each group of three digits:
/// `thousands(-1234567)` is `"-1,234,567"`.
public fun thousands(n: i64): string => "${n:,d}"

/// A size in bytes with the largest binary unit that keeps it at one or more:
/// `bytes(1536)` is `"1.5 KiB"`, `bytes(12)` is `"12 B"`. One decimal, and
/// none for plain bytes.
public fun bytes(n: i64): string {
  val negative = n < 0
  var value = if (negative) -(n.toF64()) else n.toF64()
  val units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"]
  var unit = 0
  val kibibyte = bytes.KiB.toF64()
  loop (value >= kibibyte && unit < units.len() - 1) {
    value /= kibibyte
    unit += 1
  }
  val name = units.at(unit) ?: panic("fmt.bytes: the unit index stays inside the list")
  val sign = if (negative) "-" else ""
  if (unit == 0) "$sign${value.toFixed(0)} $name" else "$sign${value.toFixed(1)} $name"
}

/// A fraction as a percentage with `digits` decimals (one by default):
/// `percent(0.256)` is `"25.6%"`, `percent(1.0, 0)` is `"100%"`.
public fun percent(fraction: f64, digits: i64 = 1): string => (fraction * 100.0).toFixed(digits) + "%"

/// The number with its English ordinal suffix: `ordinal(1)` is `"1st"`,
/// `ordinal(12)` is `"12th"`, `ordinal(23)` is `"23rd"`.
public fun ordinal(n: i64): string {
  val remainder = n % 100
  val last = if (remainder < 0) -remainder else remainder
  val suffix = if (last in 11..13) "th" else when (last % 10) {
    1    => "st"
    2    => "nd"
    3    => "rd"
    else => "th"
  }
  "$n$suffix"
}

/// A duration by its largest units, at most three of them:
/// `duration(Duration.seconds(3725))` is `"1h 2m 5s"`; under a second it is
/// `"250ms"` (or `"12us"`, `"7ns"`); zero is `"0s"`.
public fun duration(d: Duration): string {
  if (d.isZero()) return "0s"
  val sign = if (d.isNegative()) "-" else ""
  var rest = d.abs().toNanos()
  if (rest < 1000) return "$sign${rest}ns"
  if (rest < 1000000) return "$sign${rest / 1000}us"
  if (rest < 1000000000) return "$sign${rest / 1000000}ms"
  val parts: MutableList<string> = []
  val units = [(86400000000000, "d"), (3600000000000, "h"), (60000000000, "m"), (1000000000, "s")]
  loop ((size, name) in units) {
    if (rest >= size && parts.len() < 3) {
      parts.push("${rest / size}$name")
      rest = rest % size
    }
  }
  sign + parts.join(" ")
}

/// Rows as aligned columns, two spaces apart. A column is as wide as its
/// widest cell, measured in characters. Columns listed in `right` are
/// right-aligned (numbers); the rest are left-aligned. With `header: true`
/// the first row is underlined.
///
/// ```veles
/// use fmt
///
/// println(fmt.table([["tea", "3.50"], ["coffee", "12.00"]], right: [1]))
/// ```
public fun table(rows: List<List<string>>, right: List<i64> = [], header: bool = false): string {
  val widths: MutableList<i64> = []
  loop (row in rows) {
    loop (i in 0..<row.len()) {
      val cell = row.at(i).chars().len()
      if (i >= widths.len()) widths.push(cell) else if (cell > widths.at(i)) widths.set(i, cell)
    }
  }
  val lines: MutableList<string> = []
  loop (r in 0..<rows.len()) {
    val row = rows.at(r)
    val cells: MutableList<string> = []
    loop (i in 0..<widths.len()) {
      val width = widths.at(i) ?: 0
      val cell = row.at(i) ?: ""
      val gap = " ".repeat(width - cell.chars().len())
      cells.push(if (right.contains(i)) gap + cell else cell + gap)
    }
    lines.push(trimEnd(cells.join("  ")))
    if (header && r == 0) {
      lines.push(trimEnd(widths.map(w => "-".repeat(w)).join("  ")))
    }
  }
  lines.join("\n")
}

fun trimEnd(s: string): string {
  var end = s.len()
  loop (end > 0 && s.byteAt(end - 1) == ' ') end -= 1
  s.substring(0, end) ?: s
}
