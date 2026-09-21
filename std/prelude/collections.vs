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

/// An empty double-ended queue: `val q = deque<i64>()`, or
/// `val q: Deque<i64> = deque()`.
public fun deque<T>(): Deque<T> = Deque(state: &DequeState<T>())

/// A double-ended queue over a ring buffer: O(1) at both ends, so it serves
/// as a FIFO queue (`addLast` / `removeFirst`), a stack, or a sliding
/// window. Amortised O(1) growth; `at(i)` is O(1).
public struct Deque<T> {
  state: *DequeState<T>

  /// Number of elements.
  public fun len(): i64 = self.state.size

  /// True when there are no elements.
  public fun isEmpty(): bool = self.state.size == 0

  /// Appends `x` at the back.
  public fun addLast(x: T) {
    val s = self.state
    self.reserve()
    s.buf.set((s.head + s.size) % s.buf.len(), x)
    s.size += 1
  }

  /// Prepends `x` at the front.
  public fun addFirst(x: T) {
    val s = self.state
    self.reserve()
    val cap = s.buf.len()
    s.head = (s.head + cap - 1) % cap
    s.buf.set(s.head, x)
    s.size += 1
  }

  /// Removes and returns the front element, or `null` when empty.
  public fun removeFirst(): T? {
    val s = self.state
    if (s.size == 0) return null
    val x = s.buf.atOrPanic(s.head)
    s.buf.set(s.head, null)
    s.head = (s.head + 1) % s.buf.len()
    s.size -= 1
    x
  }

  /// Removes and returns the back element, or `null` when empty.
  public fun removeLast(): T? {
    val s = self.state
    if (s.size == 0) return null
    s.size -= 1
    val i = (s.head + s.size) % s.buf.len()
    val x = s.buf.atOrPanic(i)
    s.buf.set(i, null)
    x
  }

  /// The front element, or `null` when empty.
  public fun first(): T? = self.at(0)

  /// The back element, or `null` when empty.
  public fun last(): T? = self.at(-1)

  /// The element `i` places from the front, or `null` when `i` is out of
  /// range; a negative `i` counts from the back, so `at(-1)` is the last.
  public fun at(i: i64): T? {
    val s = self.state
    val k = if (i < 0) i + s.size else i
    if (k < 0 || k >= s.size) return null
    s.buf.atOrPanic((s.head + k) % s.buf.len())
  }

  /// Removes every element.
  public fun clear() {
    val s = self.state
    s.buf.clear()
    s.head = 0
    s.size = 0
  }

  /// The elements front to back.
  public fun toList(): List<T> {
    val s = self.state
    var out: MutableList<T> = []
    loop (i in 0..<s.size) {
      out.push(s.buf.atOrPanic((s.head + i) % s.buf.len()) ?: panic("deque: a live slot is empty"))
    }
    out.toList()
  }

  /// Makes room for one more element: when the ring is full, the live
  /// elements move to the start of a buffer twice the size.
  fun reserve() {
    val s = self.state
    val cap = s.buf.len()
    if (s.size < cap) return
    val fresh = MutableList<T?>.repeat(null, if (cap == 0) 4 else cap * 2)
    loop (i in 0..<s.size) fresh.set(i, s.buf.atOrPanic((s.head + i) % cap))
    s.buf = fresh
    s.head = 0
  }

  impl Iterable {
    type Iter = ListIter<T>
    fun iterator(): ListIter<T> = ListIter(list: self.toList())
  }

  impl Display {
    fun toString(): string = "[${self.toList().join(", ")}]"
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

/// An empty min-priority queue over the elements' natural order:
/// `val pq = priorityQueue<i64>()`; `pop()` yields the smallest first.
public fun priorityQueue<T: Comparable>(): PriorityQueue<T> =
  PriorityQueue(state: &PriorityQueueState<T>(compare: (a, b) => a.compareTo(b)))

/// An empty priority queue ordered by `compare`, which returns
/// `Ordering.Less` when its first argument should come out first. For the largest
/// first: `priorityQueueBy<i64>((a, b) => b.compareTo(a))`.
public fun priorityQueueBy<T>(compare: fun(T, T): Ordering): PriorityQueue<T> =
  PriorityQueue(state: &PriorityQueueState<T>(compare))

/// A priority queue over a binary heap: `push` and `pop` are O(log n),
/// `peek` O(1). Elements that compare equal come out in no particular order.
public struct PriorityQueue<T> {
  state: *PriorityQueueState<T>

  /// Number of elements.
  public fun len(): i64 = self.state.items.len()

  /// True when there are no elements.
  public fun isEmpty(): bool = self.state.items.len() == 0

  /// The element that `pop` would return, or `null` when empty.
  public fun peek(): T? = self.state.items.first()

  /// Adds `x`.
  public fun push(x: T) {
    val items = self.state.items
    items.push(x)
    self.siftUp(items.len() - 1)
  }

  /// Removes and returns the first element in the queue's order, or `null`
  /// when empty.
  public fun pop(): T? {
    val items = self.state.items
    val top = items.first() ?: return null
    val last = items.pop() ?: return null
    if (items.len() > 0) {
      items.set(0, last)
      self.siftDown(0)
    }
    top
  }

  /// Removes every element.
  public fun clear() {
    self.state.items.clear()
  }

  /// The elements in heap order — the first is the smallest, the rest are
  /// not sorted.
  public fun toList(): List<T> = self.state.items.toList()

  fun siftUp(from: i64) {
    val s = self.state
    var i = from
    loop (i > 0) {
      val parent = (i - 1) / 2
      if (s.compare(s.items.atOrPanic(i), s.items.atOrPanic(parent)) >= 0) break
      s.items.swap(i, parent)
      i = parent
    }
  }

  fun siftDown(from: i64) {
    val s = self.state
    val n = s.items.len()
    var i = from
    loop {
      val left = 2 * i + 1
      val right = left + 1
      var smallest = i
      if (left < n && s.compare(s.items.atOrPanic(left), s.items.atOrPanic(smallest)) < 0) smallest = left
      if (right < n && s.compare(s.items.atOrPanic(right), s.items.atOrPanic(smallest)) < 0) smallest = right
      if (smallest == i) break
      s.items.swap(i, smallest)
      i = smallest
    }
  }

  impl Display {
    fun toString(): string = "[${self.state.items.join(", ")}]"
  }
}
