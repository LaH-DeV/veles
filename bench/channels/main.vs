// Tasks: 200k values through a channel between two tasks.
use io { println }, time

fun produce(ch: Channel<i64>) {
  loop (i in 0..<200000) ch.send(i)
  ch.close()
}

fun main() {
  val ch = Channel<i64>(capacity: 64)
  val sw = time.Stopwatch.start()
  var sum: i64 = 0
  scope {
    async produce(ch)
    loop {
      val v = await ch.recv() ?: break
      sum += v
    }
  }
  println("BENCH channels 200000 ${sw.elapsed().toNanos()} $sum")
}
