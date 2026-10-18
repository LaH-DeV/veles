use io { println }

struct Job {
  id: i64
}
struct Done {
  id:     i64
  worker: i64
}

fun worker(id: i64, jobs: Channel<Job>, results: Channel<Done>) {
  loop {
    val job = await jobs.recv()
    if (job == null) break
    await sleep(Duration.millis(1))
    results.send(Done(id: job.id, worker: id))
  }
}

fun square(x: i64): i64 {
  await sleep(Duration.millis(2))
  x * x
}

error Boom {
  n: i64
}

fun mayFail(n: i64): i64 throws Boom {
  await sleep(Duration.millis(1))
  if (n == 2) throw Boom(n)
  n * 10
}

fun main() throws {
  // D16/D34: channels are methods; scope joins its children
  val jobs = Channel<Job>(capacity: 8)
  val results = Channel<Done>(capacity: 8)
  scope {
    async worker(1, jobs, results)
    async worker(2, jobs, results)
    loop (i in 1..4) {
      jobs.send(Job(id: i))
    }
    jobs.close()
    var got: MutableList<i64> = []
    loop (_ in 1..4) {
      val d = await results.recv()
      if (d != null) got.push(d.id)
    }
    println("results ${got.sorted()}")
  }

  // task handles and await
  scope {
    val a = async square(3)
    val b = async square(4)
    println("squares ${await a} ${await b}")
  }

  // D156: one handle needs no scope, but every path must await it;
  // awaitAll waits for several and gives their values
  val c = async square(6)
  val (d, e) = awaitAll(async square(7), async square(8))
  println("awaited ${await c} $d $e")

  // D156: a scope is a value, and workers: bounds how many children run
  // at once
  val running: Atomic<i64> = Atomic(value: 0)
  val seen = Channel<i64>(capacity: 16)
  val launched = scope(workers: 2) {
    loop (_ in 0..<8) async limitedJob(running, seen)
    8
  }
  seen.close()
  var peak: i64 = 0
  loop {
    val n = await seen.recv() ?: break
    peak = peak.max(n)
  }
  println("$launched jobs, at most $peak at once")

  // D156: a lazy mapConcurrent keeps two calls under way and gives the
  // results in order; the loop owns its tasks
  loop (s in [1, 2, 3, 4].iter().mapConcurrent(x => square(x), workers: 2)) {
    println("lazy $s")
  }

  // D36: gather collects every outcome as a tuple of Results
  val (r1, r2, r3) = gather {
    async mayFail(1)
    async mayFail(2)
    async square(5)
  }
  println("gather $r1 $r2 $r3")

  // D38/D169: select — the first ready arm wins
  val ch = Channel<string>(capacity: 1)
  scope {
    async producer(ch)
    val winner = select {
      val msg = ch.recv()        => "message ${msg ?: "closed"}"
      sleep(Duration.seconds(1)) => "timeout"
    }
    println("race $winner")
    val second = select {
      val msg = ch.recv()       => "message ${msg ?: "closed"}"
      sleep(Duration.millis(5)) => "timeout"
    }
    println("race $second")
  }

  // two selects waiting on one channel: the first also waits on another,
  // wins there, and the second still hears the shared channel close
  val first = Channel<string>(capacity: 1)
  val shared = Channel<string>(capacity: 1)
  scope {
    val a = async raceTwo(first, shared)
    val b = async raceOne(shared)
    await sleep(Duration.millis(2))
    first.send("first")
    println("race ${await a}")
    shared.close()
    println("race ${await b}")
  }

  // a loop launches from one site; whichever of its tasks fails, the scope
  // rethrows that task's error
  when (val r = launchInLoop()) {
    is Ok  => println("loop: no error")
    is Err => println("loop: Boom(${r.n})")
  }

  // fail-fast: the first child error cancels siblings and propagates
  scope {
    async mayFail(1)
    async mayFail(2)
    async slowLoop()
  }
  println("unreachable")
}

fun launchInLoop() throws Boom {
  scope {
    loop (i in 1..3) {
      async mayFail(i)  // the second launch throws
    }
  }
}

fun raceTwo(a: Channel<string>, b: Channel<string>): string => select {
  val msg = a.recv() => "a: ${msg ?: "closed"}"
  val msg = b.recv() => "b: ${msg ?: "closed"}"
}

fun raceOne(ch: Channel<string>): string => select {
  val msg = ch.recv()        => "shared: ${msg ?: "closed"}"
  sleep(Duration.seconds(5)) => "timeout"
}

/// One job of a limited scope: reports how many jobs run with it.
fun limitedJob(running: Atomic<i64>, seen: Channel<i64>) {
  seen.send(running.add(1))
  await sleep(Duration.millis(2))
  running.sub(1)
}

fun producer(ch: Channel<string>) {
  await sleep(Duration.millis(2))
  ch.send("hello")
}

fun slowLoop() {
  loop (i in 0..<100) {
    println("Printing slowLoop iteration $i")
    await sleep(Duration.millis(500))  // well after mayFail(2) fails: cancelled after iteration 0
  }
  println("slowLoop finished (should have been cancelled)")
}
