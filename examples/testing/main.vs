use io

error Mismatch {
  expected: i64
  actual:   i64
}

fun add(a: i64, b: i64) = a + b

fun expectEq(expected: i64, actual: i64) throws Mismatch {
  if (expected != actual) throw Mismatch(expected, actual)
}

@test
fun additionWorks() throws Mismatch {
  try expectEq(4, add(2, 2))
}

@test
fun tasksRunInTests() throws Mismatch {
  val ch = Channel<i64>(capacity: 1)
  scope {
    ch.send(41)
    val v = await ch.recv()
    try expectEq(42, (v ?: 0) + 1)
  }
}

@test
fun thisOneFails() throws Mismatch {
  try expectEq(1, add(1, 1))
}

@deprecated("use add")
fun plus(a: i64, b: i64) = add(a, b)

@mustUse
fun important(): i64 = 7

fun main() {
  io.println("${plus(1, 2)} ${important()}")
}

@test
fun panicsAreReported() {
  val xs = [1]
  io.println("${xs.atOrPanic(3)}")
}
