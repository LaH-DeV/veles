// Branches, loops with labels, ranges, and `when` over values.
use io

fun classify(n: i64): string => when {
  n < 0 => "negative"
  n == 0 => "zero"
  n < 10 => "small"
  else => "large"
}

fun digits(n: i64): string => when (n) {
  1 => "one"
  2, 3 => "few"
  else => "many"
}

fun main() {
  var total = 0
  loop :outer (i in 1..5) {
    loop (j in 0..<i) {
      if (j == 3) continue outer
      if (i * j > 8) break outer
      total += i * j
    }
  }
  var n = 10
  loop (n > 0) n -= 3
  var steps = 0
  loop {
    steps += 1
    if (steps == 4) break
  }
  loop (k in (0..10).step(5)) io.println("step $k")
  io.println("$total $n $steps ${classify(-2)} ${classify(0)} ${classify(4)} ${digits(3)} ${digits(9)}")
}
