// D150: an executor's members have names of their own; its functions may
// be variadic, and a hop types each trailing argument as an element.
use io

executor gl {
  var count: i64 = 0
  fun count(): i64 => 1 // error: already declared in executor
  public fun total(xs: u8...): i64 => xs.len()
  public fun bump(): i64 {
    count += 1
    count
  }
}

fun main() {
  io.println("${gl.bump()} ${gl.total(1, 2)}")
}
