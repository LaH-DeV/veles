// D88: the attribute takes no arguments and marks functions only.
use io

@caller_location(1) // error: @caller_location takes no arguments
fun withArgument(n: i64): i64 = n

@caller_location // error: @caller_location applies to functions
struct Other {
  n: i64
}

fun main() {
  io.println("${withArgument(2)}")
}
