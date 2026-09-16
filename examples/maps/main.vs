use io

struct Point { x: i64, y: i64 }

fun wordCounts(text: List<string>): Map<string, i64> {
  var counts: MutableMap<string, i64> = mut [:]
  loop (w in text) {
    counts[w] = (counts[w] ?: 0) + 1
  }
  counts.toMap()
}

fun main() {
  val ages = ["ann": 41, "bob": 29]
  io.println("$ages len ${ages.len()} ann ${ages["ann"] ?: -1} zed ${ages["zed"] ?: -1} has ${ages.containsKey("bob")}")
  io.println("keys ${ages.keys()} values ${ages.values()} entries ${ages.entries()}")
  val counts = wordCounts(["a", "b", "a", "c", "a", "b"])
  io.println("$counts")
  loop ((word, n) in counts) { io.println("$word=$n") }
  var m = MutableMap<Point, string>()
  m[Point(x: 1, y: 2)] = "first"
  m.set(Point(x: 1, y: 2), "replaced")
  m[Point(x: 3, y: 4)] = "second"
  io.println("$m ${m.get(Point(x: 1, y: 2)) ?: "?"} removed ${m.remove(Point(x: 3, y: 4))} ${m.remove(Point(x: 9, y: 9))} $m")
  var seen = MutableSet<i64>()
  loop (x in [3, 1, 3, 2, 1]) {
    if (!seen.add(x as i64)) io.println("dup $x")
  }
  io.println("$seen ${seen.contains(2)} ${seen.len()} ${seen.toList()}")
  var byKey: MutableMap<(i64, bool), List<string>> = [:]
  byKey[(1, true)] = ["x"]
  byKey[(1, false)] = ["y", "z"]
  io.println("$byKey ${byKey[(1, false)] ?: []}")
  val nested: Map<string, Map<string, i64>> = ["outer": ["inner": 7]]
  io.println("${nested["outer"]?.get("inner") ?: 0}")
  loop (k in seen) { io.print("$k ") }
  io.println("")
  var big: MutableMap<i64, i64> = [:]
  loop (i in 0..<1000) { val k = i as i64; big[k] = k * k }
  loop (i in 0..<500) { big.remove((i * 2) as i64) }
  io.println("${big.len()} ${big[999] ?: -1} ${big[998] ?: -1}")
  var stock = mut ["apples": 3, "pears": 0]
  stock["plums"] = 5
  stock.remove("pears")
  val tags = mut ["fresh"]
  tags.push("local")
  io.println("$stock $tags")
}
