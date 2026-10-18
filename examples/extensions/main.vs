// A pipeline without a pipe operator: the extensions of `strx` read as
// methods, so a chain goes left to right on any type (D173).
use io { println }
use strx

fun main() {
  println("  Hello Big World ".slug().shout())
  println("${[1, 2, 3].isSorted()} ${[3, 1, 2].isSorted()}")
  println("${(150).clamped(0, 100)} ${(-5).clamped(0, 100)}")
  println("${"   ".isBlank()} ${"x".isBlank()}")
}
