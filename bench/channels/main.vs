// Tasks: 200k values through a channel between two tasks.
use io { println }
use time

fun produce(ch: Channel<i64>) {
  loop (i in 0..<200000) ch.send(i)
  ch.close()
}

fun main() {
  val ch = Channel<i64>(capacity: 64)
  val (sum, elapsed) = time.Stopwatch.measure(() => {
    var sum: i64 = 0
    scope {
      async produce(ch)
      loop (val v = await ch.recv()) {
        sum += v
      }
    }
    sum
  })
  println("BENCH channels 200000 ${elapsed.toNanos()} $sum")
}
