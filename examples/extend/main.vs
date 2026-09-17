// `extend` adds inherent methods to a type the package declares (D23);
// the prelude uses it to give string, List and Range most of their methods.
use io

struct Vec2 {
  x: f64
  y: f64

  impl Show {
    fun show(): string = "(${self.x}, ${self.y})"
  }
}

struct Stack<T> {
  items: MutableList<T>
}

trait Show {
  fun show(): string
}

extend Vec2 {
  fun length(): f64 = (self.x * self.x + self.y * self.y).sqrt()
  mut fun scale(k: f64) {
    self.x *= k
    self.y *= k
  }
}

extend<T> Stack<T> {
  fun push(x: T) {
    self.items.push(x)
  }
  fun pop(): T? = self.items.pop()
  fun depth(): i64 = self.items.len()
}

// a bounded block: only stacks of showable things can render themselves
extend<T: Show> Stack<T> {
  fun render(): string = self.items.map(x => x.show()).join(" ")
}

fun main() {
  var v = Vec2(x: 3.0, y: 4.0)
  io.println("length ${v.length()}")
  v.scale(2.0)
  io.println("scaled ${v.show()} length ${v.length()}")

  val s = Stack(items: [Vec2(x: 1.0, y: 0.0)])
  s.push(Vec2(x: 0.0, y: 1.0))
  io.println("depth ${s.depth()}: ${s.render()}")
  io.println("popped ${s.pop()?.show()} depth ${s.depth()}")

  // the prelude's extend blocks
  val line = "  alpha, beta ,gamma  "
  val names = line.trim().split(",").map(n => n.trim())
  io.println("${names} ${names.join("|")} ${"ab".repeat(2).toUpper()}")
  io.println("${names.map(n => n.padEnd(6, ".")).join("")}")
  val xs = [4, 8, 15, 16, 23, 42]
  io.println("${xs.take(2)} ${xs.drop(4)} ${xs.sum()} ${xs.max()} ${xs.at(-1)} ${xs.chunked(4)}")
  io.println("${xs.count(x => x % 2 == 0)} ${xs.zip(names)} ${[3, 1, 3, 2, 1].distinct()}")
  var m = mut [3, 1, 2]
  m.insert(1, 9)
  m.sort()
  io.println("$m removed ${m.removeAt(0)} -> $m")
  io.println("${(1..10).step(3).toList()} ${(0..<5).reversed().toList()} ${(1..10).len()} ${(1..10).contains(11)}")
  io.println("${"3.5e2".toF64()} ${"x".toF64()} ${"héllo".bytes().len()} ${"héllo".bytes().decodeUtf8()}")
}
