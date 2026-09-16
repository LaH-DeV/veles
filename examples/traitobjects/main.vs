use io

trait Shape {
  fun area(): f64
  fun name(): string
  fun describe(): string = "${self.name()} with area ${self.area()}"
}

struct Circle { r: f64 }
struct Square { side: f64 }

impl Shape for Circle {
  fun area(): f64 = 3.0 * self.r * self.r
  fun name(): string = "circle"
}

impl Shape for Square {
  fun area(): f64 = self.side * self.side
  fun name(): string = "square"
  override fun describe(): string = "a square of side ${self.side}"
}

trait Counter {
  mut fun bump(): i64
}

struct Clicks { n: i64 = 0 }
impl Counter for Clicks {
  mut fun bump(): i64 { self.n += 1; self.n }
}

fun total(shapes: List<Shape>): f64 {
  var sum = 0.0
  loop (s in shapes) { sum += s.area() }
  sum
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  loop (s in shapes) { io.println(s.describe()) }
  io.println("total ${total(shapes)}")
  val one: Shape = Circle(r: 2.0)
  io.println("${one.name()} ${one.area()} $one")
  var c: Counter = Clicks()
  c.bump()
  io.println("bumped ${c.bump()}")
  val holder = (c, "pair")
  io.println("bumped again ${holder.0.bump()}")
}
