// Searching — indices are i64, "not found" is null (D5), never -1.

/// Linear scan: O(n), works on unsorted input.
pub fun linearSearch(xs: List<i64>, target: i64): i64? {
  loop ((i, x) in xs.iter().enumerate()) {
    if (x == target) return i
  }
  null
}

/// Binary search on a sorted list: O(log n).
pub fun binarySearch(xs: List<i64>, target: i64): i64? {
  var lo = 0
  var hi = xs.len() - 1
  loop (lo <= hi) {
    val mid = lo + (hi - lo) / 2
    val v = xs.atOrPanic(mid)
    when {
      v == target => return mid
      v < target  => lo = mid + 1
      else        => hi = mid - 1
    }
  }
  null
}

/// First index whose element is >= target (the insertion point that keeps
/// the list sorted); xs.len() when every element is smaller.
pub fun lowerBound(xs: List<i64>, target: i64): i64 {
  var lo = 0
  var hi = xs.len()
  loop (lo < hi) {
    val mid = lo + (hi - lo) / 2
    if (xs.atOrPanic(mid) < target) lo = mid + 1 else hi = mid
  }
  lo
}

/// Recursive binary search over an explicit range, for comparison.
pub fun binarySearchRec(xs: List<i64>, target: i64, lo: i64, hi: i64): i64? {
  if (lo > hi) return null
  val mid = lo + (hi - lo) / 2
  val v = xs.atOrPanic(mid)
  when {
    v == target => mid
    v < target  => binarySearchRec(xs, target, mid + 1, hi)
    else        => binarySearchRec(xs, target, lo, mid - 1)
  }
}

/// Two-pointer: does a sorted list contain a pair summing to target?
pub fun pairWithSum(xs: List<i64>, target: i64): (i64, i64)? {
  var i = 0
  var j = xs.len() - 1
  loop (i < j) {
    val s = xs.atOrPanic(i) + xs.atOrPanic(j)
    when {
      s == target => return (xs.atOrPanic(i), xs.atOrPanic(j))
      s < target  => i += 1
      else        => j -= 1
    }
  }
  null
}
