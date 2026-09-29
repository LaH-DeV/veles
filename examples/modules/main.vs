use io { println }
use geometry as geo

fun main() {
  val p = geo.Point(x: 3.0, y: 4.0)
  println("dist ${geo.distance(p, geo.origin())}")
  println("origin ${geo.origin()}")
}
