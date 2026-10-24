use io

error Failed {
  why: string
}

fun risky(n: i64): i64 throws Failed {
  if (n < 0) throw Failed(why: "negative")
  n * 2
}

fun twice<T>(x: T, f: fun(T): T): T => f(f(x))

fun main() throws Failed {
  val n = try risky(4)
  val m = twice(n, v => v + 1)
  val other = async risky(m)
  io.println("${risky(1).toString()}")
  io.println("${await other}")
}
