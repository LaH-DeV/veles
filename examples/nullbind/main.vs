// nullbind: `if (val x = e)` (D95) runs code only when a nullable has a value,
// with the value bound — in the rest of the condition and in the branch, and
// nowhere else. Bindings chain with `&&`, and the chain stops at the first
// operand that fails: a later value is not even computed.
use io { println }

struct Cookie {
  name:   string
  maxAge: Duration? = null
  domain: string? = null
}

fun header(name: string): string? = if (name == "n") "42" else null

fun trace(text: string, result: bool): bool {
  println("  tested $text")
  result
}

fun line(c: Cookie): string {
  val out = StringBuilder()
  out.append(c.name)
  if (val age = c.maxAge) out.append("; Max-Age=${age.toSeconds()}")
  if (val d = c.domain) out.append("; Domain=$d") else out.append("; no domain")
  out.toString()
}

// a binding and a test on it in one condition, and an early return that a
// lambda-based `?.let` could not make
fun firstBig(name: string): string {
  if (val n = header(name)?.toInt() && n > 40) return "big $n"
  "small or missing"
}

fun main() {
  println(line(Cookie(name: "a")))
  println(line(Cookie(name: "a", maxAge: Duration.seconds(5), domain: "x.org")))
  println(firstBig("n"))
  println(firstBig("zz"))

  // as an expression
  val label = if (val n = header("n")) "n=$n" else "none"
  println(label)

  // evaluation order: a failed first operand skips the binding's value
  if (trace("first", false) && val v = header("n")) println("never $v")
  if (trace("first", true) && val v = header("n") && trace("third", v == "42")) println("all three, v=$v")

  // two bindings in one chain, the second using the first
  if (val a = header("n") && val b = a.toInt() && b > 1) println("a=$a b=$b")

  // an else-if chain: the first binding is not visible in the else
  if (val x = header("zz")) println("x $x") else if (val y = header("n")) println("y $y") else println("neither")
}
