// Reading a `lazy val` (D210) once it is built: a 256-entry table built on
// the first read, then read 50 million times through a function that is not
// inlined, as a keyword or an operator table is read by a lexer. The read is
// an acquire load of the ready flag, a branch and the value; Go's reference
// is a sync.OnceValue global.
use io { println }
use time

fun buildTable(): List<i64> {
  val out: MutableList<i64> = []
  loop (i in 0..<256) out.push(i * 7 % 13)
  out
}

lazy val table = buildTable()

@noinline
fun lookup(i: i64): i64 => table.at(i % 256) ?: 0

fun main() {
  val n = 50000000
  val clock = time.Stopwatch.start()
  var check: i64 = 0
  loop (i in 0..<n) check += lookup(i)
  val elapsed = clock.elapsed()
  println("BENCH lazyval $n ${elapsed.toNanos()} $check")
}
