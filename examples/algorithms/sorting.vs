// Sorting — every function returns a new List and leaves its input alone
// (List is immutable, D25); the in-place work happens on a MutableList copy,
// swapping with the prelude's `swap(i, j)`.

/// Bubble sort: O(n²), stops early when a pass makes no swap.
public fun bubbleSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  val n = out.len()
  loop (i in 0..<n) {
    var swapped = false
    loop (j in 0..<n - i - 1) {
      val a = out.at(j) ?: panic("bubbleSort: j + 1 < n - i, inside the list")
      val b = out.at(j + 1) ?: panic("bubbleSort: j + 1 < n - i, inside the list")
      if (a > b) {
        out.swap(j, j + 1)
        swapped = true
      }
    }
    if (!swapped) break
  }
  out.toList()
}

/// Insertion sort, generic over anything Comparable: O(n²), fast on
/// nearly-sorted input.
public fun insertionSort<T: Comparable>(xs: List<T>): List<T> {
  val out = xs.toMutable()
  loop (i in 1..<out.len()) {
    val key = out.at(i)
    var j = i - 1
    loop (j >= 0) {
      val prev = out.at(j) ?: panic("insertionSort: 0 <= j < i, inside the list")
      if (prev <= key) break
      out.set(j + 1, prev)
      j -= 1
    }
    out.set(j + 1, key)
  }
  out.toList()
}

/// Selection sort: O(n²) comparisons, at most n swaps.
public fun selectionSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  val n = out.len()
  loop (i in 0..<n) {
    var minAt = i
    loop (j in i + 1..<n) {
      val candidate = out.at(j) ?: panic("selectionSort: j < n")
      val best = out.at(minAt) ?: panic("selectionSort: minAt is an earlier j or i")
      if (candidate < best) minAt = j
    }
    if (minAt != i) out.swap(i, minAt)
  }
  out.toList()
}

/// Merge sort: O(n log n), stable, recursive on halves.
public fun mergeSort(xs: List<i64>): List<i64> {
  if (xs.len() <= 1) return xs
  val mid = xs.len() / 2
  merge(mergeSort(xs.take(mid)), mergeSort(xs.drop(mid)))
}

/// Interleaves two sorted lists; `<=` keeps equal elements in their
/// original order, which is what makes the sort stable.
fun merge(left: List<i64>, right: List<i64>): List<i64> {
  val out: MutableList<i64> = []
  var i = 0
  var j = 0
  loop (i < left.len() && j < right.len()) {
    if (left.at(i) <= right.at(j)) {
      out.push(left.at(i))
      i += 1
    } else {
      out.push(right.at(j))
      j += 1
    }
  }
  out.addAll(left.drop(i))
  out.addAll(right.drop(j))
  out.toList()
}

/// Quick sort: O(n log n) average, in place on a copy, Lomuto partition.
public fun quickSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  quickSortRange(out, 0, out.len() - 1)
  out.toList()
}

fun quickSortRange(xs: MutableList<i64>, lo: i64, hi: i64) {
  if (lo >= hi) return
  val pivot = xs.at(hi) ?: panic("quickSortRange: lo < hi, and both are inside the list")
  var store = lo
  loop (i in lo..<hi) {
    val x = xs.at(i) ?: panic("quickSortRange: lo <= i < hi")
    if (x < pivot) {
      xs.swap(i, store)
      store += 1
    }
  }
  xs.swap(store, hi)
  quickSortRange(xs, lo, store - 1)
  quickSortRange(xs, store + 1, hi)
}

/// Counting sort for small non-negative keys: O(n + k).
public fun countingSort(xs: List<i64>, maxValue: i64): List<i64> {
  val counts = MutableList<i64>.repeat(0, maxValue + 1)
  loop (x in xs) {
    val slot = counts.ref(x) ?: panic("countingSort: $x is outside 0..$maxValue")
    *slot += 1
  }
  val out: MutableList<i64> = []
  var v: i64 = 0
  loop (c in counts) {
    loop (_ in 0..<c) out.push(v)
    v += 1
  }
  out.toList()
}

public fun isSorted(xs: List<i64>): bool {
  var prev: i64? = null
  loop (x in xs) {
    if (prev != null && prev > x) return false
    prev = x
  }
  true
}
