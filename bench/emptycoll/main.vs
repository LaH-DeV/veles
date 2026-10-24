// Creating empty collections (D187): two million small records, each with an
// empty MutableList, an empty MutableMap and an empty List field; one in
// eight is used. The cost is what a value that stays empty takes.
use io { println }
use time

struct Record {
  items: MutableList<i64> = []
  index: MutableMap<i64, i64> = [:]
  tags:  List<i64> = []
}

fun main() {
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (i in 0..<2000000) {
      val r = Record()
      if (i % 8 == 0) {
        r.items.push(i)
        r.index.set(i, i)
      }
      sum += r.items.len() + r.index.len() + r.tags.len()
    }
    sum
  })
  println("BENCH emptycoll 2000000 ${elapsed.toNanos()} $sum")
}
