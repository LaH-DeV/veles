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
  public const fun len(): i64 => this.bytes.len()

  /// True when nothing has been appended.
  public const fun isEmpty(): bool => this.bytes.len() == 0

  /// Empties the builder.
  public const fun clear() {
    this.bytes.clear()
  }

  /// Removes leading and trailing ASCII whitespace, in place.
  public const fun trim() {
    listTrimText(this.bytes)
  }

  /// Lower-cases the ASCII letters, in place.
  public const fun toLower() {
    listAsciiCase(this.bytes, false)
  }

  /// Upper-cases the ASCII letters, in place.
  public const fun toUpper() {
    listAsciiCase(this.bytes, true)
  }

  /// Replaces every occurrence of `old` with `new`, in place: no new buffer
  /// unless `new` is longer than `old`. An empty `old` changes nothing.
  public const fun replace(old: string, new: string) {
    if (!old.isEmpty()) listReplaceText(this.bytes, old, new)
  }

  /// Inserts `text` at byte index `at`, which is `0..len()` and on a
  /// character's edge.
  @caller_location
  public const fun insert(at: i64, text: string) {
    if (at < 0 || at > this.bytes.len()) panic("insert: index $at out of bounds for a builder of ${this.bytes.len()} bytes")
    if (((this.bytes.at(at) ?: 0) & 0xC0) == 0x80) panic("insert: index $at is inside a character")
    listInsertText(this.bytes, at, text)
  }

  /// The accumulated text.
  public const fun toString(): string => this.bytes.decodeUtf8() ?: panic("StringBuilder: invalid UTF-8")

  // the text an `edit` ends with (D207): the bytes themselves, not a copy;
  // the builder is frozen, and nothing changes it again
  const fun takeText(): string => listTakeText(this.bytes) ?: panic("StringBuilder: invalid UTF-8")
}

// the builder `text.edit({ … })` runs its block on (D207): one copy of the text
const fun textBuilder(text: string): StringBuilder {
  val builder = StringBuilder()
  builder.reserve(text.len())
  builder.append(text)
  builder
}

// the text an `edit` gives back: the builder's bytes, shared (D207)
const fun builderText(builder: StringBuilder): string => builder.takeText()
