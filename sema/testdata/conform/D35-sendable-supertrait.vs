// D35 addendum (2026-10-04): a trait may require Sendable. Its objects are
// Sendable, so they cross task boundaries; every implementor must be
// Sendable itself, and an implementor that is not cannot be boxed.
use io

trait Sink : Sendable {
  fun put(n: i64)
}

struct Shared {
  items: MutableList<i64>

  implement Sink { // error: 'Shared' requires Sendable through 'Sink'
    fun put(n: i64) {
      this.items.push(n)
    }
  }
}

struct Plain {
  total: i64

  implement Sink {
    fun put(n: i64) {}
  }
}

struct Wrap<T> {
  value: T

  implement Sink {
    fun put(n: i64) {}
  }
}

fun forward(s: Sink) suspends {
  scope {
    val t = async s.put(1)
    await t
  }
}

fun main() {
  val s: Sink = Plain(total: 0)
  forward(s)
  io.println("ok")
}
