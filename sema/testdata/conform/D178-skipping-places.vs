// D178: @Skip stands on a test or a suite; a test takes no other attribute.
use io

@Skip("on a function") // error: '@Skip' stands on test, suite, and cannot stand on a function
fun plain() { }

@inline // error: a test takes annotations (@Skip, @SkipIf), not '@inline' (D178)
test "plain attribute" { }

@noinline // error: a suite takes annotations (@Skip, @SkipIf), not '@noinline' (D178)
suite "plain" {
  test "inside" { }
}

fun main() {
  io.println("places")
}
