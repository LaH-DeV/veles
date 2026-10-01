// D103: a one-task `gather` is that task's Result; `async` runs a
// `sendable fun` value like a named function.
use io

error Boom { }

fun work(n: i64): i64 throws Boom = if (n < 0) throw Boom() else n

fun one(): bool {
  val r = gather { async work(1) }
  val wrong = r.0 // error: before accessing '0'
  r is Ok
}

fun two(): bool {
  val (a, b) = gather {
    async work(1)
    async work(2)
  }
  a is Ok && b is Ok
}

fun values(f: sendable fun(i64): i64, g: fun(i64): i64): i64 {
  scope {
    val a = async f(1)
    val b = async g(2) // error: 'g' is not a sendable function, so it cannot cross into a task
    return await a
  }
}

fun notACall(xs: List<i64>) {
  scope {
    async xs.len() // error: 'async' starts a call
  }
}

fun main() {
  io.println("${one()} ${two()} ${values(n => n, n => n)}")
}
