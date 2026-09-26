use io

/**
 * A simple example of structs, including a recursive struct.
 * It also shows how to use a mutable struct and a pointer to it.
 */
struct Counter {
  var n: i64 = 0
  step:  i64 = 1

  fun get(): i64 = this.n
  /** testing doc for bump */
  fun bump() {
    this.n += this.step
  }
  fun withStep(s: i64): Counter = Counter(n: this.n, step: s)
}

struct Node {
  value: i64
  next:  (*Node)?
}

fun sum(n: (*Node)?): i64 {
  var total = 0
  var cur = n
  loop (cur != null) {
    total += cur.value
    cur = cur.next
  }
  total
}

fun main() {
  var c = Counter()
  c.bump()
  c.bump()
  val d = c.withStep(10)
  var e = d
  e.bump()
  io.println("c=${c.get()} d=${d.get()} e=${e.get()}")
  val list = Node(value: 1, next: &Node(value: 2, next: &Node(value: 3, next: null)))
  io.println("sum ${sum(&list)}")
  val p = &c
  p.bump()
  io.println("through pointer ${c.get()} ${p.n}")
  val pair = (1, "two")
  val (x, y) = pair
  io.println("tuple $x $y ${pair.1}")
  var xs: MutableList<i64> = []
  loop (i in 0..<5) {
    xs.push(i * i)
  }
  io.println("squares ${xs} len ${xs.len()} last ${xs.at(4) ?: panic("five squares were pushed")}")
}
