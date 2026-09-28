// D78: the test words take what they say they take; plus two call rules
// (D45 inference, D57 enum functions).
use io

error Boom { }
error Bust { }

enum Color { Red, Green }

fun fact(n: i64) = if (n == 0) 1 else n * fact(n - 1) // error: cannot infer the return type of recursive function 'fact'
fun enumCall(): Color? = Color.nope() // error: enum 'Color' has no function 'nope'

test "words" {
  expect(1 == 1, 2 == 2) // error: 'expect' takes one argument: fun expect(condition: bool)
  expectThrows<Boom, Bust>(() => 1) // error: 'expectThrows' takes one error type // error: nothing in this function can throw
  expectThrows(5) // error: 'expectThrows' takes a function to call, like '() => parse(text)'; found 'i64'
  expectThrows((x: i64) => x) // error: 'expectThrows' calls its function with no arguments; wrap the call
}

fun main() {
  io.println("${fact(3)}")
}
