// Prelude — List, MutableList and Range methods (D25, D29, D46). In scope
// in every file (D24).
//
// The compiler provides the primitives (`len`, `at`, `atOrPanic`, `set`, `push`, `pop`,
// `sorted`, the eager `map`/`filter`/`fold`, ...); the rest is Veles.

extend<T> List<T> {
  /// The element at `i`, or `d` when `i` is out of range; `self.at(i) ?: d`.
  pub fun atOrDefault(i: i64, d: T): T = self.at(i) ?: d

  /// The first `n` elements (all of them when `n` exceeds the length).
  pub fun take(n: i64): List<T> = self.slice(0, n)

  /// The elements after the first `n`.
  pub fun drop(n: i64): List<T> = self.slice(n, self.len())

  /// A new list of these elements followed by `other`'s.
  pub fun concat(other: List<T>): List<T> {
    var out: MutableList<T> = []
    out.addAll(self)
    out.addAll(other)
    out.toList()
  }

  /// The elements in `from..<to`, clamped to the list; empty when `from >= to`.
  pub fun slice(from: i64, to: i64): List<T> {
    val lo = from.max(0)
    val hi = to.min(self.len())
    var out: MutableList<T> = []
    loop (i in lo..<hi) {
      out.push(self.atOrPanic(i))
    }
    out.toList()
  }

  /// The index of the first element `pred` accepts, or -1.
  pub fun indexOfFirst(pred: fun(T): bool): i64 {
    loop (i in 0..<self.len()) {
      if (pred(self.atOrPanic(i))) return i
    }
    -1
  }

  /// The number of elements `pred` accepts.
  pub fun count(pred: fun(T): bool): i64 {
    var n: i64 = 0
    loop (x in self) {
      if (pred(x)) n += 1
    }
    n
  }

  /// Pairs of elements at the same index, as long as the shorter list.
  pub fun zip<U>(other: List<U>): List<(T, U)> {
    var out: MutableList<(T, U)> = []
    val n = self.len().min(other.len())
    loop (i in 0..<n) {
      out.push((self.atOrPanic(i), other.atOrPanic(i)))
    }
    out.toList()
  }

  /// The lists `f` returns for each element, concatenated.
  pub fun flatMap<U>(f: fun(T): List<U>): List<U> {
    var out: MutableList<U> = []
    loop (x in self) {
      loop (y in f(x)) {
        out.push(y)
      }
    }
    out.toList()
  }

  /// The elements with duplicates removed, first occurrences kept in order.
  pub fun distinct(): List<T> {
    var seen = MutableSet<T>()
    var out: MutableList<T> = []
    loop (x in self) {
      if (seen.add(x)) out.push(x)
    }
    out.toList()
  }

  /// Consecutive pieces of `n` elements; the last may be shorter.
  pub fun chunked(n: i64): List<List<T>> {
    if (n <= 0) panic("chunked: size must be positive, got $n")
    var out: MutableList<List<T>> = []
    var i: i64 = 0
    loop (i < self.len()) {
      out.push(self.slice(i, i + n))
      i += n
    }
    out.toList()
  }

  /// Every window of `n` consecutive elements (none when the list is shorter).
  pub fun windowed(n: i64): List<List<T>> {
    if (n <= 0) panic("windowed: size must be positive, got $n")
    var out: MutableList<List<T>> = []
    loop (i in 0..(self.len() - n)) {
      out.push(self.slice(i, i + n))
    }
    out.toList()
  }
}

