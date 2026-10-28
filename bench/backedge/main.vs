// The back edge of every loop (D66, D145): one load of the attention line
// and a branch that is never taken unless a collection, a cancellation or
// a request to yield wants the loop. What it costs in tight loops (plan
// F21), in a plain function and in a suspending one: a chain where each
// round needs the last (a multiply-add, the poll can hide beside it) and a
// sum whose rounds are independent (what a vectorizer would take).
//
// Go's loops have no poll (its preemption is a signal) and Go does not
// vectorize, so the ratio is not the poll's price. That price, against the
// same loops in C through the same clang, was the whole gap (bench/results.md,
// 2026-10-27); since D218 these loops poll once per chunk, at C's speed.
use io { println }
use time

fun chain(n: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<n) x = x *% 6364136223846793005 +% i
  x
}

fun sum(n: i64): i64 {
  var total: i64 = 0
  loop (i in 0..<n) total +%= i *% i
  total
}

fun chainSuspending(n: i64): i64 {
  if (n < 0) await sleep(Duration.millis(1))  // never: it only may suspend
  var x: i64 = 1
  loop (i in 0..<n) x = x *% 6364136223846793005 +% i
  x
}

fun sumSuspending(n: i64): i64 {
  if (n < 0) await sleep(Duration.millis(1))
  var total: i64 = 0
  loop (i in 0..<n) total +%= i *% i
  total
}

fun main() {
  val n: i64 = 500000000
  val (result, elapsed) = time.measure(() => chain(n) +% sum(n) +% chainSuspending(n) +% sumSuspending(n))
  println("BENCH backedge ${4 * n} ${elapsed.toNanos()} $result")
}
