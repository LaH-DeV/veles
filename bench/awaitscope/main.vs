// The twin of `awaitall`: the same 50 000 pairs, each in a `scope` with
// two handles awaited one after the other — the form D156's `awaitAll`
// replaces, kept to show the two cost the same.
use io { println }
use time

fun work(i: i64): i64 =>
  (0..<50).iter().fold(0, (acc, k) => acc + (i + k) % 7)

fun pair(i: i64): i64 => scope {
  val a = async work(i)
  val b = async work(i + 1)
  await a + await b
}

fun main() {
  val (sum, elapsed) = time.Stopwatch.measure(() => {
    var sum: i64 = 0
    loop (i in 0..<50000) sum += pair(i)
    sum
  })
  println("BENCH awaitscope 50000 ${elapsed.toNanos()} $sum")
}
