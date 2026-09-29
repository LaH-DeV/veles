use io { println }
use mathlib, mathlib.geometry

fun main() {
  val p = geometry.Point(x: 3.0, y: 4.0)
  println("${mathlib.twice(21)} ${geometry.norm(p)}")
}
