pub fun distance(a: Point, b: Point): f64 {
  val dx = a.x - b.x
  val dy = a.y - b.y
  (dx * dx + dy * dy).sqrt()      // built in: a single instruction, no C, no unsafe
}
