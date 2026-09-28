// D13: a `when` used as a value gives one from every arm; a constant
// pattern needs a subject that compares with '=='.
use io

struct Handler {
  f: fun(i64): i64
}

fun emptyArm(n: i64) {
  val m = when (n) { // error: every arm of a 'when' used as a value must produce a value
    1    => 5
    else => { }
  }
  io.println("$m")
}

fun constantOnFunction(h: Handler) {
  when (h) {
    1    => io.println("one") // error: values of type 'Handler' cannot be matched against a constant; match its fields, or test it with a guard
    else => io.println("other")
  }
}

fun main() {
  emptyArm(1)
}
