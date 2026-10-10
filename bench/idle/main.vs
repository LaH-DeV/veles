// Parked tasks: 100k tasks each waiting on one channel, then released —
// what a parked task costs in memory (concurrency review B4: at most 250
// bytes) and how long parking and waking them all takes. The memory is the
// collector's count of live bytes, after a collection, while every task is
// parked, less the count before they were started; Go's reference reports
// its heap and stacks the same way.
use io { println }
use time

extern "C" {
  fun veles_gc_collect()
  fun veles_gc_live_bytes(): i64
}

fun liveBytes(): i64 {
  // SAFETY: a collection, then the collector's count; neither keeps anything
  unsafe {
    veles_gc_collect()
    veles_gc_live_bytes()
  }
}

fun waiter(ch: Channel<i64>, parked: Atomic<i64>): i64 {
  val _ = parked.add(1)
  await ch.recv() ?: 0
}

fun main() {
  val n: i64 = 100000
  val ch = Channel<i64>()
  val parked = Atomic(value: 0)
  val before = liveBytes()
  val stopwatch = time.Stopwatch.start()
  var sum: i64 = 0
  var perTask: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    tasks.reserve(n)
    loop (_ in 0..<n) tasks.push(async waiter(ch, parked))
    loop (parked.load() < n) yieldNow()
    // the last ones counted themselves and are on their way to the channel
    loop (_ in 0..<100) yieldNow()
    perTask = (liveBytes() - before) / n
    loop (i in 0..<n) ch.send(i)
    loop (t in tasks) sum += await t
  }
  println("BENCH idle $n ${stopwatch.elapsed().toNanos()} $sum")
  println("MEMORY idle $perTask")
}
