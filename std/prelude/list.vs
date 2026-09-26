// Prelude — List, MutableList and Range methods (D25, D29, D46). In scope
// in every file (D24).
//
// The compiler provides the primitives (`len`, `at`, `set`, `push`, `pop`,
// `sorted`, the eager `map`/`filter`/`fold`, ...); the rest is Veles.

extend<T> List<T> {
  /// The element at `i`, or `d` when `i` is out of range; `this.at(i) ?: d`.
  public fun atOrDefault(i: i64, d: T): T = this.at(i) ?: d

  /// The first `n` elements (all of them when `n` exceeds the length).
  public fun take(n: i64): List<T> = this.slice(0, n)

  /// The elements after the first `n`.
  public fun drop(n: i64): List<T> = this.slice(n, this.len())

  /// A new list of these elements followed by `other`'s.
  public fun concat(other: List<T>): List<T> {
    var out: MutableList<T> = []
    out.addAll(this)
    out.addAll(other)
    out.toList()
  }

  /// The valid indexes, `0..<len()`. In `loop (i in xs.indices())` the
  /// checker knows each `i` is in range, so `xs.at(i)` is a `T` (D62).
  public fun indices(): Range<i64> = 0..<this.len()

  /// The elements in `from..<to`, clamped to the list; empty when `from >= to`.
  public fun slice(from: i64, to: i64): List<T> {
    val lo = from.max(0)
    val hi = to.min(this.len())
    var out: MutableList<T> = []
    loop (i in lo..<hi) {
      out.push(this.at(i) ?: panic("slice: lo..<hi was clamped to the list"))
    }
    out.toList()
  }

  /// The index of the first element `pred` accepts, or -1.
  public fun indexOfFirst(pred: fun(T): bool): i64 {
    var i: i64 = 0
    loop (x in this) {
      if (pred(x)) return i
      i += 1
    }
    -1
  }

  /// The number of elements `pred` accepts.
  public fun count(pred: fun(T): bool): i64 {
    var n: i64 = 0
    loop (x in this) {
      if (pred(x)) n += 1
    }
    n
  }

  /// Each element with its index: `["a", "b"].enumerate()` is
  /// `[(0, "a"), (1, "b")]`. Eager, like `map`; `xs.iter().enumerate()` is
  /// the lazy form.
  public fun enumerate(): List<(i64, T)> {
    var out: MutableList<(i64, T)> = []
    var i: i64 = 0
    loop (x in this) {
      out.push((i, x))
      i += 1
    }
    out
  }

  /// Pairs of elements at the same index, as long as the shorter list.
  public fun zip<U>(other: List<U>): List<(T, U)> {
    var out: MutableList<(T, U)> = []
    var i: i64 = 0
    loop (a in this) {
      val b = other.at(i) ?: break
      out.push((a, b))
      i += 1
    }
    out.toList()
  }

  /// The lists `f` returns for each element, concatenated.
  public fun flatMap<U>(f: fun(T): List<U>): List<U> {
    var out: MutableList<U> = []
    loop (x in this) {
      loop (y in f(x)) {
        out.push(y)
      }
    }
    out.toList()
  }

  /// The elements with duplicates removed, first occurrences kept in order.
  public fun distinct(): List<T> {
    var seen = MutableSet<T>()
    var out: MutableList<T> = []
    loop (x in this) {
      if (seen.add(x)) out.push(x)
    }
    out.toList()
  }

  /// Consecutive pieces of `n` elements; the last may be shorter.
  public fun chunked(n: i64): List<List<T>> {
    if (n <= 0) panic("chunked: size must be positive, got $n")
    var out: MutableList<List<T>> = []
    var i: i64 = 0
    loop (i < this.len()) {
      out.push(this.slice(i, i + n))
      i += n
    }
    out.toList()
  }

  /// Every window of `n` consecutive elements (none when the list is shorter).
  public fun windowed(n: i64): List<List<T>> {
    if (n <= 0) panic("windowed: size must be positive, got $n")
    var out: MutableList<List<T>> = []
    loop (i in 0..(this.len() - n)) {
      out.push(this.slice(i, i + n))
    }
    out.toList()
  }
}

