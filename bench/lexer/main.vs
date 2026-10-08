// Compiler-shaped: tokenise a large source text byte by byte (identifiers,
// a keyword table, numbers, strings, comments, punctuation) into a list of
// token structs. The text is built before the clock starts.
use io { println }
use time

enum Kind {
  Ident
  Keyword
  Number
  Str
  Comment
  Punct
}

struct Token {
  kind:  Kind
  start: i64
  end:   i64
}

val keywords: Map<string, i64> = [
  "fun": 1, "val": 2, "var": 3, "if": 4, "else": 5, "return": 6,
  "loop": 7, "when": 8, "struct": 9, "enum": 10, "use": 11, "throws": 12,
]

fun isIdentStart(b: u8): bool => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b >= 128
fun isDigit(b: u8): bool => b >= '0' && b <= '9'

fun tokenise(src: string): MutableList<Token> {
  val out: MutableList<Token> = []
  val n = src.len()
  var i = 0
  loop (i < n) {
    val b = src.byteAt(i)
    val start = i
    if (b == ' ' || b == '\n' || b == '\t') {
      i += 1
    } else if (isIdentStart(b)) {
      loop (i < n && (isIdentStart(src.byteAt(i)) || isDigit(src.byteAt(i)))) i += 1
      val word = src.substring(start, i) ?: ""
      val kind = if (keywords.get(word) == null) Kind.Ident else Kind.Keyword
      out.push(Token(kind, start, end: i))
    } else if (isDigit(b)) {
      loop (i < n && (isDigit(src.byteAt(i)) || src.byteAt(i) == '.')) i += 1
      out.push(Token(kind: Kind.Number, start, end: i))
    } else if (b == '"') {
      i += 1
      loop (i < n && src.byteAt(i) != '"') i += 1
      i += 1
      out.push(Token(kind: Kind.Str, start, end: i))
    } else if (b == '/' && i + 1 < n && src.byteAt(i + 1) == '/') {
      loop (i < n && src.byteAt(i) != '\n') i += 1
      out.push(Token(kind: Kind.Comment, start, end: i))
    } else {
      i += 1
      out.push(Token(kind: Kind.Punct, start, end: i))
    }
  }
  out
}

fun main() {
  val sb = StringBuilder()
  loop (i in 0..<20000) {
    sb.append("fun add$i(a: i64, b: i64): i64 {\n")
    sb.append("  val x = a + b * $i // scale\n")
    sb.append("  if (x > 10) return x else return \"s$i\"\n")
    sb.append("}\n")
  }
  val src = sb.toString()

  val stopwatch = time.Stopwatch.start()
  var check: i64 = 0
  var tokens: i64 = 0
  loop (_ in 0..<3) {
    val toks = tokenise(src)
    tokens += toks.len()
    loop (token in toks) check += (token.end - token.start) * (token.kind.value + 1)
  }
  println("BENCH lexer $tokens ${stopwatch.elapsed().toNanos()} $check")
}
