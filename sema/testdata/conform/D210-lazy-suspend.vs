// D210: the initializer of a lazy value runs inside whichever read comes
// first, which does not wait for it: it may not suspend.
use io

fun sleepy(): i64 {
  await sleep(Duration.millis(1))
  1
}

lazy val slept = sleepy()  // error: the initializer of 'lazy val slept' cannot throw or suspend

fun main() {
  io.println("${slept}")
}
