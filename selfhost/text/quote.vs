// Text for messages and dumps: strings quoted with escapes for what is not printable, and
// the nearest name to a misspelt one.
//
// The tables that say which characters are printable or may start a name are
// the lexer's (generated from the Unicode data); they belong in a module of
// their own once the generator moves (checklist §11).
use utf8
use lexer { PRINT }

// whether r is printable: a letter, mark, number, punctuation or symbol, or
// the ASCII space
fun isPrint(r: i64): bool {
  if (r < 0x80) return r >= 0x20 && r < 0x7F
  val i = PRINT.partitionPoint(range => range.1 < r)
  val (first, _) = PRINT.at(i) ?: return false
  first <= r
}

fun hex(n: i64, width: i64): string => n.toString(radix: 16).padStart(width, "0")

/// `text` in double quotes, as the tree shows a string: with
/// `"` and `\` escaped, the C escapes for the control characters that have
/// one, `\xNN` for the other ASCII ones, and `\uNNNN` / `\UNNNNNNNN` for any
/// character above ASCII that is not printable.
fun quote(text: string): string {
  val out = StringBuilder()
  out.append("\"")
  var i: i64 = 0
  loop (i < text.len()) {
    val rune = utf8.decode(text, i) ?: panic("quote: a string is valid UTF-8 at every character start")
    val r = rune.code
    i += rune.size
    if (r == '"'.toI64() || r == '\\'.toI64()) {
      out.append("\\")
      out.append(utf8.char(r))
      continue
    }
    if (isPrint(r)) {
      out.append(utf8.char(r))
      continue
    }
    when (r) {
      0x07 => out.append("\\a")
      0x08 => out.append("\\b")
      0x0C => out.append("\\f")
      0x0A => out.append("\\n")
      0x0D => out.append("\\r")
      0x09 => out.append("\\t")
      0x0B => out.append("\\v")
      else => {
        if (r < 0x20 || r == 0x7F) {
          out.append("\\x${hex(r, 2)}")
        } else if (r < 0x10000) {
          out.append("\\u${hex(r, 4)}")
        } else {
          out.append("\\U${hex(r, 8)}")
        }
      }
    }
  }
  out.append("\"")
  out.toString()
}
