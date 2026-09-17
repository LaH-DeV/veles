use io

sealed trait Shape {
  fun area(): f64
  fun describe(): string = "shape with area ${self.area()}"
}
struct Circle : Shape {
  r: f64

  impl Shape {
    fun area(): f64 = 3.0 * self.r * self.r
    override fun describe(): string = "circle r=${self.r}"
  }
}
struct Rect : Shape {
  w: f64
  h: f64

  impl Shape {
    fun area(): f64 = self.w * self.h
  }
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)]
  loop (s in shapes) {
    io.println("${s.describe()} -> ${s.area()}")
  }
  val total = shapes.fold(0.0, (acc, s) => acc + s.area())
  io.println("total $total")
}
