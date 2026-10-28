// `cede` (D197): a suspension point of its own, lowered to the runtime's
// sleep of zero — one trip through the run queue, then the cancellation
// and fail-fast checks every resume makes. A function that cedes suspends,
// and so does its caller.
use io

fun checksum(data: List<i64>, every: i64): i64 {
  var sum = 0
  loop (i in 0..<data.len()) {
    sum = (sum * 31 + data.at(i)) % 1000003
    if (i % every == every - 1) cede
  }
  sum
}

fun main() {
  val data = MutableList<i64>.make(100, i => i * i)
  io.println("${checksum(data, 16)}")
  cede
  io.println("resumed")
}
