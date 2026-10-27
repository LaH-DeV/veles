// D105: `reserve(n)` on every growable container — room for n in all,
// never shrinking, never changing the contents or their order.
use io { println }

fun main() {
  val sb = StringBuilder()
  sb.append("ab")
  sb.reserve(1000)
  sb.reserve(-1)  // a negative n does nothing
  loop (i in 0..<3) sb.append("$i")
  println("builder ${sb.toString()} ${sb.len()}")

  val m: MutableMap<string, i64> = ["one": 1, "two": 2, "three": 3]
  m.remove("two")  // a dead entry, compacted by reserve
  m.reserve(100)
  loop (i in 0..<5) m.set("k$i", i)
  m.reserve(2)  // never shrinks
  println("map ${m.keys()} ${m.len()} ${m.get("three")}")

  val s: MutableSet<i64> = [3, 1, 2]
  s.reserve(64)
  loop (i in 10..<13) s.add(i)
  println("set ${s.toList()} ${s.contains(1)}")

  val d = Deque<i64>()
  d.addLast(1)
  d.addFirst(0)
  d.reserve(50)
  loop (i in 2..<6) d.addLast(i)
  d.reserve(3)
  println("deque ${d.toList()} ${d.first()} ${d.last()}")

  val xs: MutableList<i64> = [1]
  xs.reserve(10)
  xs.push(2)
  println("list $xs")
}
