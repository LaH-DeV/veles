// Source files, positions and diagnostics (veles-selfhost-frontend-plan.md,
// P0). Every span the lexer and parser make is turned into `path:line:col`
// here, so this is checked first and on its own: the harness compares
// `position` on every offset of every corpus file.

/// One source file in memory, with the byte offset of the start of each
/// line, found once.
struct File {
  path:       string
  content:    string
  lineStarts: List<i64>
  /// The bytes of a file that is not UTF-8 that do not decode, by offset
  /// (D199); `content` holds a `?` in place of each, so offsets agree.
  notUtf8: List<(at: i64, byte: u8)> = []

  /// Reads the line table of `content`. Lines end at `\n`; a `\r` before it
  /// belongs to the line (`line` drops it).
  static fun of(path: string, content: string): File {
    val starts: MutableList<i64> = [0]
    loop (i in 0..<content.len()) {
      if (content.byteAt(i) == '\n') starts.push(i + 1)
    }
    File(path, content, lineStarts: starts)
  }

  /// The 1-based line and column of a byte offset. An offset past the end
  /// counts on from the last line's start.
  fun position(offset: i64): (i64, i64) {
    val i = (this.lineStarts.partitionPoint(start => start <= offset) - 1).max(0)
    val start = this.lineStarts.at(i) ?: 0  // i is in range: partitionPoint is at most the length, and 0 when empty
    (i + 1, offset - start + 1)
  }

  /// How many lines the file has: one more than it has `\n`s.
  fun lineCount(): i64 => this.lineStarts.len()

  /// The text of line `n` (1-based) without its line break and any `\r`s
  /// before it; empty for a line that does not exist.
  fun line(n: i64): string {
    val start = this.lineStarts.at(n - 1) ?: return ""
    val end = if (n < this.lineStarts.len()) (this.lineStarts.at(n) ?: 0) - 1 else this.content.len()
    // both ends are at a line start or a `\n`, so the cut cannot split a character
    var text = this.content.slice(start, end)
    loop (text.endsWith("\r")) text = text.slice(0, text.len() - 1)
    text
  }
}

/// A half-open byte range in a file; one without a file is the compiler's
/// own (`<builtin>`).
struct Span {
  file:  (*File)?  // shared by every span of the file: a pointer, not a copy per node (self-host plan §5)
  start: i64
  end:   i64

  /// The span of nothing the source holds: a node that was not written.
  static fun none(): Span => Span(file: null, start: 0, end: 0)

  fun isValid(): bool => this.file != null

  /// From this span's start to the end of `other`.
  fun to(other: Span): Span {
    if (!this.isValid()) return other
    if (!other.isValid()) return this
    Span(file: this.file, start: this.start, end: other.end)
  }

  implement Display {
    fun toString(): string {
      val file = this.file ?: return "<builtin>"
      val (line, col) = file.position(this.start)
      "${file.path}:$line:$col"
    }
  }
}

enum Severity {
  Error
  Warning
  Note
}

/// What a phase reports about the program, at a place in it.
struct Diagnostic {
  severity: Severity
  span:     Span
  message:  string

  implement Display {
    fun toString(): string {
      val word = when (this.severity) {
        Severity.Error   => "error"
        Severity.Warning => "warning"
        Severity.Note    => "note"
      }
      "${this.span}: $word: ${this.message}"
    }
  }
}

/// The diagnostics of a run, in the order they were reported. A phase
/// records a problem and goes on where it can; nothing panics on a user's
/// mistake.
struct Diagnostics {
  items: MutableList<Diagnostic> = []

  fun errorAt(span: Span, message: string) {
    this.items.push(Diagnostic(severity: Severity.Error, span, message))
  }

  fun warnAt(span: Span, message: string) {
    this.items.push(Diagnostic(severity: Severity.Warning, span, message))
  }

  fun hasErrors(): bool => this.items.any(d => d.severity == Severity.Error)

  fun errorCount(): i64 => this.items.count(d => d.severity == Severity.Error)

