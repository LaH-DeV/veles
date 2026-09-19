// Prelude — concurrency helpers (D2/D16/D35). The worker pool written once:
// a bounded number of tasks pulling work from a channel and reporting on
// another, inside a scope that joins them. Programs that need a stream, a
// pipeline or per-worker state write that machine themselves; these cover
// "do this to every element, at most n at a time".

/// Result of one job, tagged with its index so the output keeps the input order.
struct Indexed<R> {
  index: i64
  value: R
}

/// A pool worker: takes indexes until the channel closes, applies `f`, reports.
fun <T, R, E> poolWorker(items: List<T>, f: sendable fun(T): R suspends throws E, jobs: Channel<i64>, results: Channel<Indexed<R>>) throws E {
  loop {
    val i = await jobs.recv() ?: break
    val value = try f(items.atOrPanic(i))
    results.send(Indexed(index: i, value))
  }
}

/// Feeds every index, then closes the channel so the workers finish.
fun feedIndexes(n: i64, jobs: Channel<i64>) {
  loop (i in 0..<n) {
    jobs.send(i)
  }
  jobs.close()
}

extend<T: Sendable> List<T> {
  /// Calls `f` on every element with at most `workers` calls in flight and
  /// returns the results in the list's order. `f` runs in pool tasks, so it
  /// must be sendable: it may capture only vals of Sendable types (D35). A
  /// throwing `f` makes the whole call throw — the first error cancels the
  /// remaining work, as for the eager adapters (D46).
  public fun mapConcurrent<R: Sendable, E>(f: sendable fun(T): R suspends throws E, workers: i64 = 4): List<R> throws E {
    if (workers < 1) panic("mapConcurrent: workers must be at least 1, got $workers")
    val n = self.len()
    val slots: MutableList<R?> = MutableList<R?>.make(n, _ => null)
    if (n == 0) return []
    val jobs = Channel<i64>(capacity: workers)
    val results = Channel<Indexed<R>>(capacity: workers)
    results.closeAfter(n)
    scope {
      loop (_ in 1..workers.min(n)) {
        async poolWorker(self, f, jobs, results)
      }
      async feedIndexes(n, jobs)
      loop {
        val r = await results.recv() ?: break
        slots.set(r.index, r.value)
      }
    }
    slots.map(v => v ?: panic("mapConcurrent: a slot was never filled"))
  }

  /// `mapConcurrent` for its effects: runs `f` on every element, at most
  /// `workers` at a time, and returns when all have finished.
  public fun forEachConcurrent<E>(f: sendable fun(T) suspends throws E, workers: i64 = 4) throws E {
    val _ = try self.mapConcurrent(x => {
      try f(x)
      true
    }, workers: workers)
  }
}

/// Calls `f`; the named function `async` needs for a function value.
fun <R, E> invoke(f: sendable fun(): R suspends throws E): R throws E = try f()

/// `withTimeout` ran out of time.
public error Timeout {
  public millis: i64
  fun message(): string = "timed out after ${self.millis} ms"
}

/// Runs `f` with a time limit: its result, or a `Timeout` error when `ms`
/// milliseconds pass first. The task running `f` is then cancelled and
/// unwinds — its `with` cleanups run — before `withTimeout` throws; an
/// error `f` throws in time is rethrown, so the call throws `E | Timeout`.
///
/// ```veles
/// val line = try withTimeout(5000, () => try conn.readLine())
/// ```
public fun <R: Sendable, E> withTimeout(ms: i64, f: sendable fun(): R suspends throws E): R throws E | Timeout {
  scope {
    val t = async invoke(f)
    race {
      val r = await t => return try r
      // leaving the scope by a throw cancels `t` and waits for it to unwind
      sleep(ms)       => throw Timeout(millis: ms)
    }
  }
}
