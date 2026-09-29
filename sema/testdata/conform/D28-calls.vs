// D28 and D44: calls — arguments by position then by name, what is
// callable, and what needs `unsafe`.
use io

extern "C" {
  fun abs(x: i32): i32
}

unsafe fun risky(): i64 = 1

fun pair(a: i64, b: i64): i64 = a + b
fun first<T>(xs: List<T>): T? = xs.at(0)
fun make<T>(): List<T> = []

struct Point {
  x: i64
  y: i64
  fun norm(): i64 = this.x + this.y
}

struct Box<T> {
  item: T
}

sealed trait Shape

struct Dot : Shape { }

fun positionalAfterNamed(): i64 = pair(a: 1, 2) // error: positional argument after a named argument
fun tooMany(): i64 = pair(1, 2, 3) // error: too many arguments to 'pair': expected 2
fun unknownName(): i64 = pair(1, c: 2) // error: 'pair' has no parameter named 'c'
fun missing(): i64 = pair(1) // error: missing argument 'b' in call to 'pair'
fun externOutside(): i32 = abs(-1) // error: calling extern "C" function 'abs' requires an 'unsafe' block
fun unsafeOutside(): i64 = risky() // error: calling 'unsafe fun risky' requires an 'unsafe' block
fun mismatch(): i64? = first(3) // error: cannot infer type parameters: argument of type 'i64' does not match parameter type 'List<T>'
fun uninferred() {
  val _ = make() // error: cannot infer type parameter 'T' of 'make'
}
fun genericStruct() {
  val _ = Box // error: 'Box' is a type, not a value
}
fun sealedCall(): Shape = Shape() // error: 'Shape' is a sealed trait; construct one of its variants, e.g. 'Shape.Dot(...)'
fun basicCall(): i64 = i64(3) // error: 'i64' is not callable; convert with a method: 'x.toI64()'
fun notCallable(n: i64): i64 = n(1) // error: 'n' has type 'i64' and is not callable
fun valueNotCallable(): i64 = (1 + 2)(3) // error: value of type 'i64' is not callable
fun byPosition(): Point = Point(1, 2) // error: construct 'Point' by field name
fun valueArity(f: fun(i64): i64): i64 = f(1, 2) // error: function value takes 1 argument, got 2
fun builtinArity(xs: List<string>): string? = xs.at(1, 2) // error: 'at' takes 1 argument: at(i: i64): string?
fun valueNamed(f: fun(i64): i64): i64 = f(x: 1) // error: named arguments apply only to direct calls of named functions
fun genericValue() {
  val _ = first // error: generic function 'first' cannot be used as a value; wrap the call in a lambda, whose parameter types pick the instance: '(x) => first(x)'
}
fun unsafeValue() {
  val _ = risky // error: unsafe or extern function 'risky' cannot be used as a value
}
fun popImmutable(xs: List<i64>): i64? = xs.pop() // error: cannot pop from an immutable List; use MutableList
fun clearImmutable(xs: List<i64>) {
  xs.clear() // error: cannot clear an immutable List; use MutableList
}
fun reserveImmutable(xs: List<i64>) {
  xs.reserve(8) // error: cannot reserve room in an immutable List; use MutableList (D25)
}
fun sealedMethod(s: Shape): i64 = s.area() // error: no method 'area' on sealed trait 'Shape'
fun panicTwo(): i64 = panic("a", "b") // error: 'panic' takes one argument: the message
fun rawMethod(p: *raw i64): i64 = p.norm() // error: raw pointers have no methods

fun main() {
  io.println("${Point(x: 1, y: 2).norm()} ${Dot()}")
}
