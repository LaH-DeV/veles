use support
public struct Point {
  public x: f64
  public y: f64
}
public fun norm(p: Point): f64 => support.root(p.x * p.x + p.y * p.y)
