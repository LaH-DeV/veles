// D166: labels are for reading — a labelled tuple has the unlabelled one's
// layout, and `divmod` and `divmodLabelled` compile to the same body. One
// instance of `swap` serves both. Only printing differs: a labelled tuple's
// text helper writes its labels.
use io

fun divmod(a: i64, b: i64): (i64, i64) => (a / b, a % b)

fun divmodLabelled(a: i64, b: i64): (quotient: i64, remainder: i64) => (a / b, a % b)

fun swap<A, B>(pair: (A, B)): (B, A) => (pair.1, pair.0)

fun main() {
  val plain = divmod(7, 2)
  val named = divmodLabelled(7, 2)
  val back = swap(named)
  io.println("$plain $named ${named.quotient + plain.1} ${swap(plain)} $back ${plain == named}")
}
