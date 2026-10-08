// D113 part 3: `const fun` — what the compiler may run, checked at the
// function's declaration, and what a failing evaluation reports.
use io

val counter = 5

fun plain(n: i64): i64 => n + 1

const fun double(n: i64): i64 => n * 2

const fun callsConst(n: i64): i64 => double(double(n))
const OK: i64 = callsConst(3)

const fun callsPlain(n: i64): i64 => plain(n) // error: it calls 'plain', which is not a 'const fun'
const fun readsGlobal(): i64 => counter // error: it reads the module-level 'counter'
const fun printsStuff(n: i64): i64 {
  io.println("hi") // error: it calls 'println', which is not a 'const fun'
  n
}
const fun usesLambda(n: i64): i64 {
  val f = (x: i64) => plain(x) // error: it calls 'plain', which is not a 'const fun'
  f(n)
}
trait Named {
  fun name(): string
}
struct Dog {
  implement Named {
    fun name(): string => "dog"
  }
}
const fun usesTraitObject(): string { // error: a trait object is not supported in a 'const fun' yet
  val n: Named = Dog()
  n.name()
}
const fun throwsOne(n: i64): i64 throws { // error: a 'const fun' cannot throw yet
  n
}
const fun usesUnsafe(n: i64): i64 { // error: a 'const fun' cannot use 'unsafe'
  // SAFETY: nothing here touches memory; the rule is about the keyword
  unsafe {
    n
  }
}

const fun boom(n: i64): i64 {
  if (n == 0) panic("reached zero")
  boom(n - 1)
}
const fun deep(n: i64): i64 => deep(n + 1)
const fun spin(): i64 {
  var i = 0
  loop {
    i += 1
    if (i < 0) return i
  }
}
const fun outOfRange(): i64 {
  val xs: List<i64> = [1, 2]
  xs.at(5) ?: panic("no element")
}
const fun divides(a: i64, b: i64): i64 => a / b
const fun overflows(a: i64): i64 => a * 4611686018427387904

const A: i64 = boom(3) // error: panic in a constant: reached zero
const B: i64 = deep(0) // error: 'deep' recursed more than 4096 calls deep
const C: i64 = spin() // error: evaluating the constant 'C' took more than
const D: i64 = outOfRange() // error: panic in a constant: no element
const E: i64 = divides(1, 0) // error: division by zero in a constant
const F: i64 = overflows(2) // error: constant overflow
const G: i64 = callsPlain(1)
const H: i64 = plain(1) // error: a constant cannot call 'plain': it is not a 'const fun'

// the checks a run-time failure becomes: every one is an error at the constant
const fun setPast(): i64 {
  val xs: MutableList<i64> = [1]
  xs.set(5, 2)
  xs.len()
}
const fun bytePast(): u8 => "abc".byteAt(9)
const fun absMin(): i8 {
  val a: i8 = -128
  a.abs()
}
const fun modZero(n: i64): i64 => n.mod(0)
const fun modMinus(): i8 {
  val a: i8 = -128
  a.mod(-1)
}
const fun powNegative(): i64 => 2.pow(-1)
const fun powHuge(): i64 => 2.pow(100)
const fun powBig(): i64 => 10.pow(30)
const fun showsPointer(): string {
  var n = 5
  val p = &n
  "${p}"
}

const I: i64 = setPast() // error: index 5 is out of range for a list of length 1
const J: u8 = bytePast() // error: byte index 9 is out of range for a string of length 3
const K: i8 = absMin() // error: constant overflow: abs of -128 is 128
const L: i64 = modZero(4) // error: division by zero in a constant: 4 mod 0
const M: i8 = modMinus() // error: constant overflow: -128 mod -1
const N: i64 = powNegative() // error: a negative exponent in a constant: 2.pow(-1)
const O: i64 = powHuge() // error: constant overflow: 2.pow(100) does not fit 'i64'
const P: i64 = powBig() // error: constant overflow: 10.pow(30) is 1000000000000000000000000000000
const R: string = showsPointer() // error: interpolating a '*i64' is not a constant expression
const S: i64 = ((x: i64) => x + 1)(1)
const S2: i64 = ((x: i64) => plain(x))(1) // error: a constant cannot call 'plain'
const fun variantOk(): i64 {
  val r: Result<i64, Panic> = Ok(4)
  when (r) {
    is Ok(v) => v
    is Err   => 0
  }
}
const V: i64 = variantOk() // error: a sealed variant is not a constant expression
const Q: List<i64> = [1, 2].map(x => x + 1)
const Q2: List<i64> = [1, 2].map(x => plain(x)) // error: a constant cannot call 'plain'

fun main() {
  io.println("${OK}")
}
