// D21/D57 and the operators: literals fit their type, operators are defined
// for the types that have them, `as` converts numbers only.
use io

enum Color { Red, Green }

fun tooLarge(): i64 = 99999999999999999999 // error: integer literal does not fit in 64 bits; a float ('1e20') holds it approximately
fun negativeUnsigned(): u8 = -1 // error: negative literal for unsigned type 'u8'
fun doesNotFit(): i8 = 300 // error: literal 300 does not fit in 'i8', which holds -128..127
fun belowRange(): i8 = -129 // error: literal -129 does not fit in 'i8', which holds -128..127
fun byteRange(): u8 = 256 // error: literal 256 does not fit in 'u8', which holds 0..255
fun negateText(): string = -"x" // error: cannot negate a value of type 'string'
fun derefNumber(n: i64): i64 = *n // error: cannot dereference a value of type 'i64'
fun multiplyText(): string = "a" * "b" // error: operator '*' is not defined for 'string'
fun subtractBools(): bool = true - false // error: operator '-' is not defined for 'bool'
struct Money {
  cents: i64
}
fun addMoney(a: Money, b: Money): Money = a + b // error: operator '+' is not defined for 'Money'; implement 'Addable' (fun plus(other: R): Out) to give it one (D71)
fun floatRemainder(x: f64): f64 = x % 2.0 // error: '%' is not defined for floats; 'x.mod(y)' is the remainder, in 0.0..<|y|
fun wrapFloat(x: f64): f64 = x +% 1.0 // error: wrapping operator '+%' is only defined for integers
fun colorNumber(c: Color): i64 = c as i64 // error: an enum is not its number — read it with '.value'
fun numberColor(n: i64): Color = n as Color // error: look the member up with 'Color.fromValue(n)'
fun textNumber(s: string): i64 = s as i64 // error: 'as' converts between numeric types only
fun floatRange(): Range<f64> = 1.0..2.0 // error: ranges are over integers, found 'f64'
fun emptyList() {
  val xs = [] // error: cannot infer the element type of an empty list
}
fun ifValue(b: bool): i64 = if (b) 1 // error: 'if' used as a value needs an 'else' branch
fun mixedBranches(b: bool) {
  val _ = if (b) 1 else "one" // error: branches have incompatible types 'i64' and 'string'
}
fun nullCompare(n: i64): bool = n == null // error: comparing a non-nullable 'i64' with null is always false

fun main() {
  io.println("operators")
}
