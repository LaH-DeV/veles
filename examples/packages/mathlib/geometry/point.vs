use internal
public struct Point {
  public x: f64
  public y: f64
}
public fun norm(p: Point): f64 = internal.root(p.x * p.x + p.y * p.y)
