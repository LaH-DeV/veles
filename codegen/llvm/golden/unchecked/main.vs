// D114 per build profile: `getUnchecked`, `setUnchecked` and
// `byteAtUnchecked` keep a bounds check (a panic naming the unchecked
// access) in debug and compile to a plain load or store in release.
use io

fun get(xs: List<i64>, i: i64): i64 {
  // SAFETY: callers pass an index below xs.len()
  unsafe {
    xs.getUnchecked(i)
  }
}

fun put(xs: *List<i64>, i: i64, v: i64) {
  // SAFETY: callers pass an index below xs.len()
  unsafe {
    xs.setUnchecked(i, v)
  }
}

fun byte(s: string, i: i64): u8 {
  // SAFETY: callers pass an index below s.len()
  unsafe {
    s.byteAtUnchecked(i)
  }
}

fun main() {
  val xs: List<i64> = [3, 5, 7]
  put(&xs, 1, get(xs, 0) + get(xs, 2))
  io.println("${xs} ${byte("veles", 1)}")
}
