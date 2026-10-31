// D229, D230: expectEqual, expect's message and expectNear take what they say.
use io

test "shapes" {
  expect(1 == 1, "a", "b") // error: 'expect' takes a condition and, if it helps, a message shown when it fails
  expectEqual(1) // error: 'expectEqual' takes the expected value, then the actual one
  expectNear(0.1, 0.2) // error: 'expectNear' takes the expected value, the actual one and a named tolerance
  expectNear(1, 2, within: 1) // error: 'expectNear' compares two floats of one type within a tolerance of that type
  fail("a", "b") // error: 'fail' takes one argument: fun fail(why: string): Never
}

fun main() {
  io.println("ok")
}
