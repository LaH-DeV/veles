// A recursion limit the way a parser uses one (checklist §2,
// veles-selfhost-frontend-plan.md §4.3).
//
// This is a recursive descent over text someone else wrote, which is a walk
// of unknown depth: `((((((...1...))))))` costs one frame per bracket, and
// Veles runs on the C stack and does not grow it. Without a bound, a file
// of 50 000 brackets is a segmentation fault — the one failure mode a
// program that reads other people's input may not have. With `recursion.Depth` it is
// an ordinary error, reported where it happened, in the sentence every
// other limit in the library uses.

use io { println }, recursion

/// The grammar refused the input because it nested too far.
error TooDeep {
  public limit: i64
  /// The byte offset the parser had reached.
  public at: i64
  fun message(): string = "at byte ${this.at}: ${recursion.tooDeepMessage(this.limit)}"
}

/// `expr := term ('+' term)*` and `term := digits | '(' expr ')'`, which is
/// the smallest grammar with a recursive production.
struct Calc {
  src:     string
  var pos: i64 = 0
  // deliberately small, so the example's output fits on a page; a real
  // parser passes `recursion.maxRecursionDepth` or a fraction of it
  depth: recursion.Depth = recursion.Depth(limit: 16)

  fun parseExpr(): i64 throws TooDeep {
    // every path out of this body leaves the level it entered, except the
    // throw — which abandons the whole walk, counter and all
    if (!this.depth.enter()) throw TooDeep(limit: this.depth.limit, at: this.pos)
    var total = try this.parseTerm()
    loop (this.peek() == '+') {
      this.pos += 1
      total += try this.parseTerm()
    }
    this.depth.leave()
    total
  }

  fun parseTerm(): i64 throws TooDeep {
    if (this.peek() == '(') {
      this.pos += 1
      val inner = try this.parseExpr()
      if (this.peek() == ')') this.pos += 1
      return inner
    }
    var n: i64 = 0
    loop (isDigit(this.peek())) {
      n = n * 10 + (this.peek() - '0').toI64()
      this.pos += 1
    }
    n
  }

  fun peek(): u8 = if (this.pos < this.src.len()) this.src.byteAt(this.pos) else 0

  /// How deep this parse actually went — the number a benchmark reports and
  /// the one that says whether a limit is anywhere near being reached.
  fun deepest(): i64 = this.depth.deepest()

  /// Whether the walk unwound cleanly: every `enter` had its `leave`.
  fun balanced(): bool = this.depth.depth() == 0
}

fun isDigit(b: u8): bool = b >= '0' && b <= '9'

fun run(text: string, shown: string) {
  var calc = Calc(src: text)
  when (calc.parseExpr()) {
    is Ok(value) => println("  ${shown.padEnd(18)} = ${"$value".padEnd(6)} deepest ${calc.deepest()}, balanced ${calc.balanced()}")
    is Err(e)    => println("  ${shown.padEnd(18)} ! ${e.message()}")
  }
}

fun main() {
  println("-- within the limit --")
  run("1+2+3", "1+2+3")
  run("(1+(2+3))", "(1+(2+3))")
  run("((((((1))))))", "((((((1))))))")

  println("-- past it --")
  val deep = "(".repeat(40) + "1" + ")".repeat(40)
  run(deep, "40 brackets")
  val justOver = "(".repeat(16) + "1" + ")".repeat(16)
  run(justOver, "16 brackets")
  val justUnder = "(".repeat(15) + "1" + ")".repeat(15)
  run(justUnder, "15 brackets")

  println("-- the counter on its own --")
  var d = recursion.Depth(limit: 3)
  println("  limit ${d.limit}, at ${d.depth()}")
  println("  three enters: ${d.enter()} ${d.enter()} ${d.enter()}, now at ${d.depth()}")
  println("  the fourth:   ${d.enter()}, still at ${d.depth()}")
  d.leave()
  println("  after a leave: at ${d.depth()}, and a fourth enter is ${d.enter()}")
  println("  deepest ${d.deepest()}")
  d.leave()
  d.leave()
  d.leave()
  d.leave()
  println("  an extra leave does not go negative: ${d.depth()}")
  println("  and does not let the next enter run past the limit: ${d.enter()} at ${d.depth()}")
  d.reset()
  println("  after reset: at ${d.depth()}, deepest ${d.deepest()}")
}
