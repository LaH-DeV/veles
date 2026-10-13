// Prelude — iteration (D42, D46). In scope in every file (D24).
//
// `loop (x in c)` desugars through Iterable; lazy adapters are default
// bodies on Iterator, so every iterator gets them for free (D53).

public trait Iterator {
  type Item
  fun next(): Item?

  fun map<U>(f: fun(Item): U): MapIter<Self, U> => MapIter(inner: this, f)
  fun filter(f: fun(Item): bool): FilterIter<Self> => FilterIter(inner: this, f)
  fun take(n: i64): TakeIter<Self> => TakeIter(inner: this, remaining: n)
  fun skip(n: i64): SkipIter<Self> => SkipIter(inner: this, remaining: n)
  fun enumerate(): EnumerateIter<Self> => EnumerateIter(inner: this, index: 0)
  fun zip<J: Iterator>(other: J): ZipIter<Self, J> => ZipIter(a: this, b: other)

  /// Calls `f` on the elements with up to `workers` calls under way, and
  /// gives the results in the elements' order as they are asked for (D156):
  /// lazy where `List.mapConcurrent` is eager, so a loop over it handles
  /// one result while the next ones are computed. The elements are taken
  /// from this iterator in the task that iterates; `f` runs in tasks the
  /// value holds, so a `loop` over it — or a `with` — owns them: when it
  /// ends, early or not, the calls still under way are cancelled and joined.
  /// The first error `f` throws fails the block that owns them.
  ///
  /// ```veles
  /// loop (page in urls.iter().mapConcurrent(u => try fetch(u), workers: 8)) {
  ///   save(page)
  /// }
  /// ```
  fun mapConcurrent<R, E>(f: sendable fun(Item): R suspends throws E, workers: i64 = 4): ConcurrentMap<Self, R, E> {
    if (workers < 1) panic("mapConcurrent: workers must be at least 1, got $workers")
    val jobs = Channel<Item>(capacity: workers)
    val results = Channel<R>(capacity: workers)
    ConcurrentMap(source: this, workers, jobs, results, runner: async runConcurrentMap(f, jobs, results))
  }

  fun toList(): List<Item> {
    var out: MutableList<Item> = []
    loop {
      val v = this.next()
      if (v == null) break
      out.push(v)
    }
    out.toList()
  }

  fun toSet(): Set<Item> => this.toList().toSet()

  fun count(): i64 {
    var n: i64 = 0
    loop {
      if (this.next() == null) break
      n += 1
    }
    n
  }

  fun fold<A>(init: A, f: fun(A, Item): A): A {
    var acc = init
    loop {
      val v = this.next()
      if (v == null) break
      acc = f(acc, v)
    }
    acc
  }

  fun forEach(f: fun(Item)) {
    loop {
      val v = this.next()
      if (v == null) break
      f(v)
    }
  }

  fun any(f: fun(Item): bool): bool {
    loop {
      val v = this.next()
      if (v == null) return false
      if (f(v)) return true
    }
  }

  fun all(f: fun(Item): bool): bool {
    loop {
      val v = this.next()
      if (v == null) return true
      if (!f(v)) return false
    }
  }

  fun find(f: fun(Item): bool): Item? {
    loop {
      val v = this.next()
      if (v == null) return null
      if (f(v)) return v
    }
  }

  fun last(): Item? {
    var found: Item? = null
    loop {
      val v = this.next()
      if (v == null) break
      found = v
    }
    found
  }
}

public trait Iterable {
  type Iter: Iterator
  fun iterator(): Iter
  // `xs.iter()` reads better at the head of a pipeline; same thing.
  fun iter(): Iter => this.iterator()
}

/// An iterator whose `next` may wait (D42): `loop (x in it)` drives it as
/// it drives an Iterator, and the function the loop is in suspends.
public trait AsyncIterator {
  type Item
  fun next(): Item? suspends
}

// ---------------------------------------------------------------------------
// adapters

/// What `iter.mapConcurrent(f, workers:)` gives (D156). It holds the task
/// running the calls, so it is received by a `loop` or a `with`.
public struct ConcurrentMap<I: Iterator, R, E> {
  source:  I
  workers: i64
  jobs:    Channel<I.Item>
  results: Channel<R>
  runner:  Task<Result<(), E>>
  /// elements handed over whose results have not been taken yet
  var sent: i64 = 0
  /// the source has elements left
  var open: bool = true

