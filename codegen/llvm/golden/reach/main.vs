// Only what the program can reach is emitted: `unused` and `Square.perimeter`
// have no definition in main.ll, while the methods reached only through a
// trait object's table, a function passed by name and a lambda all do.
use io

trait Shape {
  fun area(): f64
}

struct Circle {
  r: f64

  implement Shape {
    fun area(): f64 => 3.0 * this.r * this.r
  }
}

struct Square {
  side: f64

  fun perimeter(): f64 => 4.0 * this.side

  implement Shape {
    fun area(): f64 => this.side * this.side
  }
}

fun unused(x: i64): i64 => x * 3

fun twice(x: i64): i64 => x * 2

fun apply(f: fun(i64): i64, x: i64): i64 => f(x)

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  loop (s in shapes) {
    io.println("${s.area()}")
  }
  val k = 5
  io.println("${apply(twice, 10)} ${apply(x => x + k, 10)}")
}
