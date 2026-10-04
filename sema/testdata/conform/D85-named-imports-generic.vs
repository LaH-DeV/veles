// D85: a name used only in a generic function nobody calls is not reported
// unused — that body is checked only when instantiated, so the use has not
// been looked up (std/http's `screen` imports `field` for such a body).
use io { println }
use log { info }

fun neverCalled<T>(x: T): T {
  info("seen")
  return x
}

fun main() {
  println("hello")
}
