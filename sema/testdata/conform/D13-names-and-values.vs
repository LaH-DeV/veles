// What a name may stand for as a value, and the variant constructors
// (D4, D5, D13, D23, D25, D40).
use io

sealed trait Shape

struct Circle : Shape {
  r: i64
}

struct Dot : Shape { }

sealed trait Maybe<T>

struct Nothing<T> : Maybe<T> { }

struct Holder {
  n: i64
  static val limit: i64 = 3
}

error Boom { }

fun thisOutside(): i64 => this.n // error: 'this' is only inside a method; a function outside a type takes the value as a parameter
fun moduleValue() {
  val _ = io // error: 'io' is a module, not a value
}
fun variantWithoutFields(): Shape => Shape.Circle // error: variant 'Shape.Circle' needs its fields: 'Shape.Circle(...)'
fun missingStatic(): i64 => Holder.nope // error: 'Holder' has no static 'nope'
fun brackets(n: i64): i64 => n[0] // error: '[...]' after a value of type 'i64' is not indexing
fun genericVariant() {
  val _ = Nothing // error: cannot infer the type arguments of 'Nothing' here
}
fun noneWithArgument(): i64? => None(1) // error: 'None' takes no arguments
fun someWithTwo(): i64? => Some(1, 2) // error: 'Some' takes exactly one argument
fun okWithTwo(): Result<i64, Boom> => Ok(1, 2) // error: 'Ok' takes exactly one argument
fun errWithoutThrows(): i64 {
  Err(Boom()) // error: 'Err(...)' here would fail the function, but it is not declared 'throws'
}
fun effects(f: fun(i64): i64 throws Boom): fun(i64): i64 => f // error: a function value does not take on 'suspends' or 'throws'

fun mayFail(): i64 throws Boom => if (Holder.limit > 5) throw Boom() else 1
val raised: i64 = try mayFail() // error: a global initializer cannot fail

fun main() {
  io.println("${Holder.limit} $raised")
}
