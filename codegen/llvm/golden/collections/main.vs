// Lists, maps and sets: reads are copies, writes go through references.
use io

struct Counter {
  var n: i64
}

fun main() {
  val xs = [3, 1, 2]
  var ml: List<Counter> = []
  ml.push(Counter(n: 1))
  ml.push(Counter(n: 5))
  if (ml.len() == 2) ml.at(0).n += 10  // proven: a pointer, not a nullable one
  ml.set(1, Counter(n: 7))
  var m: Map<string, i64> = [:]
  m.set("a", 1)
  m.set("b", 2)
  *(m.at("a") ?: panic("a was set above")) += 40
  m.remove("b")
  var s = Set<i64>()
  s.add(4)
  s.add(4)
  io.println("${xs.get(1) ?: -1} ${xs.get(9) ?: -1} ${xs.sorted()} ${ml.get(0)?.n} ${ml.get(1)?.n}")
  io.println("${m.get("a") ?: 0} ${m.get("b") ?: 0} ${m.len()} ${s.len()} ${s.contains(4)}")
}
