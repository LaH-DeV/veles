// D166: tuple elements may be labelled. A label reads an element; labels do
// not change layout, equality or the generic instance. Unlabelled converts to
// labelled and back; two different label lists do not.
use io

fun divide(a: i64, b: i64): (quotient: i64, remainder: i64) => (a / b, a % b)

fun point(): (x: i64, y: i64) => (x: 1, y: 2)

fun swapped(): (y: i64, x: i64) => point() // error: type mismatch: expected '(y: i64, x: i64)', found '(x: i64, y: i64)'; the labels differ — drop them to keep the positions ('(v.0, v.1)'), or reorder the elements (D166)

fun mirrored(p: (x: i64, y: i64)): (y: i64, x: i64) => p // error: type mismatch: expected '(y: i64, x: i64)', found '(x: i64, y: i64)'; the labels differ — drop them to keep the positions ('(p.0, p.1)'), or reorder the elements (D166)

fun pick<T>(x: T): T => x

fun main() {
  val r = divide(7, 2)
  io.println("${r.quotient} ${r.remainder} ${r.0}")
  val (q, rest) = divide(9, 4)
  val plain: (i64, i64) = r
  val back: (quotient: i64, remainder: i64) = plain
  val lit = (key: "a", value: 1)
  io.println("$q $rest ${plain.0} ${back.quotient} ${lit.key} ${pick(r).remainder}")
  io.println("${r.missing}") // error: tuple '(quotient: i64, remainder: i64)' has no element 'missing'; its labels are 'quotient', 'remainder'
  io.println("${plain.quotient}") // error: tuple of 2 elements has no element 'quotient'
  val wrong: (a: i64, b: i64) = (b: 1, a: 2) // error: type mismatch: expected '(a: i64, b: i64)', found '(b: i64, a: i64)'; the labels differ — drop them to keep the positions ('(1, 2)'), or reorder the elements (D166)
  val entries = ["apple": 3].entries()
  val turned: List<(value: string, key: i64)> = entries // error: type mismatch: expected 'List<(value: string, key: i64)>', found 'List<(key: string, value: i64)>'; the labels of the tuples inside differ, and converting would change what the elements mean — convert the elements one by one (D166)
  val loose: List<(string, i64)> = entries
  // a tuple key is not a lambda's parameters; a lambda value still parses
  val grid: Map<(x: i64, y: i64), string> = [(0, 0): "origin", (x: 1, y: 2): "point"]
  val scaled = [(1, 2): (n: i64): i64 => n * 2]
  io.println("${grid.len()} ${scaled.len()}")
  io.println("${r == plain} ${wrong.a} ${turned.len()} ${loose.len()}")
}
