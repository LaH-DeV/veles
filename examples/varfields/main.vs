// Mutability lives on the field (D22): a bare field is set once by the
// constructor; a `var` field may be assigned — by a method through
// `self`, through any binding, through a pointer. `val` and `var` on a
// binding only say whether the name can be rebound.
use io

struct Counter {
  label: string   // never changes
  var n: i64 = 0  // the mutable part

  fun bump() {
    self.n += 1
  }
  /// A closure over `self` keeps pointing at the same counter, so the
  /// counter is moved to the heap for it (the receiver pass, D10).
  fun ticker(): fun(): i64 = () => {
    self.n += 1
    self.n
  }
  fun handle(): *Counter = &self
}

sealed trait Shape {
  fun grow(by: f64)
  fun area(): f64
}
struct Circle : Shape {
  var r: f64
  implement Shape {
    fun grow(by: f64) {
      self.r += by
    }
    fun area(): f64 = 3.0 * self.r * self.r
  }
}
struct Square : Shape {
  var side: f64
  implement Shape {
    fun grow(by: f64) {
      self.side += by
    }
    fun area(): f64 = self.side * self.side
  }
}

struct Cache {
  var last: Counter? = null
  fun remember(c: Counter) {
    self.last = c
  }
  /// `self.last` is narrowed after the test and stays narrowed across a
  /// call that cannot assign it; `remember` can, so the fact is dropped.
  fun peek(): string = if (self.last != null) "${self.last.label}=${self.last.n}" else "-"
}

fun make(): fun(): i64 {
  val c = Counter(label: "boxed")
  c.ticker()
}

fun main() {
  val c = Counter(label: "a")  // a val: the name is fixed, the var field is not
  c.bump()
  c.n += 10
  var d = c  // a copy: independent from here on
  d.bump()
  io.println("${c.label} ${c.n} ${d.n}")

  val tick = make()  // the Counter outlived make()
  io.println("${tick()} ${tick()} ${tick()}")

  val p = c.handle()  // a pointer into c's storage
  p.bump()
  io.println("${c.n} ${p.n}")

  var shape: Shape = Circle(r: 1.0)
  shape.grow(1.0)  // dispatched on the variant in place
  io.println("${shape.area()}")

  val cache = Cache()
  io.println(cache.peek())
  cache.remember(c)
  cache.last?.bump()  // through `?.` into the field itself
  io.println(cache.peek())

  val counters: MutableList<Counter> = [Counter(label: "x")]
  counters.refOrPanic(0).bump()
  loop (&k in counters) k.n *= 10
  io.println("${counters.map(k => k.n)}")
}
