use io { println }
fun main() {
  var x: i32 = 2147483647
  val y = x - 1
  println("y $y")
  x += 1
  println("x $x")
}
