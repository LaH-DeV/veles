// D121: what a program may do with an `Array<T, N>`, and what it may not.
struct Buf<const N: i64> {
  data: Array<u8, N>
}

fun sum<const N: i64>(a: Array<i64, N>): i64 = a.fold(0, (s, x) => s + x)

fun zeros<const N: i64>(): Array<i64, N> = Array.make(0)

fun firstOf<T, const N: i64>(a: Array<T, N>) {
}

fun main() {
  val short: Array<i64, 3> = [1, 2] // error: 'Array<i64, 3>' holds 3 elements, this literal has 2 // warning: 'short' is never used
  val four: Array<i64, 4> = [1, 2, 3, 4]
  val _ = four.at(9) // error: index 9 is out of range for 'Array<i64, 4>', so 'at' is always null
  val _ = four.push(5) // error: an Array has a fixed length
  four.set(0, 1) // error: cannot assign to 'four': it is a 'val'
  val _ = Array.make(0) // error: cannot tell which array to make
  val three: Array<i64, 3> = four // error: expected 'Array<i64, 3>', found 'Array<i64, 4>' // warning: 'three' is never used
  val _ = Array<i64, 4>.make() // error: 'make' takes the value every element starts as
  val none: Array<i64, 0> = [] // warning: 'none' is never used
  val bare: Array<i64, 2> = mut [1, 2] // error: 'mut' makes a MutableList // warning: 'bare' is never used
  loop (&x in four) { // error: cannot assign to 'four': it is a 'val'
    *x = *x + 1
  }
  val _ = sum([1, 2, 3]) + sum(four) + zeros<4>().len()
  val _ = ["a", "b"].toArray<2>()
  val _ = ["a"].toArray() // error: 'toArray' needs the length as its type argument
  val _ = ["a"].toArray<string>() // error: 'toArray' takes a length, not the type 'string'
  val _ = ["a"].toArray<1>(1) // error: 'toArray' takes no arguments
  val _ = Buf<8>(data: Array.make(0))
  val _ = Buf(data: Array.make(0)) // error: cannot infer type parameter 'N' of 'Buf' // error: cannot tell which array to make
  val _ = sum([1.5, 2.5]) // error: expected
  firstOf([]) // error: cannot infer the element type of an empty array literal
  val pairs: Array<(i64, i64), 1> = [(1, 2)]
  loop ((&p, q) in pairs) { // error: '&' binds an array element as a whole // warning: 'p' is never used // warning: 'q' is never used
  }
}
