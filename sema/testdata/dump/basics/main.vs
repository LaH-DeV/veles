use io

struct Point {
  x: i64
  y: i64

  fun sum(): i64 => this.x + this.y
}

fun add(a: i64, b: i64): i64 => a + b

fun main() {
  val p = Point(x: 1, y: 2)
  var total = add(p.x, p.y)
  loop (i in 0..<3) total += i
  io.println("${total} ${p.sum()}")
}
