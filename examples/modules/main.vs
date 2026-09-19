use io
use geometry as geo

fun main() {
  val p = geo.Point(x: 3.0, y: 4.0)
  io.println("dist ${geo.distance(p, geo.origin())}")
  io.println("origin ${geo.origin()}")
}
