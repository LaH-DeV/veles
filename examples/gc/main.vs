use io

struct Node {
  value: i64
  next:  (*Node)?
}

fun build(n: i64): (*Node)? {
  var head: (*Node)? = null
  loop (i in 0..<n) {
    head = &Node(value: i, next: head)
  }
  head
}

fun sum(list: (*Node)?): i64 {
  var total: i64 = 0
  var cur = list
  loop (cur != null) {
    total += cur.value
    cur = cur.next
  }
  total
}

trait Named {
  fun name(): string
}
struct Thing {
  label: string
}
impl Named for Thing {
  fun name(): string = self.label
}

fun main() {
  // long-lived data must survive many collections while garbage churns
  val keep = build(2000)
  var keepMap: MutableMap<string, List<i64>> = [:]
  loop (i in 0..<50) {
    keepMap["k$i"] = [i as i64, (i * 2) as i64]
  }
  val boxes: List<Named> = [Thing(label: "a"), Thing(label: "b")]
  var n: i64 = 0
  val counter = () => {
    n += 1
    n
  }
  var checksum: i64 = 0
  loop (round in 0..<200) {
    val garbage = build(500)
    val strs = (0..<200).iterator().map(x => "item $x $round").toList()
    var m: MutableMap<i64, string> = [:]
    loop (s in strs) {
      m[s.len()] = s
    }
    checksum += sum(garbage) + m.len() as i64 + counter()
  }
  val words: MutableList<string> = []
  loop (i in 0..<2000) {
    words.push("w" + "$i")
  }
  io.println("keep ${sum(keep)} checksum $checksum")
  var total: i64 = 0
  loop ((_, v) in keepMap) {
    total += v[0] + v[1]
  }
  io.println("map ${keepMap.len()} $total ${keepMap["k7"] ?: []}")
  io.println("${boxes.map(b => b.name())} ${counter()} ${words.len()} ${words[1999]}")
}
