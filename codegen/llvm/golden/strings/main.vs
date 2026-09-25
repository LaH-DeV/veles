// String building: interpolation of every printable kind, byte access,
// and the prelude's string methods.
use io

fun main() {
  val name = "veles"
  val n = 42
  val f = 2.5
  val ok = true
  val t = (1, "two")
  val s = "  a,b,c  "
  io.println("$name $n $f $ok $t ${name.len()} ${name.byteAt(0)} ${name.toUpper()}")
  io.println("${s.trim().split(",")} ${name.replace("e", "E")} ${name.indexOf("l")} ${"ab".repeat(3)}")
  val sb = stringBuilder()
  sb.append("x")
  sb.appendByte('y')
  io.println(sb.toString())
}
