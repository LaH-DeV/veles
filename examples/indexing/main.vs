// Element access is methods only (D25): brackets build literals and
// nothing else. Reads return VALUES — `at` is the checked read (`T?`),
// `atOrPanic` the unchecked one (`T`); maps have `get` / `getOrPanic` /
// `getOrDefault`. Writes go through `set`, or through a REFERENCE to the
// element: `ref` (`(*T)?`), `refOrPanic` (`*T`) and `loop (&x in xs)`.
// A read and a write never look alike, so `val t = xs.atOrPanic(0)` and
// `xs.refOrPanic(0)` say what they do.
use io

struct Counter {
  n: i64 = 0

  mut fun bump() {
    self.n += 1
  }

  fun show(): string = "n=${self.n}"
}

var picks = 0
fun pick(): i64 {
  picks += 1
  0
}

fun main() {
  val xs = [10, 20, 30]
  io.println("${xs.at(1)} ${xs.at(-1)} ${xs.at(3)} ${xs.atOrDefault(3, -1)}")
  io.println("${xs.atOrPanic(0)} ${xs.atOrPanic(-1)}")

  var ys: MutableList<i64> = [1, 2, 3]
  ys.set(0, 5)
  ys.set(-1, ys.atOrPanic(-1) * 10)
  // a primitive element is changed through its reference
  *ys.refOrPanic(1) += 40
  *ys.refOrPanic(-1) *= 2
  io.println("$ys")
  // `fill` overwrites every element; `repeat` builds a list of copies
  ys.fill(1)
  io.println("$ys ${MutableList<string>.repeat("ab", 2)}")

  // a value struct read out of a list is a copy: changing the copy needs
  // a `var`, and leaves the list alone
  val counters: MutableList<Counter> = [Counter(), Counter()]
  var copy = counters.atOrPanic(0)
  copy.n = 7
  io.println("${copy.n} ${counters.atOrPanic(0).n}")
  // `refOrPanic` is a pointer to the element itself, so the struct is
  // changed where it lives; a `val` may hold the pointer
  counters.refOrPanic(1).bump()
  counters.refOrPanic(0).n = 7
  val p = counters.refOrPanic(1)
  p.n += 100
  p.bump()
  // `ref` is the nullable pointer: `?.` reaches the element when the index
  // is in range and does nothing when it is not
  counters.ref(1)?.bump()
  counters.ref(-1)?.bump()
  counters.ref(7)?.bump()
  counters.ref(0)?.n += 1
  counters.ref(9)?.n = 1000
  io.println("${counters.atOrPanic(0).n} ${counters.atOrPanic(1).n} ${counters.at(1)?.show()} ${counters.at(9)?.show()}")
  // `loop (&x in xs)` visits every element in place
  loop (&c in counters) c.bump()
  loop (&n in ys) *n += 1
  io.println("${counters.map(c => c.n)} $ys")

  val grid = [[1, 2], [3, 4]]
  io.println("${grid.atOrPanic(1).atOrPanic(0)}")

  val ages = ["ann": 41, "bob": 29]
  io.println("${ages.get("ann")} ${ages.get("zed")} ${ages.getOrDefault("zed", 0)} ${ages.getOrPanic("bob")} ${ages.containsKey("zed")}")
  var stock: MutableMap<string, i64> = [:]
  stock.set("pears", 5)
  *stock.refOrPanic("pears") += 1
  // a map entry is reached the same way: `refOrPanic` and `ref(k)?.` point
  // at the stored value, so a value struct is updated in the map
  val tally: MutableMap<string, Counter> = ["hits": Counter()]
  tally.refOrPanic("hits").bump()
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
  *ys.refOrPanic(pick()) += 1
  *stock.refOrPanic(if (pick() == 0) "pears" else "") += 1
  io.println("$picks ${ys.atOrPanic(0)} ${stock.getOrPanic("pears")}")

  // out of range is a panic, never a thrown error
  io.println("${ages.getOrPanic("zed")}")
}
