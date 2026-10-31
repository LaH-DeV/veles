/// Unicode code-point properties, generated from the repository's pinned Go
/// Unicode tables. `utf8` handles encoding and decoding; this module handles
/// what a decoded code point means.
use utf8

fun inRanges(ranges: List<i64>, code: i64): bool {
  var low = 0
  var high = ranges.len() / 2
  loop (low < high) {
    val middle = (low + high) / 2
    val first = ranges.at(middle * 2) ?: return false
    val last = ranges.at(middle * 2 + 1) ?: return false
    if (code < first) high = middle
    else if (code > last) low = middle + 1
    else return true
  }
  false
}

fun inPairs(ranges: List<(i64, i64)>, code: i64): bool {
  val i = ranges.partitionPoint(range => range.1 < code)
  val (first, _) = ranges.at(i) ?: return false
  first <= code
}

/// True for a letter in Unicode general category `L`.
public fun isLetter(code: i64): bool => hasProperty("L", code) == true

/// True for a decimal digit in Unicode general category `Nd`.
public fun isDecimalDigit(code: i64): bool => hasProperty("Nd", code) == true

/// True for a code point in the Unicode `White_Space` property.
public fun isWhiteSpace(code: i64): bool {
  if (!utf8.isScalar(code)) return false
  inRanges(unicodeSpace, code)
}

/// True for Go's printable set: letters, marks, numbers, punctuation,
/// symbols and ASCII space.
public fun isPrintable(code: i64): bool {
  if (!utf8.isScalar(code)) return false
  if (code in 0x20..0x7E) return true
  inPairs(PRINT, code)
}

/// True for `_` or a Unicode `ID_Start` code point, as Veles identifiers are.
public fun isIdentifierStart(code: i64): bool {
  if (code == 95) return true
  if (code < 0x80) return code >= 65 && code <= 90 || code >= 97 && code <= 122
  if (!utf8.isScalar(code)) return false
  inPairs(ID_START, code)
}

/// True for `_` or a Unicode `ID_Continue` code point, as Veles identifiers are.
public fun isIdentifierContinue(code: i64): bool {
  if (code == 95) return true
  if (code < 0x80) return code >= 65 && code <= 90 || code >= 97 && code <= 122 || code >= 48 && code <= 57
  if (!utf8.isScalar(code)) return false
  inPairs(ID_CONTINUE, code)
}

/// Whether `code` belongs to a Unicode general category or script such as
/// `L`, `Lu`, `Nd`, `Greek` or `Han`; `null` when `name` is unknown.
public fun hasProperty(name: string, code: i64): bool? {
  val ranges = unicodeProperty(name) ?: return null
  if (!utf8.isScalar(code)) return false
  inRanges(ranges, code)
}
