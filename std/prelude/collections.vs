// Prelude — Deque and PriorityQueue. Both are values like List (D234):
// assignment, passing and storing copy, and the copy is made at the first
// change — their storage is a List, which copies on write (D231). Sharing
// one is a pointer, `&queue`.

// ---------------------------------------------------------------------------
// Deque

/// A double-ended queue over a ring buffer: O(1) at both ends, so it serves
/// as a FIFO queue (`addLast` / `removeFirst`), a stack, or a sliding
/// window. Amortised O(1) growth; `get(i)` and `at(i)` are O(1). `Deque<i64>()` makes
/// an empty one.
public struct Deque<T> {
  /// A ring: the live elements are the `size` slots starting at `head`,
  /// wrapping at `buf.len()`, which is the capacity. A free slot holds
  /// `null` — `T?` keeps that apart from a stored `null` when `T` is itself
  /// nullable — so a removed element is released at once.
  private var buf:  List<T?> = []
  private var head: i64 = 0
  private var size: i64 = 0

  /// Number of elements.
  public fun len(): i64 => this.size

  /// True when there are no elements.
  public fun isEmpty(): bool => this.size == 0

  /// Appends `x` at the back.
  public fun addLast(x: T) {
    this.grow()
    this.buf.set(this.pos(this.size), x)
    this.size += 1
  }

  /// Prepends `x` at the front.
  public fun addFirst(x: T) {
    this.grow()
    val cap = this.buf.len()
    this.head = if (this.head == 0) cap - 1 else this.head - 1
    this.buf.set(this.head, x)
    this.size += 1
  }

  /// Removes and returns the front element, or `null` when empty.
  public fun removeFirst(): T? {
    if (this.size == 0) return null
    val x = this.slot(0)
    this.buf.set(this.head, null)
    this.head = this.pos(1)
    this.size -= 1
    x
  }

  /// Removes and returns the back element, or `null` when empty.
  public fun removeLast(): T? {
    if (this.size == 0) return null
    this.size -= 1
    val x = this.slot(this.size)
    this.buf.set(this.pos(this.size), null)
    x
  }

  /// The front element, or `null` when empty.
  public fun first(): T? => this.get(0)

  /// The back element, or `null` when empty.
  public fun last(): T? => this.get(-1)

  /// A copy of the element `i` places from the front, or `null` when `i`
  /// is out of range; a negative `i` counts from the back, so `get(-1)` is
  /// the last.
  public fun get(i: i64): T? {
    val k = if (i < 0) i + this.size else i
    if (k < 0 || k >= this.size) return null
    this.slot(k)
  }

  /// A pointer to the element `i` places from the front, or `null` when `i`
  /// is out of range; a negative `i` counts from the back. `d.at(0)?.n += 1`
  /// changes the element in place; `get` gives a copy (D231).
  public fun at(i: i64): (*T)? {
    val k = if (i < 0) i + this.size else i
    if (k < 0 || k >= this.size) return null
    val slot = this.buf.at(this.pos(k)) ?: panic("deque: a position below the capacity is inside the buffer")
    slotPayload(slot)
  }

  /// The slot `k` places from the front, for `k` in `0..<capacity`; it
  /// holds an element when `k` is in `0..<size`.
  fun slot(k: i64): T? {
    this.buf.get(this.pos(k)) ?: panic("deque: a position below the capacity is inside the buffer")
  }

  /// The buffer index `k` places from the front, for `k` in
  /// `0..<capacity`: `head + k` wrapped once — a compare, where `%` is a
  /// division on every access.
  fun pos(k: i64): i64 {
    val i = this.head + k
    if (i >= this.buf.len()) i - this.buf.len() else i
  }

  /// Removes every element.
  public fun clear() {
    this.buf.clear()
    this.head = 0
    this.size = 0
  }

  /// The elements front to back.
  public fun toList(): List<T> {
    var out: List<T> = []
    loop (i in 0..<this.size) {
      out.push(this.slot(i) ?: panic("deque: a live slot is empty"))
    }
    out
  }

  /// Makes room for `n` elements in all, so adding up to `n` does not grow
  /// the buffer; never shrinks, never changes the contents (D105).
  public fun reserve(n: i64) {
    if (n > this.buf.len()) this.moveTo(n)
  }

  // Makes room for one more element: when the ring is full, the live
  // elements move to the start of a buffer twice the size.
  fun grow() {
    val cap = this.buf.len()
    if (this.size < cap) return
    this.moveTo(if (cap == 0) 4 else cap * 2)
  }

  // the live elements, in order, at the start of a buffer of `cap` slots
  fun moveTo(cap: i64) {
    val fresh: List<T?> = []
    loop (_ in 0..<cap) fresh.push(null)
    loop (i in 0..<this.size) fresh.set(i, this.slot(i))
    this.buf = fresh
    this.head = 0
  }

