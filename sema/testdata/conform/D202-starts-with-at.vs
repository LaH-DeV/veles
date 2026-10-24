// D202: startsWith(prefix, startIndex = 0) — the second argument is a byte
// index; its name, when written, is `startIndex`.
fun arrows(s: string): List<bool> => [
  s.startsWith("=>", 1),
  s.startsWith("=>", startIndex: 1),
  s.startsWith(prefix: "=>", startIndex: 1),
  s.startsWith("=>", at: 1),  // error: startsWith has no parameter 'at' here: argument 2 is 'startIndex'
  s.startsWith("=>", 1, 2),   // error: startsWith takes a prefix and an optional startIndex, got 3 arguments: write s.startsWith(prefix) or s.startsWith(prefix, startIndex)
  s.startsWith("=>", "1"), // error: type mismatch: expected 'i64', found 'string'
]

fun main() {
  arrows("a=>b")
}
