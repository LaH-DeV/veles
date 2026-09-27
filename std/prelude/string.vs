// Prelude — string methods (D18/D19). In scope in every file (D24).
//
// The compiler provides the primitives (`len`, `byteAt`, `bytes`,
// `substring`, `chars`, ...); everything here is built on them in Veles.
// Whitespace and case handling are ASCII-only for now.

extern "C" {
  fun veles_string_find(s: string, part: string, from: i64): i64
  fun veles_parse_f64(s: string): f64
}

fun isAsciiSpace(b: u8): bool = b == 32 || b == 9 || b == 10 || b == 13 || b == 12 || b == 11

extend string {
  /// Byte index of the first occurrence of `part` at or after `from`, or -1.
  public fun indexOf(part: string, from: i64 = 0): i64 = unsafe {
    veles_string_find(this, part, from)
  }

  /// Byte index of the last occurrence of `part`, or -1.
  public fun lastIndexOf(part: string): i64 {
    var last: i64 = -1
    var at = this.indexOf(part)
    loop (at >= 0) {
      last = at
      at = this.indexOf(part, from: at + 1)
    }
    last
  }

  /// The text without leading ASCII whitespace.
  public fun trimStart(): string {
    var i: i64 = 0
    loop (i < this.len() && isAsciiSpace(this.byteAt(i))) {
      i += 1
    }
    this.substring(i, this.len()) ?: this
  }

  /// The text without trailing ASCII whitespace.
  public fun trimEnd(): string {
    var j = this.len()
    loop (j > 0 && isAsciiSpace(this.byteAt(j - 1))) {
      j -= 1
    }
    this.substring(0, j) ?: this
  }

  /// The text without leading or trailing ASCII whitespace.
  public fun trim(): string = this.trimStart().trimEnd()

  /// The pieces between occurrences of `sep`. An empty `sep` yields the
  /// code points; a `sep` that never occurs yields the whole text.
  public fun split(sep: string): List<string> {
    if (sep.isEmpty()) return this.chars()
    stringSplit(this, sep)
  }

  /// The text before and after the first `sep`, or `null` when `sep` does
  /// not occur: `"key=a=b".splitOnce("=")` is `("key", "a=b")`. An empty
  /// `sep` splits before the first character: `("", text)`.
  public fun splitOnce(sep: string): (string, string)? {
    val at = this.indexOf(sep)
    if (at < 0) return null
    val before = this.substring(0, at) ?: panic("splitOnce: indexOf found sep inside the text")
    val after = this.substring(at + sep.len(), this.len()) ?: panic("splitOnce: indexOf found sep inside the text")
    (before, after)
  }

  /// The lines of the text, split on `\n`; a trailing `\r` on each line and
  /// a final empty line are dropped.
  public fun lines(): List<string> {
    var out: MutableList<string> = []
    loop (line in this.split("\n")) {
      if (line.endsWith("\r")) {
        out.push(line.substring(0, line.len() - 1) ?: line)
      } else {
        out.push(line)
      }
    }
    if (out.len() > 0 && out.last().isEmpty()) out.pop()
    out.toList()
  }

  /// The text with every occurrence of `old` replaced by `new`.
  public fun replace(old: string, new: string): string {
    if (old.isEmpty()) return this
    val parts = this.split(old)
    if (parts.len() == 1) return this
    parts.join(new)
  }

  /// The text repeated `n` times (empty for `n <= 0`).
  public fun repeat(n: i64): string {
    var out: MutableList<u8> = []
    loop (_ in 0..<n) {
      listAppendText(out, this)
    }
    out.decodeUtf8() ?: ""
  }

  /// Copy with ASCII letters upper-cased.
  public fun toUpper(): string {
    var out: MutableList<u8> = []
    loop (b in this.bytes()) {
      out.push(if (b >= 97 && b <= 122) b - 32 else b)
    }
    out.decodeUtf8() ?: this
  }

  /// Copy with ASCII letters lower-cased.
  public fun toLower(): string {
    var out: MutableList<u8> = []
    loop (b in this.bytes()) {
      out.push(if (b >= 65 && b <= 90) b + 32 else b)
    }
    out.decodeUtf8() ?: this
  }

  /// The text preceded by `pad` until it is at least `width` bytes long.
  public fun padStart(width: i64, pad: string = " "): string {
    if (pad.isEmpty() || this.len() >= width) return this
    var s = this
    loop (s.len() < width) {
      s = pad + s
    }
    s
  }

  /// The text followed by `pad` until it is at least `width` bytes long.
  public fun padEnd(width: i64, pad: string = " "): string {
    if (pad.isEmpty() || this.len() >= width) return this
    var s = this
    loop (s.len() < width) {
      s = s + pad
    }
    s
  }

  /// Parses a decimal number with an optional fraction and exponent, or
  /// `null` when the text is not one. The result is the f64 *nearest* the
  /// decimal value, correctly rounded, so the text an f64 prints as reads
  /// back as the same f64 — what makes a JSON float survive a round trip.
  public fun toF64(): f64? {
    val s = this.trim()
    if (s.isEmpty()) return null
    // the grammar is checked here; the value comes from the C library's
    // strtod, which rounds correctly where summing digits in floating
    // point does not (examples/fuzz found `-1.9223372036854771` drifting)
    var i: i64 = 0
    if (s.byteAt(0) == 45 || s.byteAt(0) == 43) i = 1
    var digits: i64 = 0
    loop (i < s.len() && isDigit(s.byteAt(i))) {
      i += 1
      digits += 1
    }
    if (i < s.len() && s.byteAt(i) == 46) {
      i += 1
      loop (i < s.len() && isDigit(s.byteAt(i))) {
        i += 1
        digits += 1
      }
    }
    if (digits == 0) return null
    if (i < s.len() && (s.byteAt(i) == 101 || s.byteAt(i) == 69)) {
      i += 1
      if (i < s.len() && (s.byteAt(i) == 45 || s.byteAt(i) == 43)) i += 1
      val expStart = i
      loop (i < s.len() && isDigit(s.byteAt(i))) i += 1
      if (i == expStart) return null
    }
    if (i != s.len()) return null
    unsafe {
      veles_parse_f64(s)
    }
  }
}

fun isDigit(b: u8): bool = b >= 48 && b <= 57
