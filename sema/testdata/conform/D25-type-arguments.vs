// Built-in generic types take a fixed number of type arguments; a type
// parameter, a plain struct and a module take none.
use io

struct Plain {
  n: i64
}

trait Convert<T> {
  fun convert(): T
}

fun a(xs: List<i64, i64>) { } // error: List takes exactly one type argument
fun b(m: Map<string>) { } // error: Map takes two type arguments
fun c(s: Set<i64, i64>) { } // error: Set takes one type argument
fun d(r: Range<i64, i64>) { } // error: Range takes exactly one type argument
fun e(ch: Channel<i64, i64>) { } // error: Channel takes one type argument
fun f(t: Task<i64, i64>) { } // error: Task takes one type argument
fun g<T>(x: T<i64>) { } // error: type parameter 'T' cannot take type arguments
fun h(p: Plain<i64>) { } // error: 'Plain' is not generic: write it without '<...>'
fun q(x: main.X) { } // error: 'main' is not a module
fun k(x: io.Nothing) { } // error: module 'io' has no declaration 'Nothing'
fun m(x: Plain.Inner) { } // error: 'Plain' is not a module or sealed trait
fun n(x: Option<i64, i64>) { } // error: Option takes exactly one type argument
fun projected<I: Iterable>(x: I.Iter<i64>) { } // error: 'I.Iter' cannot take type arguments
fun genericTrait(x: Convert<i64>) { } // error: generic traits as types are not supported

// a literal argument takes the type the expected result binds (B11)
fun ident<T>(x: T): T = x
fun pairOf<T>(a: T, b: T): List<T> = [a, b]
fun fromExpected(): i64 {
  val small: i8 = ident(12)
  val bytes: List<u8> = pairOf(1, 255)
  val half: f32 = ident(0.5)
  small.toI64() + bytes.len()
}

fun main() {
  io.println("types ${fromExpected()}")
}
