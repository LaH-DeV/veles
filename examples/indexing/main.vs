// Element access is methods only (D25): brackets build literals and
// nothing else. Reads return VALUES — `at` is the checked read, a `T?`,
// or a `T` where the checker can see the index is in range (D62); maps
// have `get` / `getOrDefault`. Writes go through `set`, or through a
// REFERENCE to the element: `ref` (`(*T)?`) and `loop (&x in xs)`. A read
// that cannot fail says why where it is written: `?: panic("…")`.
use io

struct Counter {
  var n: i64 = 0

  fun bump() {
    this.n += 1
  }

  fun show(): string = "n=${this.n}"
}

val picks = Atomic(value: 0)
fun pick(): i64 {
  val _ = picks.update(n => n + 1)
  0
}

fun main() {
  val xs = [10, 20, 30]
  io.println("${xs.at(1)} ${xs.at(-1)} ${xs.at(3)} ${xs.atOrDefault(3, -1)}")
  if (xs.len() >= 3) {
    val first: i64 = xs.at(0)  // i64, not i64?: the length was checked
    io.println("$first ${xs.at(-1)}")
  }

  var ys: MutableList<i64> = [1, 2, 3]
  ys.set(0, 5)
  ys.set(-1, (ys.at(-1) ?: panic("ys has three elements")) * 10)
  // a primitive element is changed through its reference
  val second = ys.ref(1) ?: panic("ys has three elements")
  *second += 40
  val third = ys.ref(-1) ?: panic("ys has three elements")
  *third *= 2
  io.println("$ys")
  // `fill` overwrites every element; `repeat` builds a list of copies
  ys.fill(1)
  io.println("$ys ${MutableList<string>.repeat("ab", 2)}")

  // a value struct read out of a list is a copy: changing the copy needs
  // a `var`, and leaves the list alone
  val counters: MutableList<Counter> = [Counter(), Counter()]
  var copy = counters.at(0) ?: panic("two counters")
  copy.n = 7
  io.println("${copy.n} ${counters.at(0)?.n}")
  // `ref` is a pointer to the element itself, so the struct is changed
  // where it lives; `?.` reaches it when the index is in range and does
  // nothing when it is not
  counters.ref(1)?.bump()
  counters.ref(0)?.n = 7
  val p = counters.ref(1) ?: panic("two counters")
  p.n += 100
  p.bump()
  counters.ref(1)?.bump()
  counters.ref(-1)?.bump()
  counters.ref(7)?.bump()
  counters.ref(0)?.n += 1
  counters.ref(9)?.n = 1000
  io.println("${counters.at(0)?.n} ${counters.at(1)?.n} ${counters.at(1)?.show()} ${counters.at(9)?.show()}")
  // `loop (&x in xs)` visits every element in place
  loop (&c in counters) c.bump()
  loop (&n in ys) *n += 1
  io.println("${counters.map(c => c.n)} $ys")

  // a list pattern checks the shape and binds the parts in one step
  val grid = [[1, 2], [3, 4]]
  val [_, [bottomLeft, _]] = grid else panic("grid is 2×2")
  io.println("$bottomLeft")

  val ages = ["ann": 41, "bob": 29]
  io.println("${ages.get("ann")} ${ages.get("zed")} ${ages.getOrDefault("zed", 0)} ${ages.get("bob")} ${ages.containsKey("zed")}")
  var stock: MutableMap<string, i64> = [:]
  stock.set("pears", 5)
  stock.set("pears", stock.getOrDefault("pears", 0) + 1)
  // a map entry is reached the same way: `ref(k)` points at the stored
  // value, so a value struct is updated in the map
  val tally: MutableMap<string, Counter> = ["hits": Counter()]
  val hits = tally.ref("hits") ?: panic("tally was built with 'hits'")
  hits.bump()
  tally.ref("hits")?.bump()
  tally.ref("hits")?.n += 10
  tally.ref("misses")?.bump()
  loop ((_, &c) in tally) c.bump()
  io.println("$stock $tally")

  // a nullable variable is written through after its null test, or with `?.`
  var maybe: Counter? = Counter()
  if (maybe != null) maybe.n = 5
  maybe?.n += 1
  var nothing: Counter? = null
  nothing?.n = 1
  io.println("$maybe $nothing")

  // a compound assignment locates its place once: the index or key
  // expression runs a single time
  *(ys.ref(pick()) ?: panic("ys is not empty")) += 1
  *(stock.ref(if (pick() == 0) "pears" else "") ?: panic("pears are in stock")) += 1
  io.println("${picks.load()} ${ys.at(0)} ${stock.get("pears")}")

  // a missing entry the program cannot do without is a panic, with the
  // reason — never a thrown error
  io.println("${ages.get("zed") ?: panic("no age for zed")}")
}
