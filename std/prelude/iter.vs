// Prelude — iteration (D42, D46). In scope in every file (D24).
//
// `loop (x in c)` desugars through Iterable; lazy adapters are default
// bodies on Iterator, so every iterator gets them for free (D53).

pub trait Iterator {
  type Item
  mut fun next(): Item?

  fun map<U>(f: fun(Item): U): MapIter<Self, U> = MapIter(inner: self, f: f)
  fun filter(f: fun(Item): bool): FilterIter<Self> = FilterIter(inner: self, f: f)
  fun take(n: i64): TakeIter<Self> = TakeIter(inner: self, remaining: n)
  fun skip(n: i64): SkipIter<Self> = SkipIter(inner: self, remaining: n)
  fun enumerate(): EnumerateIter<Self> = EnumerateIter(inner: self, index: 0)
  fun zip<J: Iterator>(other: J): ZipIter<Self, J> = ZipIter(a: self, b: other)

  mut fun toList(): List<Item> {
    var out: MutableList<Item> = []
    loop {
      val v = self.next()
      if (v == null) break
      out.push(v)
    }
    out.toList()
  }

  mut fun count(): i64 {
    var n: i64 = 0
    loop {
      if (self.next() == null) break
      n += 1
    }
    n
  }

  mut fun fold<A>(init: A, f: fun(A, Item): A): A {
    var acc = init
    loop {
      val v = self.next()
      if (v == null) break
      acc = f(acc, v)
    }
    acc
  }

  mut fun forEach(f: fun(Item)) {
    loop {
      val v = self.next()
      if (v == null) break
      f(v)
    }
  }

  mut fun any(f: fun(Item): bool): bool {
    loop {
      val v = self.next()
      if (v == null) return false
      if (f(v)) return true
    }
  }

  mut fun all(f: fun(Item): bool): bool {
    loop {
      val v = self.next()
      if (v == null) return true
      if (!f(v)) return false
    }
  }

  mut fun find(f: fun(Item): bool): Item? {
    loop {
      val v = self.next()
      if (v == null) return null
      if (f(v)) return v
    }
  }

  mut fun last(): Item? {
    var found: Item? = null
    loop {
      val v = self.next()
      if (v == null) break
      found = v
    }
    found
  }
}

pub trait Iterable {
  type Iter: Iterator
  fun iterator(): Iter
  // `xs.iter()` reads better at the head of a pipeline; same thing.
  fun iter(): Iter = self.iterator()
}

// ---------------------------------------------------------------------------
// adapters

pub struct MapIter<I: Iterator, U> {
  inner: I
  f: fun(I::Item): U
}

impl<I: Iterator, U> Iterator for MapIter<I, U> {
  type Item = U
  mut fun next(): U? {
    val v = self.inner.next()
    if (v == null) return null
    self.f(v)
  }
}

pub struct FilterIter<I: Iterator> {
  inner: I
  f: fun(I::Item): bool
}

impl<I: Iterator> Iterator for FilterIter<I> {
  type Item = I::Item
  mut fun next(): I::Item? {
    loop {
      val v = self.inner.next()
      if (v == null) return null
      if (self.f(v)) return v
    }
  }
}

pub struct TakeIter<I: Iterator> {
  inner: I
  remaining: i64
}

impl<I: Iterator> Iterator for TakeIter<I> {
  type Item = I::Item
  mut fun next(): I::Item? {
    if (self.remaining <= 0) return null
    self.remaining -= 1
    self.inner.next()
  }
}

pub struct SkipIter<I: Iterator> {
  inner: I
  remaining: i64
}

impl<I: Iterator> Iterator for SkipIter<I> {
  type Item = I::Item
  mut fun next(): I::Item? {
    loop (self.remaining > 0) {
      self.remaining -= 1
      if (self.inner.next() == null) return null
    }
    self.inner.next()
  }
}

pub struct EnumerateIter<I: Iterator> {
  inner: I
  index: i64
}

impl<I: Iterator> Iterator for EnumerateIter<I> {
  type Item = (i64, I::Item)
  mut fun next(): (i64, I::Item)? {
    val v = self.inner.next()
    if (v == null) return null
    val i = self.index
    self.index += 1
    (i, v)
  }
}

pub struct ZipIter<A: Iterator, B: Iterator> {
  a: A
  b: B
}

impl<A: Iterator, B: Iterator> Iterator for ZipIter<A, B> {
  type Item = (A::Item, B::Item)
  mut fun next(): (A::Item, B::Item)? {
    val x = self.a.next()
    if (x == null) return null
    val y = self.b.next()
    if (y == null) return null
    (x, y)
  }
}

// ---------------------------------------------------------------------------
// iterators over the builtin collections

pub struct ListIter<T> {
  list: List<T>
  index: i64 = 0
}

impl<T> Iterator for ListIter<T> {
  type Item = T
  mut fun next(): T? {
    if (self.index >= self.list.len()) return null
    val v = self.list[self.index]
    self.index += 1
    v
  }
}

impl<T> Iterable for List<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> = ListIter(list: self)
}

impl<T> Iterable for MutableList<T> {
  type Iter = ListIter<T>
  fun iterator(): ListIter<T> = ListIter(list: self.toList())
}

pub struct RangeIter<T> {
  current: T
  hi: T
  inclusive: bool
}

impl<T> Iterator for RangeIter<T> {
  type Item = T
  mut fun next(): T? {
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

impl<T> Iterable for Range<T> {
  type Iter = RangeIter<T>
  fun iterator(): RangeIter<T> = RangeIter(current: self.lo, hi: self.hi, inclusive: self.inclusive)
}
