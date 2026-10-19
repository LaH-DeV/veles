// A log analyser: parse structured lines with a regular expression, count
// the problems per module, find the slowest request and hide the e-mail
// addresses before the text is shared. The patterns are /…/ literals, checked
// while compiling (D182); one pattern comes from outside and goes through
// Regex.compile, which reports a bad one instead of failing.
use io
use regex { Match, Regex } extend { findAll }

val sample = """
  2026-10-19 09:15:02 INFO  [http] GET /users/42 200 took 18ms
  2026-10-19 09:15:03 WARN  [db] slow query on orders (840ms) for ana@example.com
  2026-10-19 09:15:04 ERROR [db] connection reset by peer
  2026-10-19 09:15:04 INFO  [http] POST /orders 201 took 95ms
  this line is not a log line at all
  2026-10-19 09:15:07 ERROR [http] GET /reports/7 500 took 2210ms user bob.smith@mail.example.org
  2026-10-19 09:15:09 ERROR [cache] eviction storm, 3 retries
  """

val line = /^(?<date>\d{4}-\d\d-\d\d) (?<time>[\d:]+) (?<level>[A-Z]+)\s+\[(?<module>\w+)\] (?<message>.*)$/
val duration = /(\d+)ms/
val email = /[\w.+-]+@[\w-]+(\.[\w-]+)+/

fun main() {
  val errors = MutableMap<string, i64>()
  var slowest = 0
  var slowestLine = ""
  var skipped = 0
  loop (text in sample.lines()) {
    val found = line.find(text)
    if (found == null) {
      skipped += 1
      continue
    }
    val m: Match = found
    val part = m.group("module") ?: "?"
    if (m.group("level") == "ERROR") errors.set(part, (errors.get(part) ?: 0) + 1)
    val ms = duration.find(m.group("message") ?: "")?.groupAt(1)?.toInt() ?: 0
    if (ms > slowest) {
      slowest = ms
      slowestLine = text
    }
  }
  io.println("lines that are not log lines: $skipped")
  loop ((part, count) in errors.entries().sortedBy(e => e.0)) {
    io.println("errors in $part: $count")
  }
  io.println("slowest: ${slowest}ms")
  io.println(slowestLine)

  io.println("--- shared version")
  loop (text in sample.lines()) {
    io.println(email.replaceAll(text, "<email>"))
  }

  io.println("--- patterns from outside")
  loop (pattern in [`\bERROR\b`, `(unclosed`, `\d+(ms|s)`, `(\w+)\1`]) {
    val re = Regex.compile(pattern) catch (e) {
      io.println("refused $pattern: ${e.message}")
      continue
    }
    io.println("$pattern matches ${sample.findAll(re).len()} times")
  }

  io.println("--- Unicode, and a trusted pattern that asks for more")
  io.println("words: ${"héllo wörld 日本語".findAll(/\p{L}+/).len()}")
  val doubled = Regex.compile(`\b(\w+) \1\b`, backtracking: true) catch (e) {
    panic(e.message)
  }
  io.println("repeated words: ${"it is is a a test".findAll(doubled).len()}")

  io.println("--- no input blows it up")
  val hostile = Regex.compile(`(a+)+$`) catch (e) {
    panic(e.message)
  }
  io.println("${hostile.test("a".repeat(5000) + "b")}")
}
