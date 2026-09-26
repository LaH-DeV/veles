// List patterns (D62): a list is matched by its shape, so a length check
// and the reads it guards are one step — there is no index to get wrong
// and nothing to panic on. `[a, b]` matches exactly two elements, `..`
// any number of them, and `..rest` binds those as a new List.
use io

sealed trait Shape
struct Circle : Shape {
  radius: f64
}
struct Square : Shape {
  side: f64
}

fun describe(args: List<string>): string = when (args) {
  []                  => "no arguments"
  ["help"]            => "help"
  [cmd]               => "just $cmd"
  ["run", target, ..] => "run $target"
  [cmd, ..rest]       => "$cmd with ${rest.len()} more"
}

fun ends(xs: List<i64>): string {
  val [first, .., last] = xs else return "fewer than two"
  "$first..$last"
}

fun middle(xs: List<i64>): List<i64> {
  val [_, ..inner, _] = xs else return []
  inner
}

// the jwt shape: the length check and the reads are the same statement
fun split3(token: string): string {
  val [header, payload, sig] = token.split(".") else return "malformed"
  "h=$header p=$payload s=$sig"
}

fun maybe(xs: List<i64>?): string = when (xs) {
  null    => "null"
  []      => "empty"
  [x, ..] => "starts with $x"
}

fun area(shapes: List<Shape>): f64 = when (shapes) {
  [Circle(radius)]   => radius * radius * 3.0
  [Square(side), ..] => side * side
  else               => 0.0
}

fun main() {
  io.println(describe([]))
  io.println(describe(["help"]))
  io.println(describe(["build"]))
  io.println(describe(["run", "tests", "-v"]))
  io.println(describe(["run"]))
  io.println(describe(["fmt", "a", "b"]))

  io.println(ends([1, 2, 3, 4]))
  io.println(ends([7, 8]))
  io.println(ends([9]))

  io.println("${middle([1, 2, 3, 4, 5])} ${middle([1, 2])} ${middle([1])}")

  io.println(split3("aa.bb.cc"))
  io.println(split3("aa.bb"))
  io.println(split3("a.b.c.d"))

  io.println(maybe(null))
  io.println(maybe([]))
  io.println(maybe([5, 6]))

  io.println("${area([Circle(radius: 1.0)])} ${area([Square(side: 2.0), Circle(radius: 1.0)])} ${area([])}")

  // nested: a list of pairs, and a list inside a tuple
  val pairs = [(1, "one"), (2, "two")]
  val [(n, name), ..] = pairs else return
  io.println("$n $name")
  val tagged = ("sum", [3, 4])
  when (tagged) {
    (label, [a, b]) => io.println("$label ${a + b}")
    else            => io.println("other")
  }

  // the names are copies taken at the match: a later push does not reach them
  var live: MutableList<i64> = [1, 2, 3]
  val [head, ..tail] = live else return
  live.push(4)
  live.set(0, 100)
  io.println("$head $tail ${live.len()}")
}
