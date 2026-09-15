use io

struct Job { id: i32 }
struct Done { id: i32, worker: i32 }

fun worker(id: i32, jobs: Channel<Job>, results: Channel<Done>) {
  loop {
    val job = await jobs.recv()
    if (job == null) break
    await sleep(1)
    results.send(Done(id: job.id, worker: id))
  }
}

fun square(x: i32): i32 {
  await sleep(2)
  x * x
}

struct Boom { n: i32 }

fun mayFail(n: i32): i32 throws Boom {
  await sleep(1)
  if (n == 2) throw Boom(n: n)
  n * 10
}

fun main() throws {
  // D16/D34: channels are methods; scope joins its children
  val jobs = Channel<Job>(capacity: 8)
  val results = Channel<Done>(capacity: 8)
  scope {
    async worker(1, jobs, results)
    async worker(2, jobs, results)
    loop (i in 1..4) { jobs.send(Job(id: i)) }
    jobs.close()
    var got: MutableList<i32> = []
    loop (i in 1..4) {
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
      val msg = ch.recv() => "message ${msg ?: "closed"}"
      sleep(1000) => "timeout"
    }
    io.println("race $winner")
    val second = race {
      val msg = ch.recv() => "message ${msg ?: "closed"}"
      sleep(5) => "timeout"
    }
    io.println("race $second")
  }

  // fail-fast: the first child error cancels siblings and propagates
  scope {
    async mayFail(1)
    async mayFail(2)
    async slowLoop()
  }
  io.println("unreachable")
}

fun producer(ch: Channel<string>) {
  await sleep(2)
  ch.send("hello")
}

fun slowLoop() {
  loop (i in 0..<100) {
    io.println("Printing slowLoop iteration $i");
    await sleep(1)
  }
  io.println("slowLoop finished (should have been cancelled)")
}
