use io
use os
use time

fun main() throws IoError {
  if (os.args().at(0) == "--osrun-child") {
    io.print("h\u{E9}llo\n".repeat(16384))
    io.eprintln("error".repeat(8192))
    return
  }
  val program = os.program()
  val (sum, elapsed) = try time.measure(() => {
    var sum: i64 = 0
    loop (_ in 0..<20) {
      val output = try os.run(program, ["--osrun-child"])
      sum += output.stdout.len() + output.stderr.len()
    }
    sum
  })
  io.println("BENCH osrun 20 ${elapsed.toNanos()} $sum")
}