// Comparison strategies (D48): the natural order comes from Comparable;
extend<T> List<T> {
  /// The index of the last element, -1 when empty: `if (i < xs.lastIndex()) i += 1`.
  public fun lastIndex(): i64 = this.len() - 1

  /// The distinct elements as an immutable set: `xs.map(t => t.0).toSet()`.
  public fun toSet(): Set<T> = this.toMutableSet().toSet()

  /// The distinct elements as a mutable set.
  public fun toMutableSet(): MutableSet<T> {
    val out = MutableSet<T>()
    loop (x in this) out.add(x)
    out
  }
}

// any other order is a comparator (`sortedWith`, `minWith`) or a key
// (`sortedBy`, `minBy`, `distinctBy`) passed at the call.
extend<T> List<T> {
  /// A copy sorted by `compare`, which returns `Ordering.Less` when its
  /// first argument sorts first: `xs.sortedWith((a, b) => b.compareTo(a))`
  /// is descending. Stable (equal elements keep their order); O(n log n).
  public fun sortedWith(compare: fun(T, T): Ordering): List<T> {
    val n = this.len()
    var src = this.toMutable()
    if (n < 2) return src.toList()
    var dst: MutableList<T> = this.toMutable()
    // bottom-up merge sort: runs of `width` merged into `dst`, then swapped
    var width: i64 = 1
    loop (width < n) {
      var lo: i64 = 0
      loop (lo < n) {
        val mid = if (lo + width < n) lo + width else n
        val hi = if (lo + 2 * width < n) lo + 2 * width else n
        var i = lo
        var j = mid
        var k = lo
        loop (k < hi) {
          if (i < mid && (j >= hi || compare(src.at(i) ?: panic("sortedWith: i < mid <= n"), src.at(j) ?: panic("sortedWith: j < hi <= n")) <= 0)) {
            dst.set(k, src.at(i) ?: panic("sortedWith: i < mid <= n"))
            i += 1
          } else {
            dst.set(k, src.at(j) ?: panic("sortedWith: j < hi <= n"))
            j += 1
          }
          k += 1
        }
        lo += 2 * width
      }
      (src, dst) = (dst, src)
      width *= 2
    }
    src.toList()
  }

  /// A copy sorted from largest to smallest key.
  public fun sortedByDescending<K: Comparable>(key: fun(T): K): List<T> =
    this.sortedWith((a, b) => key(b).compareTo(key(a)))

  /// The element with the smallest key, or `null` when empty; the first
  /// of several equal keys.
  public fun minBy<K: Comparable>(key: fun(T): K): T? {
    var best = this.first() ?: return null
    var bestKey = key(best)
    loop (x in this.drop(1)) {
      val k = key(x)
      if (k < bestKey) {
        best = x
        bestKey = k
      }
    }
    best
  }

  /// The element with the largest key, or `null` when empty.
  public fun maxBy<K: Comparable>(key: fun(T): K): T? {
    var best = this.first() ?: return null
    var bestKey = key(best)
    loop (x in this.drop(1)) {
      val k = key(x)
      if (k > bestKey) {
        best = x
        bestKey = k
      }
    }
    best
  }

  /// The element that `compare` places first, or `null` when empty.
  public fun minWith(compare: fun(T, T): Ordering): T? {
    var best = this.first() ?: return null
    loop (x in this.drop(1)) {
      if (compare(x, best) < 0) best = x
    }
    best
  }

  /// The element that `compare` places last, or `null` when empty.
  public fun maxWith(compare: fun(T, T): Ordering): T? {
    var best = this.first() ?: return null
    loop (x in this.drop(1)) {
      if (compare(x, best) > 0) best = x
    }
    best
  }

  /// The elements whose keys are distinct, first occurrence kept:
  /// `names.distinctBy(n => n.toLower())`.
  public fun distinctBy<K>(key: fun(T): K): List<T> {
    var seen = MutableSet<K>()
    var out: MutableList<T> = []
    loop (x in this) {
      if (seen.add(key(x))) out.push(x)
    }
    out.toList()
  }
}

