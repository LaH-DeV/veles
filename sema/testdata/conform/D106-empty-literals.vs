// D106: an empty literal still needs its type; the message names the kind
// the binding needs, and a fix writes it when a later use decides it.
use io

fun total(xs: List<f64>): f64 = xs.sum()

fun pushed() {
  var xs = [] // error: write it: 'var xs: MutableList<i64> = []'
  xs.push(1)
  io.println("$xs")
}

fun passed(): f64 {
  val xs = [] // error: write it: 'val xs: List<f64> = []'
  total(xs)
}

fun mapped() {
  val m = [:] // error: write it: 'val m: MutableMap<string, bool> = [:]'
  m.set("a", true)
  io.println("$m")
}

fun undecided() {
  val xs = [] // error: annotate it, e.g. 'val xs: List<i64> = []'
  io.println("$xs")
}

fun main() {
  pushed()
  io.println("${passed()}")
  mapped()
  undecided()
}
