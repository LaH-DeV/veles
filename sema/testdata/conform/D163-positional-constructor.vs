// D163: a constructor that takes exactly one argument — one field the call
// may supply, or an `init` with one parameter — takes it positionally. Two
// parameters, a field with a default included, keep construction by name.
use io

error UsageError { message: string }

struct Meters { value: i64 }

struct Labelled {
  value: i64
  label: string = ""
}

struct Celsius {
  degrees: f64
  init(fahrenheit: f64) {
    this.degrees = (fahrenheit - 32.0) * 5.0 / 9.0
  }
}

struct Boxed<T> { item: T }

sealed trait Expr
struct Num : Expr { n: i64 }
struct Neg : Expr { inner: *Expr }

extern struct Fd { fd: i32 }

fun check(arg: string): i64 throws UsageError {
  if (arg.isEmpty()) throw UsageError("no argument")
  arg.len()
}

fun main() {
  val m = Meters(3)
  val named = Meters(value: 3)
  val value = 4
  val punned = Meters(value)
  val c = Celsius(212.0)
  val b = Boxed("text")                   // Boxed<string>, inferred from the argument
  val counter = Mutex(0)                  // Mutex's init has one parameter
  val e: Expr = Num(1)
  val _ = Neg(&e)
  val l = Labelled(1) // error: construct 'Labelled' by field name: 'value: 1'
  val two = Meters(1, 2) // error: construct 'Meters' by field name
  val fd = Fd(3) // error: construct 'Fd' by field name: 'fd: 3'
  io.println("${m.value} ${named.value} ${punned.value} ${c.degrees} ${b.item} ${counter.get()}")
  io.println("${check("x") ?? 0}")
}
