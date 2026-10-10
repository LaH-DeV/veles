// Prelude — telling tasks that something happened (D146): Event, Watch and
// Broadcast. Each keeps its state behind a Mutex and wakes waiters through a
// signal channel that is closed when there is news, and replaced by a fresh
// one for the next; a closed channel releases every task receiving from it,
// so a waiter is one channel wait and cancellable like any.
//
// Every waiting operation `m` (`wait`, `changed`, `recv`) is written as two
// non-public halves the compiler also uses in a `race` arm (sema
// raceLowering): `mSignal()` — under the lock, a channel that is ready now
// when there is something to take, otherwise the live signal — and
// `mTake()`, which takes it. A closed signal always leaves something to
// take, so a woken waiter never comes back empty.

// a channel that is ready at once: news is already there
fun readySignal(): Channel<bool> {
  val c = Channel<bool>(capacity: 1)
  c.close()
  c
}

fun newSignal(): Channel<bool> => Channel<bool>(capacity: 1)

/// A flag tasks wait for (D146): `set()` releases every waiting task and
/// lets every later `wait` through, until `reset()`. Copies of an `Event`
/// are the same event.
///
/// ```veles
/// val ready = Event()
/// // in the tasks that need it:
/// await ready.wait()
/// // in the one that provides it:
/// ready.set()
/// ```
public struct Event {
  private state: Mutex<EventState> = Mutex(value: EventState(set: false, signal: newSignal()))

  /// Sets the event: every task waiting goes on, and so does every `wait`
  /// until `reset()`. Setting a set event does nothing.
  public fun set() {
    with s = this.state.lock()
    if (!s.set) {
      s.set = true
      s.signal.close()
    }
  }

  /// Clears the event: a `wait` from now on waits for the next `set()`.
  public fun reset() {
    with s = this.state.lock()
    if (s.set) {
      s.set = false
      s.signal = newSignal()
    }
  }

  /// Whether the event is set at this moment.
  public fun isSet(): bool => this.state.get().set

  /// Waits until the event is set; returns at once when it already is. It
  /// always suspends, so it is awaited: `await ready.wait()`; a cancelled
  /// task stops waiting, and it can be a `race` arm.
  public fun wait() {
    val signal = this.waitSignal()
    val _ = await signal.recv()
    this.waitTake()
  }

  fun waitSignal(): Channel<bool> => this.state.get().signal

  fun waitTake() { }
}

struct EventState {
  var set:    bool
  var signal: Channel<bool>
}

/// The latest value of something that changes — a configuration, a
/// status — which tasks read when they like and wait on for a change
/// (D146). Copies of a `Watch` share the value; each copy remembers the
/// version it last saw, so a task holding its own copy (a parameter) waits
/// for changes it has not seen.
///
/// ```veles
/// val settings = Watch(value: initial)
/// settings.set(next)              // anywhere
/// // in a task that follows it:
/// loop {
///   await settings.changed()
///   apply(settings.get())
/// }
/// ```
public struct Watch<T> {
  private state:    Mutex<WatchState<T>>
  private var seen: i64 = 0

  init(value: T) {
    this.state = Mutex(value: WatchState(value, version: 0, signal: newSignal()))
  }

  /// Replaces the value; every task waiting in `changed()` goes on.
  public fun set(value: T) {
    with s = this.state.lock()
    s.value = value
    s.version += 1
    s.signal.close()
    s.signal = newSignal()
  }

  /// The latest value. A later `changed()` on this copy waits for a `set`
  /// after this read.
  public fun get(): T {
    with s = this.state.lock()
    this.seen = s.version
    s.value
  }

  /// Waits until the value has been set since this copy's last `get()` or
  /// `changed()` — at once when it already has. It always suspends, so it
  /// is awaited: `await settings.changed()`; a cancelled task stops waiting,
  /// and it can be a `race` arm.
  public fun changed() {
    val signal = this.changedSignal()
    val _ = await signal.recv()
    this.changedTake()
  }

  fun changedSignal(): Channel<bool> {
    with s = this.state.lock()
    if (s.version != this.seen) return readySignal()
    s.signal
  }

  fun changedTake() {
    with s = this.state.lock()
    this.seen = s.version
  }
}

