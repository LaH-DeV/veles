// D109: `with expr` holds and closes a value the block does not name; in
// the block form, named and unnamed items mix; a bare existing name hands
// its closing to the `with`.
use io

struct Res {
  n: i64

  implement Closeable {
    fun close() { }
  }
}

struct NoClose {
  n: i64
}

fun open(n: i64): Res = Res(n)

fun held() {
  with open(1)
  with (r = open(2), open(3)) {
    io.println("${r.n}")
  }
  with _ = open(4)
  io.println("done")
}

fun takenOver(): Res {
  val conn = open(1)
  with conn
  io.println("${conn.n}")
  return conn // error: 'conn' cannot be returned: it is closed when its 'with' block ends
}

fun closedTwice() {
  val conn = open(1)
  with conn
  conn.close() // error: 'conn' is closed when its 'with' block ends
}

fun lastStatement() {
  io.println("x")
  with open(1) // warning: 'open(1)' is closed as soon as it is opened
}

fun notCloseable() {
  with NoClose(n: 1) // error: 'NoClose' is not Closeable
  io.println("after")
}

fun main() {
  held()
  closedTwice()
  lastStatement()
  notCloseable()
  io.println("${takenOver().n}")
}
