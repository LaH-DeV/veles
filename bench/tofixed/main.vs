use io { println }
use time

fun main() {
  val (check, elapsed) = time.measure(() => {
    var check: i64 = 0
    loop (i in 0..<200000) {
      val value = (i % 10000).toF64() / 4.0
      val digits = if (i % 2 == 0) 2 else 30
      check += value.toFixed(digits).len()
    }
    check
  })
  println("BENCH tofixed 200000 ${elapsed.toNanos()} $check")
}
