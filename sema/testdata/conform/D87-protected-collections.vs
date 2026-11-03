// D87, D231: a `protected var` field of a collection type is looked at or
// copied from outside its type — methods that do not change it, a loop
// head, interpolation, binding, passing, returning — never changed in place.
use io

struct Bag {
  protected var items: List<i64> = []
  protected var index: Map<string, i64> = [:]
  protected var seen:  Set<i64> = []
  var open:            List<i64> = []

  fun add(n: i64) {
    this.items.push(n)
    this.index.set("$n", n)
    this.seen.add(n)
  }
  fun total(): i64 {
    var sum = 0
    loop (x in this.items) {
      sum += x
    }
    sum
  }
}

fun looks(b: Bag): i64 {
  val n = b.items.len()
  val first = b.items.first() ?: 0
  val known = b.seen.contains(3)
  var sum = 0
  loop (x in b.items) {
    sum += x
  }
  io.println("${b.items} ${b.index.len()} $known")
  n + first + sum
}

fun copies(b: Bag): List<i64> => b.items  // a copy (D231)

fun pushes(b: Bag) {
  b.items.push(1)  // error: cannot call 'push' on 'Bag.items': the field is 'protected var'
}

fun clears(b: Bag) {
  b.index.clear()  // error: cannot call 'clear' on 'Bag.index': the field is 'protected var'
}

fun adds(b: Bag) {
  b.seen.add(1)  // error: cannot call 'add' on 'Bag.seen': the field is 'protected var'
}

fun swaps(b: Bag) {
  b.items.swap(0, 1)  // error: cannot call 'swap' on 'Bag.items': the field is 'protected var'
}

fun refs(b: Bag) {
  val _ = b.items.at(0)  // error: cannot call 'at' on 'Bag.items': the field is 'protected var'
}

fun binds(b: Bag) {
  var taken = b.items  // a copy of its own
  taken.push(1)
  io.println("$taken")
}

fun returns(b: Bag): List<i64> => b.items

fun passes(b: Bag) {
  consume(b.items)
}

fun consume(xs: List<i64>) { }

fun loopsByRef(b: Bag) {
  loop (&x in b.items) {  // error: cannot walk 'Bag.items' by reference: the field is 'protected var'
    *x += 1
  }
}

fun unprotected(b: Bag) {
  b.open.push(1)
  val taken = b.open
  taken.push(2)
}

fun main() {
  io.println("protected collections")
}
