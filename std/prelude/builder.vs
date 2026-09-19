// Prelude — StringBuilder: builds text piece by piece in linear time,
// where repeated `+` would copy the whole string each time.

/// Accumulates text. `stringBuilder()` starts one; `toString()` reads it.
public fun stringBuilder(): StringBuilder = StringBuilder()

public struct StringBuilder {
  bytes: MutableList<u8> = []

  /// Appends `s`.
  public fun append(s: string) {
    loop (b in s.bytes()) {
      self.bytes.push(b)
    }
  }

  /// Appends `s` and a newline.
  public fun appendLine(s: string = "") {
    self.append(s)
    self.bytes.push(10)
  }

  /// Appends one byte of UTF-8 — for code that walks a string with `byteAt`
  /// and copies most of it through (`toString` checks the result is text).
  public fun appendByte(b: u8) {
    self.bytes.push(b)
  }

  /// Length in bytes so far.
  public fun len(): i64 = self.bytes.len()

  /// True when nothing has been appended.
  public fun isEmpty(): bool = self.bytes.len() == 0

  /// Empties the builder.
  public fun clear() {
    self.bytes.clear()
  }

  /// The accumulated text.
  public fun toString(): string = self.bytes.decodeUtf8() ?: panic("StringBuilder: invalid UTF-8")
}
