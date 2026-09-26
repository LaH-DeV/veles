// Dynamic programming: a table, a recurrence, and a read-off at the end.

/// A rows×cols table of `value`; `make` builds a fresh row per
/// slot where `repeat` would share one row between all of them.
fun grid(rows: i64, cols: i64, value: i64): MutableList<MutableList<i64>> =
  MutableList<MutableList<i64>>.make(rows, _ => MutableList<i64>.repeat(value, cols))

fun cell(dp: MutableList<MutableList<i64>>, row: i64, col: i64): i64 {
  val r = dp.at(row) ?: panic("grid: row $row is outside the table")
  r.at(col) ?: panic("grid: column $col is outside the table")
}

fun setCell(dp: MutableList<MutableList<i64>>, row: i64, col: i64, v: i64) {
  val r = dp.at(row) ?: panic("grid: row $row is outside the table")
  r.set(col, v)
}

/// Length of the longest common subsequence of two strings (by bytes).
public fun lcsLength(a: string, b: string): i64 {
  val n = a.len()
  val m = b.len()
  val dp = grid(n + 1, m + 1, 0)
  loop (i in 1..n) {
    loop (j in 1..m) {
      val v = if (a.byteAt(i - 1) == b.byteAt(j - 1)) {
        cell(dp, i - 1, j - 1) + 1
      } else {
        cell(dp, i - 1, j).max(cell(dp, i, j - 1))
      }
      setCell(dp, i, j, v)
    }
  }
  cell(dp, n, m)
}

/// Levenshtein distance: insert, delete or replace one byte at a time.
public fun editDistance(a: string, b: string): i64 {
  val n = a.len()
  val m = b.len()
  val dp = grid(n + 1, m + 1, 0)
  loop (i in 0..n) setCell(dp, i, 0, i)
  loop (j in 0..m) setCell(dp, 0, j, j)
  loop (i in 1..n) {
    loop (j in 1..m) {
      val cost = if (a.byteAt(i - 1) == b.byteAt(j - 1)) 0 else 1
      val best = (cell(dp, i - 1, j) + 1)
        .min(cell(dp, i, j - 1) + 1)
        .min(cell(dp, i - 1, j - 1) + cost)
      setCell(dp, i, j, best)
    }
  }
  cell(dp, n, m)
}

/// 0/1 knapsack: best total value within `capacity`, one row per item.
public fun knapsack(weights: List<i64>, values: List<i64>, capacity: i64): i64 {
  val best = MutableList<i64>.repeat(0, capacity + 1)
  loop ((weight, value) in weights.zip(values)) {
    // walk capacities downwards so each item is used at most once
    loop (c in (weight..capacity).reversed()) {
      val keep = best.at(c) ?: panic("knapsack: weight <= c <= capacity")
      val take = best.at(c - weight) ?: panic("knapsack: weight <= c <= capacity")
      best.set(c, keep.max(take + value))
    }
  }
  best.at(capacity) ?: panic("knapsack: the table has capacity + 1 slots")
}

/// Fewest coins that make `amount`; null when it cannot be made.
public fun coinChange(coins: List<i64>, amount: i64): i64? {
  val fewest = MutableList<i64?>.repeat(null, amount + 1)
  fewest.set(0, 0)
  loop (target in 1..amount) {
    loop (coin in coins) {
      if (coin > target) continue
      val slot = fewest.at(target - coin) ?: panic("coinChange: 0 <= target - coin < target")
      val rest = slot ?: continue
      val known = fewest.at(target) ?: panic("coinChange: target <= amount")
      if (known == null || rest + 1 < known) fewest.set(target, rest + 1)
    }
  }
  fewest.at(amount) ?: panic("coinChange: the table has amount + 1 slots")
}

/// Longest strictly increasing subsequence, O(n²).
public fun longestIncreasing(xs: List<i64>): i64 {
  val best = MutableList<i64>.repeat(1, xs.len())
  loop (i in 1..<xs.len()) {
    loop (j in 0..<i) {
      val earlier = xs.at(j) ?: panic("longestIncreasing: j < i < len")
      if (earlier < xs.at(i)) {
        val here = best.at(i) ?: panic("longestIncreasing: best has one slot per element")
        val before = best.at(j) ?: panic("longestIncreasing: best has one slot per element")
        best.set(i, here.max(before + 1))
      }
    }
  }
  best.max() ?: 0
}

/// Kadane: largest sum of a contiguous run.
public fun maxSubarraySum(xs: List<i64>): i64 {
  var bestEndingHere = 0
  var best = xs.first() ?: 0
  loop (x in xs) {
    bestEndingHere = (bestEndingHere + x).max(x)
    best = best.max(bestEndingHere)
  }
  best
}

/// Number of ways to climb `n` stairs taking 1 or 2 steps at a time.
public fun climbStairs(n: i64): i64 {
  var a = 1
  var b = 1
  loop (_ in 0..<n) (a, b) = (b, a + b)
  a
}
