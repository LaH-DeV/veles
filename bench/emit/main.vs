// Compiler-shaped: emit text the way a code generator does — many short
// lines of textual IR appended to one StringBuilder, with interpolated
// numbers and names — then read the result back once.
use io { println }, time

fun emitFunction(sb: StringBuilder, index: i64) {
  sb.appendLine("define i64 @f$index(i64 %a, i64 %b) {")
  sb.appendLine("entry:")
  loop (t in 0..<40) {
    val prev = if (t == 0) "%a" else "%t${t - 1}"
    sb.appendLine("  %t$t = add i64 $prev, ${t * 3 + index % 11}")
  }
  sb.appendLine("  %cmp = icmp slt i64 %t39, %b")
  sb.appendLine("  br i1 %cmp, label %yes, label %no")
  sb.appendLine("yes:")
  sb.appendLine("  ret i64 %t39")
  sb.appendLine("no:")
  sb.appendLine("  ret i64 %b")
  sb.appendLine("}")
}

fun main() {
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  var lines: i64 = 0
  loop (_ in 0..<4) {
    val sb = StringBuilder()
    loop (f in 0..<5000) emitFunction(sb, f)
    val text = sb.toString()
    check += text.len()
    lines += 50 * 5000
  }
  println("BENCH emit $lines ${sw.elapsed().toNanos()} $check")
}
