// Plan S8: a debug build describes the program to a debugger. Each
// function has a subprogram, each instruction the line and column of the
// statement it comes from (a block's last expression included), and each
// local and parameter a variable with its type: numbers and bools as base
// types, a string as its bytes and length, a struct with its fields.
use io

struct Point {
  x: i64
  y: i64
}

fun distance(a: Point, b: Point): i64 {
  val dx = a.x - b.x
  val dy = a.y - b.y
  dx * dx + dy * dy
}

fun main() {
  val far = Point(x: 3, y: 4)
  val label = "far"
  var total: i64 = 0
  loop (i in 0..<2) {
    total += distance(Point(x: 0, y: 0), far) + i
  }
  val ok = total > 0
  io.println("$label $total $ok")
}
