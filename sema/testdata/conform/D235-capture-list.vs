// D235: a capture list `[a, b = expr]` copies the listed values when the
// lambda is made; the rest is shared. Each call starts from its own copy,
// which the body may change in place but not rebind. What the list may name.
use io

struct Counter {
  var n: i64

  fun read(): i64 => this.n
  fun bump() { this.n += 1 }
}

fun fill(into: *List<i64>) {
  into.push(1)
}

struct Total {
  var sum: i64

  fun add(x: i64) { this.sum += x }
}

fun helper(): i64 => 1

fun main() {
  val xs: List<i64> = [1]
  val cs: List<Counter> = [Counter(n: 1)]
  val c = Counter(n: 0)
  var total = Total(sum: 0)
  val p = &total
  val reads = [xs, c, cs, first = xs.get(0) ?: 0] () => xs.len() + c.read() + first + (cs.at(0)?.n ?: 0)
  val writes = [p] (x: i64) => p.add(x)                    // a copied pointer writes through
  val nested = [xs] () => [1].map(y => y + xs.len())
  val changes = [xs, c] () => {                            // the call's own copies
    xs.push(2)
    fill(&xs)
    c.bump()
    val q = xs.at(0) ?: panic("one")
    *q = 5
    xs.len() + c.n
  }
  var k = 0
  val assigns = [k] () => { k = 5 }                        // error: cannot assign to 'k': a capture-list copy is a 'val' of each call
  val unknown = [nope] () => 1                             // error: 'nope' in the capture list is not a variable in scope
  val notVar = [helper] () => 1                            // error: 'helper' in the capture list is not a variable; write 'helper = …'
  val twice = [xs, xs] () => xs.len()                      // error: 'xs' is listed twice in the capture list
  val unused = [xs] () => 1                                // warning: 'xs' is copied into the lambda but the lambda never uses it
  writes(1)
  io.println("${reads()} ${nested()} ${changes()} ${assigns} ${unknown} ${notVar} ${twice()} ${unused()} $k")
}
