// D16/D34: a global's initializer runs before any task can, so it cannot
// open a scope of tasks, even inside a block of its value.
use io

fun two(): i64 = 2

val scoped: i64 = if (true) {
  scope { // error: 'scope' cannot appear in a global initializer; compute the value in 'main' and pass it down
    async two()
  }
  1
} else 0

// nor start one with `with t = async f()` (D100)
val background: i64 = with (t = async two()) { 1 } // error: 'async' cannot appear in a global initializer

fun main() {
  io.println("$scoped $background")
}
