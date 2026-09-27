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

  implement Named {
    fun name(): string = this.label
  }
}

fun main() {
  // long-lived data must survive many collections while garbage churns
  val keep = build(2000)
  var keepMap: MutableMap<string, List<i64>> = [:]
  loop (i in 0..<50) {
    keepMap.set("k$i", [i as i64, (i * 2) as i64])
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
      m.set(s.len(), s)
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
    val [a, b] = v else panic("gc: every entry holds two numbers")
    total += a + b
  }
  io.println("map ${keepMap.len()} $total ${keepMap.get("k7") ?: []}")
  io.println("${boxes.map(b => b.name())} ${counter()} ${words.len()} ${words.at(1999) ?: panic("gc: 2000 words were pushed")}")

  // many small short-lived objects between two collections: linear time.
  // The allocator once scanned every slot of every full span on each
  // allocation, which made this loop take minutes (bench/results.md).
  var bytes: i64 = 0
  loop (i in 0..<400000) bytes += "item-$i".bytes().len()
  io.println("small objects $bytes")

  // text built from many pieces: linear time too. `join` once appended to
  // an accumulator that it copied whole each time (80 000 numbers: 38 s),
  // and `replace` is a split and a join.
  val numbers = (0..<200000).iterator().toList()
  val joined = numbers.join(",")
  val replaced = joined.replace(",", "; ")
  val sb = stringBuilder()
  loop (n in numbers) sb.append("$n,")
  io.println("text ${joined.len()} ${replaced.len()} ${sb.len()} ${replaced.substring(0, 12) ?: ""}")
}
