// Reading a version from text.

/// Digits and nothing else.
fun isNumeric(text: string): bool {
  if (text.isEmpty()) return false
  loop (byte in text.bytes()) {
    if (!byte.isAsciiDigit()) return false
  }
  true
}

/// A whole number below 2^32 without a leading zero, or null.
fun readNumber(text: string): i64? {
  if (!isNumeric(text) || text.len() > 10 || (text.len() > 1 && text.startsWith("0"))) return null
  val value = text.toInt() ?: return null
  if (value > 4294967295) null else value
}

/// Letters, digits, `.` and `-`: what a pre-release may be made of.
fun isPreCharacters(text: string): bool {
  loop (byte in text.bytes()) {
    if (!byte.isAsciiAlphanumeric() && byte != '.' && byte != '-') return false
  }
  true
}

/// The identifiers of a pre-release (`rc.1`), or why it is not one: none empty,
/// a number without a leading zero.
fun readPre(text: string, whole: string): List<string> throws VersionError {
  if (text.isEmpty() || !isPreCharacters(text)) {
    throw VersionError(message: "${quoted(whole)}: the pre-release part after '-' is letters, digits, '.' and '-'")
  }
  val identifiers = text.split(".")
  loop (identifier in identifiers) {
    if (identifier.isEmpty() || (isNumeric(identifier) && identifier.len() > 1 && identifier.startsWith("0"))) {
      throw VersionError(message: "${quoted(whole)}: pre-release identifiers are separated by '.', none empty, a number without a leading zero")
    }
  }
  identifiers
}

/// `1.4.2`, `v1.4.2`, `1.4.2-rc.1`.
fun readVersion(text: string): Version throws VersionError {
  val shown = quoted(text)
  val bare = if (text.startsWith("v")) text.substring(1, text.len()) ?: "" else text
  val (core, preText) = bare.splitOnce("-") ?: (bare, "")
  val pre: List<string> = if (bare.indexOf("-") >= 0) try readPre(preText, text) else []
  if (core.indexOf("+") >= 0) throw VersionError(message: "$shown: build metadata ('+…') is not used in versions")
  val parts = core.split(".")
  if (parts.len() != 3) throw VersionError(message: "$shown is not a version: write major.minor.patch, e.g. 1.4.2")
  val numbers: MutableList<i64> = []
  loop (part in parts) {
    numbers.push(readNumber(part) ?: throw VersionError(message: "$shown is not a version: ${quoted(part)} is not a number without leading zeros"))
  }
  val [major, minor, patch] = numbers.toList() else throw VersionError(message: "$shown is not a version")
  Version(major, minor, patch, pre)
}
