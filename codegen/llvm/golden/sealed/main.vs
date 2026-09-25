// A sealed type through a pointer, dispatch by `when`, smart casts,
// nullable values and results.
use io

sealed trait Expr {
  fun eval(): i64 = when (self) {
    is Num(v) => v
    is Add(l, r) => l.eval() + r.eval()
    is Neg(e) => -e.eval()
  }
}
struct Num : Expr { v: i64 }
struct Add : Expr { l: *Expr; r: *Expr }
struct Neg : Expr { e: *Expr }

error TooBig { n: i64 }

fun check(n: i64): i64 throws TooBig = if (n > 100) throw TooBig(n) else n

fun half(n: i64): i64? = if (n % 2 == 0) n / 2 else null

fun main() {
  val one: Expr = Num(v: 1)
  val two: Expr = Num(v: 2)
  val sum: Expr = Add(l: &one, r: &two)
  val e: Expr = Neg(e: &sum)
  io.println("${e.eval()}")
  val h = half(10) ?: -1
  val g = half(7)?.toString() ?: "odd"
  io.println("$h $g")
  when (val r = check(500)) {
    is Ok => io.println("ok $r")
    is Err => io.println("too big ${r.n}")
  }
  val fine = check(5)
  if (fine.ok) io.println("fine $fine")
}
