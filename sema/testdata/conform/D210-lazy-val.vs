// D210: a `lazy val` is built on its first read, once. Its initializer is
// a sendable function that may not throw or suspend, its type is Sendable,
// it is ordered with the other module values (a cycle is an error), and a
// constant cannot read it.
use io

error Missing { }

fun fetch(): i64 throws Missing => throw Missing()

lazy val fine = [1, 2, 3].toSet()
lazy val thrown = try fetch()                  // error: the initializer of 'lazy val thrown' cannot throw or suspend
lazy val table: MutableMap<string, i64> = [:]  // error: 'lazy val table' has type 'MutableMap<string, i64>', which is not Sendable

lazy val a: i64 = b + 1  // error: initialization cycle: 'a' is read while its own initializer is computed
lazy val b: i64 = a + 1

const fromLazy = fine.len() + 1  // error: 'fine' is a lazy value, built at run time on its first read; a constant cannot read it

struct Tz {
  name: string

  static lazy val utc = Tz(name: "UTC")
}

fun main() {
  io.println("${fine.len()} ${thrown} ${a} ${fromLazy} ${Tz.utc.name}")
}
