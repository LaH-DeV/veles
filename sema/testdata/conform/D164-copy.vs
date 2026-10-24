// D164: `copy` is generated for every struct, error and sealed variant: the
// fields it changes are named, the rest keep the receiver's values, and
// `init` runs again. It is refused where a copy would duplicate what must
// be unique, and gives way to a hand-written `copy` or a field named so.
use io

struct Point { x: i64; y: i64 = 0 }

struct Rect {
  w: i64
  h: i64
  area: i64
  init {
    this.area = this.w * this.h
  }
}

struct Celsius {
  degrees: f64
  init(fahrenheit: f64) {
    this.degrees = (fahrenheit - 32.0) * 5.0 / 9.0
  }
}

struct Versioned {
  n: i64
  fun copy(): Versioned => Versioned(n: this.n + 1)
}

struct Named {
  copy: i64 // warning: a field named 'copy' takes the place of the generated 'copy' method, so 'Named' has none (D164)
}

struct Worker {
  job: Task<()>
}

struct Res {
  n: i64
  implement Closeable { fun close() { } }
}

extern union Bits { i: i64; f: f64 }

error Failed { code: i64; detail: string = "" }

fun idle() {
  await sleep(Duration.millis(1))
}

fun copies(w: Worker) {
  val p = Point(x: 1, y: 2)
  val q = p.copy(x: 5)
  val r = Rect(w: 2, h: 3).copy(h: 4)
  val e = Failed(code: 1).copy(detail: "again")
  val v = Versioned(n: 1).copy()          // the hand-written one
  io.println("${q.x} ${r.area} ${e.detail} ${v.n}")
  val a = p.copy(5) // error: 'copy' takes the fields it changes by name, e.g. 'p.copy(x: 5)' (D164)
  val b = p.copy(z: 1) // error: 'Point' has no field 'z' for 'copy' to change
  val c = p.copy(x: 1, x: 2) // error: 'copy' changes field 'x' twice
  val d = Rect(w: 1, h: 1).copy(area: 9) // error: field 'area' is assigned by the 'init' block of 'Rect', which a copy runs again; it cannot be named (D164)
  val f = Celsius(32.0).copy() // error: 'Celsius' cannot be copied: its 'init' takes parameters, and a copy has no value to pass them
  val g = p.copy<i64>(x: 1) // error: 'copy' takes no type arguments (D164)
  val h = w.copy() // error: 'Worker' holds a running task, which a copy would duplicate
  val i = Bits(i: 1).copy(i: 2) // error: an extern union has no 'copy'
  with res = Res(n: 1)
  val j = res.copy(n: 2) // error: 'res' is bound by 'with', which closes it when the block ends; a copy would be a second handle to the same resource (D100, D164)
  val k = Named(copy: 1).copy(copy: 2) // error: no method 'copy' on type 'Named'; 'copy' is a field, not a method
}

fun main() {
  with w = Worker(job: async idle())
  copies(w)
}
