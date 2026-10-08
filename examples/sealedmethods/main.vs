use io { println }

sealed trait Shape {
  fun area(): f64
  fun describe(): string => "shape with area ${this.area()}"
}
struct Circle : Shape {
  r: f64

  implement Shape {
    fun area(): f64 => 3.0 * this.r * this.r
    override fun describe(): string => "circle r=${this.r}"
  }
}
struct Rect : Shape {
  w: f64
  h: f64

  implement Shape {
    fun area(): f64 => this.w * this.h
  }
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)]
  loop (s in shapes) {
    println("${s.describe()} -> ${s.area()}")
  }
  val total = shapes.fold(0.0, (acc, s) => acc + s.area())
  println("total $total")
}
