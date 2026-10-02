// D114: the unchecked accesses are callable only inside `unsafe`; the
// message names the checked spelling. `setUnchecked` needs a MutableList.
use io

fun outsideUnsafe(xs: MutableList<i64>, s: string): i64 {
  val a = xs.atUnchecked(0) // error: 'atUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'at(i)', or wrap the call in 'unsafe { }' with a '// SAFETY:' comment saying why the index is in range
  xs.setUnchecked(0, 1) // error: 'setUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'set(i, v)'
  val b = s.byteAtUnchecked(0) // error: 'byteAtUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'byteAt(i)'
  a + b.toI64()
}

fun immutable(xs: List<i64>) {
  // SAFETY: the list is not empty
  unsafe { xs.setUnchecked(0, 1) } // error: cannot set an element of an immutable List; use MutableList (D25)
}

fun inside(xs: MutableList<i64>): i64 {
  if (xs.isEmpty()) return 0
  // SAFETY: the list is not empty
  unsafe {
    xs.setUnchecked(0, xs.atUnchecked(0) + 1)
    xs.atUnchecked(0)
  }
}

fun main() {
  val xs: MutableList<i64> = [1]
  io.println("${outsideUnsafe(xs, "a")} ${inside(xs)}")
  immutable([1])
}
