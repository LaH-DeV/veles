// D100: where the statement form `with x = e` is refused, each error naming
// the block form.
use io

struct Res {
  n: i64

  implement Closeable {
    fun close() { }
  }
}

with r = Res(n: 1) // error: a module has no end; open it inside a function

fun exprBody(): i64 = with r = Res(n: 1) // error: 'with x = e' is a statement

fun braceless(flag: bool) {
  if (flag) with r = Res(n: 1) // error: a body without braces has nothing after it
  io.println("after")
}

fun operand(): i64 {
  val n = with r = Res(n: 2) // error: 'with x = e' is a statement
  return n
}

fun arm(k: i64) {
  when (k) {
    1 => with r = Res(n: 3) // error: 'with x = e' is a statement
    else => { }
  }
}

fun twoBindings() {
  with a = Res(n: 1), b = Res(n: 2) // error: 'with x = e' binds one resource
  io.println("${a.n}")
}
