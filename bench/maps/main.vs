// A hash map: 200k inserts then 200k lookups, five rounds.
use io { println }
use time

fun main() {
  val (sum, elapsed) = time.Stopwatch.measure(() => {
    var sum: i64 = 0
    loop (_ in 0..<5) {
      var m: MutableMap<i64, i64> = [:]
      loop (k in 0..<200000) m.set(k * 2654435761 % 1000000007, k)
      loop (k in 0..<200000) sum += m.get(k * 2654435761 % 1000000007) ?: 0
    }
    sum
  })
  println("BENCH maps 2000000 ${elapsed.toNanos()} $sum")
}