extend<T> MutableList<T> {
  /// Sorts in place by `compare` (see `sortedWith`).
  public fun sortWith(compare: fun(T, T): Ordering) {
    var i: i64 = 0
    loop (x in this.sortedWith(compare)) {
      this.set(i, x)
      i += 1
    }
  }
}

// Ordering needs Comparable: numbers and strings implement it in the
// prelude, a struct by `implement Comparable`. The bound is checked at the call
// site, so a list of anything else reports the error there.
extend<T: Comparable> List<T> {
  /// A copy sorted from largest to smallest.
  public fun sortedDescending(): List<T> = this.sorted().reversed()

  /// The smallest element, or `null` when empty.
  public fun min(): T? {
    if (this.isEmpty()) return null
    var best = this.at(0)
    loop (x in this) {
      if (x < best) best = x
    }
    best
  }

  /// The largest element, or `null` when empty.
  public fun max(): T? {
    if (this.isEmpty()) return null
    var best = this.at(0)
    loop (x in this) {
      if (x > best) best = x
    }
    best
  }
}

// Binary search (D48). Everything here needs the list to be *in order* —
// sorted by the same order the search uses — and none of it checks, because
// checking is the linear scan the search exists to avoid. On a list that is
// not in order the answer is simply wrong; it is never a panic and never a
// read out of range.
//
// All five are one loop, `partitionPoint`, asked five ways: the list is a
// run of elements before the answer and a run from the answer on, and every
// step halves whichever run the middle belongs to. `lo + (hi - lo) / 2`
// rather than `(lo + hi) / 2`, so a list long enough to overflow the sum
// would still work (the classic bug, and free to avoid).
extend<T> List<T> {
  /// The first index where `pred` stops being true, or the length when it
  /// never does. The list must be *partitioned* by `pred`: every element it
  /// accepts before every element it does not.
  ///
  /// This is the general form — a line table looks up the line containing a
  /// byte offset with `starts.partitionPoint(s => s <= offset) - 1` — and
  /// the one to reach for when the question is not "where is this element"
  /// but "where does the run end".
  public fun partitionPoint(pred: fun(T): bool): i64 {
    var lo: i64 = 0
    var hi = this.len()
    loop (lo < hi) {
      val mid = lo + (hi - lo) / 2
      val x = this.at(mid) ?: panic("partitionPoint: lo <= mid < hi <= len")
      if (pred(x)) lo = mid + 1 else hi = mid
    }
    lo
  }

  /// The index of the first element `compare` calls `Equal`, or -1 — the
  /// same answer shape as `indexOf`, in log n comparisons instead of n.
  ///
  /// `compare(element)` answers where the element sits *relative to what is
  /// being looked for*: `Less` when it sorts before it, `Greater` when
  /// after. So a search of a list ordered by `id` is
  /// `rows.binarySearchWith(r => r.id.compareTo(wanted))`, and a descending
  /// list flips the arguments: `xs.binarySearchWith(x => wanted.compareTo(x))`.
  public fun binarySearchWith(compare: fun(T): Ordering): i64 {
    val at = this.partitionPoint(x => compare(x) < 0)
    val found = this.at(at) ?: return -1  // at == len(): nothing is Equal
    if (compare(found) == Ordering.Equal) at else -1
  }

  /// The index of the first element whose key equals `target`, or -1:
  /// `people.binarySearchBy(p => p.name, "ann")` on a list sorted by name.
  public fun binarySearchBy<K: Comparable>(key: fun(T): K, target: K): i64 =
    this.binarySearchWith(x => key(x).compareTo(target))
}

extend<T: Comparable> List<T> {
  /// The index of the first element equal to `x`, or -1. The *first*, not
  /// any: a list with duplicates gives the same answer every time.
  public fun binarySearch(x: T): i64 {
    val at = this.lowerBound(x)
    val found = this.at(at) ?: return -1  // at == len(): x belongs at the end
    if (found.compareTo(x) == Ordering.Equal) at else -1
  }

  /// The first index whose element is not less than `x` — where `x` belongs
  /// if it is inserted, keeping the order and going before any equals.
  /// `[10, 20, 20, 30].lowerBound(20)` is 1.
  public fun lowerBound(x: T): i64 = this.partitionPoint(e => e < x)

  /// The first index whose element is greater than `x` — where `x` belongs
  /// if it goes after any equals. `[10, 20, 20, 30].upperBound(20)` is 3, so
  /// `upperBound(x) - lowerBound(x)` is how many times `x` occurs.
  public fun upperBound(x: T): i64 = this.partitionPoint(e => e <= x)
}

