use io

// Labeled loops (§4b), subjectless when (D13), ranges (D29)
fun primes(limit: i32): List<i32> {
  var found: MutableList<i32> = []
  loop :outer (n in 2..limit) {
    loop (p in found) {
      if (n % p == 0) continue outer
      if (p * p > n) break
    }
    found.push(n)
  }
  found.toList()
}

fun sign(n: i32): string = when {
  n < 0 => "negative"
  n == 0 => "zero"
  else => "positive"
}

// Generic struct with methods, generic function with a trait bound (D6/D8)
trait Area {
  fun area(): f64
  fun describe(): string = "area ${self.area()}"
}

struct Square { side: f64 }
impl Area for Square {
  fun area(): f64 = self.side * self.side
  override fun describe(): string = "square ${self.side}: " + "area ${self.area()}"
}

struct Pair<A, B> {
  first: A
  second: B
  fun swap(): Pair<B, A> = Pair(first: self.second, second: self.first)
}

fun <T: Area> total(xs: List<T>): f64 {
  var sum = 0.0
  loop (x in xs) { sum += x.area() }
  sum
}

// Wrapping arithmetic (D21), casts, tuples (D37)
fun wrap(): (i32, i64, u8) {
  val big: i32 = 2147483647
  val w = big +% 1
  val widened = big as i64 + 1
  val small = 300 as u8
  (w, widened, small)
}

// Smart casts on vars and sealed (D5/D13), pointer auto-deref (D39)
sealed trait Expr
struct Num : Expr { value: i64 }
struct Add : Expr { left: *Expr, right: *Expr }
struct Mul : Expr { left: *Expr, right: *Expr }

fun eval(e: Expr): i64 = when (e) {
  is Num(value) => value
  is Add(left, right) => eval(*left) + eval(*right)
  is Mul(left, right) => eval(*left) * eval(*right)
}

fun show(e: *Expr): string = when (e) {
  is Expr.Num => "${e.value}"
  is Expr.Add => "(${show(e.left)} + ${show(e.right)})"
  is Expr.Mul => "${show(e.left)} * ${show(e.right)}"
}

fun main() {
  io.println("primes ${primes(30)}")
  io.println("${sign(-5)} ${sign(0)} ${sign(7)}")
  val sq = Square(side: 3.0)
  io.println(sq.describe())
  io.println("total ${total([Square(side: 1.0), Square(side: 2.0)])}")
  val p = Pair(first: 1, second: "one")
  val s = p.swap()
  io.println("pair $p swapped $s ${s.first.len()}")
  val (w, widened, small) = wrap()
  io.println("wrap $w $widened $small")
  val tree = Add(left: &Num(value: 2), right: &Mul(left: &Num(value: 3), right: &Num(value: 4)))
  io.println("${show(&tree)} = ${eval(tree)}")
  var name: string? = null
  if (name == null) name = "filled"
  io.println("name ${name.len()} $name")
  val eq = Pair(first: 1, second: 2) == Pair(first: 1, second: 2)
  val neq = Num(value: 1) == Num(value: 2)
  io.println("eq $eq $neq ${(1, "a") == (1, "a")} ${"abc" < "abd"}")
  var counted = 0
  loop {
    counted += 1
    if (counted == 3) break
  }
  io.println("counted $counted ${"héllo".len()} ${"hello".substring(1, 3) ?: "?"} ${"42".toInt() ?: 0}")
}
