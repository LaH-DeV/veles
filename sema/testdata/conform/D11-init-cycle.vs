// Module-level values are computed in the order they read each other —
// through the functions their initializers call too — so a cycle has no
// first value and is an error.
use io

val a: i64 = b + 1 // error: initialization cycle: 'a' reads 'b', which reads 'a'
val b: i64 = a + 1

val e: i64 = twice() // error: initialization cycle: 'e' reads itself through its call of 'twice', before it has a value
fun twice(): i64 = viaHelper() * 2
fun viaHelper(): i64 = e

val late: i64 = early() + 1
fun early(): i64 = base * 2
val base: i64 = "four".len()

fun main() {
  io.println("${a} ${e} ${late}")
}
