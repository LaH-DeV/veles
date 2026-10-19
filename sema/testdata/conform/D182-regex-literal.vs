// D182: a /…/ literal is checked while compiling.
use io
use regex

fun main() {
  val good = /(?<year>\d{4})-(?<month>\d\d)/i
  val unicode = /\p{L}+\s\p{Greek}*/u
  val folding = /(?iu)straße/
  val unclosed = /(open/ // error: cannot be compiled: missing ')'
  val backref = /(\w)\1/ // error: backreferences are not supported
  val lookahead = /a(?=b)/ // error: lookahead is not supported
  val nope = /\p{Nope}/ // error: unknown Unicode class or script
  val half = 10 / 2 / 5
  io.println("${good.test("2026-10")} ${unicode.test("x")} ${folding.test("x")} ${unclosed.test("x")} ${backref.test("x")} ${lookahead.test("x")} ${nope.test("x")} $half")
}
