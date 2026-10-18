// D178: skip, skipIf, @Skip and @SkipIf; what a program may not do with them.
use io

const slow = true

@Skip("not ready")
test "annotated" {
  fail("never runs")
}

@SkipIf(slow, "too slow here")
test "conditional" {
  fail("never runs")
}

@Skip("a whole suite")
suite "off" {
  test "inside" { fail("never runs") }
}

suite "gated" {
  skipIf(slow, "gate")
  skip("again")
  test "inside" { fail("never runs") }
}

test "words" {
  skip() // error: 'skip' takes the reason: fun skip(why: string): Never
  skipIf(true) // error: 'skipIf' takes a condition and the reason: fun skipIf(condition: bool, why: string)
  skipIf(1, "x") // error: type mismatch: expected 'bool', found 'i64'
  skip(3) // error: type mismatch: expected 'string', found 'i64'
}

fun main() {
  io.println("skipping")
}
