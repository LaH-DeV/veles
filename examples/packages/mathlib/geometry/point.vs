use internal
pub struct Point {
  pub x: f64
  pub y: f64
}
pub fun norm(p: Point): f64 = internal.root(p.x * p.x + p.y * p.y)
