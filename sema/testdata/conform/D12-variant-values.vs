// D12: a variant is a type of its own. Constructed where that type is
// expected, the value is the variant, its fields are read and assigned
// directly, and it becomes the sealed trait wherever the trait is expected.
// With no such expectation a construction is the trait, as it always was.
use io

sealed trait Shape
struct Circle : Shape {
  var radius: f64
}
struct Rect : Shape {
  w: f64
}

fun doubled(c: Circle): Circle {
  var bigger: Circle = c
  bigger.radius = bigger.radius * 2.0
  bigger
}

fun area(s: Shape): f64 => when (s) {
  is Circle => s.radius * s.radius * 3.0
  is Rect   => s.w * s.w
}

fun main() {
  var c: Circle = Circle(radius: 1.0)
  c.radius = 2.0
  val maybe: Circle? = Circle(radius: 4.0)
  val grown = doubled(Circle(radius: 3.0))
  var s = Circle(radius: 5.0) // the trait: another variant may follow
  s = Rect(w: 2.0)
  val wrong: Circle = Rect(w: 1.0) // error: type mismatch: expected 'Circle', found 'Shape'
  io.println("${area(c)} ${area(grown)} ${maybe?.radius ?: 0.0} ${area(s)} $wrong")
}
