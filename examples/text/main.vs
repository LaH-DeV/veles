// Splitting text and printing numbers: the runtime's fast paths (one pass
// for `split`, the shortest round-trip digits without a printf search, the
// exact decimal-to-float path) pinned against the cases where a fast path
// usually goes wrong.
use io

fun show(xs: List<string>): string = "${xs.len()}:" + xs.map(x => "[$x]").join("")

fun main() {
  // split: empty text, separators at both ends and next to each other, a
  // separator longer than one byte or than the text, non-ASCII
  io.println(show("".split(",")))
  io.println(show(",".split(",")))
  io.println(show("a,,b,".split(",")))
  io.println(show("a::b:::c".split("::")))
  io.println(show("ab".split("abc")))
  io.println(show("żółw→kot→pies".split("→")))
  io.println(show("x".split("")))

  // floats out: shortest digits that read back, plain notation for
  // ordinary magnitudes, an exponent outside them
  io.println("${0.1 + 0.2} ${1.0 / 3.0} ${123456.789} ${-0.0} ${0.0001} ${1e-5} ${1e14} ${1e15} ${2.5e-3}")
  io.println("${641755631438386.25} ${894.50421142578125} ${9007199254740993.0}")

  // floats in: the same numbers read back to the same values
  loop (text in ["0.1", "123456.789", "-0.0", "1e-5", "2.5e-3", "641755631438386.2", "1.7976931348623157e308", "4.9e-324", "0.30000000000000004"]) {
    val v = text.toF64() ?: 0.0
    io.println("$text -> $v")
  }
}
