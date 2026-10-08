// Prelude — Deque and PriorityQueue. Both are reference types like
// MutableList (D25): a `val` binding can grow them, and passing one to a
// function shares it. The state lives behind a GC pointer (D31) so that the
// bookkeeping fields move with the buffer.

// ---------------------------------------------------------------------------
// Deque

struct DequeState<T> {
  /// A ring: the live elements are the `size` slots starting at `head`,
  /// wrapping at `buf.len()`, which is the capacity. A free slot holds
  /// `null` — `T?` keeps that apart from a stored `null` when `T` is itself
  /// nullable — so a removed element is released at once.
  var buf:  MutableList<T?> = []
  var head: i64 = 0
  var size: i64 = 0
}

/// A double-ended queue over a ring buffer: O(1) at both ends, so it serves
/// as a FIFO queue (`addLast` / `removeFirst`), a stack, or a sliding
/// window. Amortised O(1) growth; `at(i)` is O(1). `Deque<i64>()` makes
/// an empty one.
public struct Deque<T> {
  private state: *DequeState<T> = &DequeState<T>()

  /// Number of elements.
  public fun len(): i64 => this.state.size

  /// True when there are no elements.
  public fun isEmpty(): bool => this.state.size == 0

  /// Appends `x` at the back.
  public fun addLast(x: T) {
    val s = this.state
    this.grow()
    s.buf.set((s.head + s.size) % s.buf.len(), x)
    s.size += 1
  }

  /// Prepends `x` at the front.
  public fun addFirst(x: T) {
    val s = this.state
    this.grow()
    val cap = s.buf.len()
    s.head = (s.head + cap - 1) % cap
    s.buf.set(s.head, x)
    s.size += 1
  }

  /// Removes and returns the front element, or `null` when empty.
  public fun removeFirst(): T? {
    val s = this.state
    if (s.size == 0) return null
    val x = this.slot(0)
    s.buf.set(s.head, null)
    s.head = (s.head + 1) % s.buf.len()
    s.size -= 1
    x
  }

  /// Removes and returns the back element, or `null` when empty.
  public fun removeLast(): T? {
    val s = this.state
    if (s.size == 0) return null
    s.size -= 1
    val x = this.slot(s.size)
    s.buf.set((s.head + s.size) % s.buf.len(), null)
    x
  }

  /// The front element, or `null` when empty.
  public fun first(): T? => this.at(0)

  /// The back element, or `null` when empty.
  public fun last(): T? => this.at(-1)

  /// The element `i` places from the front, or `null` when `i` is out of
  /// range; a negative `i` counts from the back, so `at(-1)` is the last.
  public fun at(i: i64): T? {
    val s = this.state
    val k = if (i < 0) i + s.size else i
    if (k < 0 || k >= s.size) return null
    this.slot(k)
  }

  /// The slot `k` places from the front. Every position is taken mod the
  /// capacity, so it is inside the buffer whatever `k` is; it holds an
  /// element when `k` is in `0..<size`.
  fun slot(k: i64): T? {
    val s = this.state
    s.buf.at((s.head + k) % s.buf.len()) ?: panic("deque: a position taken mod the capacity is inside the buffer")
  }

  /// Removes every element.
  public fun clear() {
    val s = this.state
    s.buf.clear()
    s.head = 0
    s.size = 0
  }

  /// The elements front to back.
  public fun toList(): List<T> {
    val s = this.state
    var out: MutableList<T> = []
    loop (i in 0..<s.size) {
      out.push(this.slot(i) ?: panic("deque: a live slot is empty"))
    }
    out.toList()
  }

  /// Makes room for `n` elements in all, so adding up to `n` does not grow
  /// the buffer; never shrinks, never changes the contents (D105).
  public fun reserve(n: i64) {
    if (n > this.state.buf.len()) this.moveTo(n)
  }

  // Makes room for one more element: when the ring is full, the live
  // elements move to the start of a buffer twice the size.
  fun grow() {
    val cap = this.state.buf.len()
    if (this.state.size < cap) return
    this.moveTo(if (cap == 0) 4 else cap * 2)
  }

