// D156 parts 3 and 6: `scope(workers: n)` and `gather(workers: n)` run at
// most n children at once, with or without `on:`; a placed or limited
// scope whose body starts no task of its own is refused.
use io

fun work(n: i64): i64 {
  await sleep(Duration.millis(1))
  n
}

fun decode(n: i64): i64 => n * 2

fun limited(xs: List<i64>) {
  scope(workers: 4) {
    loop (x in xs) async work(x)
  }
}

fun placedLimited(cpu: Executor, xs: List<i64>) {
  scope(on: cpu, workers: 2) {
    loop (x in xs) async work(x)
  }
}

fun computed(xs: List<i64>) {
  scope(workers: xs.len() / 2 + 1) {
    loop (x in xs) async work(x)
  }
}

fun gathered(): i64 {
  val (a, b) = gather(workers: 1) {
    async work(1)
    async work(2)
  }
  (a ?? 0) + (b ?? 0)
}

fun zero() {
  scope(workers: 0) { // error: 'workers' must be at least 1
    async work(1)
  }
}

fun nothingPlaced(cpu: Executor): i64 {
  var r: i64 = 0
  scope(on: cpu) { // error: this body runs in the calling task
    r = decode(4)
  }
  r
}

fun nothingGathered(cpu: Executor) {
  gather(on: cpu) { // error: this body runs in the calling task
    io.println("${decode(4)}")
  }
}

fun nothingLimited() {
  scope(workers: 2) { // error: remove 'workers:'
    io.println("x")
  }
}

fun nestedOnly(cpu: Executor) {
  scope(on: cpu) { // error: this body runs in the calling task
    scope {
      async work(1)
    }
  }
}

fun main() {
  limited([1, 2, 3])
  io.println("${gathered()}")
}
