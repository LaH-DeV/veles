// Text: build a 200k-item string with a StringBuilder, then split it.
use io, time

fun main() {
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  loop (_ in 0..<5) {
    val sb = stringBuilder()
    loop (i in 0..<200000) sb.append("item-$i,")
    val parts = sb.toString().split(",")
    val part = parts.at(123456) ?: panic("strings: the text has 200000 parts")
    check += parts.len() + part.len()
  }
  io.println("BENCH strings 1000000 ${sw.elapsed().toNanos()} $check")
}
