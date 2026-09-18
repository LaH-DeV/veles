// Classic data structures built from the prelude collections and pointers.

/// LIFO on top of a MutableList.
pub struct Stack<T> {
  items: MutableList<T> = []

  fun push(x: T) {
    self.items.push(x)
  }
  fun pop(): T? = self.items.pop()
  fun peek(): T? = self.items.last()
  fun len(): i64 = self.items.len()
  fun isEmpty(): bool = self.items.len() == 0
}

/// FIFO with a moving head; compacts once half the buffer is dead.
pub struct Queue<T> {
  items: MutableList<T> = []
  head:  i64 = 0

  fun enqueue(x: T) {
    self.items.push(x)
  }

  mut fun dequeue(): T? {
    val x = self.items.at(self.head) ?: return null
    self.head += 1
    if (self.head * 2 >= self.items.len()) {
      val rest = self.items.drop(self.head)
      self.items.clear()
      self.items.addAll(rest)
      self.head = 0
    }
    x
  }

  fun len(): i64 = self.items.len() - self.head
}

/// Balanced parentheses — the textbook stack use.
pub fun balanced(s: string): bool {
  var stack = Stack<u8>()
  loop (i in 0..<s.len()) {
    val b = s.byteAt(i)
    when (b) {
      '(', '[', '{' => stack.push(b)
      ')'           => {
        if (stack.pop() != '(') return false
      }
      ']' => {
        if (stack.pop() != '[') return false
      }
      '}' => {
        if (stack.pop() != '{') return false
      }
      else => { }
    }
  }
  stack.isEmpty()
}

/// Binary search tree of i64, nodes linked by GC pointers (D10/D31).
pub struct TreeNode {
  value: i64
  left:  (*TreeNode)? = null
  right: (*TreeNode)? = null
}

pub struct Bst {
  root: (*TreeNode)? = null
  size: i64 = 0

  mut fun insert(v: i64) {
    val r = self.root
    if (r == null) {
      self.root = &TreeNode(value: v)
      self.size += 1
      return
    }
    var cur = r
    loop {
      if (v == cur.value) return
      if (v < cur.value) {
        val l = cur.left
        if (l == null) {
          cur.left = &TreeNode(value: v)
          self.size += 1
          return
        }
        cur = l
      } else {
        val rr = cur.right
        if (rr == null) {
          cur.right = &TreeNode(value: v)
          self.size += 1
          return
        }
        cur = rr
      }
    }
  }

  fun contains(v: i64): bool {
    var cur = self.root
    loop (cur != null) {
      if (v == cur.value) return true
      cur = if (v < cur.value) cur.left else cur.right
    }
    false
  }

  /// In-order walk yields the values sorted.
  fun inOrder(): List<i64> {
    var out: MutableList<i64> = []
    walk(self.root, out)
    out.toList()
  }

  fun height(): i64 = heightOf(self.root)

  fun min(): i64? {
    var cur = self.root ?: return null
    loop {
      val l = cur.left ?: return cur.value
      cur = l
    }
  }
}

fun walk(node: (*TreeNode)?, out: MutableList<i64>) {
  if (node == null) return
  walk(node.left, out)
  out.push(node.value)
  walk(node.right, out)
}

fun heightOf(node: (*TreeNode)?): i64 =
  if (node == null) 0 else 1 + heightOf(node.left).max(heightOf(node.right))

/// Binary min-heap in a list: parent of i is (i - 1) / 2.
pub struct MinHeap {
  items: MutableList<i64> = []

  fun push(x: i64) {
    self.items.push(x)
    var i = self.items.len() - 1
    loop (i > 0) {
      val parent = (i - 1) / 2
      if (self.items.atOrPanic(parent) <= self.items.atOrPanic(i)) break
      self.swap(parent, i)
      i = parent
    }
  }

  fun pop(): i64? {
    val n = self.items.len()
    if (n == 0) return null
    val top = self.items.atOrPanic(0)
    val last = self.items.pop() ?: return null
    if (n == 1) return top
    self.items.set(0, last)
    var i = 0
    loop {
      val l = 2 * i + 1
      val r = l + 1
      var smallest = i
      if (l < n - 1 && self.items.atOrPanic(l) < self.items.atOrPanic(smallest)) smallest = l
      if (r < n - 1 && self.items.atOrPanic(r) < self.items.atOrPanic(smallest)) smallest = r
      if (smallest == i) break
      self.swap(i, smallest)
      i = smallest
    }
    top
  }

  fun peek(): i64? = self.items.first()
  fun len(): i64 = self.items.len()

  fun swap(i: i64, j: i64) {
    (self.items.atOrPanic(i), self.items.atOrPanic(j)) = (self.items.atOrPanic(j), self.items.atOrPanic(i))
  }
}

/// Heap sort, as a demonstration of the heap: O(n log n).
pub fun heapSort(xs: List<i64>): List<i64> {
  var heap = MinHeap()
  loop (x in xs) heap.push(x)
  var out: MutableList<i64> = []
  loop {
    val v = heap.pop() ?: break
    out.push(v)
  }
  out.toList()
}
