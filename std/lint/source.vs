// Generated from selfhost/source/source.vs by `go test ./selfhost -run LintAPI -update`; edit the source, not this file.

/// One source file in memory, with the byte offset of the start of each
/// line, found once.
public struct File {
  path:       string
  content:    string
  lineStarts: List<i64>

  /// Reads the line table of `content`. Lines end at `\n`; a `\r` before it
  /// belongs to the line (`line` drops it).
  public static fun of(path: string, content: string): File {
    val starts: MutableList<i64> = [0]
    loop (i in 0..<content.len()) {
      if (content.byteAt(i) == '\n') starts.push(i + 1)
    }
    File(path, content, lineStarts: starts)
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
    var text = this.content.slice(start, end)
    loop (text.endsWith("\r")) text = text.slice(0, text.len() - 1)
    text
  }
}

/// A half-open byte range in a file; one without a file is the compiler's
/// own (`<builtin>`).
public struct Span {
  file:  File?
  start: i64
  end:   i64

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
