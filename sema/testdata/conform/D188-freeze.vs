// D188, superseded by D231: a collection is a value, so `freeze()` and
// `isFrozen()` are gone; each is an error with a fix that drops the call.
use io

fun lists(): List<i64> {
  val xs: List<i64> = [1]
  xs.freeze()  // error: 'freeze' was removed (D231): a collection is a value, which no other name can change; 'xs' is enough
  xs
}

fun maps(): bool {
  val m: Map<string, i64> = ["a": 1]
  m.isFrozen()  // error: 'isFrozen' was removed (D231)
}

fun sets(): Set<i64> {
  val s: Set<i64> = [1]
  s.freeze()  // error: 'freeze' was removed (D231)
  s
}

fun main() {
  io.println("${lists()} ${maps()} ${sets()}")
}