extend List<i64> {
  /// The sum of the elements (0 for an empty list).
  public fun sum(): i64 {
    var total: i64 = 0
    loop (x in this) {
      total += x
    }
    total
  }
}

extend List<f64> {
  /// The sum of the elements (0.0 for an empty list).
  public fun sum(): f64 {
    var total = 0.0
    loop (x in this) {
      total += x
    }
    total
  }
}

// `repeat` and `fill` copy one value into every slot, so the value must be
// one that copies cleanly: a mutable list, a pointer or a closure would be
// shared by all the slots instead (D35 Sendable), and `make` is the way to
// build a fresh one per slot.
extend<T: Sendable> MutableList<T> {
  /// A list of `count` copies of `x`: `MutableList<bool>.repeat(false, n)`.
  public static fun repeat(x: T, count: i64): MutableList<T> {
    var out: MutableList<T> = []
    loop (_ in 0..<count) out.push(x)
    out
  }

  /// Overwrites every element with `x`; the length does not change.
  public fun fill(x: T) {
    loop (i in 0..<this.len()) this.set(i, x)
  }
}

extend<T> MutableList<T> {
  /// A list of `n` elements where slot `i` holds `init(i)`:
  /// `MutableList<MutableList<i64>>.make(n, _ => [])`.
  public static fun make(n: i64, init: fun(i64): T): MutableList<T> {
    var out: MutableList<T> = []
    loop (i in 0..<n) out.push(init(i))
    out
  }

  /// Exchanges the elements at `i` and `j`.
  public fun swap(i: i64, j: i64) {
    val a = this.at(i) ?: panic("swap: index $i out of bounds for list of length ${this.len()}")
    val b = this.at(j) ?: panic("swap: index $j out of bounds for list of length ${this.len()}")
    this.set(i, b)
    this.set(j, a)
  }

  /// Inserts `x` at index `i`, shifting the rest up; `i == len()` appends.
  public fun insert(i: i64, x: T) {
    if (i < 0 || i > this.len()) panic("insert: index $i out of bounds for list of length ${this.len()}")
    this.push(x)
    var j = this.len() - 1
    loop (j > i) {
      this.set(j, this.at(j - 1) ?: panic("insert: i < j < len"))
      j -= 1
    }
    this.set(i, x)
  }

  /// Removes and returns the element at index `i`, shifting the rest down.
  public fun removeAt(i: i64): T {
    val removed = this.at(i) ?: panic("removeAt: index $i out of bounds for list of length ${this.len()}")
    loop (j in i..<(this.len() - 1)) {
      this.set(j, this.at(j + 1) ?: panic("removeAt: j + 1 < len"))
    }
    this.pop()
    removed
  }

  /// Appends every element of `xs`.
  public fun addAll(xs: List<T>) {
    loop (x in xs) {
      this.push(x)
    }
  }

  /// Sorts in place (elements must be Comparable).
  public fun sort() {
    var i: i64 = 0
    loop (x in this.sorted()) {
      this.set(i, x)
      i += 1
    }
  }
}

/// Steps through a range by a fixed increment, in either direction.
public struct RangeStepIter<T> {
  var current: T
  last:        T
  step:        T
  up:          bool
  var done:    bool = false

  implement Iterator {
    type Item = T
    fun next(): T? {
      if (this.done) return null
      if (this.up && this.current > this.last) return null
      if (!this.up && this.current < this.last) return null
      val v = this.current
      // stop rather than wrap when the next step would leave the type's range
      if (this.up) {
        if (this.last -% this.current < this.step) this.done = true else this.current += this.step
      } else {
        if (this.current -% this.last < this.step) this.done = true else this.current -= this.step
      }
      v
    }
  }

