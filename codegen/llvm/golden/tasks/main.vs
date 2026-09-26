// Suspending functions (the coroutine transform), a scope with children,
// a channel, and `with` cleanup on the way out; the non-suspending channel
// forms, and two executor regressions that used to hang.
use io

struct Resource {
  name: string
  implement Closeable { fun close() { io.println("closed ${this.name}") } }
}

fun slow(n: i64): i64 {
  await sleep(Duration.millis(1))
  return n * 2
}

fun produce(ch: Channel<i64>) {
  loop (i in 1..3) ch.send(i)
  ch.close()
}

fun main() {
  with (r = Resource(name: "db")) {
    scope {
      val a = async slow(4)
      val b = async slow(5)
      io.println("${await a} ${await b}")
    }
  }
  val ch = Channel<i64>(capacity: 1)
  var sum = 0
  scope {
    async produce(ch)
    loop {
      val v = await ch.recv() ?: break
      sum += v
    }
  }
  io.println("sum $sum")

  // trySend/tryRecv never wait
  val box = Channel<string>(capacity: 2)
  io.println("${box.trySend("a")} ${box.trySend("b")} ${box.trySend("c")} ${box.tryRecv() ?: "-"} ${box.tryRecv() ?: "-"} ${box.tryRecv() ?: "-"}")

  // main sleeps while its child finishes: the finish wakes the scope's
  // owner early, and the sleeper must go back to sleep, not vanish
  val slowly = Channel<i64>(capacity: 1)
  var got = 0
  scope {
    async produce(slowly)
    loop (got < 6) {
      await sleep(Duration.millis(1))
      got += await slowly.recv() ?: 0
    }
  }
  io.println("slept $got")

  // polling with tryRecv: a zero sleep yields, so the producer gets a turn
  val polled = Channel<i64>(capacity: 1)
  var total = 0
  scope {
    async produce(polled)
    loop (total < 6) {
      val v = polled.tryRecv()
      if (v != null) total += v else await sleep(Duration.zero)
    }
  }
  io.println("polled $total")
}
