// Tasks: 64 allocation-heavy jobs (lists, sorting, a map each) run at
// once — how the work spreads over cores with one shared heap (D66).
use io { println }, time

fun work(seed: i64): i64 {
  var total: i64 = 0
  val words: MutableMap<string, i64> = [:]
  loop (round in 0..<200) {
    val xs: MutableList<i64> = []
    loop (i in 0..<200) {
      xs.push((seed * 31 + i * 7 + round) % 1000)
    }
    val sorted = xs.sorted()
    total += sorted.at(100) ?: 0
    val key = "k${(seed + round) % 17}"
    words.set(key, (words.get(key) ?: 0) + 1)
  }
  total + words.len()
}

fun worker(seed: i64, out: Channel<i64>) {
  out.send(work(seed))
}

fun main() {
  val sw = time.Stopwatch.start()
  val out = Channel<i64>(capacity: 64)
  var sum: i64 = 0
  scope {
    loop (s in 0..<64) {
      async worker(s, out)
    }
    loop (_ in 0..<64) {
      sum += (await out.recv()) ?: 0
    }
  }
  println("BENCH parallel 64 ${sw.elapsed().toNanos()} $sum")
}
