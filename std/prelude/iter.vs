// Prelude — iteration (D42, D46). In scope in every file (D24).
//
// `loop (x in c)` desugars through Iterable; lazy adapters are default
// bodies on Iterator, so every iterator gets them for free (D53).

public trait Iterator {
  type Item
  fun next(): Item?

  fun map<U>(f: fun(Item): U): MapIter<Self, U> = MapIter(inner: self, f)
  fun filter(f: fun(Item): bool): FilterIter<Self> = FilterIter(inner: self, f)
  fun take(n: i64): TakeIter<Self> = TakeIter(inner: self, remaining: n)
  fun skip(n: i64): SkipIter<Self> = SkipIter(inner: self, remaining: n)
  fun enumerate(): EnumerateIter<Self> = EnumerateIter(inner: self, index: 0)
  fun zip<J: Iterator>(other: J): ZipIter<Self, J> = ZipIter(a: self, b: other)

  fun toList(): List<Item> {
    var out: MutableList<Item> = []
    loop {
      val v = self.next()
      if (v == null) break
      out.push(v)
    }
    out.toList()
  }

  fun toSet(): Set<Item> = self.toList().toSet()

  fun count(): i64 {
    var n: i64 = 0
    loop {
      if (self.next() == null) break
      n += 1
    }
    n
  }

  fun fold<A>(init: A, f: fun(A, Item): A): A {
    var acc = init
    loop {
      val v = self.next()
      if (v == null) break
      acc = f(acc, v)
    }
    acc
  }

  fun forEach(f: fun(Item)) {
    loop {
      val v = self.next()
      if (v == null) break
      f(v)
    }
  }

  fun any(f: fun(Item): bool): bool {
    loop {
      val v = self.next()
      if (v == null) return false
      if (f(v)) return true
    }
  }

  fun all(f: fun(Item): bool): bool {
    loop {
      val v = self.next()
      if (v == null) return true
      if (!f(v)) return false
    }
  }

  fun find(f: fun(Item): bool): Item? {
    loop {
      val v = self.next()
      if (v == null) return null
      if (f(v)) return v
    }
  }

  fun last(): Item? {
    var found: Item? = null
    loop {
      val v = self.next()
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
  fun iter(): Iter = self.iterator()
}

// ---------------------------------------------------------------------------
// adapters

public struct MapIter<I: Iterator, U> {
  inner: I
  f:     fun(I.Item): U

  implement Iterator {
    type Item = U
    fun next(): U? {
      val v = self.inner.next()
      if (v == null) return null
      self.f(v)
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
        val v = self.inner.next()
        if (v == null) return null
        if (self.f(v)) return v
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
      if (self.remaining <= 0) return null
      self.remaining -= 1
      self.inner.next()
    }
  }
}

public struct SkipIter<I: Iterator> {
  inner:         I
  var remaining: i64

  implement Iterator {
    type Item = I.Item
    fun next(): I.Item? {
      loop (self.remaining > 0) {
        self.remaining -= 1
        if (self.inner.next() == null) return null
      }
      self.inner.next()
    }
  }
}

public struct EnumerateIter<I: Iterator> {
  inner:     I
  var index: i64

  implement Iterator {
    type Item = (i64, I.Item)
    fun next(): (i64, I.Item)? {
      val v = self.inner.next()
      if (v == null) return null
      val i = self.index
      self.index += 1
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
      val x = self.a.next()
      if (x == null) return null
      val y = self.b.next()
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
      val v = self.list.at(self.index) ?: return null
      self.index += 1
      v
    }
  }
}

implement<T> Iterable for List<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> = ListIter(list: self)
}

implement<T> Iterable for MutableList<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> = ListIter(list: self.toList())
}

public struct RangeIter<T> {
  var current: T
  hi:          T
  inclusive:   bool

  implement Iterator {
    type Item = T
    fun next(): T? {
      if (self.inclusive) {
        if (self.current > self.hi) return null
      } else {
        if (self.current >= self.hi) return null
      }
      val v = self.current
      self.current = self.current +% 1
      v
    }
  }
}

implement<T> Iterable for Range<T> {
  type Iter = RangeIter<T>
  fun iterator(): RangeIter<T> = RangeIter(current: self.lo, hi: self.hi, inclusive: self.inclusive)
}
