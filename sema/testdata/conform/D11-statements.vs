// D11/D37/D42/D43: bindings, returns, loops, assignment and `with`.
use io

struct Point {
  x: i64
  var y: i64
}

struct NoClose {
  n: i64
}

struct Counter {
  var n: i64 = 0
  implement Iterator {
    type Item = i64
    fun next(): i64? {
      this.n += 1
      if (this.n > 3) null else this.n
    }
  }
}

val limit: i64 = 3
const pi: f64 = 3.14
const bad: i64 = limit + 1 // error: 'limit' is a 'val', computed at run time, not a constant

fun redeclared() {
  val a = 1
  val a = 2 // error: 'a' is already declared in this scope // warning: 'a' is never used
  io.println("$a")
}
fun bodyHasValue() => 1 + 2
fun unitBody() {
  io.println("x")
}
fun declaredNothing(): () => 5 // error: function 'declaredNothing' returns nothing but its body has type 'i64'; add a return type
fun missingReturn(): i64 { // error: missing return: function 'missingReturn' must return a value of type 'i64'
  io.println("no value")
}
fun localFun() {
  fun inner() { } // error: local functions are not supported; use a lambda or a module-level function
}
fun breakOutside() {
  break // error: 'break' outside of a loop
}
fun unknownLabel() {
  loop { // warning: this loop never repeats
    break outer // error: no enclosing loop labelled 'outer'
  }
}
fun localConst() {
  const c = 1 // error: 'const' is only allowed at module level
  io.println("$c")
}
fun bindUnit() {
  val _ = unitBody() // error: cannot bind a value of type '()'
}
fun bindNever() {
  val _ = panic("no") // error: initializer never produces a value
}
fun destructureNumber() {
  val (a, b) = 5 // error: cannot destructure a value of type 'i64'; only tuples destructure positionally
}
fun destructureCount() {
  val (a, b) = (1, 2, 3) // error: tuple has 3 elements but 2 names are bound
}
fun returnNothing(): i64 {
  return // error: missing return value: this function returns 'i64'; write 'return' followed by one
}
fun returnSomething() {
  return 5 // error: this function returns nothing, but this 'return' has a value of type 'i64'; declare the result type (': i64') or return without a value
}
fun assignUnknown() {
  missing = 1 // error: unknown name 'missing'
}
fun assignGlobal() {
  limit = 4 // error: cannot assign to 'limit': it is a module-level 'val'; state every task shares is a 'var' behind a lock (D66)
}
fun assignFunction() {
  unitBody = 1 // error: 'unitBody' is not assignable
}
fun assignSafe(p: Point?) {
  var q = p
  q?.y = 1
}
fun assignCall() {
  bodyHasValue() = 1 // error: expression is not assignable
}
fun refRange() {
  loop (&i in 0..3) { } // error: a range yields values // warning: 'i' is never used
}
fun refSet(s: Set<i64>) {
  loop (&x in s) { } // error: '&' cannot bind a set element in place // warning: 'x' is never used
}
fun refMapKey(m: MutableMap<string, i64>) {
  loop ((&k, v) in m) { } // error: a map key cannot be changed in place // warning: 'k' is never used // warning: 'v' is never used
}
fun refIterator() {
  loop (&x in Counter()) { } // error: an iterator yields values // warning: 'x' is never used
}
fun notIterable() {
  loop (x in 5) { } // error: cannot iterate over 'i64': it implements neither Iterable nor Iterator
}
fun typedLoopVar(xs: List<i64>) {
  val (a: string, b) = (1, 2) // error: 'a' is bound to a value of type 'i64', not 'string'
  io.println("$a $b $xs")
}
fun withNotCloseable() {
  with (r = NoClose(n: 1)) { } // error: 'NoClose' is not Closeable; 'with' resources must implement Closeable
}

fun main() {
  io.println("${bodyHasValue()} $pi")
}
