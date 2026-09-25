// SHA-256 over 1 MiB, 16 times: the hash is plain Veles (std/crypto).
use crypto, io, time

fun main() {
  var data: MutableList<u8> = []
  loop (i in 0..<1048576) data.push((i * 31 % 256) as u8)
  val bytes = data.toList()
  val sw = time.Stopwatch.start()
  var last = ""
  loop (_ in 0..<16) last = "${crypto.sha256(bytes)}"
  io.println("BENCH sha256 16 ${sw.elapsed().toNanos()} ${last.substring(0, 16) ?: last}")
}
