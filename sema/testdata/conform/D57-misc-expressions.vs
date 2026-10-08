// Assorted expression rules: `??` and `?!` (D61), enums (D57), tuples
// (D37), variants (D13), `None`/`Ok` inference (D4/D5), sets (D25),
// `@mustUse` (D51), and what a global initializer may not do.
use io

enum Color { Red, Green }

sealed trait Shape

struct Circle : Shape {
  r: i64
}

error Boom { }

struct Temp {
  c: i64
  fun reading(): i64 => this.c
}

@mustUse
fun important(): i64 => 42

fun mayFail(n: i64): i64 throws Boom => if (n > 0) throw Boom() else n

fun tryWithFallback(): i64 => try mayFail(1) ?? 0 // error: '??' handles the error itself; drop the 'try'
fun fallbackOnNumber(): i64 => 5 ?? 0 // error: '??' needs a Result on its left
fun orFailNumber(x: i64?): i64 throws Boom => try x ?! 5 // error: the right operand of '?!' must be an error
fun enumFunctionValue() {
  val _ = Color.values // error: 'Color.values' is a function; call it: 'Color.values(...)'
}
fun enumNoFunction(): Color? => Color.nope() // error: enum 'Color' has no function 'nope'
fun enumMethodValue(c: Color) {
  val _ = c.toString // error: 'toString' is a function; call it: '.toString(...)'
}
fun tupleElement(p: (i64, i64)): i64 => p.5 // error: tuple of 2 elements has no element '5'
fun sealedField(s: Shape): i64 => s.r // error: 'Shape' is a sealed trait; match on its variants with 'when' before accessing 'r'
fun noneUninferred() {
  val _ = None // error: cannot infer the type of 'None' here; annotate the binding
}
fun okUninferred() {
  val _ = Ok(1) // error: cannot infer the Result type of 'Ok(...)' here
}
fun setUnionTwo(a: Set<i64>, b: Set<i64>) {
  val _ = a.union(b, b) // error: 'union' takes 1 argument
}
fun negateUnsigned(x: u8): u8 => -x // error: cannot negate a value of type 'u8'
fun matchFunction(f: fun(): i64): i64 => when (f) {
  null => 0 // error: 'null' pattern on a non-nullable subject
  else => 1
}
fun unused() {
  important() // error: result of 'important' must be used (@mustUse)
}
fun isWithFields(s: Shape): bool => s is Circle(r) // error: destructuring patterns are only allowed in 'when' arms

val gathered = gather { // error: 'gather' cannot appear in a global initializer
  async important()
}

fun main() {
  io.println("${Temp(c: 1).reading()} ${gathered}")
}
