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
fun poolWorker<T, R, E>(items: List<T>, f: sendable fun(T): R suspends throws E, jobs: Channel<i64>, results: Channel<Indexed<R>>) throws E {
  loop {
    val i = await jobs.recv() ?: break
    val item = items.at(i) ?: panic("mapConcurrent: feedIndexes sends only 0..<items.len()")
    val value = try f(item)
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
    val n = this.len()
    val slots: MutableList<R?> = MutableList<R?>.make(n, _ => null)
    if (n == 0) return []
    val jobs = Channel<i64>(capacity: workers)
    val results = Channel<Indexed<R>>(capacity: workers)
    results.closeAfter(n)
    scope {
      loop (_ in 1..workers.min(n)) {
        async poolWorker(this, f, jobs, results)
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
    val _ = try this.mapConcurrent(x => {
      try f(x)
      true
    }, workers: workers)
  }
}

/// `withTimeout` ran out of time.
public error Timeout {
  /// The limit that was reached — not how long the call actually took.
  public limit: Duration
  fun message(): string => "timed out after ${this.limit}"
}

/// Runs `f` with a time limit: its result, or a `Timeout` error when
/// `limit` passes first. The task running `f` is then cancelled and
/// unwinds — its `with` cleanups run — before `withTimeout` throws; an
/// error `f` throws in time is rethrown, so the call throws `E | Timeout`.
///
/// ```veles
/// val line = try withTimeout(Duration.seconds(5), () => try conn.readLine(max: 8192))
/// ```
public fun withTimeout<R: Sendable, E>(limit: Duration, f: sendable fun(): R suspends throws E): R throws E | Timeout {
  scope {
    val t = async f()
    race {
      // an error of `f` is the scope's: it fails the scope, never this arm (D141)
      val r = await t => return r
      // leaving the scope by a throw cancels `t` and waits for it to unwind
      sleep(limit)    => throw Timeout(limit)
    }
  }
}

/// Calls `f` until it returns, and returns what it returns. After a thrown
/// error it waits `delay` and tries again, up to `times` calls in all; then
/// it throws the last error. A panic is not retried, and a cancelled task
/// stops at the wait. Panics when `times` is less than 1 (D110).
///
/// ```veles
/// val body = try retry(3, () => try http.get(url), delay: Duration.millis(200))
/// ```
public fun retry<R, E>(times: i64, f: fun(): R suspends throws E, delay: Duration = Duration.zero): R throws E {
  if (times < 1) panic("retry: times must be at least 1, got $times")
  var left = times
  loop {
    do {
      return try f()
    } catch (e) {
      left -= 1
      if (left == 0) throw e
    }
    await sleep(delay)
  }
}

/// At most `permits` holders at a time: a limit on connections, open
/// files or requests in flight that tasks share. `acquire` waits for a
/// permit; the `Permit` it returns gives it back when closed, so it is
/// held with `with` (D109, D110):
///
/// ```veles
/// val db = Semaphore(permits: 8)
/// // in each task:
/// with db.acquire()
/// try query(conn, sql)
/// ```
public struct Semaphore {
  private free: Channel<bool>

  /// Panics when `permits` is less than 1.

  init(permits: i64) {
    if (permits < 1) panic("Semaphore: permits must be at least 1, got $permits")
    this.free = Channel<bool>(capacity: permits)
    loop (_ in 0..<permits) {
      val _ = this.free.trySend(true)
    }
  }

  /// A permit, waiting until one is free; a cancelled task stops waiting.
  public fun acquire(): Permit {
    val _ = await this.free.recv()
    Permit(free: this.free)
  }

  /// A permit if one is free now, or `null`; never waits.
  public fun tryAcquire(): Permit? {
    val _ = this.free.tryRecv() ?: return null
    Permit(free: this.free)
  }

  /// How many permits are free at this moment — by the time the caller
  /// looks, another task may have taken one.
  public fun available(): i64 => this.free.len()
}

/// One of a `Semaphore`'s permits; closing it gives it back. A second
/// close does nothing.
public struct Permit {
  private free:     Channel<bool>
  private returned: Atomic<bool> = Atomic(value: false)

  implement Closeable {
    fun close() {
      if (!this.returned.swap(true)) {
        val _ = this.free.trySend(true)
      }
    }
  }
}

extend<T> Channel<T> {
  /// Receives until the channel is closed and drained, calling `f` on each
  /// value; an error from `f` stops it and is thrown (D110).
  public fun forEach<E>(f: fun(T) suspends throws E) throws E {
    loop {
      val v = await this.recv() ?: break
      try f(v)
    }
  }

  /// Every value until the channel is closed and drained, in order.
  public fun toList(): List<T> {
    val out = MutableList<T>()
    loop {
      val v = await this.recv() ?: break
      out.push(v)
    }
    out.toList()
  }
}
