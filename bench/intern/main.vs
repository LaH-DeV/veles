// Compiler-shaped: intern identifiers. A million names drawn from a pool of
// 50k distinct ones go through a string-keyed map that hands out dense ids
// and keeps the names in a list, the way a symbol table does.
use io { println }
use time

struct Interner {
  ids:   MutableMap<string, i64> = [:]
  names: MutableList<string> = []

  fun intern(name: string): i64 {
    val known = this.ids.get(name)
    if (known != null) return known
    val id = this.names.len()
    this.ids.set(name, id)
    this.names.push(name)
    id
  }
}

fun main() {
  // the spellings are built before the clock starts
  val pool: MutableList<string> = []
  loop (i in 0..<50000) pool.push("sym_${i * 7919 % 100003}_x")
  val stream: MutableList<string> = []
  var x: i64 = 88172645463325252
  loop (_ in 0..<1000000) {
    x = x ^ (x << 13)
    x = x ^ (x >> 7)
    x = x ^ (x << 17)
    stream.push(pool.at((x & 0x7fffffff) % 50000) ?: "")
  }

  val sw = time.Stopwatch.start()
  var check: i64 = 0
  var interner = Interner()
  loop (s in stream) check += interner.intern(s)
  check += interner.names.len()
  println("BENCH intern 1000000 ${sw.elapsed().toNanos()} $check")
}
