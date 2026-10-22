// D156 part 2: `awaitAll(t1, …, tn)` waits for every task and gives the
// tuple of their values; `awaitAll(tasks)` gives a List. Inline launches
// belong to the awaitAll, which is the await of each handle passed to it.
use io

error Boom { n: i64 }

fun work(n: i64): i64 throws Boom {
  await sleep(Duration.millis(1))
  if (n < 0) throw Boom(n)
  n * 10
}

fun name(n: i64): string {
  await sleep(Duration.millis(1))
  "n$n"
}

fun plain(n: i64): i64 {
  await sleep(Duration.millis(1))
  n
}

fun pair(): (i64, string) throws Boom {
  val a = async work(1)
  val b = async name(2)
  awaitAll(a, b)
}

fun inline(): i64 throws Boom {
  val (x, y) = awaitAll(async work(1), async work(2))
  x + y
}

fun three(): string throws Boom {
  val (x, s, y) = awaitAll(async work(1), async name(2), async plain(3))
  "$x $s $y"
}

fun mixed(): i64 throws Boom {
  val a = async work(1)
  val (x, y) = awaitAll(a, async plain(2))
  x + y
}

fun listed(): i64 throws Boom {
  scope {
    val tasks: MutableList<Task<i64>> = []
    loop (i in 0..<4) tasks.push(async work(i))
    return awaitAll(tasks).sum()
  }
}

fun inScope(): i64 throws Boom {
  scope {
    val a = async work(1)
    val b = async work(2)
    val (x, y) = awaitAll(a, b)
    return x + y
  }
}

fun notTask(): i64 throws Boom {
  val (x, y) = awaitAll(work(1), async work(2)) // error: 'work(1)' finishes before 'awaitAll' starts
  x + y
}

fun awaited(): i64 throws Boom {
  val (x, y) = await awaitAll(async work(1), async work(2)) // error: 'awaitAll' waits; remove 'await'
  x + y
}

fun single(): i64 throws Boom {
  val a = async work(1)
  val other = plain(2)
  awaitAll(a) + other // warning: use 'await a'
}

fun notAHandle(): i64 throws Boom {
  val n = 3
  val (x, y) = awaitAll(n, async work(2)) // error: 'awaitAll' waits for tasks, and 'n' is a 'i64'
  x + y
}

val early = awaitAll(async plain(1), async plain(2)) // error: 'awaitAll' cannot appear in a global initializer

fun none() {
  awaitAll() // error: 'awaitAll' waits for tasks
}

fun quiet(): i64 {
  val (x, y) = awaitAll(async work(1), async plain(2)) // error: but the function is not declared 'throws'
  x + y
}

fun main() throws Boom {
  io.println("${inline().toString()} ${three().toString()} ${listed().toString()}")
}
