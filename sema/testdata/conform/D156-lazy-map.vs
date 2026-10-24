// D156 part 4: `iter.mapConcurrent(f, workers:)` is lazy and holds tasks;
// a loop over it receives it (no `with`), anything else still needs `with`.
use io

error Boom { n: i64 }

fun download(n: i64): string throws Boom {
  await sleep(Duration.millis(1))
  if (n < 0) throw Boom(n)
  "<$n>"
}

fun plain(n: i64): i64 {
  await sleep(Duration.millis(1))
  n
}

fun pipelined(xs: List<i64>): string throws Boom {
  var out = ""
  loop (s in xs.iter().mapConcurrent(x => try download(x), workers: 2)) {
    out = out + s
  }
  out
}

fun outcomes(xs: List<i64>): i64 {
  var failed = 0
  loop (r in xs.iter().mapConcurrent(x => download(x))) {
    if (r is Err) failed += 1
  }
  failed
}

fun bound(xs: List<i64>): i64 {
  with it = xs.iter().mapConcurrent(x => plain(x), workers: 3)
  var total: i64 = 0
  loop (n in it) total += n
  total
}

fun stopped(xs: List<i64>): i64 {
  var first: i64 = 0
  loop (n in xs.iter().mapConcurrent(x => plain(x))) {
    first = n
    break
  }
  first
}

fun kept(xs: List<i64>): i64 {
  val _ = xs.iter().mapConcurrent(x => plain(x)) // error: holds a running task, so it must be received
  0
}

fun notSendable(xs: List<i64>): i64 {
  val seen: MutableList<i64> = []
  var total: i64 = 0
  loop (n in xs.iter().mapConcurrent(x => { // error: not Sendable
    seen.push(x)
    x
  })) {
    total += n
  }
  total
}

fun main() throws Boom {
  io.println("${try pipelined([1, 2, 3])} ${outcomes([1, -2])} ${bound([1, 2])}")
}
