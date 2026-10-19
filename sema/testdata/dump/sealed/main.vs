use io

sealed trait Shape

struct Circle : Shape {
  r: f64
}

struct Rect : Shape {
  w: f64
  h: f64
}

fun area(s: Shape): f64 => when (s) {
  is Circle => 3.0 * s.r * s.r
  is Rect => s.w * s.h
}

fun first(xs: List<string>): string? => xs.at(0)

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)]
  loop (s in shapes) io.println("${area(s)}")
  io.println(first(["a", "b"]) ?: "none")
}
