// D95: `if (val x = e && ...)` binds a nullable's non-null value for the rest
// of the condition and the then-branch — and only there.
use io

fun find(name: string): string? => if (name == "n") "42" else null

fun binds() {
  // the name is the non-null value in the rest of the chain and the branch
  if (val n = find("n") && n.len() > 1) io.println(n.toUpper())
  if (val a = find("a")) io.println(a) else io.println("none")
  val label = if (val n = find("n")) n else "-"
  io.println(label)
}

fun notInElse() {
  if (val a = find("a")) io.println(a) else io.println(a) // error: unknown name 'a'
}

fun notAfter() {
  if (val a = find("a")) io.println(a)
  io.println(a) // error: unknown name 'a'
}

fun notNullable() {
  if (val n = 5) io.println("$n") // error: 'val n = ...' in a condition needs a value that can be null, and this is a 'i64'; a plain 'val' binds it (D95)
}

fun onlyInIf() {
  val flag = (val b = find("b")) // error: 'val b = ...' is only allowed in the condition of an 'if', joined to the rest by '&&' (D95)
  io.println("$flag")
}

fun notUnderOr() {
  if (val a = find("a") || find("b") != null) io.println("x") // error: 'val a = ...' is only allowed in the condition of an 'if', joined to the rest by '&&' (D95)
}

fun neverRead() {
  if (val a = find("a")) io.println("present") // warning: 'a' is never used
}

fun main() {
  binds()
}
