// Line statistics over any io.Stream: the same `summarize` reads a file and a
// TCP connection (D128). The connection is served by a task that sends the
// file's text, so the call through the trait object really suspends while the
// bytes are on their way.

use fs
use io { println }
use net
use os
use path

struct Stats {
  var lines:   i64
  var words:   i64
  var longest: i64
}

fun summarize(s: io.Stream): Stats throws IoError | io.TooLong {
  var stats = Stats(lines: 0, words: 0, longest: 0)
  loop {
    val line = try s.readLine(4096)
    if (line == null) break
    stats.lines += 1
    stats.words += line.split(" ").filter(w => !w.isEmpty()).len()
    stats.longest = stats.longest.max(line.len())
  }
  stats
}

fun report(label: string, s: Stats) {
  println("$label: ${s.lines} lines, ${s.words} words, longest ${s.longest}")
}

// sends `text`, then closes its side so the reader sees the end
fun serve(listener: net.Listener, text: string) suspends throws IoError {
  with conn = try listener.accept()
  val s: io.Stream = conn
  try s.writeText(text)
  try s.shutdownWrite()
}

fun main() throws IoError | io.TooLong {
  val text = "the quick brown fox\njumps over\n\nthe lazy dog again and again\n"
  val file = path.join(os.tempDir(), "veles-streams-example.txt")
  try fs.writeFile(file, text)

  with f = try fs.open(file)
  report("file", try summarize(f))

  with listener = try net.listen()
  scope {
    val server = async serve(listener, text)
    with conn = try net.connect("127.0.0.1", listener.port())
    report("socket", try summarize(conn))
    await server
  }
  try fs.remove(file)
}
