// D113: a `const fun` whose body does not check is reported once, where the
// mistake is; the constant that calls it fails silently with it (it once
// added "this (*sema.UnitConst) is not a constant expression").
const fun digit(c: string): i64 {
  val n = "0123456789".indexOf(c) ?: -1 // error: '?:' needs a nullable left operand, found 'i64'
  n
}

// a mismatch at an argument: the call is still built, but the function is
// not run
const fun twice(s: string): string {
  val b = StringBuilder()
  b.append("ab".substring(0, 1)) // error: type mismatch: expected 'string', found 'string?'
  b.append(s)
  b.toString()
}

const D: i64 = digit("7")
const T: string = twice("x")

fun main() {
}
