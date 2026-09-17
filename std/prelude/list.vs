// Prelude — List, MutableList and Range methods (D25, D29, D46). In scope
// in every file (D24).
//
// The compiler provides the primitives (`len`, `[i]`, `at`, `push`, `pop`,
// `sorted`, the eager `map`/`filter`/`fold`, ...); the rest is Veles.

extend<T> List<T> {
  /// The first `n` elements (all of them when `n` exceeds the length).
  pub fun take(n: i64): List<T> = self.slice(0, n)

  /// The elements after the first `n`.
  pub fun drop(n: i64): List<T> = self.slice(n, self.len())

  /// The elements in `from..<to`, clamped to the list; empty when `from >= to`.
  pub fun slice(from: i64, to: i64): List<T> {
    val lo = from.max(0)
    val hi = to.min(self.len())
    var out: MutableList<T> = []
    loop (i in lo..<hi) {
      out.push(self[i])
    }
    out.toList()
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
      out.push((self[i], other[i]))
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

  /// A copy sorted from largest to smallest (elements must be numbers or strings).
  pub fun sortedDescending(): List<T> = self.sorted().reversed()

  /// The smallest element, or `null` when empty (elements must be numbers or strings).
  pub fun min(): T? {
    if (self.isEmpty()) return null
    var best = self[0]
    loop (x in self) {
      if (x < best) best = x
    }
    best
  }

  /// The largest element, or `null` when empty (elements must be numbers or strings).
  pub fun max(): T? {
    if (self.isEmpty()) return null
    var best = self[0]
    loop (x in self) {
      if (x > best) best = x
    }
    best
  }

  /// The elements as text, joined with `sep`.
  pub fun join(sep: string): string = self.joinToString(sep)
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

extend<T> MutableList<T> {
  /// Inserts `x` at index `i`, shifting the rest up; `i == len()` appends.
  pub mut fun insert(i: i64, x: T) {
    if (i < 0 || i > self.len()) panic("insert: index $i out of bounds for list of length ${self.len()}")
    self.push(x)
    var j = self.len() - 1
    loop (j > i) {
      self[j] = self[j - 1]
      j -= 1
    }
    self[i] = x
  }

  /// Removes and returns the element at index `i`, shifting the rest down.
  pub mut fun removeAt(i: i64): T {
    val removed = self[i]
    loop (j in i..<(self.len() - 1)) {
      self[j] = self[j + 1]
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

  /// Sorts in place (elements must be numbers or strings).
  pub mut fun sort() {
    val sorted = self.sorted()
    loop (i in 0..<self.len()) {
      self[i] = sorted[i]
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
}

impl<T> Iterator for RangeStepIter<T> {
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
    RangeStepIter(current: self.lo, last: last, step: step, up: true, done: last < self.lo)
  }

  /// The values from the high end down to the low end.
  pub fun reversed(): RangeStepIter<T> {
    val last = if (self.inclusive) self.hi else self.hi -% 1
    RangeStepIter(current: last, last: self.lo, step: 1, up: false, done: last < self.lo)
  }
}
