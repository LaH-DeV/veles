use io
fun main() {
  var x: i32 = 2147483647
  val y = x - 1
  io.println("y $y")
  x += 1
  io.println("x $x")
}
