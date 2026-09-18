// Dynamic programming: a table, a recurrence, and a read-off at the end.

/// A rows×cols table filled with `fill`.
fun table(rows: i64, cols: i64, fill: i64): MutableList<MutableList<i64>> {
  var t: MutableList<MutableList<i64>> = []
  loop (_ in 0..<rows) {
    var row: MutableList<i64> = []
    loop (_ in 0..<cols) row.push(fill)
    t.push(row)
  }
  t
}

fun cell(t: MutableList<MutableList<i64>>, r: i64, c: i64): i64 = t.atOrPanic(r).atOrPanic(c)

fun setCell(t: MutableList<MutableList<i64>>, r: i64, c: i64, v: i64) {
  t.atOrPanic(r).set(c, v)
}

/// Length of the longest common subsequence of two strings (by bytes).
pub fun lcsLength(a: string, b: string): i64 {
  val n = a.len()
  val m = b.len()
  var t = table(n + 1, m + 1, 0)
  loop (i in 1..n) {
    loop (j in 1..m) {
      val v = if (a.byteAt(i - 1) == b.byteAt(j - 1)) {
        cell(t, i - 1, j - 1) + 1
      } else {
        cell(t, i - 1, j).max(cell(t, i, j - 1))
      }
      setCell(t, i, j, v)
    }
  }
  cell(t, n, m)
}

/// Levenshtein distance: insert, delete or replace one byte at a time.
pub fun editDistance(a: string, b: string): i64 {
  val n = a.len()
  val m = b.len()
  var t = table(n + 1, m + 1, 0)
  loop (i in 0..n) setCell(t, i, 0, i)
  loop (j in 0..m) setCell(t, 0, j, j)
  loop (i in 1..n) {
    loop (j in 1..m) {
      val cost = if (a.byteAt(i - 1) == b.byteAt(j - 1)) 0 else 1
      val best = (cell(t, i - 1, j) + 1)
        .min(cell(t, i, j - 1) + 1)
        .min(cell(t, i - 1, j - 1) + cost)
      setCell(t, i, j, best)
    }
  }
  cell(t, n, m)
}

/// 0/1 knapsack: best total value within `capacity`, one row per item.
pub fun knapsack(weights: List<i64>, values: List<i64>, capacity: i64): i64 {
  var best: MutableList<i64> = []
  loop (_ in 0..capacity) best.push(0)
  loop ((i, w) in weights.iter().enumerate()) {
    val v = values.atOrPanic(i)
    // walk capacities downwards so each item is used at most once
    var c = capacity
    loop (c >= w) {
      best.set(c, best.atOrPanic(c).max(best.atOrPanic(c - w) + v))
      c -= 1
    }
  }
  best.atOrPanic(capacity)
}

/// Fewest coins that make `amount`; null when it cannot be made.
pub fun coinChange(coins: List<i64>, amount: i64): i64? {
  var fewest: MutableList<i64?> = []
  loop (_ in 0..amount) fewest.push(null)
  fewest.set(0, 0)
  loop (a in 1..amount) {
    loop (c in coins) {
      if (c > a) continue
      val prev = fewest.atOrPanic(a - c)
      if (prev == null) continue
      val cur = fewest.atOrPanic(a)
      if (cur == null || prev + 1 < cur) fewest.set(a, prev + 1)
    }
  }
  fewest.atOrPanic(amount)
}

/// Longest strictly increasing subsequence, O(n²).
pub fun longestIncreasing(xs: List<i64>): i64 {
  if (xs.len() == 0) return 0
  var best: MutableList<i64> = []
  loop (_ in xs) best.push(1)
  loop (i in 1..<xs.len()) {
    loop (j in 0..<i) {
      if (xs.atOrPanic(j) < xs.atOrPanic(i)) {
        best.set(i, best.atOrPanic(i).max(best.atOrPanic(j) + 1))
      }
    }
  }
  best.max() ?: 0
}

/// Kadane: largest sum of a contiguous run.
pub fun maxSubarraySum(xs: List<i64>): i64 {
  var bestEndingHere = 0
  var best = xs.first() ?: 0
  loop (x in xs) {
    bestEndingHere = (bestEndingHere + x).max(x)
    best = best.max(bestEndingHere)
  }
  best
}

/// Number of ways to climb `n` stairs taking 1 or 2 steps at a time.
pub fun climbStairs(n: i64): i64 {
  var a = 1
  var b = 1
  loop (_ in 0..<n) (a, b) = (b, a + b)
  a
}
