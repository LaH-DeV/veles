// Element access is methods only (D25, v0.24): brackets build literals and
// nothing else. `at` is the checked read, `atOrPanic` the unchecked one,
// `set` the write; maps have `get` / `getOrPanic` / `getOrDefault` / `set`.
use io

struct Counter {
  n: i64 = 0

  mut fun bump() {
    self.n += 1
  }

  fun show(): string = "n=${self.n}"
}

fun main() {
  val xs = [10, 20, 30]
  io.println("${xs.at(1)} ${xs.at(-1)} ${xs.at(3)} ${xs.atOrDefault(3, -1)}")
  io.println("${xs.atOrPanic(0)} ${xs.atOrPanic(-1)}")

  var ys: MutableList<i64> = [1, 2, 3]
  ys.set(0, 5)
  ys.set(-1, ys.atOrPanic(-1) * 10)
  io.println("$ys")

  // `atOrPanic` names the element in place: a value struct is mutated
  // where it lives, and `&` takes its address
  val counters: MutableList<Counter> = [Counter(), Counter()]
  counters.atOrPanic(1).bump()
  counters.atOrPanic(1).bump()
  counters.atOrPanic(0).n = 7
  val p = &counters.atOrPanic(1)
  p.n += 100
  // so does a `?.` call through `at` / `first` / `last`: the element is
  // reached in place when the index is in range, skipped when it is not
  counters.at(1)?.bump()
  counters.at(-1)?.bump()
  counters.at(7)?.bump()
  counters.first()?.bump()
  counters.last()?.bump()
  io.println("${counters.atOrPanic(0).n} ${counters.atOrPanic(1).n} ${counters.at(1)?.show()} ${counters.at(9)?.show()}")

  val grid = [[1, 2], [3, 4]]
  io.println("${grid.atOrPanic(1).atOrPanic(0)}")

  val ages = ["ann": 41, "bob": 29]
  io.println("${ages.get("ann")} ${ages.get("zed")} ${ages.getOrDefault("zed", 0)} ${ages.getOrPanic("bob")}")
  var stock: MutableMap<string, i64> = [:]
  stock.set("pears", 5)
  stock.set("pears", stock.getOrPanic("pears") + 1)
  io.println("$stock")

  io.println("${ages.containsKey("zed")}")
  // out of range is a panic, never a thrown error
  io.println("${ages.getOrPanic("zed")}")
}
