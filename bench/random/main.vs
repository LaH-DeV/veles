use io
use random
use time

fun main() {
  var rng = random.Rng.seeded(7)
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (index in 0..<5000000) {
      sum += rng.range(-1000000, 1000000 + index % 31)
    }
    sum
  })
  io.println("BENCH random 5000000 ${elapsed.toNanos()} $sum")
}
