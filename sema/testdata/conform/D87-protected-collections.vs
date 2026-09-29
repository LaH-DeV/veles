// D87: a `protected var` field of a mutable-collection type is looked at from
// outside its type — immutable methods, a loop head, interpolation — never
// taken or changed.
use io

struct Bag {
  public protected var items: MutableList<i64> = []
  public protected var index: MutableMap<string, i64> = [:]
  public protected var seen: MutableSet<i64> = []
  public var open: MutableList<i64> = []

  public fun add(n: i64) {
    this.items.push(n)
    this.index.set("$n", n)
    this.seen.add(n)
  }
  public fun total(): i64 {
    var sum = 0
    loop (x in this.items) { sum += x }
    sum
  }
}

fun looks(b: Bag): i64 {
  val n = b.items.len()
  val first = b.items.first() ?: 0
  val known = b.seen.contains(3)
  var sum = 0
  loop (x in b.items) { sum += x }
  io.println("${b.items} ${b.index.len()} $known")
  n + first + sum
}

fun copies(b: Bag): List<i64> = b.items.toList()

fun pushes(b: Bag) {
  b.items.push(1) // error: cannot call 'push' on 'Bag.items': the field is 'protected var'
}

fun clears(b: Bag) {
  b.index.clear() // error: cannot call 'clear' on 'Bag.index': the field is 'protected var'
}

fun adds(b: Bag) {
  b.seen.add(1) // error: cannot call 'add' on 'Bag.seen': the field is 'protected var'
}

fun swaps(b: Bag) {
  b.items.swap(0, 1) // error: cannot call 'swap' on 'Bag.items': the field is 'protected var'
}

fun refs(b: Bag) {
  val _ = b.items.ref(0) // error: cannot call 'ref' on 'Bag.items': the field is 'protected var'
}

fun binds(b: Bag) {
  val taken = b.items // error: cannot take 'Bag.items' out of 'Bag': the field is 'protected var'
  taken.push(1)
}

fun returns(b: Bag): MutableList<i64> = b.items // error: cannot take 'Bag.items' out of 'Bag': the field is 'protected var'

fun passes(b: Bag) {
  consume(b.items) // error: cannot take 'Bag.items' out of 'Bag': the field is 'protected var'
}

fun consume(xs: MutableList<i64>) {}

fun loopsByRef(b: Bag) {
  loop (&x in b.items) { *x += 1 } // error: cannot take 'Bag.items' out of 'Bag': the field is 'protected var'
}

fun unprotected(b: Bag) {
  b.open.push(1)
  val taken = b.open
  taken.push(2)
}

fun main() {
  io.println("protected collections")
}
