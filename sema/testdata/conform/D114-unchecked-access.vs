// D114: the unchecked accesses are callable only inside `unsafe`; the
// message names the checked spelling.
use io

fun outsideUnsafe(xs: List<i64>, s: string): i64 {
  val a = xs.getUnchecked(0)     // error: 'getUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'get(i)', or wrap the call in 'unsafe { }' with a '// SAFETY:' comment saying why the index is in range
  xs.setUnchecked(0, 1)         // error: 'setUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'set(i, v)'
  val b = s.byteAtUnchecked(0)  // error: 'byteAtUnchecked' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write 'byteAt(i)'
  a + b.toI64()
}



fun inside(xs: List<i64>): i64 {
  if (xs.isEmpty()) return 0
  // SAFETY: the list is not empty
  unsafe {
    xs.setUnchecked(0, xs.getUnchecked(0) + 1)
    xs.getUnchecked(0)
  }
}

fun main() {
  val xs: List<i64> = [1]
  io.println("${outsideUnsafe(xs, "a")} ${inside(xs)}")
}
