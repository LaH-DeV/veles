// `lazy val` (D210): a module value or a type's static value built on its
// first read, once, by whichever task gets there first; the others wait for
// it. A value nobody reads is never built, and a panic in the initializer is
// raised again by every later read.
use io { println }

val builds = Atomic(value: 0)

/// The words a lexer would look up: built when the first word is checked.
lazy val keywords = buildKeywords()

fun buildKeywords(): Set<string> {
  builds.add(1)
  println("building the keyword table")
  ["fun", "val", "var", "loop", "when", "struct"].toSet()
}

fun isKeyword(word: string): bool => keywords.contains(word)

fun count(word: string, found: Atomic<i64>) {
  if (isKeyword(word)) found.add(1)
}

/// A report nothing asks for: never built.
lazy val report: List<string> = buildReport()

fun buildReport(): List<string> {
  println("never printed")
  []
}

struct Zone {
  name:   string
  offset: i64

  /// Loaded once, on the first `Zone.utc`.
  static lazy val utc = Zone.load("UTC")

  static fun load(name: string): Zone {
    println("loading $name")
    Zone(name, offset: 0)
  }
}

lazy val broken: i64 = parseTable("12;x;4")

fun readBroken(): i64 => broken + 1

fun parseTable(text: string): i64 {
  var total = 0
  loop (part in text.split(";")) total += i64.parse(part) ?: panic("'$part' is not a number")
  total
}

fun main() {
  println("start")
  val words = ["fun", "if", "val", "x", "loop", "when", "y", "struct"]
  val found = Atomic(value: 0)
  scope {
    loop (word in words) {
      async count(word, found)
    }
  }
  println("${found.load()} keywords among ${words.len()} words; the table was built ${builds.load()} time")
  println("${Zone.utc.name} ${Zone.utc.offset}")
  loop (round in 1..2) {
    val r = gather {
      async readBroken()
    }
    when (r) {
      is Ok(v)  => println("round $round: $v")
      is Err(e) => println("round $round: ${e.message()}")
    }
  }
}
