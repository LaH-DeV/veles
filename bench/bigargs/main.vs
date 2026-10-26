// Passing large values to functions: a 144-byte struct of numbers and a
// struct holding a 16-element array, each handed by value to a function
// that reads two fields, 20 million times (the plan B32 step 3 shape: an
// AST node or a parser state passed down a call).
use io { println }
use time

struct Wide {
  a: i64
  b: i64
  c: i64
  d: i64
  e: i64
  f: i64
  g: i64
  h: i64
  i: i64
  j: i64
  k: i64
  l: i64
  m: i64
  n: i64
  o: i64
  p: i64
  q: i64
  r: i64
}

struct Framed {
  head: i64
  body: Array<i64, 16>
  tail: i64
}

@noinline
fun readWide(w: Wide, k: i64): i64 => w.a + w.r + k

@noinline
fun readFramed(f: Framed, k: i64): i64 => f.head + f.tail + k

fun main() {
  val n = 20000000
  val wide = Wide(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10, k: 11, l: 12, m: 13, n: 14, o: 15, p: 16, q: 17, r: 18)
  val framed = Framed(head: 3, body: Array<i64, 16>.make(5), tail: 4)
  // a stopwatch, not time.measure: a lambda would capture the two values
  val clock = time.Stopwatch.start()
  var check: i64 = 0
  loop (k in 0..<n) check += readWide(wide, k)
  loop (k in 0..<n) check += readFramed(framed, k)
  val elapsed = clock.elapsed()
  println("BENCH bigargs ${2 * n} ${elapsed.toNanos()} $check")
}
