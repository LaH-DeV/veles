// D78: a suite's helpers share one namespace.
use io

suite "parser" {
  test fun helper() { }
  test fun helper() { } // error: 'helper' is already declared in this suite
}

fun main() {
  io.println("suites")
}