  // the live elements, in order, at the start of a buffer of `cap` slots
  fun moveTo(cap: i64) {
    val s = this.state
    val fresh = MutableList<T?>.repeat(null, cap)
    loop (i in 0..<s.size) fresh.set(i, this.slot(i))
    s.buf = fresh
    s.head = 0
  }

  implement Iterable {
    type Iter = ListIter<T>
    fun iterator(): ListIter<T> => ListIter(list: this.toList())
  }

  implement Display {
    fun toString(): string => "[${this.toList().join(", ")}]"
  }
}

// ---------------------------------------------------------------------------
// PriorityQueue

struct PriorityQueueState<T> {
  /// A binary heap: the smallest element (by `compare`) is at index 0 and
  /// the children of `i` are at `2i + 1` and `2i + 2`.
  items:   MutableList<T> = []
  compare: fun(T, T): Ordering
}

/// A priority queue over a binary heap: `push` and `pop` are O(log n),
/// `peek` O(1). Elements that compare equal come out in no particular order.
///
/// `PriorityQueue<i64>.natural()` pops the smallest first;
/// `PriorityQueue<Job>(compare: (a, b) => a.due.compareTo(b.due))` orders
/// by `compare`, which returns `Ordering.Less` when its first argument
/// should come out first — for the largest first,
/// `PriorityQueue<i64>(compare: (a, b) => b.compareTo(a))`.
public struct PriorityQueue<T> {
  private state: *PriorityQueueState<T>

  init(compare: fun(T, T): Ordering) {
    this.state = &PriorityQueueState<T>(compare)
  }

  /// Number of elements.
  public fun len(): i64 => this.state.items.len()

  /// True when there are no elements.
  public fun isEmpty(): bool => this.state.items.len() == 0

  /// The element that `pop` would return, or `null` when empty.
  public fun peek(): T? => this.state.items.first()

  /// Adds `x`.
  public fun push(x: T) {
    val items = this.state.items
    items.push(x)
    this.siftUp(items.len() - 1)
  }

  /// Removes and returns the first element in the queue's order, or `null`
  /// when empty.
  public fun pop(): T? {
    val items = this.state.items
    val top = items.first() ?: return null
    val last = items.pop() ?: return null
    if (items.len() > 0) {
      items.set(0, last)
      this.siftDown(0)
    }
    top
  }

  /// Removes every element.
  public fun clear() {
    this.state.items.clear()
  }

  /// The elements in heap order — the first is the smallest, the rest are
  /// not sorted.
  public fun toList(): List<T> => this.state.items.toList()

  /// The element at heap position `i`; the sifts only ask for positions
  /// below the heap's size.
  fun item(i: i64): T => this.state.items.at(i) ?: panic("heap: a sift reads only positions below the size")

  fun siftUp(from: i64) {
    val s = this.state
    var i = from
    loop (i > 0) {
      val parent = (i - 1) / 2
      if (s.compare(this.item(i), this.item(parent)) >= 0) break
      s.items.swap(i, parent)
      i = parent
    }
  }

  fun siftDown(from: i64) {
    val s = this.state
    val n = s.items.len()
    var i = from
    loop {
      val left = 2 * i + 1
      val right = left + 1
      var smallest = i
      if (left < n && s.compare(this.item(left), this.item(smallest)) < 0) smallest = left
      if (right < n && s.compare(this.item(right), this.item(smallest)) < 0) smallest = right
      if (smallest == i) break
      s.items.swap(i, smallest)
      i = smallest
    }
  }

  implement Display {
    fun toString(): string => "[${this.state.items.join(", ")}]"
  }
}

extend<T: Comparable> PriorityQueue<T> {
  /// An empty queue over the elements' natural order: `pop()` yields the
  /// smallest first.
  public static fun natural(): PriorityQueue<T> => PriorityQueue<T>(compare: (a, b) => a.compareTo(b))
}
