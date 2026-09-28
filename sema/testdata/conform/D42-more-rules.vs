// More rules the checker enforces: statics and variants by name (D12/D23),
// generic aliases (D55), loops over ranges and lists (D42), lambdas that
// cannot be typed (D40), removed index forms (D25), list adapters (D46),
// `race` (D34), let-else destructuring (D37/D61), trait values (D9), and
// what a global initializer may not do.
use io

sealed trait Shape

struct Circle : Shape {
  r: i64
}

struct Temp {
  c: i64
}

struct Box<T> {
  item: T
}

type Wrap<T> = Box<T>

trait Show {
  fun show(): string
}

val returned: i64 = if (Temp(c: 1).c > 0) 1 else return // error: 'return' outside of a function
val raced = race { // error: 'race' cannot appear in a global initializer
  sleep(Duration.millis(1)) => 1
}

fun noStatic(): i64 = Temp.nope // error: 'Temp' has no static 'nope'
fun noVariant(): Shape = Shape.Square(1) // error: 'Shape' has no variant 'Square'
fun wrapped() {
  val _ = Wrap(item: 1) // error: 'Wrap' is generic; write the type arguments, e.g. 'Wrap<T>(...)'
}
fun rangeFloat(r: Range<f64>) {
  loop (x in r) { // error: cannot iterate a range of 'f64'
    io.println("$x")
  }
}
fun refPairs(ps: MutableList<(i64, i64)>) {
  loop ((&a, b) in ps) { // error: '&' binds a list element as a whole: 'loop (&x in xs)'
    io.println("$a $b")
  }
}
fun untypedLambda() {
  val f = x => x // error: cannot infer the type of lambda parameter 'x' // warning: 'f' is never used
}
fun untypedTuple() {
  val f = ((a, b)) => a // error: cannot infer the type of this tuple parameter // warning: 'f' is never used
}
fun assignList() {
  val xs = [1, 2]
  xs[0] = 3 // error: is not index assignment // error: cannot assign into an immutable List; use MutableList
}
fun assignMap() {
  val m = ["a": 1]
  m["a"] = 2 // error: is not index assignment // error: cannot assign into an immutable Map; use MutableMap
}
fun filterWrong(xs: List<i64>, f: fun(i64): i64) {
  val _ = xs.filter(f) // error: expected a function returning 'bool', found 'i64'
}
fun mapTwo(xs: List<i64>, f: fun(i64, i64): i64) {
  val _ = xs.map(f) // error: expected a function taking 1 argument, found 'fun(i64, i64): i64'
}
fun filterIsArgs(ss: List<Shape>) {
  val _ = ss.filterIs(1) // error: 'filterIs' takes no arguments; the variant is its type argument
}
fun filterIsNone(ss: List<Shape>) {
  val _ = ss.filterIs() // error: 'filterIs' needs one type argument, a variant of 'Shape'
}
fun raceBindTwo(ch: Channel<i64>) {
  val _ = race {
    val (a, b) = ch.recv() => 1 // error: race arms bind a single name
  }
}
fun raceEmpty() {
  val _ = race { } // error: 'race' needs at least one arm
}
fun letElse(x: i64?) {
  val (a, b) = x else { return } // error: cannot destructure a value of type 'i64' into 2 names
  io.println("$a $b")
}
fun notShow(): string = describe(Temp(c: 1)) // error: type 'Temp' does not implement trait 'Show', so it cannot be used as a 'Show' value
fun describe(s: Show): string = s.show()

fun main() {
  io.println("more rules")
}
