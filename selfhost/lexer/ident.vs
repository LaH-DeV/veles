// Identifier characters (UAX #31) and how a diagnostic names a character.
// The predicates are shared with the bootstrap lexer through std/unicode.
use unicode
use utf8

fun isIDStart(r: i64): bool => unicode.isIdentifierStart(r)

fun isIDContinue(r: i64): bool => unicode.isIdentifierContinue(r)

/// The byte length of the identifier-start character at `src[i]`, or 0.
fun identStartLen(src: string, i: i64): i64 {
  if (i >= src.len()) return 0
  val b = src.byteAt(i)
  if (b < 0x80) return if (unicode.isIdentifierStart(b.toI64())) 1 else 0
  val rune = utf8.decode(src, i) ?: return 0
  if (isIDStart(rune.code)) rune.size else 0
}

/// identStartLen for the characters after the first.
fun identContinueLen(src: string, i: i64): i64 {
  if (i >= src.len()) return 0
  val b = src.byteAt(i)
  if (b < 0x80) return if (unicode.isIdentifierContinue(b.toI64())) 1 else 0
  val rune = utf8.decode(src, i) ?: return 0
  if (isIDContinue(rune.code)) rune.size else 0
}

/// Where the identifier that starts at `i` ends.
fun identEnd(src: string, i: i64): i64 {
  var end = i + identStartLen(src, i)
  loop {
    val n = identContinueLen(src, end)
    if (n == 0) return end
    end += n
  }
}

/// The characters that reorder how text is displayed: the embeddings and
/// overrides, the isolates, the three marks — Trojan source.
fun isBidiControl(r: i64): bool =>
  (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || r == 0x200E || r == 0x200F || r == 0x061C

/// `U+XXXX`: four hex digits at least, upper case.
fun codePoint(r: i64): string => "U+${hexUpper(r).padStart(4, "0")}"

fun hexUpper(n: i64): string => n.toString(radix: 16).toUpper()

/// A character as itself in a message: printable ASCII and everything above.
fun showAsItself(r: i64): bool => r >= 0x20 && r != 0x7F

/// An unexpected character, spelled so that an invisible one is named.
fun describeRune(r: i64): string {
  if (isBidiControl(r)) return "${codePoint(r)}, a bidirectional control character, which makes text display in a different order from how it compiles"
  if (r == 0x00A0 || r == 0x202F || r == 0x2007) return "${codePoint(r)}, a no-break space; use an ordinary space"
  if (r == 0xFEFF) return "U+FEFF, a byte-order mark, which may only begin a file"
  if (r >= 0x80 && isInvisible(r)) return "${codePoint(r)}, an invisible character"
  if (showAsItself(r)) return "'${utf8.char(r)}' (${codePoint(r)})"
  codePoint(r)
}

fun isInvisible(r: i64): bool =>
  unicode.hasProperty("Cf", r) == true ||
  unicode.hasProperty("Zs", r) == true ||
  unicode.hasProperty("Zl", r) == true ||
  unicode.hasProperty("Zp", r) == true
