// Compiler-shaped: build a large tree of a sealed family (one heap node per
// variant, children behind pointers), then walk it with `when` twice — once
// to evaluate and once to count. The walk is the shape of every compiler pass.
use io { println }
use time

sealed trait Expr
struct Num : Expr {
  value: i64
}
struct Neg : Expr {
  operand: *Expr
}
struct Add : Expr {
  left:  *Expr
  right: *Expr
}
struct Mul : Expr {
  left:  *Expr
  right: *Expr
}
struct Cond : Expr {
  test: *Expr
  yes:  *Expr
  no:   *Expr
}

fun build(depth: i64, seed: i64): Expr {
  if (depth == 0) return Num(value: seed % 7 + 1)
  val s = seed * 1103515245 + 12345
  val pick = (s >> 8) % 5
  val a = build(depth - 1, s % 1000003)
  val b = build(depth - 1, (s >> 3) % 1000003)
  when (pick) {
    0    => Neg(operand: &a)
    1    => Add(left: &a, right: &b)
    2    => Mul(left: &a, right: &b)
    3    => Cond(test: &a, yes: &b, no: &a)
    else => Add(left: &b, right: &a)
  }
}

fun eval(e: Expr): i64 => when (e) {
  is Num(value)          => value
  is Neg(operand)        => 0 - eval(*operand)
  is Add(left, right)    => (eval(*left) + eval(*right)) % 1000003
  is Mul(left, right)    => (eval(*left) * eval(*right)) % 1000003
  is Cond(test, yes, no) => if (eval(*test) % 2 == 0) eval(*yes) else eval(*no)
}

fun count(e: Expr): i64 => when (e) {
  is Num(_)              => 1
  is Neg(operand)        => 1 + count(*operand)
  is Add(left, right)    => 1 + count(*left) + count(*right)
  is Mul(left, right)    => 1 + count(*left) + count(*right)
  is Cond(test, yes, no) => 1 + count(*test) + count(*yes) + count(*no)
}

fun main() {
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  var nodes: i64 = 0
  loop (r in 0..<6) {
    val tree = build(17, r + 1)
    nodes += count(tree)
    loop (_ in 0..<4) check += eval(tree)
  }
  println("BENCH ast $nodes ${sw.elapsed().toNanos()} $check")
}
