// Trait objects (a vtable call), a generic function instantiated twice,
// default methods, and Display used by interpolation.
use io

trait Shape {
  fun area(): f64
  fun describe(): string => "area ${this.area()}"
}

struct Square {
  side: f64
  implement Shape { fun area(): f64 => this.side * this.side }
}

struct Circle {
  r: f64
  implement Shape {
    fun area(): f64 => 3.0 * this.r * this.r
    override fun describe(): string => "circle"
  }
}

struct Point {
  x: i64
  y: i64
  implement Display { fun toString(): string => "(${this.x}, ${this.y})" }
}

fun largest<T: Comparable>(xs: List<T>): T? => xs.max()

fun main() {
  val shapes: List<Shape> = [Square(side: 2.0), Circle(r: 1.0)]
  loop (s in shapes) io.println(s.describe())
  io.println("${largest([3, 9, 4]) ?: 0} ${largest(["b", "a"]) ?: ""} ${Point(x: 1, y: 2)}")
}
