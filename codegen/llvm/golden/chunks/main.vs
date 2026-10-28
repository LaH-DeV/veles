// D218: which loops look for attention once per chunk. A counted loop
// whose body calls nothing is cut into chunks (chunk.inner, chunk.looked);
// one that calls a function, waits, cedes or holds a loop of its own looks
// on every round, as does a loop on a condition.
use io

fun sumSquares(n: i64): i64 {
  var total: i64 = 0
  loop (i in 0..<n) total +%= i *% i
  total
}

fun firstNegative(xs: List<i64>): i64 {
  loop (i in 0..<xs.len()) {
    if (xs.at(i) < 0) return i
  }
  -1
}

fun twice(x: i64): i64 => x * 2

fun withCall(n: i64): i64 {
  var total: i64 = 0
  loop (i in 0..<n) total += twice(i)
  total
}

fun ceding(n: i64): i64 {
  var total: i64 = 0
  loop (i in 0..<n) {
    total += i
    cede
  }
  total
}

fun halvings(n: i64): i64 {
  var steps: i64 = 0
  var v = n
  loop (v > 1) {
    v /= 2
    steps += 1
  }
  steps
}

fun main() {
  io.println("${sumSquares(100)} ${firstNegative([3, 1, -4, 1])} ${withCall(10)} ${ceding(10)} ${halvings(1024)}")
}
