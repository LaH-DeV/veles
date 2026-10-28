// A task that ticks beside busy loops (plan F21, D145, D197): how late does
// it get to run? One busy task per worker computes, and a ticker sleeps
// 1 ms at a time and keeps the worst and the mean of how much later than
// that it woke. A loop in a suspending function gives its thread up once
// it has run ~10 ms while others wait (D145); a plain function's cannot
// (it has no frame to resume); `cede` gives it up where it is written.
//
// Without arguments (the runner): the suspending loops, against Go's
// goroutines, which the Go runtime preempts asynchronously after 10 ms.
// `veles run bench/preempt -- all` prints every case of plan F21 (or name
// the cases to run):
//
//   idle         the ticker alone: the timer's own resolution
//   suspending   busy loops in suspending functions (the automatic yield)
//   plain        busy loops in plain functions (no yield is possible)
//   cede         a `cede` at every iteration
//   cede4096     a `cede` every 4096 iterations
//   thread       a one-thread executor (D150) running a busy suspending
//                loop and the ticker
use io { println }
use os
use time

// the work: every busy task does `rounds` of it
fun step(x: i64, i: i64): i64 => (x * 31 + i) % 1000003

fun busySuspending(rounds: i64): i64 {
  if (rounds < 0) await sleep(Duration.millis(1))  // never: it only may suspend
  var x: i64 = 1
  loop (i in 0..<rounds) x = step(x, i)
  x
}

fun busyPlain(rounds: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<rounds) x = step(x, i)
  x
}

fun busyCede(rounds: i64, every: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<rounds) {
    x = step(x, i)
    if (i % every == 0) cede
  }
  x
}

struct Lateness {
  worst: Atomic<i64>
  total: Atomic<i64>
  ticks: Atomic<i64>
}

// a tick in flight when the busy tasks end is counted too: a ticker that
// could not run at all shows as one tick as late as the whole run
fun ticker(stop: Atomic<bool>, started: Channel<bool>, late: Lateness) {
  started.send(true)
  loop (!stop.load()) {
    val sw = time.Stopwatch.start()
    await sleep(Duration.millis(1))
    val ns = sw.elapsed().toNanos() - 1000000
    late.total.add(ns)
    late.ticks.add(1)
    late.worst.update(w => w.max(ns))
  }
}

fun busy(kind: string, rounds: i64): i64 => when (kind) {
  "plain"    => busyPlain(rounds)
  "cede"     => busyCede(rounds, 1)
  "cede4096" => busyCede(rounds, 4096)
  else       => busySuspending(rounds)
}

// busy tasks, one per worker, beside the ticker; the checksum of their work
fun scenario(kind: string, workers: i64, rounds: i64, late: Lateness): i64 {
  val stop = Atomic(value: false)
  val started = Channel<bool>(capacity: 1)
  var sum: i64 = 0
  scope {
    async ticker(stop, started, late)
    await started.recv()  // the ticker is ticking before the busy tasks start
    if (kind == "idle") {
      await sleep(Duration.millis(300))
    } else {
      val tasks: MutableList<Task<i64>> = []
      loop (_ in 0..<workers) tasks.push(async busy(kind, rounds))
      loop (t in tasks) sum += await t
    }
    stop.store(true)
  }
  sum
}

fun onThread(rounds: i64, late: Lateness): i64 throws ThreadError {
  with th = try Executor.thread(name: "one")
  th.run(() => scenario("suspending", 1, rounds, late))
}

fun ms(ns: i64): string {
  val tenths = (ns + 50000) / 100000
  "${tenths / 10}.${tenths % 10} ms"
}

fun report(kind: string, workers: i64, rounds: i64) {
  val late = Lateness(worst: Atomic(value: 0), total: Atomic(value: 0), ticks: Atomic(value: 0))
  val sw = time.Stopwatch.start()
  val sum = if (kind == "thread") {
    onThread(rounds, late) catch (e) {
      panic("no thread: ${e.message()}")
    }
  } else {
    scenario(kind, workers, rounds, late)
  }
  val elapsed = sw.elapsed().toNanos()
  val ticks = late.ticks.load()
  val mean = if (ticks > 0) late.total.load() / ticks else 0
  println("| $kind | $workers | ${ms(elapsed)} | $ticks | ${ms(late.worst.load())} | ${ms(mean)} | $sum |")
}

fun main() {
  val workers = Executor.defaultThreads()
  val rounds: i64 = 100000000
  val kinds = ["idle", "suspending", "plain", "cede", "cede4096", "thread"]
  val asked = os.args().filter(a => a == "all" || kinds.contains(a))
  if (!asked.isEmpty()) {
    println("| case | busy tasks | elapsed | ticks | worst lateness | mean lateness | checksum |")
    println("|---|---:|---:|---:|---:|---:|---:|")
    loop (kind in kinds) {
      if (!asked.contains("all") && !asked.contains(kind)) continue
      // a cede at every iteration is a trip through the run queue each:
      // a hundredth of the work
      report(kind, workers, if (kind == "cede") rounds / 100 else rounds)
    }
    return
  }
  val late = Lateness(worst: Atomic(value: 0), total: Atomic(value: 0), ticks: Atomic(value: 0))
  val sw = time.Stopwatch.start()
  val sum = scenario("suspending", workers, rounds, late)
  println("BENCH preempt ${workers * rounds} ${sw.elapsed().toNanos()} $sum")
  println("LATENCY preempt ${late.worst.load()}")
}
