use io.{ println }

struct Counter {
  n: i32 = 0
  step: i32 = 1

  fun get(): i32 = self.n
  mut fun bump() { self.n += self.step }
  fun withStep(s: i32): Counter = Counter(n: self.n, step: s)
}

struct Node { value: i32, next: (*Node)? }

fun sum(n: (*Node)?): i32 {
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
  println("c=${c.get()} d=${d.get()} e=${e.get()}")
  val list = Node(value: 1, next: &Node(value: 2, next: &Node(value: 3, next: null)))
  println("sum ${sum(&list)}")
  val p = &c
  p.bump()
  println("through pointer ${c.get()} ${p.n}")
  val pair = (1, "two")
  val (x, y) = pair
  println("tuple $x $y ${pair.1}")
  var xs: MutableList<i32> = []
  loop (i in 0..<5) { xs.push(i * i) }
  println("squares ${xs} len ${xs.len()} last ${xs[4]}")
}
