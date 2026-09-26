use io

trait Shape {
  fun area(): f64
  fun name(): string
  fun describe(): string = "${this.name()} with area ${this.area()}"
}

struct Circle {
  r: f64

  implement Shape {
    fun area(): f64 = 3.0 * this.r * this.r
    fun name(): string = "circle"
  }
}
struct Square {
  side: f64

  implement Shape {
    fun area(): f64 = this.side * this.side
    fun name(): string = "square"
    override fun describe(): string = "a square of side ${this.side}"
  }
}

trait Counter {
  fun bump(): i64
}

struct Clicks {
  var n: i64 = 0

  implement Counter {
    fun bump(): i64 {
      this.n += 1
      this.n
    }
  }
}

fun total(shapes: List<Shape>): f64 {
  var sum = 0.0
  loop (s in shapes) {
    sum += s.area()
  }
  sum
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  loop (s in shapes) {
    io.println(s.describe())
  }
  io.println("total ${total(shapes)}")
  val one: Shape = Circle(r: 2.0)
  io.println("${one.name()} ${one.area()} $one")
  var c: Counter = Clicks()
  c.bump()
  io.println("bumped ${c.bump()}")
  val holder = (c, "pair")
  io.println("bumped again ${holder.0.bump()}")
}
