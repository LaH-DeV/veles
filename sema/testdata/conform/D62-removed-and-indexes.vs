// D62/D25: the removed `…OrPanic` reads are reported once, whatever their
// arguments; an index is an integer.
use io

fun removedWithoutIndex(xs: List<i64>): i64 = xs.atOrPanic() // error: 'atOrPanic' was removed (D62): write '.at(…) ?: panic("why this cannot fail")'

fun floatIndex(xs: MutableList<i64>) {
  val r = xs.ref(1.5) // error: index must be an integer, found 'f64'
  io.println("${r != null}")
}

fun main() {
  floatIndex([1])
}
