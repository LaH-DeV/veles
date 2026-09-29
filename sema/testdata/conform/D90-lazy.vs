// D90: a `lazy` parameter is the standard library's for now.
use io

fun later(lazy msg: fun(): string) { // error: 'lazy' is reserved for the standard library for now (D90)
  io.println(msg())
}

fun main() {
  later("now")
}
