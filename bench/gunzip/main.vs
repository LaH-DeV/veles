// gunzip of 2 MiB of word-salad text (std/compress, plain Veles): checks the
// CRC and the length too, as the Go reference does. The compressed bytes are
// each side's own, made before the clock starts.
use compress
use io { println }
use time

val words: List<string> = ["the", "of", "and", "stream", "request", "value", "error", "handler", "buffer", "index", "result", "compile", "token", "branch", "memory", "thread", "socket", "length", "window", "symbol", "a", "to", "in", "is"]

fun text(size: i64): List<u8> {
  val out: MutableList<u8> = []
  var x: u64 = 88172645463325252
  loop (out.len() < size) {
    x = x ^ (x << 13)
    x = x ^ (x >> 7)
    x = x ^ (x << 17)
    out.addAll((words.at((x % 24).wrapI64()) ?: "a").bytes())
    out.push(if (x % 11 == 0) 10 else 32)
  }
  out
}

fun main() {
  val data = text(2097152).toList()
  val packed = compress.gzip(data, 6)
  val sw = time.Stopwatch.start()
  var back: List<u8> = []
  loop (_ in 0..<8) back = compress.gunzip(packed) ?? []
  val took = sw.elapsed().toNanos()
  var sum: i64 = 0
  loop (b in back) sum = (sum * 31 + b.toI64()) % 1000000007
  println("BENCH gunzip ${data.len() * 8} $took $sum")
}
