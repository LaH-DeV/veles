// D135 and D117 (run time): `is` on a trait object tests its concrete type,
// or whether that type implements another trait; on a concrete value the
// answer is known, and a type parameter is asked at compile time.
use io

trait Shape {
  fun area(): f64
}

trait Named {
  fun name(): string
}

trait Sized : Shape {
  fun size(): i64
}

trait Maker {
  static fun make(): Self
}

struct Circle {
  radius: f64
  implement Shape {
    fun area(): f64 = 3.0 * this.radius * this.radius
  }
  implement Named {
    fun name(): string = "circle"
  }
  implement Sized {
    fun size(): i64 = 1
  }
}

struct Plain { }

fun describe(s: Shape, z: Sized): string {
  if (s is Plain) return "plain" // error: subject has type 'Shape', which can never be 'Plain', so this never matches: 'Plain' does not implement 'Shape'
  if (s is Maker) return "maker" // error: 'Maker' cannot be a trait object
  if (z is Shape) return "sized" // warning: every 'Sized' implements 'Shape', so this test is always true
  if (s is Shape) return "shape" // warning: every 'Shape' implements 'Shape', so this test is always true
  when (s) {
    is Circle(radius) => "circle $radius"
    is Named() => "named" // error: 'Named' is a trait: there are no fields to destructure
    else => "other"
  }
}

fun generic<T>(x: T, s: Shape): string {
  if (x is Named) return "named" // error: 'T' is a type parameter, so whether it implements 'Named' is known at compile time: write 'if (T implements Named)'
  if (s is T) return "a T" // error: 'T' is a type parameter; a trait object is tested against a concrete type or a trait
  val n: i64? = 1
  if (n is T) return "an i64" // error: 'T' is a type parameter; whether a value is of type 'T' is known at compile time
  "neither"
}

fun main() {
  val c = Circle(radius: 1.0)
  val p = Plain()
  val maybe: Circle? = null
  io.println("${c is Named}") // warning: 'Circle' implements 'Named', so this test is always true; remove it
  io.println("${p is Named}") // warning: 'Plain' does not implement 'Named', so this test is always false; remove it
  io.println("${maybe is Named}") // warning: 'Circle' implements 'Named', so this test is the same as '!= null'; write that
  when (p) {
    is Named => io.println("never") // warning: 'Plain' does not implement 'Named', so this test is always false
    else => io.println("plain")
  }
  io.println(describe(c, c))
  io.println(generic(1, c))
}
