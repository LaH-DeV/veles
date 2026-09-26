// Sorting 300k pseudo-random integers (xorshift), three times.
use io, time

fun main() {
  var x: u64 = 88172645463325252
  var xs: MutableList<i64> = []
  loop (_ in 0..<300000) {
    x = x ^ (x << 13)
    x = x ^ (x >> 7)
    x = x ^ (x << 17)
    xs.push((x % 1000000) as i64)
  }
  val input = xs.toList()
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  loop (_ in 0..<3) {
    val s = input.sorted()
    if (s.len() != 300000) panic("sort: the input has 300000 numbers")
    check += s.at(0) + s.at(150000) + s.at(299999)
  }
  io.println("BENCH sort 900000 ${sw.elapsed().toNanos()} $check")
}
