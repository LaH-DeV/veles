// D156 part 5: a `scope` may be used as a value, its body's last
// expression, available once every child has finished.
use io

error Boom { n: i64 }

fun work(n: i64): i64 throws Boom {
  await sleep(Duration.millis(1))
  if (n < 0) throw Boom(n)
  n * 10
}

fun plain(n: i64): i64 {
  await sleep(Duration.millis(1))
  n
}

fun total(): i64 throws Boom {
  val n = scope {
    val a = async work(1)
    val b = async work(2)
    val x = await a
    x + await b
  }
  n + 1
}

fun pool(xs: List<i64>): i64 throws Boom {
  scope(workers: 2) {
    val tasks: MutableList<Task<i64>> = []
    loop (x in xs) tasks.push(async work(x))
    awaitAll(tasks).sum()
  }
}

fun asArgument(): string throws Boom {
  "${scope {
    val a = async work(3)
    plain(1) + await a
  }}"
}

fun placed(cpu: Executor): i64 throws Boom {
  scope(on: cpu) {
    val t = async work(4)
    await t
  }
}

fun returns(c: bool): i64 throws Boom {
  val n: i64 = scope {
    async work(1)
    if (c) return 0
    plain(2)
  }
  n
}

fun noValue(): i64 throws Boom {
  val n: i64 = scope { // error: a 'scope' used as a value ends in an expression
    async work(1)
    loop (i in 0..<3) io.println("$i")
  }
  n
}

fun handle(): i64 throws Boom {
  val _ = scope {
    async work(1) // error: this task cannot be the value of its block
  }
  0
}

fun main() throws Boom {
  io.println("${total()} ${pool([1, 2, 3])} ${asArgument()}")
}
