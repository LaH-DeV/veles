use io { println }
use greeter, mathlib, mathlib.geometry

fun main() {
  val p = geometry.Point(x: 3.0, y: 4.0)
  println("${mathlib.twice(21)} ${geometry.norm(p)} ${mathlib.sqrt(2.0)}")
  println(greeter.hello("Veles"))
}
