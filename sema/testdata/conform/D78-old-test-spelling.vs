// D78: the old `@test fun` spelling is refused with the new one; a test
// written that way with parameters is also told they have to go.
use io

@test // error: a test is written 'test "old style" { ... }' (D78)
fun oldStyle(x: i64) { // error: a test takes no parameters and returns nothing
}

fun main() {
  io.println("")
}
