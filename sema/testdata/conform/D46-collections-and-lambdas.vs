// D25/D40/D46: collection constructors and methods, and the lambdas they take.
use io

fun emptyMap() {
  val _ = [:] // error: cannot infer the type of an empty map; annotate it, e.g. 'val m: Map<string, i32> = [:]'
}
fun emptyMutable() {
  var xs = mut [] // error: cannot infer the element type of an empty list; annotate it, e.g. 'var xs: MutableList<i32> = []'
  var m = mut [:] // error: cannot infer the type of an empty map; annotate it, e.g. 'var m: MutableMap<string, i32> = [:]'
  io.println("$xs $m")
}
fun ctorWithArgs() {
  val _ = MutableMap<string, i64>(4) // error: 'MutableMap()' takes no arguments; use a literal to construct with contents
}
fun ctorUninferred() {
  val _ = MutableMap() // error: cannot infer the type arguments of 'MutableMap'
}
fun addToSet(s: Set<i64>) {
  s.add(1) // error: cannot call 'add' on an immutable Set; use MutableSet
}
fun mapValuesToUnit(m: Map<string, i64>) {
  val _ = m.mapValues(v => io.println("$v")) // error: 'mapValues' needs a function that returns a value
}
fun mapWithNumber(xs: List<i64>) {
  val _ = xs.map(5) // error: expected a function of type
}
fun mapWithTwo(xs: List<i64>) {
  val _ = xs.map((a: i64, b: i64) => a) // error: lambda takes 2 parameters but a function of 1 is expected here
}
fun mapToUnit(xs: List<i64>) {
  val _ = xs.map(x => io.println("$x")) // error: 'map' needs a function that returns a value
}
fun containsFunctions(fs: List<fun(): i64>, f: fun(): i64): bool = fs.contains(f) // error: elements of type 'fun(): i64' cannot be compared
fun declaredParam(xs: List<i64>) {
  val _ = xs.map((x: string) => x) // error: parameter 'x' is declared 'string' but 'i64' is expected here
  val f: fun(i64): i64 = (x: string) => 1 // error: parameter 'x' is declared 'string' but 'i64' is expected here
  io.println("${f(1)}")
}
fun tupleParam(ps: List<(i64, i64)>) {
  val _ = ps.map((a: string, b) => b) // error: parameter 'a' is declared 'string' but the tuple element is 'i64'
}
fun lambdaNoValue() {
  val f: fun(): i64 = () => { // error: lambda must return a value of type 'i64'
    io.println("none")
  }
  io.println("${f()}")
}

fun mapAll<T, U>(xs: List<T>, f: fun(T): U): List<U> {
  val out: MutableList<U> = []
  loop (x in xs) out.push(f(x))
  out.toList()
}

// the list is the one mistake: the lambda, typed from it, is not a second
fun builtFromAMistake(): List<i64> {
  val xs = List.generate(3, i => i) // error: no static function 'generate' on type 'List'; did you mean 'MutableList<T>.make(...)'?
  mapAll(xs, x => x * 3)
}

fun main() {
  io.println("collections")
}
