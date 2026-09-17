// Prelude — string methods (D18/D19). In scope in every file (D24).
//
// The compiler provides the primitives (`len`, `byteAt`, `bytes`,
// `substring`, `chars`, ...); everything here is built on them in Veles.
// Whitespace and case handling are ASCII-only for now.

extern "C" {
  fun veles_string_find(s: string, part: string, from: i64): i64
}

fun isAsciiSpace(b: u8): bool = b == 32 || b == 9 || b == 10 || b == 13 || b == 12 || b == 11

extend string {
  /// Byte index of the first occurrence of `part` at or after `from`, or -1.
  pub fun indexOf(part: string, from: i64 = 0): i64 = unsafe { veles_string_find(self, part, from) }

  /// Byte index of the last occurrence of `part`, or -1.
  pub fun lastIndexOf(part: string): i64 {
    var last: i64 = -1
    var at = self.indexOf(part)
    loop (at >= 0) {
      last = at
      at = self.indexOf(part, from: at + 1)
    }
    last
  }

  /// The text without leading ASCII whitespace.
  pub fun trimStart(): string {
    var i: i64 = 0
    loop (i < self.len() && isAsciiSpace(self.byteAt(i))) { i += 1 }
    self.substring(i, self.len()) ?: self
  }

  /// The text without trailing ASCII whitespace.
  pub fun trimEnd(): string {
    var j = self.len()
    loop (j > 0 && isAsciiSpace(self.byteAt(j - 1))) { j -= 1 }
    self.substring(0, j) ?: self
  }

  /// The text without leading or trailing ASCII whitespace.
  pub fun trim(): string = self.trimStart().trimEnd()

  /// The pieces between occurrences of `sep`. An empty `sep` yields the
  /// code points; a `sep` that never occurs yields the whole text.
  pub fun split(sep: string): List<string> {
    if (sep.isEmpty()) return self.chars()
    var out: MutableList<string> = []
    var start: i64 = 0
    loop {
      val at = self.indexOf(sep, from: start)
      if (at < 0) break
      out.push(self.substring(start, at) ?: "")
      start = at + sep.len()
    }
    out.push(self.substring(start, self.len()) ?: "")
    out.toList()
  }

  /// The lines of the text, split on `\n`; a trailing `\r` on each line and
  /// a final empty line are dropped.
  pub fun lines(): List<string> {
    var out: MutableList<string> = []
    loop (line in self.split("\n")) {
      if (line.endsWith("\r")) {
        out.push(line.substring(0, line.len() - 1) ?: line)
      } else {
        out.push(line)
      }
    }
    if (out.len() > 0 && (out.last() ?: "").isEmpty()) out.pop()
    out.toList()
  }

  /// The text with every occurrence of `old` replaced by `new`.
  pub fun replace(old: string, new: string): string {
    if (old.isEmpty()) return self
    val parts = self.split(old)
    if (parts.len() == 1) return self
    parts.join(new)
  }

  /// The text repeated `n` times (empty for `n <= 0`).
  pub fun repeat(n: i64): string {
    var out: MutableList<u8> = []
    val bytes = self.bytes()
    loop (_ in 0..<n) {
      loop (b in bytes) { out.push(b) }
    }
    out.decodeUtf8() ?: ""
  }

  /// Copy with ASCII letters upper-cased.
  pub fun toUpper(): string {
    var out: MutableList<u8> = []
    loop (b in self.bytes()) {
      out.push(if (b >= 97 && b <= 122) b - 32 else b)
    }
    out.decodeUtf8() ?: self
  }

  /// Copy with ASCII letters lower-cased.
  pub fun toLower(): string {
    var out: MutableList<u8> = []
    loop (b in self.bytes()) {
      out.push(if (b >= 65 && b <= 90) b + 32 else b)
    }
    out.decodeUtf8() ?: self
  }

  /// The text preceded by `pad` until it is at least `width` bytes long.
  pub fun padStart(width: i64, pad: string = " "): string {
    if (pad.isEmpty() || self.len() >= width) return self
    var s = self
    loop (s.len() < width) { s = pad + s }
    s
  }

  /// The text followed by `pad` until it is at least `width` bytes long.
  pub fun padEnd(width: i64, pad: string = " "): string {
    if (pad.isEmpty() || self.len() >= width) return self
    var s = self
    loop (s.len() < width) { s = s + pad }
    s
  }

  /// Parses a decimal number with an optional fraction and exponent, or
  /// `null` when the text is not one.
  pub fun toF64(): f64? {
    val s = self.trim()
    if (s.isEmpty()) return null
    var i: i64 = 0
    var neg = false
    if (s.byteAt(0) == 45 || s.byteAt(0) == 43) {
      neg = s.byteAt(0) == 45
      i = 1
    }
    var mantissa = 0.0
    var digits: i64 = 0
    loop (i < s.len() && isDigit(s.byteAt(i))) {
      mantissa = mantissa * 10.0 + (s.byteAt(i) - 48) as f64
      i += 1
      digits += 1
    }
    var scale: i64 = 0
    if (i < s.len() && s.byteAt(i) == 46) {
      i += 1
      loop (i < s.len() && isDigit(s.byteAt(i))) {
        mantissa = mantissa * 10.0 + (s.byteAt(i) - 48) as f64
        scale -= 1
        i += 1
        digits += 1
      }
    }
    if (digits == 0) return null
    if (i < s.len() && (s.byteAt(i) == 101 || s.byteAt(i) == 69)) {
      i += 1
      var expNeg = false
      if (i < s.len() && (s.byteAt(i) == 45 || s.byteAt(i) == 43)) {
        expNeg = s.byteAt(i) == 45
        i += 1
      }
      var exp: i64 = 0
      var expDigits: i64 = 0
      loop (i < s.len() && isDigit(s.byteAt(i))) {
        exp = exp * 10 + (s.byteAt(i) - 48) as i64
        i += 1
        expDigits += 1
      }
      if (expDigits == 0) return null
      scale += if (expNeg) -exp else exp
    }
    if (i != s.len()) return null
    val value = mantissa * 10.0.pow(scale as f64)
    if (neg) -value else value
  }
}

fun isDigit(b: u8): bool = b >= 48 && b <= 57

