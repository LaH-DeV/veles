use io

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
    io.println("results ${got.sorted()}")
  }

  // task handles and await
  scope {
    val a = async square(3)
    val b = async square(4)
    io.println("squares ${await a} ${await b}")
  }

  // D36: gather collects every outcome as a tuple of Results
  val (r1, r2, r3) = gather {
    async mayFail(1)
    async mayFail(2)
    async square(5)
  }
  io.println("gather $r1 $r2 $r3")

  // D38: race — first ready arm wins
  val ch = Channel<string>(capacity: 1)
  scope {
    async producer(ch)
    val winner = race {
      val msg = ch.recv()        => "message ${msg ?: "closed"}"
      sleep(Duration.seconds(1)) => "timeout"
    }
    io.println("race $winner")
    val second = race {
      val msg = ch.recv()       => "message ${msg ?: "closed"}"
      sleep(Duration.millis(5)) => "timeout"
    }
    io.println("race $second")
  }

  // two races waiting on one channel: the first also waits on another,
  // wins there, and the second still hears the shared channel close
  val first = Channel<string>(capacity: 1)
  val shared = Channel<string>(capacity: 1)
  scope {
    val a = async raceTwo(first, shared)
    val b = async raceOne(shared)
    await sleep(Duration.millis(2))
    first.send("first")
    io.println("race ${await a}")
    shared.close()
    io.println("race ${await b}")
  }

  // a loop launches from one site; whichever of its tasks fails, the scope
  // rethrows that task's error
  when (val r = launchInLoop()) {
    is Ok  => io.println("loop: no error")
    is Err => io.println("loop: Boom(${r.n})")
  }

  // fail-fast: the first child error cancels siblings and propagates
  scope {
    async mayFail(1)
    async mayFail(2)
    async slowLoop()
  }
  io.println("unreachable")
}

fun launchInLoop() throws Boom {
  scope {
    loop (i in 1..3) {
      async mayFail(i)  // the second launch throws
    }
  }
}

fun raceTwo(a: Channel<string>, b: Channel<string>): string = race {
  val msg = a.recv() => "a: ${msg ?: "closed"}"
  val msg = b.recv() => "b: ${msg ?: "closed"}"
}

fun raceOne(ch: Channel<string>): string = race {
  val msg = ch.recv()        => "shared: ${msg ?: "closed"}"
  sleep(Duration.seconds(5)) => "timeout"
}

fun producer(ch: Channel<string>) {
  await sleep(Duration.millis(2))
  ch.send("hello")
}

fun slowLoop() {
  loop (i in 0..<100) {
    io.println("Printing slowLoop iteration $i")
    await sleep(Duration.millis(500))  // well after mayFail(2) fails: cancelled after iteration 0
  }
  io.println("slowLoop finished (should have been cancelled)")
}
