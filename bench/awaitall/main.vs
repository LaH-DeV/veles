// Two tasks at a time, joined by `awaitAll` (D156): 50 000 pairs of short
// computations started inline and awaited together — what one `awaitAll`
// of two launches costs. `awaitscope` is the same work in a `scope` with
// two handles, the form `awaitAll` replaces.
use io { println }
use time

fun work(i: i64): i64 =>
  (0..<50).iter().fold(0, (acc, k) => acc + (i + k) % 7)

fun pair(i: i64): i64 {
  val (a, b) = awaitAll(async work(i), async work(i + 1))
  a + b
}

fun main() {
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (i in 0..<50000) sum += pair(i)
    sum
  })
  println("BENCH awaitall 50000 ${elapsed.toNanos()} $sum")
}
