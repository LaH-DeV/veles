// D102: a collection cannot change while a loop walks it.
use io

struct Bag {
  items: MutableList<i64>

  fun grow() {
    loop (x in this.items) {
      this.items.push(x) // error: 'this.items.push' changes 'this.items' while the loop at line 8 walks it
    }
  }
}

fun lists(xs: MutableList<i64>) {
  loop (x in xs) {
    if (x < 3) xs.push(x + 10) // error: 'xs.push' changes 'xs' while the loop at line 15 walks it
    xs.set(0, x) // replacing an element in place is allowed
  }
  loop (&x in xs) {
    *x += 1
    xs.removeAt(0) // error: 'xs.removeAt' changes 'xs' while the loop at line 19 walks it
  }
  loop (_ in xs) {
    loop (_ in xs) {
      xs.clear() // error: 'xs.clear' changes 'xs' while the loop at line 24 walks it // error: 'xs.clear' changes 'xs' while the loop at line 23 walks it
    }
  }
  loop (_ in xs) {
    // a lambda passed as an argument runs here
    call(() => xs.sort()) // error: 'xs.sort' changes 'xs' while the loop at line 28 walks it
    // a stored one does not, until it is called
    val _ = () => xs.pop()
    xs.reserve(100) // reserve changes neither the length nor the order
  }
  loop (i in xs.indices()) xs.push(i) // a range walks no collection
  loop (x in xs.toList()) xs.push(x) // nor does a copy
}

fun call(f: fun()) => f()

fun maps(m: MutableMap<string, i64>, s: MutableSet<i64>) {
  loop ((k, n) in m) m.set(k, n + 1) // the visited key: a replacement
  loop ((_, n) in m) m.set("new", n) // error: 'm.set' changes 'm' while the loop at line 43 walks it
  loop ((k, &n) in m) { *n += 1; m.remove(k) } // error: 'm.remove' changes 'm' while the loop at line 44 walks it
  loop (x in s) s.add(x + 1) // error: 's.add' changes 's' while the loop at line 45 walks it
}

fun deques(d: Deque<i64>) {
  loop (x in d) d.addLast(x) // error: 'd.addLast' changes 'd' while the loop at line 49 walks it
}

fun main() {
  io.println("loops")
}
