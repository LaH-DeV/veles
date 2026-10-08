// The GC under pressure: build and walk complete binary trees
// (the benchmarks-game shape), depth 4 to 16.
use io { println }
use time

struct Node {
  left:  (*Node)?
  right: (*Node)?
}

fun build(depth: i64): *Node {
  if (depth == 0) return &Node(left: null, right: null)
  val left = build(depth - 1)
  val right = build(depth - 1)
  &Node(left, right)
}

fun count(node: *Node): i64 {
  val left = node.left ?: return 1
  val right = node.right ?: return 1
  1 + count(left) + count(right)
}

fun main() {
  val stopwatch = time.Stopwatch.start()
  var check: i64 = 0
  var depth = 4
  loop (depth <= 16) {
    val trees = 1 << (16 - depth + 4)
    loop (_ in 0..<trees) check += count(build(depth))
    depth += 2
  }
  println("BENCH trees $check ${stopwatch.elapsed().toNanos()} $check")
}
