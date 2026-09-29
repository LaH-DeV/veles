use io { println }

// Labeled loops (§4b), subjectless when (D13), ranges (D29)
fun primes(limit: i64): List<i64> {
  var found: MutableList<i64> = []
  loop :outer (n in 2..limit) {
    loop (p in found) {
      if (n % p == 0) continue outer
      if (p * p > n) break
    }
    found.push(n)
  }
  found.toList()
}

fun sign(n: i64): string = when {
  n < 0  => "negative"
  n == 0 => "zero"
  else   => "positive"
}

// Generic struct with methods, generic function with a trait bound (D6/D8)
trait Area {
  fun area(): f64
  fun describe(): string = "area ${this.area()}"
}

struct Square {
  side: f64

  implement Area {
    fun area(): f64 = this.side * this.side
    override fun describe(): string = "square ${this.side}: " + "area ${this.area()}"
  }
}

struct Pair<A, B> {
  first:  A
  second: B
  fun swap(): Pair<B, A> = Pair(first: this.second, second: this.first)
}

fun total<T: Area>(xs: List<T>): f64 =
  xs.fold(0.0, (acc, shape) => acc + shape.area())

// Wrapping arithmetic (D21), casts, tuples (D37)
fun wrap(): (i32, i64, u8) {
  val big: i32 = 2147483647
  val wrapped = big +% 1
  val widened = big.toI64() + 1
  val small = 300.wrapU8()
  (wrapped, widened, small)
}

// Smart casts on vars and sealed (D5/D13), pointer auto-deref (D39)
sealed trait Expr
struct Num : Expr {
  value: i64
}
struct Add : Expr {
  left:  *Expr
  right: *Expr
}
struct Mul : Expr {
  left:  *Expr
  right: *Expr
}

fun eval(e: Expr): i64 = when (e) {
  is Num(value)       => value
  is Add(left, right) => eval(*left) + eval(*right)
  is Mul(left, right) => eval(*left) * eval(*right)
}

fun show(e: *Expr): string = when (e) {
  is Expr.Num => "${e.value}"
  is Expr.Add => "(${show(e.left)} + ${show(e.right)})"
  is Expr.Mul => "${show(e.left)} * ${show(e.right)}"
}

fun main() {
  println("primes ${primes(30)}")
  println("${sign(-5)} ${sign(0)} ${sign(7)}")
  val sq = Square(side: 3.0)
  println(sq.describe())
  println("total ${total([Square(side: 1.0), Square(side: 2.0)])}")
  val p = Pair(first: 1, second: "one")
  val s = p.swap()
  println("pair $p swapped $s ${s.first.len()}")
  val (wrapped, widened, small) = wrap()
  println("wrap $wrapped $widened $small")
  val tree = Add(left: &Num(value: 2), right: &Mul(left: &Num(value: 3), right: &Num(value: 4)))
  println("${show(&tree)} = ${eval(tree)}")
  var name: string? = null
  if (name == null) name = "filled"
  println("name ${name.len()} $name")
  val eq = Pair(first: 1, second: 2) == Pair(first: 1, second: 2)
  val neq = Num(value: 1) == Num(value: 2)
  println("eq $eq $neq ${(1, "a") == (1, "a")} ${"abc" < "abd"}")
  var counted = 0
  loop {
    counted += 1
    if (counted == 3) break
  }
  println("counted $counted ${"héllo".len()} ${"hello".substring(1, 3) ?: "?"} ${"42".toInt() ?: 0}")
}
