// `extend` adds inherent methods to a type the package declares (D23);
// the prelude uses it to give string, List and Range most of their methods.
use io { println }

struct Vec2 {
  var x: f64
  var y: f64

  implement Show {
    fun show(): string = "(${this.x}, ${this.y})"
  }
}

struct Stack<T> {
  items: MutableList<T>
}

trait Show {
  fun show(): string
}

extend Vec2 {
  fun length(): f64 = (this.x * this.x + this.y * this.y).sqrt()
  fun scale(k: f64) {
    this.x *= k
    this.y *= k
  }
}

extend<T> Stack<T> {
  fun push(x: T) {
    this.items.push(x)
  }
  fun pop(): T? = this.items.pop()
  fun depth(): i64 = this.items.len()
}

// a bounded block: only stacks of showable things can render themselves
extend<T: Show> Stack<T> {
  fun render(): string = this.items.map(x => x.show()).join(" ")
}

fun main() {
  var v = Vec2(x: 3.0, y: 4.0)
  println("length ${v.length()}")
  v.scale(2.0)
  println("scaled ${v.show()} length ${v.length()}")

  val s = Stack(items: [Vec2(x: 1.0, y: 0.0)])
  s.push(Vec2(x: 0.0, y: 1.0))
  println("depth ${s.depth()}: ${s.render()}")
  println("popped ${s.pop()?.show()} depth ${s.depth()}")

  // the prelude's extend blocks
  val line = "  alpha, beta ,gamma  "
  val names = line.trim().split(",").map(n => n.trim())
  println("${names} ${names.join("|")} ${"ab".repeat(2).toUpper()}")
  println("${names.map(n => n.padEnd(6, ".")).join("")}")
  val xs = [4, 8, 15, 16, 23, 42]
  println("${xs.take(2)} ${xs.drop(4)} ${xs.sum()} ${xs.max()} ${xs.at(-1)} ${xs.chunked(4)}")
  println("${xs.count(x => x % 2 == 0)} ${xs.zip(names)} ${[3, 1, 3, 2, 1].distinct()}")
  var m = mut [3, 1, 2]
  m.insert(1, 9)
  m.sort()
  println("$m removed ${m.removeAt(0)} -> $m")
  println("${(1..10).step(3).toList()} ${(0..<5).reversed().toList()} ${(1..10).len()} ${(1..10).contains(11)}")
  println("${"3.5e2".toF64()} ${"x".toF64()} ${"héllo".bytes().len()} ${"héllo".bytes().decodeUtf8()}")
}
