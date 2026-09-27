// Cancellation reaches through every join (D34, D43). A task parked where
// its `scope` or `gather` waits for children is cancelled by cancelling
// those children first, waiting for them to unwind — each closing what
// it holds, innermost first — and only then unwinding itself. Nothing
// outlives the block that launched it, and nothing it opened stays open.
use io, time

struct Held {
  name: string
  implement Closeable {
    fun close() {
      io.println("closed: ${this.name}")
    }
  }
}

fun slow(name: string): i64 {
  with (held = Held(name)) {
    await sleep(Duration.seconds(30))
  }
  1
}

// parked at the `gather`'s join
fun viaGather(): i64 {
  val out = (gather {
    async slow("a child of a gather")
  }).0
  out ?? -1
}

// parked at the `scope`'s join, holding a resource of its own around it.
// Its two children are cancelled together and unwind in parallel, so they
// may close in either order; both close before their owner does.
fun viaScope() {
  with (held = Held(name: "the scope's owner")) {
    scope {
      async slow("a child of a scope")
      async slow("a child of a scope")
    }
  }
}

fun twoLevels() {
  scope {
    async viaScope()
  }
}

fun main() {
  val sw = time.Stopwatch.start()
  scope {
    val g = async viaGather()
    val s = async twoLevels()
    await sleep(Duration.millis(20))
    io.println("--- cancel the gather's owner")
    g.cancel()
    await sleep(Duration.millis(20))
    io.println("--- cancel two levels above the scope")
    s.cancel()
  }
  // a child left running would have held the scope for 30 seconds
  io.println("done, nothing waited out: ${sw.elapsed() < Duration.seconds(5)}")
}