  /// Every `step`-th value of this sequence, from its start:
  /// `(0..10).reversed().step(3)` is 10, 7, 4, 1.
  public fun step(step: T): RangeStepIter<T> {
    if (step <= 0) panic("step: must be positive")
    RangeStepIter(current: this.current, last: this.last, step, up: this.up, done: this.done)
  }

  /// The same values in the opposite order: `(0..10).step(3).reversed()`
  /// is 9, 6, 3, 0.
  public fun reversed(): RangeStepIter<T> {
    if (this.done) return this
    // the last value this sequence reaches, then walk back from it
    val span = if (this.up) this.last - this.current else this.current - this.last
    val steps = span / this.step
    val end = if (this.up) this.current + steps * this.step else this.current - steps * this.step
    RangeStepIter(current: end, last: this.current, step: this.step, up: !this.up)
  }
}

extend<T> Range<T> {
  /// Number of values in the range (0 when empty).
  public fun len(): i64 {
    val last = if (this.inclusive) this.hi else this.hi -% 1
    if (last < this.lo) return 0
    (last - this.lo) as i64 + 1
  }

  /// True when `x` lies inside the range.
  public fun contains(x: T): bool {
    if (x < this.lo) return false
    if (this.inclusive) x <= this.hi else x < this.hi
  }

  /// Every `step`-th value, starting at the low end.
  public fun step(step: T): RangeStepIter<T> {
    if (step <= 0) panic("step: must be positive")
    val last = if (this.inclusive) this.hi else this.hi -% 1
    RangeStepIter(current: this.lo, last, step, up: true, done: last < this.lo)
  }

  /// The values from the high end down to the low end.
  public fun reversed(): RangeStepIter<T> {
    val last = if (this.inclusive) this.hi else this.hi -% 1
    RangeStepIter(current: last, last: this.lo, step: 1, up: false, done: last < this.lo)
  }
}

extend<T> List<T> {
  /// `map` that drops the nulls: `f` returns `U?` and the result holds the
  /// values that were present, in order (Kotlin's `mapNotNull`).
  public fun mapNotNull<U>(f: fun(T): U?): List<U> {
    val out: MutableList<U> = []
    loop (x in this) {
      val y = f(x) ?: continue
      out.push(y)
    }
    out.toList()
  }

  /// Splits the list in two: the elements `p` accepts, then the rest, each
  /// in the original order.
  public fun partition(p: fun(T): bool): (List<T>, List<T>) {
    val yes: MutableList<T> = []
    val no: MutableList<T> = []
    loop (x in this) {
      if (p(x)) yes.push(x) else no.push(x)
    }
    (yes.toList(), no.toList())
  }
}

extend<T, E> Result<T, E> {
  /// The value, or null when this is an error (Kotlin's `getOrNull`).
  public fun getOrNull(): T? = when (this) {
    is Ok(value) => value
    is Err       => null
  }

  /// The error, or null when this is a value.
  public fun errorOrNull(): E? = when (this) {
    is Err(error) => error
    is Ok         => null
  }

  /// The value, or `fallback` when this is an error.
  public fun getOrDefault(fallback: T): T = this.getOrNull() ?: fallback

  /// The same outcome with the error replaced by `f(error)` — how an error
  /// from one layer becomes one of another while keeping what it said:
  /// `try parse(text).mapError(e => BadRequest(detail: e.message()))`.
  /// When the new error does not depend on the old one, `?!` is shorter:
  /// `try parse(text) ?! BadRequest(detail: "not a user")`.
  public fun mapError<E2: Error>(f: fun(E): E2): Result<T, E2> = when (this) {
    is Ok(value)  => Ok(value)
    is Err(error) => Err(f(error))
  }
}

extend<T, E> List<Result<T, E>> {
  /// The values of the successful results, in order.
  public fun oks(): List<T> = this.mapNotNull(r => r.getOrNull())

  /// The errors of the failed results, in order.
  public fun errors(): List<E> = this.mapNotNull(r => r.errorOrNull())
}

extend<T> List<T?> {
  /// The elements that are present, in order (Kotlin's `filterNotNull`).
  public fun filterNotNull(): List<T> = this.mapNotNull(x => x)
}
