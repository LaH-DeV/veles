// D102, D231 addendum: a loop over a list, map or set walks it as it was
// when the loop began, so the body may change it; a loop that walks it by
/// reference may not change its length or order. A Deque is a value whose
// loop walks a copy, so its body may change it too (D234).
use io

struct Bag {
  items: List<i64>

  fun grow() {
    loop (x in this.items) {
      this.items.push(x)  // the loop walks the items as they were
    }
  }
}

fun lists(xs: List<i64>) {
  loop (x in xs) {
    if (x < 3) xs.push(x + 10)
    xs.set(0, x)
  }
  loop (&x in xs) {
    *x += 1
    xs.removeAt(0)  // error: 'xs.removeAt' changes 'xs' while the loop at line 22 walks it by reference
  }
  loop (&x in xs) {
    *x += 1
    xs.set(0, 1)  // replacing an element in place is allowed
    xs.reserve(100)  // reserve changes neither the length nor the order
  }
  loop (_ in xs) {
    loop (_ in xs) {
      xs.clear()
    }
  }
  loop (i in xs.indices()) xs.push(i)
  io.println("$xs")
}

fun maps(m: Map<string, i64>, s: Set<i64>) {
  loop ((_, n) in m) m.set("new", n)
  loop ((k, &n) in m) {
    *n += 1
    m.set(k, 0)  // the visited key: a replacement
    m.remove(k)  // error: 'm.remove' changes 'm' while the loop at line 42 walks it by reference
  }
  loop (x in s) s.add(x + 1)
  io.println("$m $s")
}

fun deques() {
  val d = Deque<i64>()
  d.addLast(1)
  loop (x in d) {
    d.addLast(x)  // the loop walks d as it was (D234)
  }
  io.println("$d")
}

fun main() {
  io.println("loops")
}
