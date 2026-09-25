// The GC under pressure: build and walk complete binary trees
// (the benchmarks-game shape), depth 4 to 16.
use io, time

struct Node {
  left:  (*Node)?
  right: (*Node)?
}

fun build(depth: i64): *Node {
  if (depth == 0) return &Node(left: null, right: null)
  val l = build(depth - 1)
  val r = build(depth - 1)
  &Node(left: l, right: r)
}

fun count(n: *Node): i64 {
  val l = n.left ?: return 1
  val r = n.right ?: return 1
  1 + count(l) + count(r)
}

fun main() {
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  var d = 4
  loop (d <= 16) {
    val trees = 1 << (16 - d + 4)
    loop (_ in 0..<trees) check += count(build(d))
    d += 2
  }
  io.println("BENCH trees $check ${sw.elapsed().toNanos()} $check")
}
