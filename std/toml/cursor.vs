/// The one thing wrong with a document: the line and what.
public error ParseError {
  public line:    i64
  public message: string
}

/// What a dialect of TOML reads. `Rules.full()` is TOML 1.0; `Rules.manifest()`
/// is the subset a package manifest uses, which refuses the rest with a message
/// so that a `veles.toml` cannot mean one thing here and another elsewhere.
struct Rules {
  /// `[[name]]` headers.
  arrayTables: bool
  /// Floats, dates and times, and integers in hex, octal and binary.
  richValues: bool
  /// Multi-line strings, the `\b \f \r \e \U` escapes, control characters refused.
  fullStrings: bool
  /// Definitions are final: a header cannot reopen a table a dotted key made, a
  /// dotted key cannot reach into a defined table, an inline table takes dotted keys.
  closedTables: bool

  static fun full(): Rules => Rules(arrayTables: true, richValues: true, fullStrings: true, closedTables: true)

  static fun manifest(): Rules => Rules(arrayTables: false, richValues: false, fullStrings: false, closedTables: false)
}

/// A position in the text, and the lexical moves every part of the reader makes.
struct Cursor {
  text:     string
  rules:    Rules
  var pos:  i64 = 0
  var line: i64 = 1

  fun more(): bool => this.pos < this.text.len()

  /// The byte at the position; only called where `more()` holds.
  fun peek(): u8 => this.text.byteAt(this.pos)

  /// The byte `ahead` places on, or 0 past the end.
  fun peekAt(ahead: i64): u8 => if (this.pos + ahead < this.text.len()) this.text.byteAt(this.pos + ahead) else 0

  fun fail(message: string): ParseError => ParseError(line: this.line, message)

  fun eat(byte: u8): bool {
    if (this.more() && this.peek() == byte) {
      this.pos += 1
      return true
    }
    false
  }

  /// True when `byte` stands `count` times in a row at the position.
  fun lookingAt(byte: u8, count: i64): bool {
    loop (i in 0..<count) {
      if (this.peekAt(i) != byte) return false
    }
    true
  }

  /// Skips blanks and comments; newlines too when `newlines` is set.
  fun skipSpace(newlines: bool) {
    loop (this.more()) {
      val byte = this.peek()
      if (byte == ' ' || byte == '\t') {
        this.pos += 1
      } else if (byte == '#') {
        loop (this.more() && this.peek() != '\n') this.pos += 1
      } else if (byte == '\n' && newlines) {
        this.pos += 1
        this.line += 1
      } else {
        return
      }
    }
  }

  /// Nothing but a comment may stand before the next newline.
  fun endOfLine() throws ParseError {
    this.skipSpace(false)
    if (this.more() && this.peek() != '\n') throw this.fail("unexpected text after the value")
  }

  /// The text from `start` to the position.
  fun since(start: i64): string => this.text.substring(start, this.pos) ?: panic("Cursor: the reader stops at character edges")
}
