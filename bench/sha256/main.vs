// SHA-256 over 1 MiB, 16 times: the hash is plain Veles (std/crypto).
use crypto
use io { println }
use time

fun main() {
  var data: MutableList<u8> = []
  data.reserve(1048576)
  loop (i in 0..<1048576) data.push((i * 31 % 256).wrapU8())
  val bytes = data.toList()
  val stopwatch = time.Stopwatch.start()
  var last = ""
  loop (_ in 0..<16) last = "${crypto.sha256(bytes)}"
  println("BENCH sha256 16 ${stopwatch.elapsed().toNanos()} ${last.substring(0, 16) ?: last}")
}
