// Prelude — string methods (D18/D19). In scope in every file (D24).
//
// The compiler provides the primitives (`len`, `byteAt`, `bytes`,
// `substring`, `chars`, ...); everything here is built on them in Veles.
// Whitespace and case handling are ASCII-only for now.

extern "C" {
  fun veles_string_find(s: string, part: string, from: i64): i64
  fun veles_parse_f64(s: string): f64
}

extend string {
  /// True when some byte of the text is `b` (the byte form of `b in text`, D189).
  public const fun containsByte(b: u8): bool {
    loop (i in 0..<this.len()) {
      if (this.byteAt(i) == b) return true
    }
    false
  }

  /// The bytes in `from..<to` as text, whatever the bounds: both ends are clamped to
  /// the string, a bound inside a character moves outward to the character's edge
  /// (so the piece holds every byte asked for), and `from >= to` is the empty
  /// string. Like `substring` it shares the original's bytes: no copy. Use
  /// `substring` where an invalid range must be told from an empty one (D192).
  public const fun slice(from: i64, to: i64): string {
    val n = this.len()
    var lo = if (from < 0) 0 else if (from > n) n else from
    var hi = if (to < 0) 0 else if (to > n) n else to
    if (lo >= hi) return ""
    loop (lo > 0 && (this.byteAt(lo) & 0xC0) == 0x80) lo -= 1
    loop (hi < n && (this.byteAt(hi) & 0xC0) == 0x80) hi += 1
    this.substring(lo, hi) ?: panic("slice: both bounds are in range and on character edges")
  }

  /// Byte index of the first occurrence of `part` at or after `from`, or
  /// null when there is none (D165).
  public const fun indexOf(part: string, from: i64 = 0): i64? {
    val at = unsafe {
      // SAFETY: a read-only search inside both strings' lengths; `from` is
      // clamped by the runtime, which answers -1 when `part` is absent
      veles_string_find(this, part, from)
    }
    if (at < 0) null else at
  }

  /// Byte index of the last occurrence of `part`, or null when there is
  /// none (D165).
  public const fun lastIndexOf(part: string): i64? {
    var last: i64? = null
    var at = this.indexOf(part)
    loop (at != null) {
      last = at
      at = this.indexOf(part, from: at + 1)
    }
    last
  }

  /// The text without leading ASCII whitespace.
  public const fun trimStart(): string {
    var i: i64 = 0
    loop (i < this.len() && this.byteAt(i).isAsciiSpace()) {
      i += 1
    }
    this.substring(i, this.len()) ?: this
  }

  /// The text without trailing ASCII whitespace.
  public const fun trimEnd(): string {
    var j = this.len()
    loop (j > 0 && this.byteAt(j - 1).isAsciiSpace()) {
      j -= 1
    }
    this.substring(0, j) ?: this
  }

  /// The text without leading or trailing ASCII whitespace.
  public const fun trim(): string => this.trimStart().trimEnd()

  /// The pieces between occurrences of `sep`. An empty `sep` yields the
  /// code points; a `sep` that never occurs yields the whole text.
  public const fun split(sep: string): List<string> {
    if (sep.isEmpty()) return this.chars()
    stringSplit(this, sep)
  }

  /// The text before and after the first `sep`, or `null` when `sep` does
  /// not occur: `"key=a=b".splitOnce("=")` is
  /// `(before: "key", after: "a=b")`. An empty `sep` splits before the first
  /// character: `(before: "", after: text)`.
  public const fun splitOnce(sep: string): (before: string, after: string)? {
    val at = this.indexOf(sep) ?: return null
    val before = this.substring(0, at) ?: panic("splitOnce: indexOf found sep inside the text")
    val after = this.substring(at + sep.len(), this.len()) ?: panic("splitOnce: indexOf found sep inside the text")
    (before, after)
  }

  /// The lines of the text, split on `\n`; a trailing `\r` on each line and
  /// a final empty line are dropped.
  public const fun lines(): List<string> {
    var out: MutableList<string> = []
    loop (line in this.split("\n")) {
      if (line.endsWith("\r")) {
        out.push(line.substring(0, line.len() - 1) ?: line)
      } else {
        out.push(line)
      }
    }
    if (out.len() > 0 && out.last().isEmpty()) out.pop()
    out
  }

  /// The text with every occurrence of `old` replaced by `new`.
  public const fun replace(old: string, new: string): string {
    if (old.isEmpty()) return this
    val parts = this.split(old)
    if (parts.len() == 1) return this
    parts.join(new)
  }

  /// The text repeated `n` times (empty for `n <= 0`).
  public const fun repeat(n: i64): string {
    var out: MutableList<u8> = []
    loop (_ in 0..<n) {
      listAppendText(out, this)
    }
    out.decodeUtf8() ?: ""
  }

  /// Copy with ASCII letters upper-cased.
  public const fun toUpper(): string {
    var out: MutableList<u8> = []
    loop (b in this.bytes()) {
      out.push(b.toAsciiUpper())
    }
    out.decodeUtf8() ?: this
  }

  /// Copy with ASCII letters lower-cased.
  public const fun toLower(): string {
    var out: MutableList<u8> = []
    loop (b in this.bytes()) {
      out.push(b.toAsciiLower())
    }
    out.decodeUtf8() ?: this
  }

  /// Copy with the first ASCII letter upper-cased, the rest unchanged.
  public const fun capitalize(): string {
    if (this.isEmpty()) return this
    val first = this.byteAt(0)
    if (first < 97 || first > 122) return this
    val rest = this.substring(1, this.len()) ?: ""
    val out: MutableList<u8> = [first - 32]
    listAppendText(out, rest)
    out.decodeUtf8() ?: this
  }

  /// The text preceded by `pad` until it is at least `width` bytes long.
  public const fun padStart(width: i64, pad: string = " "): string {
    if (pad.isEmpty() || this.len() >= width) return this
    var s = this
    loop (s.len() < width) {
      s = pad + s
    }
    s
  }

  /// The text followed by `pad` until it is at least `width` bytes long.
  public const fun padEnd(width: i64, pad: string = " "): string {
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
    loop (i < s.len() && s.byteAt(i).isAsciiDigit()) {
      i += 1
      digits += 1
    }
    if (i < s.len() && s.byteAt(i) == 46) {
      i += 1
      loop (i < s.len() && s.byteAt(i).isAsciiDigit()) {
        i += 1
        digits += 1
      }
    }
    if (digits == 0) return null
    if (i < s.len() && (s.byteAt(i) == 101 || s.byteAt(i) == 69)) {
      i += 1
      if (i < s.len() && (s.byteAt(i) == 45 || s.byteAt(i) == 43)) i += 1
      val expStart = i
      loop (i < s.len() && s.byteAt(i).isAsciiDigit()) i += 1
      if (i == expStart) return null
    }
    if (i != s.len()) return null
    // SAFETY: `s` was checked above to be a well-formed decimal number, and
    // strtod reads it within its length
    unsafe {
      veles_parse_f64(s)
    }
  }
}
