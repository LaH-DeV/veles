// D150: an executor's state is a value its code reads, not a function.
use io

executor gl {
  var count: i64 = 0
  public fun bump(): i64 {
    count += 1
    count
  }
  public fun misuse(): i64 => gl.count() // error: is state, not a function
}

fun main() {
  io.println("${gl.bump()} ${gl.misuse()}")
}
