// D171: "${x:spec}" — [[fill]align][sign][#][0][width][,][.precision][type] —
// is read at compile time and checked against the type of `x`.
use io

fun main() {
  val s = "tea"
  val n: i64 = 255
  val f = 3.14159
  io.println("[${s:<8}] [${s:*^9}] [${n:08x}] [${n:#b}] [${n:+,d}] [${f:>10.2}] [${f:.1%}] [${true:^6}]")
  io.println("${s:.2}") // error: format spec '.2' does not fit a value of type 'string': only a fill, an alignment and a width apply to text
  io.println("${s:x}") // error: 'x' is for numbers; text takes s, or nothing
  io.println("${f:x}") // error: a float takes f or %
  io.println("${n:.2}") // error: a precision belongs to a float
  io.println("${n:#}") // error: '#' adds a prefix to x, b and o
  io.println("${n:q}") // error: format spec 'q' is not valid: unexpected 'q'
  io.println("${n:.}") // error: '.' needs the number of digits after it
  io.println("${n:e}") // error: the type 'e' is not available yet
  io.println("${n:>${n}}") // error: a format spec is fixed text for now
}
