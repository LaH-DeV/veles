use io

fun boom(n: i64): i64 {
  await sleep(Duration.millis(1))
  val xs = [1, 2]
  xs.atOrPanic(n)
}

fun fine(): i64 {
  await sleep(Duration.millis(1))
  7
}

fun main() {
  // D52: gather captures a child's panic as Err(Panic); scope re-raises it
  val (a, b) = gather {
    async boom(5)
    async fine()
  }
  io.println("$a $b")
  when (a) {
    is Ok(v)  => io.println("ok $v")
    is Err(e) => when (e) {
      is Panic(message) => io.println("captured panic: $message")
    }
  }
  scope {
    async fine()
    async boom(9)
  }
  io.println("not reached")
}
