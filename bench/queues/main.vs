// A Deque as a 64-wide sliding window, walked with `loop` every 100 steps,
// and a PriorityQueue filled and drained: 50 rounds of 20k steps each.
use io { println }
use time

fun main() {
  val (sum, elapsed) = time.measure(() => {
    var sum: i64 = 0
    loop (_ in 0..<50) {
      val window = Deque<i64>()
      loop (i in 0..<20000) {
        window.addLast(i * 7919 % 1000)
        if (window.len() > 64) sum += window.removeFirst() ?: 0
        if (i % 100 == 0) {
          loop (x in window) sum += x
        }
      }
      val pq = PriorityQueue<i64>.natural()
      loop (i in 0..<20000) pq.push(i * 104729 % 100003)
      var last: i64 = -1
      loop {
        val x = pq.pop() ?: break
        if (x < last) panic("out of order")
        last = x
        sum += x
      }
    }
    sum
  })
  println("BENCH queues 3000000 ${elapsed.toNanos()} $sum")
}