  implement AsyncIterator {
    type Item = R
    fun next(): R? suspends {
      // keep `workers` calls under way: the elements are taken here, in
      // the task that iterates, so the source need not cross a task
      loop (this.open && this.sent < this.workers) {
        val v = this.source.next()
        if (v == null) {
          this.jobs.close()
          this.open = false
        } else {
          this.jobs.send(v)
          this.sent += 1
        }
      }
      if (this.sent == 0) return null
      this.sent -= 1
      await this.results.recv()
    }
  }
}

/// The task behind a `ConcurrentMap`: a task of `f` per element handed
/// over on `jobs`, started as it arrives, and their results sent on
/// `results` in the order the elements came.
fun runConcurrentMap<T, R, E>(f: sendable fun(T): R suspends throws E, jobs: Channel<T>, results: Channel<R>) throws E {
  scope {
    // oldest first; the iterating task keeps at most `workers` under way,
    // so taking the first off is cheap
    val running: MutableList<Task<R>> = []
    var open = true
    loop (open || !running.isEmpty()) {
      if (running.isEmpty()) {
        val job = await jobs.recv() ?: break
        running.push(async f(job))
        continue
      }
      val first = running.at(0) ?: break
      if (open) {
        // the oldest call's value, or a new element to start meanwhile
        race {
          val v = await first   => {
            val _ = running.removeAt(0)
            results.send(v)
          }
          val job = jobs.recv() => {
            if (job == null) open = false else running.push(async f(job))
          }
        }
      } else {
        val _ = running.removeAt(0)
        results.send(await first)
      }
    }
  }
  results.close()
}

public struct MapIter<I: Iterator, U> {
  inner: I
  f:     fun(I.Item): U

  implement Iterator {
    type Item = U
    fun next(): U? {
      val v = this.inner.next()
      if (v == null) return null
      this.f(v)
    }
  }
}

public struct FilterIter<I: Iterator> {
  inner: I
  f:     fun(I.Item): bool

  implement Iterator {
    type Item = I.Item
    fun next(): I.Item? {
      loop {
        val v = this.inner.next()
        if (v == null) return null
        if (this.f(v)) return v
      }
    }
  }
}

public struct TakeIter<I: Iterator> {
  inner:         I
  var remaining: i64

  implement Iterator {
    type Item = I.Item
    fun next(): I.Item? {
      if (this.remaining <= 0) return null
      this.remaining -= 1
      this.inner.next()
    }
  }
}

public struct SkipIter<I: Iterator> {
  inner:         I
  var remaining: i64

  implement Iterator {
    type Item = I.Item
    fun next(): I.Item? {
      loop (this.remaining > 0) {
        this.remaining -= 1
        if (this.inner.next() == null) return null
      }
      this.inner.next()
    }
  }
}

public struct EnumerateIter<I: Iterator> {
  inner:     I
  var index: i64

  implement Iterator {
    type Item = (i64, I.Item)
    fun next(): (i64, I.Item)? {
      val v = this.inner.next()
      if (v == null) return null
      val i = this.index
      this.index += 1
      (i, v)
    }
  }
}

public struct ZipIter<A: Iterator, B: Iterator> {
  a: A
  b: B

  implement Iterator {
    type Item = (A.Item, B.Item)
    fun next(): (A.Item, B.Item)? {
      val x = this.a.next()
      if (x == null) return null
      val y = this.b.next()
      if (y == null) return null
      (x, y)
    }
  }
}

// ---------------------------------------------------------------------------
// iterators over the builtin collections

public struct ListIter<T> {
  list:      List<T>
  var index: i64 = 0

  implement Iterator {
    type Item = T
    fun next(): T? {
      val v = this.list.at(this.index) ?: return null
      this.index += 1
      v
    }
  }
}

implement<T> Iterable for List<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> => ListIter(list: this)
}

implement<T> Iterable for MutableList<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> => ListIter(list: this.toList())
}

public struct RangeIter<T> {
  var current: T
  hi:          T
  inclusive:   bool
  // an inclusive range ends by this flag on its last value, not by
  // stepping past it: past the type's maximum, `current` would wrap
  var done: bool = false

  implement Iterator {
    type Item = T
    fun next(): T? {
      if (this.done) return null
      if (this.inclusive) {
        if (this.current > this.hi) return null
        if (this.current == this.hi) this.done = true
      } else {
        if (this.current >= this.hi) return null
      }
      val v = this.current
      this.current = this.current +% 1
      v
    }
  }
}

implement<T> Iterable for Range<T> {
  type Iter = RangeIter<T>
  fun iterator(): RangeIter<T> => RangeIter(current: this.lo, hi: this.hi, inclusive: this.inclusive)
}
