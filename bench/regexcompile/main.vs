use io
use regex
use time

fun main() throws regex.RegexError {
  val patterns = ["[[:alpha:]]+", "[[:^digit:]]+", "(?i)[[:^lower:]]+"]
  val (sum, elapsed) = try time.measure(() => {
    var sum: i64 = 0
    loop (round in 0..<3000) {
      val pattern = patterns.at(round % patterns.len()) ?: panic("the index is in range")
      val compiled = try regex.Regex.compile(pattern)
      sum += compiled.source().len() + compiled.groupCount()
    }
    sum
  })
  io.println("BENCH regexcompile 3000 ${elapsed.toNanos()} $sum")
}
