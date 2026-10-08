// Tasks: 100k short tasks, started and joined 1000 at a time — what the
// scheduler costs per task, spread over the worker threads.
use io { println }
use time

fun square(i: i64): i64 =>
  (0..<50).iter().fold(0, (acc, k) => acc + (i + k) % 7)

fun batch(from: i64, to: i64): i64 {
  var sum: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    tasks.reserve(to - from)
    loop (i in from..<to) {
      tasks.push(async square(i))
    }
    sum = tasks.fold(0, (acc, task) => acc + await task)
  }
  sum
}

fun main() {
  val stopwatch = time.Stopwatch.start()
  var sum: i64 = 0
  loop (b in 0..<100) sum += batch(b * 1000, (b + 1) * 1000)
  println("BENCH spawn 100000 ${stopwatch.elapsed().toNanos()} $sum")
}
