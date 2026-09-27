// Tasks: 100k short tasks, started and joined 1000 at a time — what the
// scheduler costs per task, spread over the worker threads.
use io, time

fun square(i: i64): i64 {
  var acc: i64 = 0
  loop (k in 0..<50) acc += (i + k) % 7
  acc
}

fun batch(from: i64): i64 {
  val tasks: MutableList<Task<i64>> = []
  scope {
    loop (i in from..<from + 1000) {
      tasks.push(async square(i))
    }
  }
  var sum: i64 = 0
  loop (t in tasks) sum += await t
  sum
}

fun main() {
  val sw = time.Stopwatch.start()
  var sum: i64 = 0
  loop (b in 0..<100) sum += batch(b * 1000)
  io.println("BENCH spawn 100000 ${sw.elapsed().toNanos()} $sum")
}
