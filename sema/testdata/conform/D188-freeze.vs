// D188: freeze() makes a mutable collection immutable in place; an immutable
// one has nothing to freeze, and the error says so.
use io

fun frozenList() {
  val xs: List<i64> = [1]
  val _ = xs.freeze() // error: 'freeze' needs a MutableList: a List is already immutable
}

fun frozenMap() {
  val m: Map<string, i64> = ["a": 1]
  val _ = m.freeze() // error: 'freeze' needs a MutableMap: a Map is already immutable
}

fun frozenSet() {
  val s: Set<i64> = [1]
  val _ = s.freeze() // error: 'freeze' needs a MutableSet: a Set is already immutable
}

fun askedList() {
  val xs: List<i64> = [1]
  val _ = xs.isFrozen() // error: 'isFrozen' needs a MutableList: a List is immutable by its type
}

fun askedMap() {
  val m: Map<string, i64> = ["a": 1]
  val _ = m.isFrozen() // error: 'isFrozen' needs a MutableMap: a Map is immutable by its type
}

fun askedSet() {
  val s: Set<i64> = [1]
  val _ = s.isFrozen() // error: 'isFrozen' needs a MutableSet: a Set is immutable by its type
}

fun main() {
  frozenList()
  frozenMap()
  frozenSet()
  askedList()
  askedMap()
  askedSet()
  io.println("done")
}
