use io { println }

// A user-defined iterable: a countdown (D42)
struct Countdown {
  from: i64

  implement Iterable {
    type Iter = CountdownIter
    fun iterator(): CountdownIter => CountdownIter(current: this.from)
  }
}
struct CountdownIter {
  var current: i64

  implement Iterator {
    type Item = i64
    fun next(): i64? {
      if (this.current <= 0) return null
      val v = this.current
      this.current -= 1
      v
    }
  }
}

fun summarize<I: Iterable>(xs: I): string {
  var parts: MutableList<string> = []
  loop (x in xs) {
    parts.push("$x")
  }
  parts.join("|")
}

fun main() {
  loop (n in Countdown(from: 3)) {
    println("t-minus $n")
  }
  println(summarize(Countdown(from: 4)))
  println(summarize([7, 8, 9]))

  // lazy adapters (D46) on iterators; materialised by toList()
  val nums = [1, 2, 3, 4, 5, 6]
  val evensSquared = nums.iter().filter(x => x % 2 == 0).map(x => x * x).toList()
  println("$evensSquared")
  println("${nums.iter().map(x => x * 10).take(2).toList()} ${nums.iter().skip(4).toList()}")
  println("${nums.iter().enumerate().map((i, x) => "$i:$x").toList()}")
  println("${nums.iter().zip(["a", "b", "c"].iter()).toList()}")
  println("count ${Countdown(from: 5).iterator().count()} sum ${nums.iter().fold(0, (a, b) => a + b)}")
  println("any ${nums.iter().any(x => x > 5)} all ${nums.iter().all(x => x > 0)} find ${nums.iter().find(x => x > 3) ?: -1} last ${nums.iter().last() ?: -1}")
  loop (x in (1..3).iterator().map(x => x * 100)) {
    println("mapped $x")
  }
  var total = 0
  Countdown(from: 3).iterator().forEach(x => total += x)
  println("total $total")
}