  /// Every diagnostic with the line it points at and a caret under the
  /// span: the text `veles check` prints, without the `veles explain`
  /// page of each diagnostic's family, which comes in a later phase.
  fun render(): string {
    val out = StringBuilder()
    loop (d in this.items) {
      out.appendLine("$d")
      val file = d.span.file ?: continue
      val (line, col) = file.position(d.span.start)
      val text = file.line(line)
      out.appendLine("  $text")
      out.appendLine("  ${caret(text, col - 1, d.span.end - d.span.start)}")
    }
    out.toString()
  }
}

/// The line under a source line that marks `width` bytes from byte `at`,
/// lined up by characters: a space for each character before the span and
/// a tab for a tab, then one `^` per character of the span on this line, at
/// least one.
fun caret(text: string, at: i64, width: i64): string {
  val start = at.min(text.len())
  val end = (start + width.max(0)).min(text.len())
  val out = StringBuilder()
  var i: i64 = 0
  loop (i < start) {
    out.append(if (text.byteAt(i) == '\t') "\t" else " ")
    i += charSize(text, i, start)
  }
  var count: i64 = 0
  i = start
  loop (i < end) {
    count += 1
    i += charSize(text, i, end)
  }
  out.append("^".repeat(count.max(1)))
  out.toString()
}

// the size of the character at i if it ends by `limit`, else 1: a byte of a
// cut-off or a partly seen character counts as a character of its own (a span need not fall on a
// character's edges)
fun charSize(text: string, i: i64, limit: i64): i64 {
  val b = text.byteAt(i)
  if (b < 0x80) return 1
  val size = when {
    b >= 0xF0 => 4
    b >= 0xE0 => 3
    b >= 0xC0 => 2
    else      => 1
  }
  if (i + size > limit) 1 else size
}

// how many bytes the UTF-8 sequence at i takes, or 0 when it does not decode
// there: Go's `utf8.DecodeRune` rules, so overlong forms, surrogates and
// code points past U+10FFFF are refused
fun utf8Size(bytes: List<u8>, i: i64): i64 {
  val b0 = bytes.at(i) ?: return 0
  if (b0 < 0x80) return 1
  // the second byte's range depends on the first; the rest are 0x80..0xBF
  val (size, lo, hi) = when {
    b0 in 0xC2..0xDF => (2, 0x80, 0xBF)
    b0 == 0xE0       => (3, 0xA0, 0xBF)
    b0 == 0xED       => (3, 0x80, 0x9F)
    b0 in 0xE1..0xEF => (3, 0x80, 0xBF)
    b0 == 0xF0       => (4, 0x90, 0xBF)
    b0 in 0xF1..0xF3 => (4, 0x80, 0xBF)
    b0 == 0xF4       => (4, 0x80, 0x8F)
    else             => return 0
  }
  val b1 = (bytes.at(i + 1) ?: return 0).toI64()
  if (b1 < lo || b1 > hi) return 0
  loop (k in 2..<size) {
    val b = bytes.at(i + k) ?: return 0
    if (b < 0x80 || b > 0xBF) return 0
  }
  size
}

/// A file as read from disk (`File.of` is for text in memory). A file that
/// is not UTF-8 keeps where each byte that does not decode is, in the steps
/// Go's `utf8.DecodeRune` takes, for the lexer to refuse (D199).
fun readSource(path: string, bytes: List<u8>): File {
  if (val text = bytes.decodeUtf8()) return File.of(path, text)
  val clean = bytes.toMutable()
  val bad: MutableList<(at: i64, byte: u8)> = []
  var i: i64 = 0
  loop (i < bytes.len()) {
    val size = utf8Size(bytes, i)
    if (size == 0) {
      bad.push((at: i, byte: bytes.at(i) ?: 0))
      clean.set(i, '?')
      i += 1
    } else {
      i += size
    }
  }
  val file = File.of(path, clean.decodeUtf8() ?: panic("readSource: every byte that does not decode was replaced"))
  File(path, content: file.content, lineStarts: file.lineStarts, notUtf8: bad)
}
