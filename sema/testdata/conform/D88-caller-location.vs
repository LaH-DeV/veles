// D88: `@caller_location` is std's for now.
use io

@caller_location // error: '@caller_location' is reserved for the standard library for now (D88)
fun checked(n: i64): i64 {
  if (n < 0) panic("negative")
  n
}

fun main() {
  io.println("${checked(1)}")
}
