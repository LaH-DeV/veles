// Dotenv files (D125): `KEY=value` lines, read literally. No interpolation: a
// `$` is a `$`, so a password is never expanded into something else.

const NEWLINE: u8 = 10
const SPACE: u8 = 32
const TAB: u8 = 9
const HASH: u8 = 35
const EQUALS: u8 = 61
const DOUBLE: u8 = 34
const SINGLE: u8 = 39
const BACKSLASH: u8 = 92

/// What a dotenv text holds: the variables, and what was wrong with the
/// lines that are not variables (`line 3: ...`).
struct Dotenv {
  vars:     Map<string, string>
  problems: List<string>
}

// A cursor over the text; `line` counts the newlines passed, for messages.
struct Scanner {
  text:     string
  var pos:  i64 = 0
  var line: i64 = 1

  fun atEnd(): bool = this.pos >= this.text.len()

  fun peek(): u8 = if (this.atEnd()) 0 else this.text.byteAt(this.pos)

  fun skipBlank() {
    loop (this.peek() == SPACE || this.peek() == TAB) {
      this.pos += 1
    }
  }

  // past the end of this line
  fun skipLine() {
    loop (!this.atEnd() && this.peek() != NEWLINE) {
      this.pos += 1
    }
    if (!this.atEnd()) {
      this.pos += 1
      this.line += 1
    }
  }

  fun slice(from: i64, to: i64): string = this.text.substring(from, to) ?: ""

  // the text up to a quote on the same or a later line, `null` when there is none;
  // a double-quoted value reads `\n`, `\t`, `\r`, `\"` and `\\`
  fun quoted(quote: u8): string? {
    val out = StringBuilder()
    loop (!this.atEnd()) {
      val b = this.peek()
      this.pos += 1
      if (b == quote) return out.toString()
      if (b == NEWLINE) this.line += 1
      if (b == BACKSLASH && quote == DOUBLE && !this.atEnd()) {
        val next = this.peek()
        this.pos += 1
        if (next == 'n') {
          out.appendByte(NEWLINE)
        } else if (next == 't') {
          out.appendByte(TAB)
        } else if (next == 'r') {
          out.appendByte(13)
        } else {
          if (next != DOUBLE && next != BACKSLASH) out.appendByte(BACKSLASH)
          out.appendByte(next)
        }
        if (next == NEWLINE) this.line += 1
        continue
      }
      out.appendByte(b)
    }
    null
  }

  // the rest of the line, up to a ` #` that starts a comment, trimmed
  fun unquoted(): string {
    val begin = this.pos
    loop (!this.atEnd() && this.peek() != NEWLINE) {
      this.pos += 1
    }
    val line = this.slice(begin, this.pos)
    var end = line.len()
    var i = 1
    loop (i < line.len()) {
      val before = line.byteAt(i - 1)
      if (line.byteAt(i) == HASH && (before == SPACE || before == TAB)) {
        end = i - 1
        break
      }
      i += 1
    }
    (line.substring(0, end) ?: line).trim()
  }
}

fun validName(name: string): bool {
  if (name.isEmpty()) return false
  var i = 0
  loop (i < name.len()) {
    val b = name.byteAt(i)
    val letter = b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b == '_'
    val digit = b >= '0' && b <= '9'
    if (!(letter || i > 0 && (digit || b == '.' || b == '-'))) return false
    i += 1
  }
  true
}

/// The variables of a dotenv text: `KEY=value`, one per line; blank lines and
/// lines starting with `#` are skipped, `export ` before a name is ignored,
/// `'…'` is literal, `"…"` reads escapes and may span lines, and an unquoted
/// value ends at the line or at ` #`. A line that is none of these is a
/// problem, and reading goes on with the next.
fun parseDotenv(text: string): Dotenv {
  val vars: MutableMap<string, string> = [:]
  val problems: MutableList<string> = []
  var scan = Scanner(text: text.replace("\r\n", "\n"))
  loop (!scan.atEnd()) {
    scan.skipBlank()
    val first = scan.peek()
    if (first == NEWLINE || first == HASH) {
      scan.skipLine()
      continue
    }
    if (scan.atEnd()) break
    val line = scan.line
    val begin = scan.pos
    loop (!scan.atEnd() && scan.peek() != EQUALS && scan.peek() != NEWLINE) {
      scan.pos += 1
    }
    if (scan.peek() != EQUALS) {
      problems.push("line $line: expected NAME=value")
      scan.skipLine()
      continue
    }
    var name = scan.slice(begin, scan.pos).trim()
    if (name.startsWith("export ") || name.startsWith("export\t")) name = (name.substring(7, name.len()) ?: "").trim()
    scan.pos += 1
    if (!validName(name)) {
      problems.push("line $line: '$name' is not a variable name")
      scan.skipLine()
      continue
    }
    val afterEquals = scan.pos
    scan.skipBlank()
    val b = scan.peek()
    if (b == DOUBLE || b == SINGLE) {
      scan.pos += 1
      val value = scan.quoted(b)
      if (value == null) {
        problems.push("line $line: the quote opened here is never closed")
        continue
      }
      scan.skipBlank()
      if (!scan.atEnd() && scan.peek() != NEWLINE && scan.peek() != HASH) {
        problems.push("line $line: text after the closing quote")
        scan.skipLine()
        continue
      }
      scan.skipLine()
      vars.set(name, value)
      continue
    }
    // `NAME= # note` is empty; `NAME=#x` is the text `#x`
    if (b == HASH && scan.pos > afterEquals) {
      scan.skipLine()
      vars.set(name, "")
      continue
    }
    vars.set(name, scan.unquoted())
    scan.skipLine()
  }
  Dotenv(vars: vars.toMap(), problems: problems.toList())
}
