// Tests (D78): `test "sentence" { }` next to the code, the test-only
// vocabulary (expect, require, expectThrows, expectPanics, fail), a
// `test fun` helper, a suite, and `assert` for an invariant in ordinary code.
// `veles test examples/testing` runs them; `veles run` leaves them out.
use io { println }

error RangeError {
  value: i64
}

fun add(a: i64, b: i64) => a + b

fun parsePort(text: string): i64 throws RangeError {
  val n = text.toInt() ?: 0
  if (n < 0 || n > 65535) throw RangeError(value: n)
  n
}

fun half(n: i64): i64? => if (n % 2 == 0) n / 2 else null

/// A helper only tests can call: it may use the vocabulary itself.
test fun expectSorted(xs: List<i64>) {
  loop (i in 1..<xs.len()) {
    val before = xs.at(i - 1) ?: 0
    expect(before <= xs.at(i))
  }
}

test "adds small numbers" {
  expect(add(2, 2) == 4)
  expect(add(-1, 1) == 0)
}

// a suite groups tests under a name; its helpers are its own
suite "ports" {
  test fun expectPort(text: string, want: i64) {
    expect(require(parsePort(text)) == want)
  }

  test "parses" {
    expectPort("8080", 8080)
    expectPort("0", 0)
  }

  test "refuses the rest" {
    expectThrows<RangeError>(() => parsePort("70000"))
    expectThrows<RangeError>(() => parsePort("-1"))
  }
}

test "halves even numbers" {
  val h = require(half(10))
  expect(h == 5)
  expect(half(3) == null)
}

test "tasks run in tests" {
  val ch = Channel<i64>(capacity: 1)
  scope {
    ch.send(41)
    val v = await ch.recv()
    expect((v ?: 0) + 1 == 42)
  }
}

test "sorts" {
  expectSorted([3, 1, 2].sorted())
}

test "panics are expected" {
  expectPanics(() => {
    val _ = [1].at(3) ?: panic("index 3 is out of range")
  })
}

@deprecated("use add")
fun plus(a: i64, b: i64) => add(a, b)

@mustUse
fun important(): i64 => 7

fun main() {
  val total = plus(1, 2)
  assert(total == 3, "plus adds")  // an invariant: panics with the reason and both sides
  println("$total ${important()}")
}
