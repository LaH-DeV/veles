// D12: every variant implements its sealed trait's methods that have no
// default, in an `implement` block of its body — told at the variant, not
// only when some call happens to dispatch to it.
use io

sealed trait Shape {
  fun area(): i64
  fun describe(): string => "a shape"
}

struct Dot : Shape {  // error: variant 'Dot' of 'Shape' does not implement 'area'; 'area' is declared in its body, outside any implement: move it inside 'implement Shape { }'
  fun area(): i64 => 0
}

struct Square : Shape {  // error: variant 'Square' of 'Shape' does not implement 'area'; add 'implement Shape { fun area(): i64 }' to its body
  side: i64
}

struct Circle : Shape {
  r: i64
  implement Shape {
    fun area(): i64 => 3 * this.r * this.r
  }
}

fun main() {
  io.println("")
}
