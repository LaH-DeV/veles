// Compress a log file the way `gzip` would, in constant memory: the file is
// read a piece at a time, each piece goes through a GzipWriter onto the .gz
// file, and a GzipReader then reads the result back to count its lines and
// check it against the original.

use compress
use fs
use io { println }
use os
use path

// a day of an imaginary service's log, one line per request
fun logLine(i: i64): string {
  val route = when (i % 4) {
    0    => "/users/${i % 97}"
    1    => "/orders/${i * 7 % 1000}"
    2    => "/healthz"
    else => "/static/app.js"
  }
  "2026-10-04T12:${i / 60 % 60}:${i % 60} GET $route 200 ${i * 13 % 900 + 100}ms\n"
}

fun main() throws IoError | compress.CompressError {
  val log = path.join(os.tempDir(), "veles-gzipstream-example.log")
  val packed = log + ".gz"
  with out = try fs.open(log, fs.FileMode.Write)
  var bytes = 0
  loop (i in 0..<20000) {
    val line = logLine(i)
    bytes += line.len()
    try out.writeText(line)
  }
  println("wrote ${log.split("/").last() ?: log}: $bytes bytes")

  // file -> gzip -> file, 16 KiB at a time
  with source = try fs.open(log)
  with target = try fs.open(packed, fs.FileMode.Write)
  var writer = compress.GzipWriter(to: target, level: 6)
  loop {
    val chunk = try source.read(16384)
    if (chunk.isEmpty()) break
    try writer.write(chunk)
  }
  try writer.finish()
  val size = try fs.stat(packed).size
  println("compressed to $size bytes (${size * 100 / bytes}%)")

  // and back, counting lines as they come out
  with again = try fs.open(packed)
  var reader = compress.GzipReader(from: again)
  var lines = 0
  var total = 0
  var last = ""
  loop {
    val chunk = try reader.read()
    if (chunk.isEmpty()) break
    total += chunk.len()
    loop (b in chunk) {
      if (b == 10) lines += 1
    }
    last = chunk.decodeUtf8() ?: last
  }
  println("read back $total bytes, $lines lines; same size: ${total == bytes}")
  println("original: ${logLine(19999).trim()}")
  println("last piece ends with: ${(last.trim().split("\n").last() ?: "")}")
  try fs.remove(log)
  try fs.remove(packed)
}
