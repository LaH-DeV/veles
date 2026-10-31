use io
use os
use regex
use time

fun main() throws regex.RegexError {
  val operations = os.args().at(0)?.toInt() ?: 50000
  if (operations < 1) panic("regex benchmark: operations must be positive")
  val compiled = try regex.Regex.compile("(a*)*b")
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (index in 0..<operations) {
      val text = if (index % 2 == 0) "aaaab" else "b"
      val found = compiled.find(text) ?: panic("the trailing b must match")
      sum += found.end() - found.start()
    }
    sum
  })
  io.println("BENCH regex $operations ${elapsed.toNanos()} $sum")
}
