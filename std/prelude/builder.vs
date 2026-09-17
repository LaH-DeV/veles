// Prelude — StringBuilder: builds text piece by piece in linear time,
// where repeated `+` would copy the whole string each time.

/// Accumulates text. `stringBuilder()` starts one; `toString()` reads it.
pub fun stringBuilder(): StringBuilder = StringBuilder()

pub struct StringBuilder {
  bytes: MutableList<u8> = []

  /// Appends `s`.
  pub fun append(s: string) {
    loop (b in s.bytes()) {
      self.bytes.push(b)
    }
  }

  /// Appends `s` and a newline.
  pub fun appendLine(s: string = "") {
    self.append(s)
    self.bytes.push(10)
  }

  /// Length in bytes so far.
  pub fun len(): i64 = self.bytes.len()

  /// True when nothing has been appended.
  pub fun isEmpty(): bool = self.bytes.len() == 0

  /// Empties the builder.
  pub fun clear() {
    self.bytes.clear()
  }

  /// The accumulated text.
  pub fun toString(): string = self.bytes.decodeUtf8() ?: panic("StringBuilder: invalid UTF-8")
}
