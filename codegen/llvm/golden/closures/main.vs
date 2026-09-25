// Lambdas capturing by reference, returned closures, and higher-order calls.
use io

fun adder(k: i64): fun(i64): i64 = x => x + k

fun apply(f: fun(i64): i64, x: i64): i64 = f(x)

fun main() {
  var count = 0
  val bump = () => { count += 1 }
  bump()
  bump()
  val add5 = adder(5)
  val doubled = [1, 2, 3].map(x => x * 2)
  val total = doubled.fold(0, (a, b) => a + b)
  io.println("$count ${apply(add5, 10)} $doubled $total")
}
