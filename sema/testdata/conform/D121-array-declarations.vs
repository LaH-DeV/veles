// D121: `Array<T, N>`, and the constants that go where a type argument does.
struct Buf<const N: i64> {
  data: Array<u8, N>
}

struct Wrong {
  a: Array<i64> // error: Array takes an element type and a length
  b: Array<i64, -1> // error: a constant argument is a length from 0 to 1073741824, not -1
  c: Array<u8, 2000000000> // error: a constant argument is a length from 0 to 1073741824, not 2000000000
  e: Buf<i64> // error: expected a constant here, found the type 'i64'
  f: List<3> // error: expected a type here, found a number
  g: Array<i64, i64> // error: expected a constant here, found the type 'i64'
  h: Array<u8, *i64> // error: expected a constant here: a number
}

fun calc<const N: i64>(a: Array<u8, N + 1>) { // error: a type argument may name the constant parameter 'N'
}


fun asType<const N: i64>(x: N) { // error: 'N' is a constant, not a type
}

fun main() {
}
