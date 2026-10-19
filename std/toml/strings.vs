use utf8

fun isControl(byte: u8): bool => byte < 0x20 || byte == 0x7F

extend Cursor {
  /// A quoted string, the quote at the position. Basic strings take escapes,
  /// literal ones do not; the full language also reads `"""…"""` and `'''…'''`.
  fun string(): string throws ParseError {
    val quote = this.peek()
    if (this.rules.fullStrings && this.lookingAt(quote, 3)) return try this.multiline(quote)
    this.pos += 1
    val out = StringBuilder()
    loop (this.more()) {
      val byte = this.peek()
      if (byte == '\n') throw this.fail("a string must close on its own line")
      if (byte == quote) {
        this.pos += 1
        return out.toString()
      }
      if (byte == '\\' && quote == '"') {
        this.pos += 1
        try this.escape(out)
      } else {
        if (this.rules.fullStrings && isControl(byte) && byte != '\t') {
          throw this.fail("a control character in a string must be escaped")
        }
        out.appendByte(byte)
      }
      this.pos += 1
    }
    throw this.fail("unterminated string")
  }

  /// `"""…"""` or `'''…'''`: a newline right after the opening is dropped, and
  /// in a basic string a backslash at the end of a line drops the white space up
  /// to the next text.
  fun multiline(quote: u8): string throws ParseError {
    this.pos += 3
    if (this.eat('\n')) this.line += 1
    val out = StringBuilder()
    loop (this.more()) {
      val byte = this.peek()
      if (byte == quote && this.lookingAt(quote, 3)) {
        // up to two more quotes belong to the text
        var extra: i64 = 0
        loop (extra < 2 && this.peekAt(3 + extra) == quote) extra += 1
        loop (_ in 0..<extra) out.appendByte(quote)
        this.pos += 3 + extra
        return out.toString()
      }
      if (byte == '\\' && quote == '"') {
        this.pos += 1
        if (this.endsLine()) {
          this.skipWhitespaceAndNewlines()
          continue
        }
        try this.escape(out)
        this.pos += 1
        continue
      }
      if (byte == '\n') this.line += 1
      if (isControl(byte) && byte != '\n' && byte != '\t') throw this.fail("a control character in a string must be escaped")
      out.appendByte(byte)
      this.pos += 1
    }
    throw this.fail("unterminated string")
  }

  /// True when only blanks stand between the position and the next newline: a
  /// line-ending backslash.
  fun endsLine(): bool {
    var ahead: i64 = 0
    loop (this.peekAt(ahead) == ' ' || this.peekAt(ahead) == '\t') ahead += 1
    this.peekAt(ahead) == '\n'
  }

  fun skipWhitespaceAndNewlines() {
    loop (this.more()) {
      val byte = this.peek()
      if (byte == '\n') {
        this.line += 1
      } else if (byte != ' ' && byte != '\t') {
        return
      }
      this.pos += 1
    }
  }

  /// The escape after a backslash, the position on its letter; leaves it on
  /// the escape's last byte.
  fun escape(out: StringBuilder) throws ParseError {
    if (!this.more()) throw this.fail("unfinished escape")
    val letter = this.peek()
    val full = this.rules.fullStrings
    when {
      letter == 'n'         => out.append("\n")
      letter == 't'         => out.append("\t")
      letter == '"'         => out.append("\"")
      letter == '\\'        => out.append("\\")
      letter == 'u'         => out.append(try this.unicodeEscape(4))
      full && letter == 'U' => out.append(try this.unicodeEscape(8))
      full && letter == 'b' => out.append(utf8.char(8))
      full && letter == 'f' => out.append(utf8.char(12))
      full && letter == 'r' => out.append("\r")
      full && letter == 'e' => out.append(utf8.char(27))
      else                  => throw this.fail("unknown escape \\${utf8.char(letter.toI64())}")
    }
  }

  /// `count` hex digits after the `u` or `U` at the position, as the character
  /// they name; leaves the position on the last digit.
  fun unicodeEscape(count: i64): string throws ParseError {
    if (this.pos + count >= this.text.len()) throw this.fail("unfinished \\u escape")
    val digits = this.text.substring(this.pos + 1, this.pos + 1 + count) ?: throw this.fail("bad \\u escape")
    val code = parseRadix(digits, 16) ?: throw this.fail("bad \\u escape")
    if (!utf8.isScalar(code)) throw this.fail("bad \\u escape")
    this.pos += count
    utf8.char(code)
  }

  /// A key: bare (`a-b_1`) or quoted.
  fun key(): string throws ParseError {
    if (this.more() && (this.peek() == '"' || this.peek() == '\'')) return try this.string()
    val start = this.pos
    loop (this.more() && isKeyByte(this.peek())) this.pos += 1
    if (start == this.pos) throw this.fail("expected a key")
    this.since(start)
  }
}

fun isKeyByte(byte: u8): bool =>
  byte.isAsciiAlphanumeric() || byte == '_' || byte == '-'