  implement Iterable {
    type Iter = DequeIter<T>
    fun iterator(): DequeIter<T> => DequeIter(buf: this.buf, head: this.head, size: this.size)
  }

  implement Display {
    fun toString(): string => "[${this.toList().join(", ")}]"
  }
}

/// The elements of a deque front to back, as they were when the iterator
/// was made: it holds a copy of the ring — O(1), the buffer copied only if
/// the deque changes meanwhile (D231, D234) — and walks it in place, with
/// no list built first.
public struct DequeIter<T> {
  buf:       List<T?>
  head:      i64
  size:      i64
  var index: i64 = 0

  implement Iterator {
    type Item = T
    fun next(): T? {
      if (this.index >= this.size) return null
      val at = this.head + this.index
      val slot = this.buf.get(if (at >= this.buf.len()) at - this.buf.len() else at) ?: panic("deque: a position below the capacity is inside the buffer")
      this.index += 1
      slot ?: panic("deque: a live slot is empty")
    }
  }
}

// ---------------------------------------------------------------------------
// PriorityQueue

/// A priority queue over a binary heap: `push` and `pop` are O(log n),
/// `peek` O(1). Elements that compare equal come out in no particular order.
///
/// `PriorityQueue<i64>.natural()` pops the smallest first;
/// `PriorityQueue<Job>(compare: (a, b) => a.due.compareTo(b.due))` orders
/// by `compare`, which returns `Ordering.Less` when its first argument
/// should come out first — for the largest first,
/// `PriorityQueue<i64>(compare: (a, b) => b.compareTo(a))`.
public struct PriorityQueue<T> {
  /// A binary heap: the smallest element (by `compare`) is at index 0 and
  /// the children of `i` are at `2i + 1` and `2i + 2`.
  private var items: List<T> = []
  /// Shared by the copies of a queue: a function cannot be changed (D234).
  /// Sendable, so a queue is Sendable when its elements are.
  private compare: sendable fun(T, T): Ordering

  init(compare: sendable fun(T, T): Ordering) {
    this.compare = compare
  }

  /// Number of elements.
  public fun len(): i64 => this.items.len()

  /// True when there are no elements.
  public fun isEmpty(): bool => this.items.len() == 0

  /// The element that `pop` would return, or `null` when empty.
  public fun peek(): T? => this.items.first()

  /// Adds `x`.
  public fun push(x: T) {
    val items = &this.items
    items.push(x)
    this.siftUp(items.len() - 1)
  }

  /// Removes and returns the first element in the queue's order, or `null`
  /// when empty.
  public fun pop(): T? {
    val items = &this.items
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
    this.items.clear()
  }

  /// A copy of this queue, with the same order (D236). O(1); the heap is
  /// copied at the first change of either queue. (Written by hand: an
  /// `init` with parameters has no generated `copy`, D164.)
  public fun copy(): PriorityQueue<T> => this

  /// The elements in heap order — the first is the smallest, the rest are
  /// not sorted.
  public fun toList(): List<T> => this.items

  /// The element at heap position `i`; the sifts only ask for positions
  /// below the heap's size.
  fun item(i: i64): T => this.items.get(i) ?: panic("heap: a sift reads only positions below the size")

  // The sifts move a hole, not the element: the element at `from` is held,
  // each element it passes moves one level into the hole, and it is stored
  // once where it stops — one write per level instead of a swap's two.

  fun siftUp(from: i64) {
    val x = this.item(from)
    var i = from
    loop (i > 0) {
      val parent = (i - 1) / 2
      val p = this.item(parent)
      if (this.compare(x, p) >= 0) break
      this.items.set(i, p)
      i = parent
    }
    this.items.set(i, x)
  }

  fun siftDown(from: i64) {
    val n = this.items.len()
    val x = this.item(from)
    var i = from
    loop {
      val left = 2 * i + 1
      if (left >= n) break
      var child = left
      var c = this.item(left)
      val right = left + 1
      if (right < n) {
        val r = this.item(right)
        if (this.compare(r, c) < 0) {
          child = right
          c = r
        }
      }
      if (this.compare(c, x) >= 0) break
      this.items.set(i, c)
      i = child
    }
    this.items.set(i, x)
  }

  implement Display {
    fun toString(): string => "[${this.items.join(", ")}]"
  }
}

extend<T: Comparable> PriorityQueue<T> {
  /// An empty queue over the elements' natural order: `pop()` yields the
  /// smallest first.
  public static fun natural(): PriorityQueue<T> => PriorityQueue<T>(compare: (a, b) => a.compareTo(b))
}
