// A small binary file format, written the careful way: records are encoded
// with big endian numbers, the file is replaced atomically (a crash never
// leaves half of it), a header is read by seeking instead of loading the
// whole file, a lock keeps a second writer out, and a backup is copied.
//
//   offset 0   4 bytes  magic "SCOR"
//   offset 4   2 bytes  version, big endian
//   offset 6   4 bytes  number of records, big endian
//   then per record: 2 bytes name length, the name (UTF-8), 8 bytes score (signed)

use fs
use io { println }
use os
use path

struct Score {
  name:  string
  score: i64
}

fun encode(scores: List<Score>): List<u8> {
  val out: MutableList<u8> = "SCOR".bytes().toMutable()
  out.pushU16Be(1.wrapU16())
  out.pushU32Be(scores.len().wrapU32())
  loop (s in scores) {
    val name = s.name.bytes()
    out.pushU16Be(name.len().wrapU16())
    out.addAll(name)
    out.pushI64Be(s.score)
  }
  out.toList()
}

// Every record, or null when the bytes are not this format.
fun decode(bytes: List<u8>): List<Score>? {
  if (bytes.slice(0, 4) != "SCOR".bytes()) return null
  val count = bytes.readU32Be(6) ?: return null
  val scores: MutableList<Score> = []
  var at = 10
  loop (_ in 0..<count.toI64()) {
    val length = (bytes.readU16Be(at) ?: return null).toI64()
    val name = bytes.slice(at + 2, at + 2 + length).decodeUtf8() ?: return null
    val score = bytes.readI64Be(at + 2 + length) ?: return null
    scores.push(Score(name, score))
    at += 2 + length + 8
  }
  scores.toList()
}

// The header is one seek and one read; the rest of the file stays on disk.
// (The file is closed on return: Windows will not replace a file that is open.)
fun peekHeader(file: string) throws IoError {
  with f = try fs.open(file)
  try f.seek(4)
  val head = try f.readExact(6)
  println("version ${head.readU16Be(0)}, ${head.readU32Be(2)} records")
}

// A writer takes the lock first; a second one that only asks is told no.
fun lockTwice(file: string) throws IoError {
  with writer = try fs.open(file, fs.FileMode.Append)
  with held = try writer.lock()
  with asker = try fs.open(file, fs.FileMode.Append)
  println("a second lock is refused while it is held: ${(try asker.tryLock()) == null}")
}

fun main() throws IoError {
  val dir = path.join(os.tempDir(), "veles-binfile-example")
  if (!fs.isDir(dir)) try fs.mkdir(dir)
  val file = path.join(dir, "scores.bin")
  val scores = [Score(name: "ada", score: 1815), Score(name: "grace", score: -1906), Score(name: "linus", score: 4294967296)]

  // replaced whole: readers see the old file or the new one, never a mixture
  try fs.writeAtomic(file, encode(scores))
  println("wrote ${try fs.stat(file).size} bytes")

  try peekHeader(file)
  try lockTwice(file)
  val decoded = decode(try fs.readBytes(file)) ?: []
  loop (s in decoded) {
    println("  ${s.name}: ${s.score}")
  }

  // an update goes through the same door, and a backup is one call
  try fs.writeAtomic(file, encode(scores.take(2)))
  val backup = file + ".bak"
  try fs.copy(file, backup)
  println("after the update: ${(decode(try fs.readBytes(backup)) ?: []).len()} records, backup ${try fs.stat(backup).size} bytes")
  println("leftovers in the directory: ${try fs.listDir(dir)}")
  try fs.remove(backup)
  try fs.remove(file)
}
