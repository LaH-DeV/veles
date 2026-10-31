use fs
use io { println }
use os
use path
use time

fun main() throws IoError {
  val dir = path.join(os.tempDir(), "veles-bench-fslist")
  if (fs.exists(dir)) {
    val previous = try fs.listDir(dir)
    loop (name in previous) try fs.remove(path.join(dir, name))
    try fs.remove(dir)
  }
  try fs.mkdir(dir)
  loop (i in 0..<100) try fs.writeFile(path.join(dir, "entry-$i"), "")

  val (check, elapsed) = try time.measure(() => {
    var check: i64 = 0
    loop (_ in 0..<500) {
      val names = try fs.listDir(dir)
      check += names.len()
      loop (name in names) check += name.len()
    }
    check
  })

  val names = try fs.listDir(dir)
  loop (name in names) try fs.remove(path.join(dir, name))
  try fs.remove(dir)
  println("BENCH fslist 50000 ${elapsed.toNanos()} $check")
}
