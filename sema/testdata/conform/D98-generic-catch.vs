// D98 in generic code: a `do` block whose failures come through a `throws E`
// it was handed. In an instance where E is Never nothing reaches the
// handler, which is not an error there — the template's handler may fall
// through, so the code after the `do` is not unreachable either.
use io

fun attempts<R, E>(times: i64, f: fun(): R throws E): R throws E {
  var left = times
  loop {
    do {
      return try f()
    } catch (e) {
      left -= 1
      if (left == 0) throw e
    }
    io.println("again")
  }
}

error Nope { }

fun failing(): i64 throws Nope = throw Nope()

fun main() {
  io.println("${attempts(3, () => 7)}")
  io.println("${attempts(2, () => try failing()) is Err}")
}
