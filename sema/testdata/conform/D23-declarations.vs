// D23 and the declaration rules: one name per member, typed parameters,
// generic names that do not shadow, `Self` only where it means something.
use io

struct Point {
  x: i64
  x: i64 // error: duplicate field 'x'
  static val origin: i64 = 0
  static val origin: i64 = 1 // error: duplicate static 'origin' on 'Point'

  fun norm(): i64 => this.x
  fun norm(): i64 => 0 // error: duplicate method 'norm' on 'Point'
}

struct Box<T> {
  item: T

  fun map<T>(f: fun(T): T): Box<T> => Box(item: f(this.item)) // error: type parameter 'T' shadows an outer type parameter
}

fun twice(a: i64, a: i64): i64 => a // error: duplicate parameter 'a'

fun selfless(): Self => 0 // error: 'Self' is only meaningful inside a struct, trait or impl

fun main() {
  io.println("${twice(1, 2)} ${Box(item: 1).item} ${Point.origin}")
}
