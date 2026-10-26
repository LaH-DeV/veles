// D214: a local `var` declared without a value is not used — read, `&`,
// captured, a method called or a field written — until it is assigned on
// every path to that point. The facts join at `if`/`when`, and an
// assignment inside a loop body does not count after the loop.
use io

struct Box {
  var n: i64
  fun bump() {
    this.n += 1
  }
}

fun pick(): bool => "x".len() > 5

fun everyBranch(): i64 {
  var a: i64
  if (pick()) a = 1 else a = 2
  var b: string
  when (pick()) {
    true  => b = "t"
    false => b = "f"
  }
  var c: i64
  if (pick()) {
    c = 1
  } else {
    panic("a path that leaves needs no value")
  }
  a + b.len() + c
}

fun thenUsed(): i64 {
  var box: Box
  box = Box(n: 1)
  box.bump()
  box.bump() // a method that writes the value leaves it assigned
  box.n = 5
  val p = &box
  var count: i64
  count = 1
  count += 1
  var xs: List<i64>
  xs = [1]
  val size = () => xs.len()
  var early: i64
  if (!pick()) return 0
  early = 3
  p.n + count + size() + early
}

fun insideLambda(): i64 {
  var m: i64
  val f = () => {
    m = 1
    m
  }
  f() + m // error: 'm' is used before it is assigned on every path to here
}

fun oneBranch(): i64 {
  var a: i64
  if (pick()) a = 1
  a // error: 'a' is used before it is assigned on every path to here
}

fun afterLoop(): i64 {
  var c: i64
  loop (i in 0..<3) c = i
  c // error: 'c' is used before it is assigned on every path to here
}

fun neverAssigned(): i64 {
  var xs: List<i64>
  var box: Box
  box.n = 4 // error: 'box' is used before it is assigned on every path to here
  var other: Box
  other.bump() // error: 'other' is used before it is assigned on every path to here
  var g: i64
  g += 1 // error: 'g' is used before it is assigned on every path to here
  var h: i64
  val f = () => h // error: 'h' is used before it is assigned on every path to here
  var k: Box
  val q = &k // error: 'k' is used before it is assigned on every path to here
  xs.len() + f() + g + q.n // error: 'xs' is used before it is assigned on every path to here
}

fun oncePerPath(): i64 {
  var a: i64
  io.println("$a") // error: 'a' is used before it is assigned on every path to here
  a + a
}

fun main() {
  io.println("${everyBranch()} ${thenUsed()} ${insideLambda()} ${oneBranch()} ${afterLoop()} ${neverAssigned()} ${oncePerPath()}")
}