struct WatchState<T> {
  var value:   T
  var version: i64
  var signal:  Channel<bool>
}

/// A subscriber fell behind a `Broadcast` by more than its capacity:
/// `missed` values were dropped before it read them, and its next `recv`
/// goes on from the oldest value still kept (D146).
public error Lagged {
  public missed: i64
}

/// One sender, many receivers, every receiver gets every value (D146): a
/// news feed, a shutdown notice, cache invalidations. `send` never waits;
/// each `subscribe()` returns a `Subscription` that receives what is sent
/// after it subscribed. The last `capacity` values are kept: a subscriber
/// further behind than that loses the oldest, and is told — its next `recv`
/// throws `Lagged`. Copies of a `Broadcast` are the same broadcast.
///
/// ```veles
/// val news = Broadcast<Article>(capacity: 64)
/// with sub = news.subscribe()
/// news.send(article)
/// val a = try await sub.recv()    // the article; null once news is closed and drained
/// ```
public struct Broadcast<T> {
  private state: Mutex<BroadcastState<T>>

  /// Panics when `capacity` is less than 1.
  init(capacity: i64) {
    if (capacity < 1) panic("Broadcast: capacity must be at least 1, got $capacity")
    this.state = Mutex(value: BroadcastState(kept: MutableList<T>(), capacity, next: 0, closed: false, signal: newSignal()))
  }

  /// Sends `value` to every subscriber; never waits. Panics after `close()`.
  public fun send(value: T) {
    with s = this.state.lock()
    if (s.closed) panic("Broadcast: send after close")
    if (s.kept.len() < s.capacity) {
      s.kept.push(value)
    } else {
      s.kept.set(s.next % s.capacity, value)
    }
    s.next += 1
    s.signal.close()
    s.signal = newSignal()
  }

  /// A subscription that receives every value sent from now on.
  public fun subscribe(): Subscription<T> {
    with s = this.state.lock()
    Subscription(state: this.state, cursor: &Cursor(at: s.next, closed: false))
  }

  /// Ends the broadcast: each subscriber receives what it has not yet read,
  /// then `null`. Closing it again does nothing.
  public fun close() {
    with s = this.state.lock()
    if (!s.closed) {
      s.closed = true
      s.signal.close()
    }
  }
}

struct BroadcastState<T> {
  // the last `capacity` values: the value numbered n (from 0, in the
  // order sent) is in slot n % capacity while it is kept
  kept:       MutableList<T>
  capacity:   i64
  var next:   i64  // the number of the next value sent
  var closed: bool
  var signal: Channel<bool>
}

struct Cursor {
  var at:     i64  // the number of the next value this subscriber reads
  var closed: bool
}

/// A `Broadcast`'s receiving end (D146), from `subscribe()`. Closing it
/// stops it: its `recv` returns `null` from then on.
public struct Subscription<T> {
  private state:  Mutex<BroadcastState<T>>
  private cursor: *Cursor

  /// The next value sent after this subscription began, waiting for one;
  /// `null` once the broadcast is closed and every value read. When this
  /// subscriber fell more than the capacity behind, it throws `Lagged` once
  /// and goes on from the oldest value kept. It always suspends, so it is
  /// awaited: `try await sub.recv()`; a cancelled task stops waiting, and it
  /// can be a `race` arm.
  public fun recv(): T? throws Lagged {
    val signal = this.recvSignal()
    val _ = await signal.recv()
    try this.recvTake()
  }

  fun recvSignal(): Channel<bool> {
    with s = this.state.lock()
    if (this.cursor.closed || this.cursor.at < s.next || s.closed) return readySignal()
    s.signal
  }

  fun recvTake(): T? throws Lagged {
    with s = this.state.lock()
    if (this.cursor.closed) return null
    val oldest = s.next - s.kept.len()
    if (this.cursor.at < oldest) {
      val missed = oldest - this.cursor.at
      this.cursor.at = oldest
      throw Lagged(missed)
    }
    if (this.cursor.at < s.next) {
      val v = s.kept.at(this.cursor.at % s.capacity) ?: panic("Broadcast: a kept value is missing")
      this.cursor.at += 1
      return v
    }
    null
  }

  implement Closeable {
    fun close() {
      this.cursor.closed = true
    }
  }
}
