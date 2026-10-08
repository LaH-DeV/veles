// Tasks: 8 independent producer/consumer pairs, 200k values each over a
// channel of their own — whether channels that share nothing also share
// no lock (D66: each channel has its own).
use io { println }
use time

fun produce(ch: Channel<i64>) {
  loop (i in 0..<200000) ch.send(i)
  ch.close()
}

fun consume(ch: Channel<i64>): i64 {
  var sum: i64 = 0
  loop {
    sum += await ch.recv() ?: break
  }
  sum
}

fun main() {
  val sw = time.Stopwatch.start()
  var total: i64 = 0
  scope {
    val sums: MutableList<Task<i64>> = []
    sums.reserve(8)
    loop (_ in 0..<8) {
      val ch = Channel<i64>(capacity: 64)
      async produce(ch)
      sums.push(async consume(ch))
    }
    loop (t in sums) total += await t
  }
  println("BENCH pipes 1600000 ${sw.elapsed().toNanos()} $total")
}
