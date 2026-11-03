// Sorting 300k pseudo-random integers (xorshift), three times.
use io { println }
use time

fun main() {
  var x: u64 = 88172645463325252
  var xs: List<i64> = []
  loop (_ in 0..<300000) {
    x ^= x << 13
    x ^= x >> 7
    x ^= x << 17
    xs.push((x % 1000000).wrapI64())
  }
  val input = xs
  val (check, elapsed) = time.measure(() => {
    var check: i64 = 0
    loop (_ in 0..<3) {
      val s = input.sorted()
      if (s.len() != 300000) panic("sort: the input has 300000 numbers")
      check += s.get(0) + s.get(150000) + s.get(299999)
    }
    check
  })
  println("BENCH sort 900000 ${elapsed.toNanos()} $check")
}
