// Prelude — what the test vocabulary (D78) is built on. Private: test code
// reaches it through the words the compiler knows, never by name.

/// Calls `body`. `expectPanics(body)` launches this in a task of its own
/// inside a `gather`, where a panic is a value (D52) instead of the end of
/// the test.
fun runTestBody<T>(body: sendable fun(): T): T => body()

/// What a failed `expect(a == b)`, `assert(a == b, why)` or
/// `expectEqual(expected, actual)` adds under the two values' lines when they
/// are long enough that a reader would have to hunt for the difference: for
/// text of several lines, the lines that differ with two of context on each
/// side, `-` for `leftName`'s and `+` for `rightName`'s; for one long line, a
/// caret under the first character that differs, the values starting at
/// `column`. `""` when the two lines above say it all. The compiler calls it
/// with two texts, or two values as printed; `quoted` says the lines above
/// show them in quotes. It runs in the compiler too (D208).
const fun testDiff(left: string, right: string, quoted: bool, column: i64, leftName: string, rightName: string): string {
  if (left == right) return ""
  if (left.contains("\n") || right.contains("\n")) return lineDiff(left.split("\n"), right.split("\n"), leftName, rightName)
  if (left.len() < 24 && right.len() < 24) return ""
  var at = 0
  loop (at < left.len() && at < right.len() && left.byteAt(at) == right.byteAt(at)) at += 1
  // the caret's column counts characters, not bytes: a byte that continues
  // a UTF-8 sequence takes no column of its own
  var offset = if (quoted) 1 else 0
  loop (i in 0..<at) {
    if ((right.byteAt(i) & 0xC0) != 0x80) offset += 1
  }
  // under the line above, where its value starts
  "\n" + " ".repeat(column + offset) + "^ differs from here"
}

/// A value a failed `expect` shows, with every line after its first moved
/// right to `column`, where the value starts: `left:  "a` then `       b"`.
const fun testIndent(text: string, column: i64): string {
  if (!text.contains("\n")) return text
  text.replace("\n", "\n" + " ".repeat(column))
}

/// The line diff of testDiff: `-` for a line only on the left, `+` for one
/// only on the right, from the longest common run of lines; long texts are
/// compared from their first difference to their last.
const fun lineDiff(left: List<string>, right: List<string>, leftName: string, rightName: string): string {
  // the equal lines at both ends need no table
  var head = 0
  loop (head < left.len() && head < right.len() && left.at(head) == right.at(head)) head += 1
  var tail = 0
  loop (tail < left.len() - head && tail < right.len() - head && left.at(left.len() - 1 - tail) == right.at(right.len() - 1 - tail)) tail += 1
  val rows = left.len() - head - tail
  val cols = right.len() - head - tail
  val out = MutableList<string>()
  out.push("\n      diff (- ${leftName}, + ${rightName}):")
  val context = 2
  if (head > context) out.push("\n          ...")
  loop (i in (if (head > context) head - context else 0)..<head) out.push("\n          " + (left.at(i) ?: ""))
  if (rows * cols > 40000) {
    // too long to align: everything between the equal ends, as removed and added
    loop (i in head..<head + rows) out.push("\n        - " + (left.at(i) ?: ""))
    loop (j in head..<head + cols) out.push("\n        + " + (right.at(j) ?: ""))
  } else {
    // common[i * (cols + 1) + j]: the longest common run of left[head + i..]
    // and right[head + j..]
    val width = cols + 1
    val common = MutableList<i64>()
    loop (_ in 0..<(rows + 1) * width) common.push(0)
    loop (i in (0..<rows).reversed()) {
      loop (j in (0..<cols).reversed()) {
        val here = if (left.at(head + i) == right.at(head + j)) {
          (common.at((i + 1) * width + j + 1) ?: 0) + 1
        } else {
          val down = common.at((i + 1) * width + j) ?: 0
          val across = common.at(i * width + j + 1) ?: 0
          if (down > across) down else across
        }
        common.set(i * width + j, here)
      }
    }
    var i = 0
    var j = 0
    loop (i < rows || j < cols) {
      when {
        i < rows && j < cols && left.at(head + i) == right.at(head + j) => {
          out.push("\n          " + (left.at(head + i) ?: ""))
          i += 1
          j += 1
        }
        i < rows && (j == cols || (common.at((i + 1) * width + j) ?: 0) >= (common.at(i * width + j + 1) ?: 0)) => {
          out.push("\n        - " + (left.at(head + i) ?: ""))
          i += 1
        }
        else => {
          out.push("\n        + " + (right.at(head + j) ?: ""))
          j += 1
        }
      }
    }
  }
  val end = left.len() - tail
  loop (i in end..<(if (tail > context) end + context else left.len())) out.push("\n          " + (left.at(i) ?: ""))
  if (tail > context) out.push("\n          ...")
  out.join("")
}

/// Skips a test, or every test of a suite, without running it (D178). The
/// reason is shown in the report: `test x ... skipped: no ::1 here`.
///
/// ```veles
/// @Skip("waiting on the new parser")
/// test "parses dates" { ... }
/// ```
///
/// For a condition that is known when compiling, `@SkipIf`; for one that is
/// known only when the test runs, `skip(why)` and `skipIf(cond, why)` in the
/// test's body.
public annotation Skip(why: string) on test, suite

/// Skips a test, or every test of a suite, when the condition holds. The
/// condition is a constant (D113): a literal or a `val` of one works, an
/// environment variable does not (use `skipIf(...)` in the body); `build.target`
/// joins them with D160.
///
/// ```veles
/// @SkipIf(isCi, "needs a display")
/// test "ipv6 loopback" { ... }
/// ```
public annotation SkipIf(cond: bool, why: string) on test, suite
