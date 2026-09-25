// A hash map: 200k inserts then 200k lookups, five rounds.
use io, time

fun main() {
  val sw = time.Stopwatch.start()
  var sum: i64 = 0
  loop (_ in 0..<5) {
    var m: MutableMap<i64, i64> = [:]
    loop (k in 0..<200000) m.set(k * 2654435761 % 1000000007, k)
    loop (k in 0..<200000) sum += m.get(k * 2654435761 % 1000000007) ?: 0
  }
  io.println("BENCH maps 2000000 ${sw.elapsed().toNanos()} $sum")
}
