// D114 per build profile: `atUnchecked`, `setUnchecked` and
// `byteAtUnchecked` keep a bounds check (a panic naming the unchecked
// access) in debug and compile to a plain load or store in release.
use io

fun get(xs: MutableList<i64>, i: i64): i64 {
  // SAFETY: callers pass an index below xs.len()
  unsafe { xs.atUnchecked(i) }
}

fun put(xs: MutableList<i64>, i: i64, v: i64) {
  // SAFETY: callers pass an index below xs.len()
  unsafe { xs.setUnchecked(i, v) }
}

fun byte(s: string, i: i64): u8 {
  // SAFETY: callers pass an index below s.len()
  unsafe { s.byteAtUnchecked(i) }
}

fun main() {
  val xs: MutableList<i64> = [3, 5, 7]
  put(xs, 1, get(xs, 0) + get(xs, 2))
  io.println("${xs} ${byte("veles", 1)}")
}
