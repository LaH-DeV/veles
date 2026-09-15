use io

sealed trait Shape
struct Circle : Shape { radius: f64 }
struct Rect   : Shape { w: f64, h: f64 }
struct Point  : Shape { }

const PI: f64 = 3.14159265358979

fun area(shape: Shape): f64 = when (shape) {
  is Shape.Circle(radius) if radius > 100.0 => 0.0
  is Shape.Circle(radius) => PI * radius * radius
  is Shape.Rect(w, h)     => w * h
  is Shape.Point          => 0.0
}

fun describe(shape: Shape): string = when (shape) {
  is Circle => "circle r=${shape.radius}"
  is Rect   => "rect ${shape.w}x${shape.h}"
  else      => "point"
}

fun main() {
  val shapes: List<Shape> = [Circle(radius: 1.0), Rect(w: 2.0, h: 3.0), Shape.Point()]
  loop (s in shapes) {
    io.println("${describe(s)} area=${area(s)}")
  }
  val c = Circle(radius: 2.0)
  if (c is Circle) {
    io.println("still a circle: $c")
  }
}
