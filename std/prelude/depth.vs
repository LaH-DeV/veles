// Prelude — the one bound on recursion depth. In scope in every file (D24).
//
// Every walk over input someone else wrote is a walk of unknown depth: a
// JSON document of 100 000 `[`, a request path that nests forever, a source
// file of 50 000 `(`. Veles runs on the C stack and does not grow it, so an
// unbounded recursive descent is a segmentation fault, not an error — the
// one failure mode a server may not have.
//
// So there is one default here, one sentence to report reaching it, and one
// counter for the walks that need one — and every decoder, encoder and
// parser in the standard library uses them. A walk that already keeps a
// stack (`std/json`, `ValueEncoder`) has its depth in hand and needs only
// the limit and `tooDeepMessage`; a walk that keeps nothing but its call
// frames (a recursive descent parser) counts them with `Depth`.

/// The default bound on how deep a recursive walk over untrusted input may
/// go: 1000 levels.
///
/// The number is a stack budget, not a taste: the smallest stack Veles is
/// expected to run on is Windows' 1 MiB default, and a level of a recursive
/// descent costs on the order of a kilobyte once its handful of frames are
/// counted. A walk whose frames are larger than that — many locals, a big
/// value returned by copy — should say so and pass a smaller `limit`; a
/// format with its own convention should follow the format (`json.Options`
/// defaults to 64, which is what JSON documents in the wild look like).
public val maxRecursionDepth: i64 = 1000

/// What every recursion limit in the standard library says when it is
/// reached — `nesting deeper than 64` — so the wording is one sentence and
/// not five. The caller wraps it in whatever its own errors carry: a byte
/// offset, a field path, a source span.
public fun tooDeepMessage(limit: i64): string = "nesting deeper than $limit"

/// How deep a recursive walk has gone, and how deep it may go.
///
/// ```veles
/// struct Parser {
///   private depth: Depth = Depth.of(500)
///
///   fun parseExpr(): Expr {
///     if (!this.depth.enter()) return this.tooDeep()
///     val e = this.parseBinary(0)
///     this.depth.leave()
///     e
///   }
/// }
/// ```
///
/// `enter` and `leave` pair like a push and a pop, and the pairing is the
/// caller's to get right: every path out of the body — an early `return`, a
/// `break`, a fall-through — leaves the level it entered. A path that
/// throws is the exception, and deliberately: a throw abandons the whole
/// walk, so the counter is abandoned with it.
///
/// The level is never negative, and `enter` past the limit does not raise
/// it, so a walk that stops on `false` and unwinds is left exactly where it
/// started.
public struct Depth {
  /// The deepest the walk may go. Reaching it is what `enter` refuses.
  public limit:      i64 = maxRecursionDepth
  private var level: i64 = 0
  private var peak:  i64 = 0

  /// A counter at level zero. This is how one is made: the fields it counts
  /// with are private, so the implicit constructor stays inside the prelude
  /// (D28).
  public static fun of(limit: i64 = maxRecursionDepth): Depth = Depth(limit)

  /// Goes one level deeper, or reports `false` when `limit` levels are
  /// already open — in which case nothing changed and there is no `leave`
  /// to pair with it.
  public fun enter(): bool {
    if (this.level >= this.limit) return false
    this.level += 1
    if (this.level > this.peak) this.peak = this.level
    true
  }

  /// Comes back one level. Ignored at level zero, so an unbalanced `leave`
  /// cannot make the next `enter` succeed past the limit.
  public fun leave() {
    if (this.level > 0) this.level -= 1
  }

  /// How many levels are open.
  public fun depth(): i64 = this.level

  /// The deepest this walk ever went — what a benchmark reports and a test
  /// asserts on. Never reset by `leave`; `reset` clears it.
  public fun deepest(): i64 = this.peak

  /// Back to nothing open, ready to walk again.
  public fun reset() {
    this.level = 0
    this.peak = 0
  }

  /// What to say when `enter` refuses: `tooDeepMessage(this.limit)`.
  public fun message(): string = tooDeepMessage(this.limit)
}
