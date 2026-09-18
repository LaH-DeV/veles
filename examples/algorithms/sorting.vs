// Sorting — every function returns a new List and leaves its input alone
// (List is immutable, D25); the in-place work happens on a MutableList copy,
// swapping with the prelude's `swap(i, j)`.

/// Bubble sort: O(n²), stops early when a pass makes no swap.
pub fun bubbleSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  val n = out.len()
  loop (i in 0..<n) {
    var swapped = false
    loop (j in 0..<n - i - 1) {
      if (out.atOrPanic(j) > out.atOrPanic(j + 1)) {
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
pub fun insertionSort<T: Comparable>(xs: List<T>): List<T> {
  val out = xs.toMutable()
  loop (i in 1..<out.len()) {
    val key = out.atOrPanic(i)
    var j = i - 1
    loop (j >= 0 && out.atOrPanic(j) > key) {
      out.set(j + 1, out.atOrPanic(j))
      j -= 1
    }
    out.set(j + 1, key)
  }
  out.toList()
}

/// Selection sort: O(n²) comparisons, at most n swaps.
pub fun selectionSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  val n = out.len()
  loop (i in 0..<n) {
    var minAt = i
    loop (j in i + 1..<n) {
      if (out.atOrPanic(j) < out.atOrPanic(minAt)) minAt = j
    }
    if (minAt != i) out.swap(i, minAt)
  }
  out.toList()
}

/// Merge sort: O(n log n), stable, recursive on halves.
pub fun mergeSort(xs: List<i64>): List<i64> {
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
    if (left.atOrPanic(i) <= right.atOrPanic(j)) {
      out.push(left.atOrPanic(i))
      i += 1
    } else {
      out.push(right.atOrPanic(j))
      j += 1
    }
  }
  out.addAll(left.drop(i))
  out.addAll(right.drop(j))
  out.toList()
}

/// Quick sort: O(n log n) average, in place on a copy, Lomuto partition.
pub fun quickSort(xs: List<i64>): List<i64> {
  val out = xs.toMutable()
  quickSortRange(out, 0, out.len() - 1)
  out.toList()
}

fun quickSortRange(xs: MutableList<i64>, lo: i64, hi: i64) {
  if (lo >= hi) return
  val pivot = xs.atOrPanic(hi)
  var store = lo
  loop (i in lo..<hi) {
    if (xs.atOrPanic(i) < pivot) {
      xs.swap(i, store)
      store += 1
    }
  }
  xs.swap(store, hi)
  quickSortRange(xs, lo, store - 1)
  quickSortRange(xs, store + 1, hi)
}

/// Counting sort for small non-negative keys: O(n + k).
pub fun countingSort(xs: List<i64>, maxValue: i64): List<i64> {
  val counts = MutableList<i64>.repeat(0, maxValue + 1)
  loop (x in xs) counts.atOrPanic(x) += 1
  val out: MutableList<i64> = []
  loop (v in 0..maxValue) {
    loop (_ in 0..<counts.atOrPanic(v)) out.push(v)
  }
  out.toList()
}

pub fun isSorted(xs: List<i64>): bool {
  loop (i in 1..<xs.len()) {
    if (xs.atOrPanic(i - 1) > xs.atOrPanic(i)) return false
  }
  true
}
