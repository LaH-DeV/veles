// D158: in a method a bare name reaches the receiver's members — locals and
// parameters first, then the members of `this`, then the module. A static
// function has no receiver; the type's static members are reachable bare in
// every function of it.
use io

fun scale(n: i64): i64 => n * 100

struct Counter {
  var count: i64 = 0
  note: string? = null
  static val step = 2

  fun bump(): i64 {
    count += step
    count
  }
  fun describe(): i64 => if (note != null) note.len() else 0
  fun withParam(count: i64): i64 => count + this.count
  fun viaLambda(): i64 {
    val f = (n: i64) => n + count
    f(1)
  }
  fun scale(n: i64): i64 => n
  fun hidden(): i64 => scale(3) // warning: 'scale' means this.scale here (D158)
  static fun make(): Counter => Counter(count: step)
  static fun broken(): i64 => count // error: 'count' is a member of 'Counter', but this function is static: it has no receiver
}

fun main() {
  var c = Counter.make()
  io.println("${c.bump()} ${c.describe()} ${c.withParam(1)} ${c.viaLambda()} ${c.hidden()}")
}
