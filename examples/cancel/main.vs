// Cancellation reaches through every join (D34, D43). A task parked where
// its `scope` or `gather` waits for children is cancelled by cancelling
// those children first, waiting for them to unwind — each closing what
// it holds, innermost first — and only then unwinding itself. Nothing
// outlives the block that launched it, and nothing it opened stays open.
use io { println }
use time

// `steps` tells main when a child has opened what it holds and when a
// child of a gather has closed it, so main cancels at the right moment
// however long a task takes to start
struct Held {
  name:  string
  steps: Channel<bool>
  implement Closeable {
    fun close() {
      println("closed: ${this.name}")
      if (this.name.contains("gather")) {
        val _ = this.steps.trySend(true)  // a close cannot wait; the channel has room
      }
    }
  }
}

fun slow(name: string, steps: Channel<bool>): i64 {
  with (held = Held(name, steps)) {
    steps.send(true)
    await sleep(Duration.seconds(30))
  }
  1
}

// parked at the `gather`'s join
fun viaGather(steps: Channel<bool>): i64 {
  val out = gather {
    async slow("a child of a gather", steps)
  }
  out ?? -1
}

// parked at the `scope`'s join, holding a resource of its own around it.
// Its two children are cancelled together and unwind in parallel, so they
// may close in either order; both close before their owner does.
fun viaScope(steps: Channel<bool>) {
  with held = Held(name: "the scope's owner", steps)
  scope {
    async slow("a child of a scope", steps)
    async slow("a child of a scope", steps)
  }
}

fun twoLevels(steps: Channel<bool>) {
  scope {
    async viaScope(steps)
  }
}

fun main() {
  val sw = time.Stopwatch.start()
  val steps = Channel<bool>(capacity: 8)
  scope {
    val g = async viaGather(steps)
    val s = async twoLevels(steps)
    loop (_ in 0..<3) {
      val _ = await steps.recv()  // the three children hold their resources
    }
    println("--- cancel the gather's owner")
    g.cancel()
    val _ = await steps.recv()  // the gather's child has closed
    println("--- cancel two levels above the scope")
    s.cancel()
  }
  // a child left running would have held the scope for 30 seconds
  println("done, nothing waited out: ${sw.elapsed() < Duration.seconds(5)}")
}
