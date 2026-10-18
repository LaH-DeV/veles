// The collector's pauses (plan F18): a live heap of 2^22 pointer-linked
// nodes (134 MB) the collector must trace at every collection, while the
// program makes and drops small lists as fast as it can. Timed: the
// churn; reported besides (PAUSE, ns): the longest the churn's loop went
// without a step — looked at every 4096 steps, so a pause, a wait for the
// collector or the OS taking the thread all count, as on the Go side —
// from after the tree is built and collected: the steady state a server
// is in.
use io { println }
use time

extern "C" {
  fun veles_gc_collect()
}

struct Node {
  left:  (*Node)?
  right: (*Node)?
  value: i64
}

fun build(depth: i64, value: i64): *Node {
  if (depth == 0) return &Node(left: null, right: null, value)
  &Node(left: build(depth - 1, value * 2), right: build(depth - 1, value * 2 + 1), value)
}

fun total(node: *Node): i64 {
  var sum = node.value
  if (val l = node.left) sum += total(l)
  if (val r = node.right) sum += total(r)
  sum
}

fun main() {
  val live = build(21, 1)
  // SAFETY: a full collection: none the build began is left running
  unsafe {
    veles_gc_collect()
  }
  val n: i64 = 20000000
  val stopwatch = time.Stopwatch.start()
  var churn: i64 = 0
  var last: i64 = 0
  var longest: i64 = 0
  loop (i in 0..<n) {
    if (i % 4096 == 0) {
      val now = stopwatch.elapsed().toNanos()
      longest = longest.max(now - last)
      last = now
    }
    val xs: MutableList<i64> = [i, i + 1, i + 2]
    churn += xs.len()
  }
  val ns = stopwatch.elapsed().toNanos()
  println("BENCH gcpause $n $ns ${total(live) + churn}")
  println("PAUSE gcpause $longest")
}
