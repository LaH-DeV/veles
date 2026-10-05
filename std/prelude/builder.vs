// Prelude — StringBuilder: builds text piece by piece in linear time,
// where repeated `+` would copy the whole string each time.

/// Accumulates text. `StringBuilder()` starts one; `toString()` reads it.
public struct StringBuilder {
  private bytes: MutableList<u8> = []

  /// Appends `s`.
  public const fun append(s: string) {
    listAppendText(this.bytes, s)
  }

  /// Appends `s` and a newline.
  public const fun appendLine(s: string = "") {
    this.append(s)
    this.bytes.push(10)
  }

  /// Appends one byte of UTF-8 — for code that walks a string with `byteAt`
  /// and copies most of it through (`toString` checks the result is text).
  public const fun appendByte(b: u8) {
    this.bytes.push(b)
  }

  /// Makes room for `n` bytes in all, so appending up to `n` does not grow
  /// the buffer; never shrinks, never changes the text (D105).
  public const fun reserve(n: i64) {
    this.bytes.reserve(n)
  }

  /// Length in bytes so far.
  public const fun len(): i64 = this.bytes.len()

  /// True when nothing has been appended.
  public const fun isEmpty(): bool = this.bytes.len() == 0

  /// Empties the builder.
  public const fun clear() {
    this.bytes.clear()
  }

  /// The accumulated text.
  public const fun toString(): string = this.bytes.decodeUtf8() ?: panic("StringBuilder: invalid UTF-8")
}
