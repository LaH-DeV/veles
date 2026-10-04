// Tests of `TaskLocal.bind` (D72, D126): a binding that lasts to the end of
// the block that holds it.

val level = TaskLocal(fallback: 0)

test fun readLevel(): i64 = level.get()

test fun readInChild(): i64 {
  var seen: i64 = -1
  scope {
    seen = await async readLevel()
  }
  seen
}

test fun early(): i64 {
  with b = level.bind(9)
  if (readLevel() == 9) return 1
  0
}

test fun failing() throws Boom {
  with b = level.bind(5)
  throw Boom()
}

error Boom { }

test fun inner(): i64 {
  with b = level.bind(2)
  readLevel()
}

test fun outer(): (i64, i64, i64) {
  with b = level.bind(1)
  val before = readLevel()
  val nested = inner()
  (before, nested, readLevel())
}

test "a binding lasts to the end of its block and gives the old value back" {
  expect(level.get() == 0)
  val (before, nested, after) = outer()
  expect(before == 1)
  expect(nested == 2)
  expect(after == 1)
  expect(level.get() == 0)
}

test "tasks started inside a binding see it" {
  with b = level.bind(7)
  expect(readInChild() == 7)
}

test "a binding ends on return and on a thrown error" {
  expect(early() == 1)
  expect(level.get() == 0)
  expect(failing() is Err)
  expect(level.get() == 0)
}
