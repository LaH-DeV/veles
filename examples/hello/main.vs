use io

fun main() {
  val sum = add(5, 7)
  printSum(sum)
  var i = 0
  loop (i < 3) {
    io.println("i = $i")
    i += 1
  }
  loop (k in 1..3) {
    io.println("k = $k")
  }
  val big = if (sum > 10) "big" else "small"
  io.println("sum is $big, double is ${sum * 2}, float ${2.5 * 2.0}, bool ${sum == 12}")
}

fun add(a: i32, b: i32) = a + b

fun printSum(sum: i32) {
  io.println("The sum is $sum")
}
