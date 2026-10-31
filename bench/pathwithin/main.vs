use io
use os
use path
use time

fun main() {
  val operations = os.args().at(0)?.toInt() ?: 200000
  if (operations < 1) panic("pathwithin benchmark: operations must be positive")
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (round in 0..<operations) {
      val candidate = if (round % 2 == 0) "/srv/www/assets/site.css" else "/srv/www2/private"
      if (path.within("/srv/www", candidate)) sum += 1
    }
    sum
  })
  io.println("BENCH pathwithin $operations ${elapsed.toNanos()} $sum")
}
