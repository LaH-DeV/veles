// D12: every variant implements its sealed trait's methods that have no
// default, in its body (D195) — told at the variant, not only when some call
// happens to dispatch to it.
use io

sealed trait Shape {
  fun area(): i64
  fun describe(): string => "a shape"
}

struct Dot : Shape {
  fun area(): i64 => 0
}

struct Square : Shape {  // error: variant 'Square' of 'Shape' does not implement 'area'; write 'fun area(): i64' in its body (D195)
  side: i64
}

struct Circle : Shape {
  r: i64
  fun area(): i64 => 3 * this.r * this.r
}

fun main() {
  io.println("")
}
