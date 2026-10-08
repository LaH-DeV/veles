// D5: a `T?` is not a `T` until it is checked for null.
use io

struct User {
  name: string
}

fun first(u: User?): string => u.name // error: may be null

fun main() {
  val u: User? = null
  io.println(first(u))
  val n: i64? = 3
  val m: i64 = n // error: type mismatch
  io.println("$m ${u?.name ?: "-"}")
}
