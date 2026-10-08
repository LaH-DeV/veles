// Source files, positions and diagnostics (veles-selfhost-frontend-plan.md,
// P0). Every span the lexer and parser make is turned into `path:line:col`
// here, so this is checked first and on its own: the harness compares
// `position` on every offset of every corpus file.

/// One source file in memory, with the byte offset of the start of each
/// line, found once.
public struct File {
  public path:    string
  public content: string
  lineStarts:     List<i64>

  /// Reads the line table of `content`. Lines end at `\n`; a `\r` before it
  /// belongs to the line (`line` drops it).
  public static fun of(path: string, content: string): File {
    val starts: MutableList<i64> = [0]
    loop (i in 0..<content.len()) {
      if (content.byteAt(i) == '\n') starts.push(i + 1)
    }
    File(path, content, lineStarts: starts.toList())
  }

  /// The 1-based line and column of a byte offset. An offset past the end
  /// counts on from the last line's start.
  public fun position(offset: i64): (i64, i64) {
    val i = (this.lineStarts.partitionPoint(start => start <= offset) - 1).max(0)
    val start = this.lineStarts.at(i) ?: 0  // i is in range: partitionPoint is at most the length, and 0 when empty
    (i + 1, offset - start + 1)
  }

  /// How many lines the file has: one more than it has `\n`s.
  public fun lineCount(): i64 => this.lineStarts.len()

  /// The text of line `n` (1-based) without its line break and any `\r`s
  /// before it; empty for a line that does not exist.
  public fun line(n: i64): string {
    val start = this.lineStarts.at(n - 1) ?: return ""
    val end = if (n < this.lineStarts.len()) (this.lineStarts.at(n) ?: 0) - 1 else this.content.len()
    // both ends are at a line start or a `\n`, so the cut cannot split a character
    var text = this.content.substring(start, end) ?: ""
    loop (text.endsWith("\r")) text = text.substring(0, text.len() - 1) ?: ""
    text
  }
}

/// A half-open byte range in a file; one without a file is the compiler's
/// own (`<builtin>`).
public struct Span {
  public file:  File?
  public start: i64
  public end:   i64

  /// The span of nothing the source holds: a node that was not written.
  public static fun none(): Span => Span(file: null, start: 0, end: 0)

  public fun isValid(): bool => this.file != null

  /// From this span's start to the end of `other`.
  public fun to(other: Span): Span {
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

public enum Severity {
  Error
  Warning
  Note
}

/// What a phase reports about the program, at a place in it.
public struct Diagnostic {
  public severity: Severity
  public span:     Span
  public message:  string

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
public struct Diagnostics {
  public items: MutableList<Diagnostic> = []

  public fun errorAt(span: Span, message: string) {
    this.items.push(Diagnostic(severity: Severity.Error, span, message))
  }

  public fun warnAt(span: Span, message: string) {
    this.items.push(Diagnostic(severity: Severity.Warning, span, message))
  }

  public fun hasErrors(): bool => this.items.any(d => d.severity == Severity.Error)

  public fun errorCount(): i64 => this.items.count(d => d.severity == Severity.Error)

  /// Every diagnostic with the line it points at and a caret under the
  /// span: the text `veles check` prints, without the `veles explain`
  /// page of each diagnostic's family, which comes in a later phase.
  public fun render(): string {
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
public fun caret(text: string, at: i64, width: i64): string {
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
  val size = if (b >= 0xF0) 4 else if (b >= 0xE0) 3 else if (b >= 0xC0) 2 else 1
  if (i + size > limit) 1 else size
}