// Comparison strategies (D48): the natural order comes from Comparable;
// any other order is a comparator (`sortedWith`, `minWith`) or a key
// (`sortedBy`, `minBy`, `distinctBy`) passed at the call.
extend<T> List<T> {
  /// A copy sorted by `compare`, which returns a negative number when its
  /// first argument sorts first: `xs.sortedWith((a, b) => b.compareTo(a))`
  /// is descending. Stable (equal elements keep their order); O(n log n).
  pub fun sortedWith(compare: fun(T, T): i64): List<T> {
    val n = self.len()
    var src = self.toMutable()
    if (n < 2) return src.toList()
    var dst: MutableList<T> = self.toMutable()
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
          if (i < mid && (j >= hi || compare(src.atOrPanic(i), src.atOrPanic(j)) <= 0)) {
            dst.set(k, src.atOrPanic(i))
            i += 1
          } else {
            dst.set(k, src.atOrPanic(j))
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
  pub fun sortedByDescending<K: Comparable>(key: fun(T): K): List<T> =
    self.sortedWith((a, b) => key(b).compareTo(key(a)))

  /// The element with the smallest key, or `null` when empty; the first
  /// of several equal keys.
  pub fun minBy<K: Comparable>(key: fun(T): K): T? {
    var best = self.first() ?: return null
    var bestKey = key(best)
    loop (x in self.drop(1)) {
      val k = key(x)
      if (k < bestKey) {
        best = x
        bestKey = k
      }
    }
    best
  }

  /// The element with the largest key, or `null` when empty.
  pub fun maxBy<K: Comparable>(key: fun(T): K): T? {
    var best = self.first() ?: return null
    var bestKey = key(best)
    loop (x in self.drop(1)) {
      val k = key(x)
      if (k > bestKey) {
        best = x
        bestKey = k
      }
    }
    best
  }

  /// The element that `compare` places first, or `null` when empty.
  pub fun minWith(compare: fun(T, T): i64): T? {
    var best = self.first() ?: return null
    loop (x in self.drop(1)) {
      if (compare(x, best) < 0) best = x
    }
    best
  }

  /// The element that `compare` places last, or `null` when empty.
  pub fun maxWith(compare: fun(T, T): i64): T? {
    var best = self.first() ?: return null
    loop (x in self.drop(1)) {
      if (compare(x, best) > 0) best = x
    }
    best
  }

  /// The elements whose keys are distinct, first occurrence kept:
  /// `names.distinctBy(n => n.toLower())`.
  pub fun distinctBy<K>(key: fun(T): K): List<T> {
    var seen = MutableSet<K>()
    var out: MutableList<T> = []
    loop (x in self) {
      if (seen.add(key(x))) out.push(x)
    }
    out.toList()
  }
}

extend<T> MutableList<T> {
  /// Sorts in place by `compare` (see `sortedWith`).
  pub mut fun sortWith(compare: fun(T, T): i64) {
    val sorted = self.sortedWith(compare)
    loop (i in 0..<self.len()) {
      self.set(i, sorted.atOrPanic(i))
    }
  }
}

// Ordering needs Comparable: numbers and strings implement it in the
// prelude, a struct by `impl Comparable`. The bound is checked at the call
// site, so a list of anything else reports the error there.
extend<T: Comparable> List<T> {
  /// A copy sorted from largest to smallest.
  pub fun sortedDescending(): List<T> = self.sorted().reversed()

  /// The smallest element, or `null` when empty.
  pub fun min(): T? {
    if (self.isEmpty()) return null
    var best = self.atOrPanic(0)
    loop (x in self) {
      if (x < best) best = x
    }
    best
  }

  /// The largest element, or `null` when empty.
  pub fun max(): T? {
    if (self.isEmpty()) return null
    var best = self.atOrPanic(0)
    loop (x in self) {
      if (x > best) best = x
    }
    best
  }
}

extend List<i64> {
  /// The sum of the elements (0 for an empty list).
  pub fun sum(): i64 {
    var total: i64 = 0
    loop (x in self) {
      total += x
    }
    total
  }
}

extend List<f64> {
  /// The sum of the elements (0.0 for an empty list).
  pub fun sum(): f64 {
    var total = 0.0
    loop (x in self) {
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
  pub static fun repeat(x: T, count: i64): MutableList<T> {
    var out: MutableList<T> = []
    loop (_ in 0..<count) out.push(x)
    out
  }

  /// Overwrites every element with `x`; the length does not change.
  pub mut fun fill(x: T) {
    loop (i in 0..<self.len()) self.set(i, x)
  }
}

extend<T> MutableList<T> {
  /// A list of `n` elements where slot `i` holds `init(i)`:
  /// `MutableList<MutableList<i64>>.make(n, _ => [])`.
  pub static fun make(n: i64, init: fun(i64): T): MutableList<T> {
    var out: MutableList<T> = []
    loop (i in 0..<n) out.push(init(i))
    out
  }

  /// Exchanges the elements at `i` and `j`.
  pub mut fun swap(i: i64, j: i64) {
    (*self.refOrPanic(i), *self.refOrPanic(j)) = (self.atOrPanic(j), self.atOrPanic(i))
  }

  /// Inserts `x` at index `i`, shifting the rest up; `i == len()` appends.
  pub mut fun insert(i: i64, x: T) {
    if (i < 0 || i > self.len()) panic("insert: index $i out of bounds for list of length ${self.len()}")
    self.push(x)
    var j = self.len() - 1
    loop (j > i) {
      self.set(j, self.atOrPanic(j - 1))
      j -= 1
    }
    self.set(i, x)
  }

  /// Removes and returns the element at index `i`, shifting the rest down.
  pub mut fun removeAt(i: i64): T {
    val removed = self.atOrPanic(i)
    loop (j in i..<(self.len() - 1)) {
      self.set(j, self.atOrPanic(j + 1))
    }
    self.pop()
    removed
  }

  /// Appends every element of `xs`.
  pub mut fun addAll(xs: List<T>) {
    loop (x in xs) {
      self.push(x)
    }
  }

  /// Sorts in place (elements must be Comparable).
  pub mut fun sort() {
    val sorted = self.sorted()
    loop (i in 0..<self.len()) {
      self.set(i, sorted.atOrPanic(i))
    }
  }
}

/// Steps through a range by a fixed increment, in either direction.
pub struct RangeStepIter<T> {
  current: T
  last:    T
  step:    T
  up:      bool
  done:    bool = false

  impl Iterator {
    type Item = T
    mut fun next(): T? {
      if (self.done) return null
      if (self.up && self.current > self.last) return null
      if (!self.up && self.current < self.last) return null
      val v = self.current
      // stop rather than wrap when the next step would leave the type's range
      if (self.up) {
        if (self.last -% self.current < self.step) self.done = true else self.current += self.step
      } else {
        if (self.current -% self.last < self.step) self.done = true else self.current -= self.step
      }
      v
    }
  }

  /// Every `step`-th value of this sequence, from its start:
  /// `(0..10).reversed().step(3)` is 10, 7, 4, 1.
  pub fun step(step: T): RangeStepIter<T> {
    if (step <= 0) panic("step: must be positive")
    RangeStepIter(current: self.current, last: self.last, step, up: self.up, done: self.done)
  }

  /// The same values in the opposite order: `(0..10).step(3).reversed()`
  /// is 9, 6, 3, 0.
  pub fun reversed(): RangeStepIter<T> {
    if (self.done) return self
    // the last value this sequence reaches, then walk back from it
    val span = if (self.up) self.last - self.current else self.current - self.last
    val steps = span / self.step
    val end = if (self.up) self.current + steps * self.step else self.current - steps * self.step
    RangeStepIter(current: end, last: self.current, step: self.step, up: !self.up)
  }
}

extend<T> Range<T> {
  /// Number of values in the range (0 when empty).
  pub fun len(): i64 {
    val last = if (self.inclusive) self.hi else self.hi -% 1
    if (last < self.lo) return 0
    (last - self.lo) as i64 + 1
  }

  /// True when `x` lies inside the range.
  pub fun contains(x: T): bool {
    if (x < self.lo) return false
    if (self.inclusive) x <= self.hi else x < self.hi
  }

  /// Every `step`-th value, starting at the low end.
  pub fun step(step: T): RangeStepIter<T> {
    if (step <= 0) panic("step: must be positive")
    val last = if (self.inclusive) self.hi else self.hi -% 1
    RangeStepIter(current: self.lo, last, step, up: true, done: last < self.lo)
  }

  /// The values from the high end down to the low end.
  pub fun reversed(): RangeStepIter<T> {
    val last = if (self.inclusive) self.hi else self.hi -% 1
    RangeStepIter(current: last, last: self.lo, step: 1, up: false, done: last < self.lo)
  }
}

extend<T> List<T> {
  /// `map` that drops the nulls: `f` returns `U?` and the result holds the
  /// values that were present, in order (Kotlin's `mapNotNull`).
  pub fun mapNotNull<U>(f: fun(T): U?): List<U> {
    val out: MutableList<U> = []
    loop (x in self) {
      val y = f(x) ?: continue
      out.push(y)
    }
    out.toList()
  }

  /// Splits the list in two: the elements `p` accepts, then the rest, each
  /// in the original order.
  pub fun partition(p: fun(T): bool): (List<T>, List<T>) {
    val yes: MutableList<T> = []
    val no: MutableList<T> = []
    loop (x in self) {
      if (p(x)) yes.push(x) else no.push(x)
    }
    (yes.toList(), no.toList())
  }
}

extend<T, E> Result<T, E> {
  /// The value, or null when this is an error (Kotlin's `getOrNull`).
  pub fun getOrNull(): T? = when (self) {
    is Ok(value) => value
    is Err       => null
  }

  /// The error, or null when this is a value.
  pub fun errorOrNull(): E? = when (self) {
    is Err(error) => error
    is Ok         => null
  }

  /// The value, or `fallback` when this is an error.
  pub fun getOrDefault(fallback: T): T = self.getOrNull() ?: fallback

  /// The same outcome with the error replaced by `f(error)` — how an error
  /// from one layer becomes one of another while keeping what it said:
  /// `try parse(text).mapError(e => BadRequest(detail: e.message()))`.
  /// When the new error does not depend on the old one, `?!` is shorter:
  /// `try parse(text) ?! BadRequest(detail: "not a user")`.
  pub fun mapError<E2: Error>(f: fun(E): E2): Result<T, E2> = when (self) {
    is Ok(value)  => Ok(value)
    is Err(error) => Err(f(error))
  }
}

extend<T, E> List<Result<T, E>> {
  /// The values of the successful results, in order.
  pub fun oks(): List<T> = self.mapNotNull(r => r.getOrNull())

  /// The errors of the failed results, in order.
  pub fun errors(): List<E> = self.mapNotNull(r => r.errorOrNull())
}

extend<T> List<T?> {
  /// The elements that are present, in order (Kotlin's `filterNotNull`).
  pub fun filterNotNull(): List<T> = self.mapNotNull(x => x)
}
