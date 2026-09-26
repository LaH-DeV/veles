// Searching — indices are i64, "not found" is null (D5), never -1.
//
// The binary searches below are written out because that is what this
// example is for. Working code calls the prelude instead, which has the
// whole family as methods on any `List`: `xs.binarySearch(t)`,
// `xs.lowerBound(t)`, `xs.upperBound(t)`, `xs.binarySearchWith(compare)`,
// `xs.binarySearchBy(key, t)`, and `xs.partitionPoint(pred)` underneath
// them all. Those answer -1 for absent, as `indexOf` does. `preludeAgrees`
// at the bottom of this file checks the two against each other.

/// Linear scan: O(n), works on unsorted input. (`xs.indexOf(target)` is the
/// built-in spelling; it answers -1 rather than null.)
public fun linearSearch(xs: List<i64>, target: i64): i64? {
  loop ((i, x) in xs.iter().enumerate()) {
    if (x == target) return i
  }
  null
}

/// Binary search on a sorted list: O(log n).
public fun binarySearch(xs: List<i64>, target: i64): i64? {
  var lo = 0
  var hi = xs.len() - 1
  loop (lo <= hi) {
    val mid = lo + (hi - lo) / 2
    val v = xs.at(mid) ?: panic("binarySearch: lo <= mid <= hi < len")
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
public fun lowerBound(xs: List<i64>, target: i64): i64 {
  var lo = 0
  var hi = xs.len()
  loop (lo < hi) {
    val mid = lo + (hi - lo) / 2
    val v = xs.at(mid) ?: panic("lowerBound: lo <= mid < hi <= len")
    if (v < target) lo = mid + 1 else hi = mid
  }
  lo
}

/// Recursive binary search over an explicit range, for comparison.
public fun binarySearchRec(xs: List<i64>, target: i64, lo: i64, hi: i64): i64? {
  if (lo > hi) return null
  val mid = lo + (hi - lo) / 2
  val v = xs.at(mid) ?: panic("binarySearchRec: lo..hi must lie inside the list")
  when {
    v == target => mid
    v < target  => binarySearchRec(xs, target, mid + 1, hi)
    else        => binarySearchRec(xs, target, lo, mid - 1)
  }
}

/// The two implementations, on every target from below the smallest element
/// to above the largest — the only honest way to compare two searches, since
/// the interesting answers are the ones for values that are *not* there.
public fun preludeAgrees(xs: List<i64>): bool {
  val sorted = xs.sorted()
  loop (target in -2..12) {
    if (sorted.lowerBound(target) != lowerBound(sorted, target)) return false
    // the prelude answers -1 where this file answers null
    val here = binarySearch(sorted, target)
    val prelude = sorted.binarySearch(target)
    if ((here != null) != (prelude >= 0)) return false
    if (prelude >= 0 && sorted.at(prelude) != target) return false
    // and it always answers the first of several equal elements
    if (prelude >= 0 && prelude != sorted.lowerBound(target)) return false
    // the half-open range [lowerBound, upperBound) is exactly the equals
    if (sorted.upperBound(target) - sorted.lowerBound(target) != sorted.count(x => x == target)) return false
    // partitionPoint is the same question with the predicate spelled out
    if (sorted.partitionPoint(x => x < target) != sorted.lowerBound(target)) return false
  }
  true
}

/// Two-pointer: does a sorted list contain a pair summing to target?
public fun pairWithSum(xs: List<i64>, target: i64): (i64, i64)? {
  var i = 0
  var j = xs.len() - 1
  loop (i < j) {
    val a = xs.at(i) ?: panic("pairWithSum: 0 <= i < j < len")
    val b = xs.at(j) ?: panic("pairWithSum: 0 <= i < j < len")
    val sum = a + b
    when {
      sum == target => return (a, b)
      sum < target  => i += 1
      else          => j -= 1
    }
  }
  null
}
