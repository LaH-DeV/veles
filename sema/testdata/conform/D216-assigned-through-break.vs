// D216: what is assigned after a loop is what is assigned on every way out
// of it. A `loop { }` with no condition ends only by `break`, so a variable
// assigned before every `break` that leaves it — a labelled one from a loop
// inside included — is assigned after it. A loop with a condition, or over a
// collection, can also end normally, where only what held before it holds.
use io

fun firstNegative(xs: List<i64>): i64 {
  var found: i64
  var i = 0
  loop {
    if (i == xs.len()) {
      found = -1
      break
    }
    if ((xs.at(i) ?: 0) < 0) {
      found = i
      break
    }
    i += 1
  }
  found
}

fun missesOne(xs: List<i64>): i64 {
  var found: i64
  var i = 0
  loop {
    if (i == xs.len()) break
    if ((xs.at(i) ?: 0) < 0) {
      found = i
      break
    }
    i += 1
  }
  found  // error: 'found' is used before it is assigned on every path to here
}

fun rowWithSeven(grid: List<List<i64>>): i64 {
  var hit: i64
  var r = 0
  loop :rows {
    if (r == grid.len()) {
      hit = -1
      break
    }
    loop (x in grid.at(r) ?: []) {
      if (x == 7) {
        hit = r
        break rows
      }
    }
    r += 1
  }
  hit
}

fun overTheList(xs: List<i64>): i64 {
  var found: i64
  loop (x in xs) {
    if (x > 0) {
      found = x
      break
    }
  }
  found  // error: 'found' is used before it is assigned on every path to here
}

fun firstPositiveOrNone(xs: List<i64>): i64 {
  var found: i64
  var i = 0
  loop {
    if (i == xs.len()) return -1
    if ((xs.at(i) ?: 0) > 0) {
      found = i
      break
    }
    i += 1
  }
  found
}

fun main() {
  io.println("${firstNegative([3, -1])} ${missesOne([1])} ${rowWithSeven([[1], [7]])} ${overTheList([1])} ${firstPositiveOrNone([0, 2])}")
}
