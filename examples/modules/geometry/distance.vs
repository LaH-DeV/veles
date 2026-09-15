extern "C" { fun sqrt(x: f64): f64 }

pub fun distance(a: Point, b: Point): f64 {
  val dx = a.x - b.x
  val dy = a.y - b.y
  unsafe { sqrt(dx * dx + dy * dy) }
}
