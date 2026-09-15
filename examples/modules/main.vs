use io.{ println }
use geometry.{ Point as P, distance, origin }

fun main() {
  val p = P(x: 3.0, y: 4.0)
  println("dist ${distance(p, origin())}")
  println("origin ${origin()}")
}
