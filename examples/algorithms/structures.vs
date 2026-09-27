// Classic data structures. The prelude already has the containers a
// program reaches for first — MutableList is a stack, Deque a queue, and
// PriorityQueue a heap — so this file shows them in use and builds the two
// that are worth writing out: a binary search tree over GC pointers and a
// binary heap, the structure behind PriorityQueue.

/// Balanced brackets — the textbook stack use, with a MutableList as the
/// stack: `push`, `pop` and `last` are all it needs.
public fun balanced(s: string): bool {
  val opener: Map<u8, u8> = [')': '(', ']': '[', '}': '{']
  val stack: MutableList<u8> = []
  loop (byte in s.bytes()) {
    when (byte) {
      '(', '[', '{' => stack.push(byte)
      ')', ']', '}' => {
        if (stack.pop() != opener.get(byte)) return false
      }
      else          => { }
    }
  }
  stack.isEmpty()
}

/// Largest element of every window of `k` — the textbook deque use. The
/// deque holds indices whose values decrease front to back, so the front
/// is always the current window's maximum: O(n) overall.
public fun slidingMax(xs: List<i64>, k: i64): List<i64> {
  val out: MutableList<i64> = []
  val window = Deque<i64>()
  loop ((i, x) in xs.iter().enumerate()) {
    // drop smaller elements from the back: they can never be a maximum again
    loop {
      val back = window.last() ?: break
      if ((xs.at(back) ?: panic("slidingMax: the window holds indexes of xs")) > x) break
      window.removeLast()
    }
    window.addLast(i)
    // drop the front once it has left the window
    if (window.first() == i - k) window.removeFirst()
    if (i >= k - 1) out.push(xs.at(window.first() ?: i) ?: panic("slidingMax: the window holds indexes of xs"))
  }
  out.toList()
}

/// Binary search tree of i64, nodes linked by GC pointers (D10/D31).
public struct TreeNode {
  value:     i64
  var left:  (*TreeNode)? = null
  var right: (*TreeNode)? = null
}

public struct Bst {
  var root: (*TreeNode)? = null
  var size: i64 = 0

  fun insert(v: i64) {
    if (this.contains(v)) return
    this.root = insertInto(this.root, v)
    this.size += 1
  }

  fun contains(v: i64): bool {
    var cur = this.root
    loop (cur != null) {
      if (v == cur.value) return true
      cur = if (v < cur.value) cur.left else cur.right
    }
    false
  }

  /// In-order walk yields the values sorted.
  fun inOrder(): List<i64> {
    val out: MutableList<i64> = []
    walk(this.root, out)
    out.toList()
  }

  fun height(): i64 = heightOf(this.root)

  /// The leftmost node holds the smallest value.
  fun min(): i64? {
    var cur = this.root ?: return null
    loop cur = cur.left ?: return cur.value
  }

  fun max(): i64? {
    var cur = this.root ?: return null
    loop cur = cur.right ?: return cur.value
  }
}

/// The subtree with `v` added; a null subtree becomes a leaf. Written as a
/// function of the subtree so the same code handles the root and any child.
fun insertInto(node: (*TreeNode)?, v: i64): *TreeNode {
  if (node == null) return &TreeNode(value: v)
  if (v < node.value) node.left = insertInto(node.left, v) else node.right = insertInto(node.right, v)
  node
}

fun walk(node: (*TreeNode)?, out: MutableList<i64>) {
  if (node == null) return
  walk(node.left, out)
  out.push(node.value)
  walk(node.right, out)
}

fun heightOf(node: (*TreeNode)?): i64 =
  if (node == null) 0 else 1 + heightOf(node.left).max(heightOf(node.right))

/// Binary min-heap in a list: the parent of `i` is `(i - 1) / 2`, its
/// children are `2i + 1` and `2i + 2`. The prelude's `PriorityQueue` is
/// this structure made generic and comparator-driven.
public struct MinHeap {
  items: MutableList<i64> = []

  /// The element at heap position `i`; the sifts read only positions below
  /// the size.
  fun item(i: i64): i64 = this.items.at(i) ?: panic("MinHeap: a sift reads only positions below the size")

  /// Appends, then sifts the new element up while it beats its parent.
  fun push(x: i64) {
    this.items.push(x)
    var i = this.items.len() - 1
    loop (i > 0) {
      val parent = (i - 1) / 2
      if (this.item(parent) <= this.item(i)) break
      this.items.swap(parent, i)
      i = parent
    }
  }

  /// Takes the root, moves the last element there and sifts it down.
  fun pop(): i64? {
    val top = this.items.first() ?: return null
    val last = this.items.pop() ?: return null
    if (this.items.isEmpty()) return top
    this.items.set(0, last)
    val n = this.items.len()
    var i = 0
    loop {
      val left = 2 * i + 1
      val right = left + 1
      var smallest = i
      if (left < n && this.item(left) < this.item(smallest)) smallest = left
      if (right < n && this.item(right) < this.item(smallest)) smallest = right
      if (smallest == i) break
      this.items.swap(i, smallest)
      i = smallest
    }
    top
  }

  fun peek(): i64? = this.items.first()
  fun len(): i64 = this.items.len()
}

/// Heap sort, as a demonstration of the heap: O(n log n).
public fun heapSort(xs: List<i64>): List<i64> {
  val heap = MinHeap()
  loop (x in xs) heap.push(x)
  val out: MutableList<i64> = []
  loop {
    val v = heap.pop() ?: break
    out.push(v)
  }
  out.toList()
}
