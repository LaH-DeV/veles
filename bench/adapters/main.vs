// The eager list adapters over 200k pseudo-random integers (xorshift),
// twenty times: map, filter, fold, any, all, find. Measures what moving
// them from compiler lowering into the prelude costs (plan B8).
use io { println }
use time

fun main() {
  var x: u64 = 88172645463325252
  var xs: MutableList<i64> = []
  loop (_ in 0..<200000) {
    x = x ^ (x << 13)
    x = x ^ (x >> 7)
    x = x ^ (x << 17)
    xs.push((x % 1000000).wrapI64())
  }
  val input = xs.toList()
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  loop (round in 0..<20) {
    val scaled = input.map(n => n * 3 + round)
    val even = scaled.filter(n => n % 2 == 0)
    check += even.fold(0, (acc, n) => acc + n % 1000)
    if (even.any(n => n > 2999990)) check += 1
    if (scaled.all(n => n >= 0)) check += 2
    check += even.find(n => n % 7 == 3) ?: -1
  }
  println("BENCH adapters 24000000 ${sw.elapsed().toNanos()} $check")
}
