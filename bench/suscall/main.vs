// Suspending calls that do not wait: 10M calls, three suspending functions
// deep, where the bottom one could sleep but never does — what calling a
// function that may suspend costs (review F3). Go's reference is the same
// three ordinary functions.
use io { println }
use time

fun leaf(i: i64): i64 {
  if (i < 0) await sleep(Duration.millis(1))
  i % 7
}

fun middle(i: i64): i64 => leaf(i) + 1

fun top(i: i64): i64 => middle(i) + 1

fun main() {
  val n: i64 = 10000000
  val stopwatch = time.Stopwatch.start()
  var sum: i64 = 0
  loop (i in 0..<n) sum += top(i)
  println("BENCH suscall $n ${stopwatch.elapsed().toNanos()} $sum")
}
