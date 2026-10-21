// Dates and times: which of the four shapes a word is, checked for its form and
// range. The word is kept as written.

fun between(x: i64, low: i64, high: i64): bool => x >= low && x <= high

fun isLeapYear(year: i64): bool => (year % 4 == 0 && year % 100 != 0) || year % 400 == 0

fun daysIn(month: i64, year: i64): i64 => when (month) {
  2           => if (isLeapYear(year)) 29 else 28
  4, 6, 9, 11 => 30
  else        => 31
}

/// `count` digits at `at` as a number, or -1.
fun digitsAt(text: string, at: i64, count: i64): i64 {
  if (at + count > text.len()) return -1
  var value: i64 = 0
  loop (i in 0..<count) {
    val byte = text.byteAt(at + i)
    if (!byte.isAsciiDigit()) return -1
    value = value * 10 + (byte - '0').toI64()
  }
  value
}

/// Where `HH:MM:SS[.fraction]` at `at` ends, or -1 when it is not one.
fun timeEnd(text: string, at: i64): i64 {
  val hour = digitsAt(text, at, 2)
  val minute = digitsAt(text, at + 3, 2)
  val second = digitsAt(text, at + 6, 2)
  val inRange = between(hour, 0, 23) && between(minute, 0, 59) && between(second, 0, 60)
  if (!inRange || text.byteAt(at + 2) != ':' || text.byteAt(at + 5) != ':') return -1
  var end = at + 8
  if (end < text.len() && text.byteAt(end) == '.') {
    end += 1
    val digitsFrom = end
    loop (end < text.len() && text.byteAt(end).isAsciiDigit()) end += 1
    if (end == digitsFrom) return -1
  }
  end
}

/// A `+HH:MM` or `-HH:MM` offset at `at` that ends the word.
fun isOffsetAt(word: string, at: i64): bool {
  if (at + 6 != word.len() || word.byteAt(at + 3) != ':') return false
  val hours = digitsAt(word, at + 1, 2)
  val minutes = digitsAt(word, at + 4, 2)
  between(hours, 0, 23) && between(minutes, 0, 59)
}

/// The shape of date or time `word` is, or null when it is neither.
fun momentShape(word: string): MomentShape? {
  val year = digitsAt(word, 0, 4)
  val startsWithDate = year >= 0 && word.len() >= 10 && word.byteAt(4) == '-' && word.byteAt(7) == '-'
  if (!startsWithDate) {
    val isTime = word.len() >= 8 && word.byteAt(2) == ':' && timeEnd(word, 0) == word.len()
    return if (isTime) MomentShape.LocalTime else null
  }
  val month = digitsAt(word, 5, 2)
  val day = digitsAt(word, 8, 2)
  if (month < 1 || month > 12 || day < 1 || day > daysIn(month, year)) return null
  if (word.len() == 10) return MomentShape.LocalDate
  val separator = word.byteAt(10)
  if (separator != 'T' && separator != 't' && separator != ' ') return null
  val end = timeEnd(word, 11)
  if (end < 0) return null
  if (end == word.len()) return MomentShape.LocalDateTime
  val zone = word.byteAt(end)
  val isUtc = zone in ['Z', 'z'] && end + 1 == word.len()
  val hasOffset = zone in ['+', '-'] && isOffsetAt(word, end)
  if (isUtc || hasOffset) MomentShape.OffsetDateTime else null
}
