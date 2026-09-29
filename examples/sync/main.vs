use io { println }

struct Counter {
  var hits: i64
}

fun bump(m: Mutex<Counter>, times: i64) {
  loop (_ in 0..<times) {
    m.withLock(c => c.hits += 1)
    await sleep(Duration.zero)
  }
}

fun main() {
  val shared = Mutex(value: Counter(hits: 0))
  val total = Atomic(value: 0)
  scope {
    async bump(shared, 5)
    async bump(shared, 7)
  }
  total.store(shared.get().hits)
  println("hits ${shared.get().hits} atomic ${total.load()} swapped ${total.swap(1)} now ${total.load()}")
}
