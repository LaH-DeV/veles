use io { print, println }

struct Point {
  x: i64
  y: i64
}

fun wordCounts(text: List<string>): Map<string, i64> {
  var counts: MutableMap<string, i64> = [:]
  loop (w in text) {
    counts.set(w, (counts.get(w) ?: 0) + 1)
  }
  counts.toMap()
}

fun main() {
  val ages = ["ann": 41, "bob": 29]
  println("$ages len ${ages.len()} ann ${ages.get("ann") ?: -1} zed ${ages.get("zed") ?: -1} has ${ages.containsKey("bob")}")
  println("keys ${ages.keys()} values ${ages.values()} entries ${ages.entries()}")
  val counts = wordCounts(["a", "b", "a", "c", "a", "b"])
  println("$counts")
  loop ((word, n) in counts) {
    println("$word=$n")
  }
  var m = MutableMap<Point, string>()
  m.set(Point(x: 1, y: 2), "first")
  m.set(Point(x: 1, y: 2), "replaced")
  m.set(Point(x: 3, y: 4), "second")
  println("$m ${m.get(Point(x: 1, y: 2)) ?: "?"} removed ${m.remove(Point(x: 3, y: 4))} ${m.remove(Point(x: 9, y: 9))} $m")
  var seen = MutableSet<i64>()
  loop (x in [3, 1, 3, 2, 1]) {
    if (!seen.add(x as i64)) println("dup $x")
  }
  println("$seen ${seen.contains(2)} ${seen.len()} ${seen.toList()}")
  var byKey: MutableMap<(i64, bool), List<string>> = [:]
  byKey.set((1, true), ["x"])
  byKey.set((1, false), ["y", "z"])
  println("$byKey ${byKey.get((1, false)) ?: []}")
  val nested: Map<string, Map<string, i64>> = ["outer": ["inner": 7]]
  println("${nested.get("outer")?.get("inner") ?: 0}")
  loop (k in seen) {
    print("$k ")
  }
  println("")
  var big: MutableMap<i64, i64> = [:]
  loop (i in 0..<1000) {
    val k = i as i64
    big.set(k, k * k)
  }
  loop (i in 0..<500) {
    big.remove((i * 2) as i64)
  }
  println("${big.len()} ${big.get(999) ?: -1} ${big.get(998) ?: -1}")
  var stock = mut ["apples": 3, "pears": 0]
  stock.set("plums", 5)
  stock.remove("pears")
  val tags = mut ["fresh"]
  tags.push("local")
  println("$stock $tags")
}
