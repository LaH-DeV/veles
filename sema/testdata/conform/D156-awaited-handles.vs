// D156 part 1: outside a scope, `val h = async f()` is a fail-fast child of
// the rest of its block and must be awaited on every normal path; a failed
// `try`, `throw`, panic or cancellation cancels and joins it instead.
// Part 1b: an `await` right after its `async` runs nothing alongside it.
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

fun helper(t: Task<i64>): i64 {
  await t
}

fun both(): i64 throws Boom {
  val a = async work(1)
  val b = async work(2)
  await a + await b
}

fun overlapped(): i64 throws Boom {
  val a = async work(1)
  val local = plain(5)
  local + await a
}

fun branches(c: bool): i64 throws Boom {
  val a = async work(1)
  if (c) {
    return await a
  }
  plain(1) + await a
}

fun inWhen(n: i64): i64 throws Boom {
  val a = async work(1)
  when (n) {
    0 => return plain(0) + await a
    else => return plain(n) + await a
  }
}

fun skipped(c: bool): i64 throws Boom {
  val a = async work(1)
  if (c) return 0 // error: 'a' is not awaited on this path
  plain(1) + await a
}

fun oneBranch(c: bool): i64 throws Boom {
  val a = async work(1) // error: 'a' is not awaited on every path to the end of its block
  var n: i64 = plain(0)
  if (c) n = await a
  n
}

fun last() throws Boom {
  val a = async work(1) // error: 'a' is not awaited on every path to the end of its block
}

fun looped(xs: List<i64>): i64 throws Boom {
  var total: i64 = 0
  loop (x in xs) {
    val t = async work(x)
    if (x == 0) continue // error: 't' is not awaited on this path
    if (x > 9) break // error: 't' is not awaited on this path
    total += plain(x) + await t
  }
  total
}

fun outside(xs: List<i64>): i64 throws Boom {
  val t = async work(1) // error: 't' is not awaited on every path to the end of its block
  var total: i64 = plain(0)
  loop (x in xs) total += await t + x
  total
}

fun innerBreak(xs: List<i64>): i64 throws Boom {
  val t = async work(1)
  var n: i64 = plain(0)
  loop (x in xs) {
    if (x > 2) break
    n += x
  }
  n + await t
}

fun raced(): i64 throws Boom {
  val t = async work(1)
  select {
    val v = t => v
    sleep(Duration.millis(100)) => throw Boom(n: 0)
  }
}

fun racedCached(): i64 throws Boom {
  val t = async work(1) // error: 't' is not awaited on every path to the end of its block
  select {
    val v = t => v
    sleep(Duration.millis(100)) => 0
  }
}

fun failsFirst(n: i64): i64 throws Boom {
  val t = async work(1)
  val m = try work(n)
  if (m > 100) throw Boom(n: m)
  m + await t
}

fun panicsFirst(n: i64): i64 throws Boom {
  val t = async work(1)
  if (n > 100) panic("too big")
  plain(n) + await t
}

fun twice(): i64 throws Boom {
  val t = async work(1)
  val x = plain(1)
  x + await t + await t
}

fun passed(): i64 throws Boom {
  val t = async work(1)
  val x = plain(1)
  x + helper(t) + await t // error: 't' must be awaited by this block, so it cannot be
}

fun stored(): i64 throws Boom {
  val t = async work(1)
  val keep: MutableList<Task<i64>> = []
  keep.push(t) // error: 't' must be awaited by this block, so it cannot be
  plain(1) + await t
}

fun captured(): i64 throws Boom {
  val t = async work(1)
  val get = () => await t // error: 't' must be awaited by this block, so it cannot be
  plain(1) + get() + await t
}

fun cancelled(): i64 throws Boom {
  val t = async work(1)
  t.cancel() // error: 't' must be awaited by this block, so it cannot be
  plain(1) + await t // error: 't' was cancelled at line
}

fun mutable(): i64 throws Boom {
  var t = async work(1) // error: a task is bound with 'val'
  plain(1) + await t
}

fun bare() {
  async plain(1) // error: 'async' must be
}

fun quiet(): i64 {
  val t = async work(1) // error: but the function is not declared 'throws'
  plain(1) + await t
}

fun inScope(): i64 throws Boom {
  scope {
    async work(1)
    val b = async work(2)
    io.println("started")
    return await b
  }
}

fun inLambda(): i64 {
  val f = () => {
    val t = async plain(1)
    plain(2) + await t
  }
  f()
}

// 1b: nothing runs alongside the task
fun immediate(): i64 throws Boom {
  val t = async work(1)
  val v = await t // warning: 'work' runs with nothing alongside it; call 'work(1)' directly
  v + 1
}

fun immediateWith(): i64 throws Boom {
  with t = async work(2)
  return await t // warning: 'work' runs with nothing alongside it
}

fun immediateArg(): i64 throws Boom {
  val t = async work(3)
  plain(await t) // warning: 'work' runs with nothing alongside it
}

fun original(): i64 throws Boom {
  scope {
    val handle = async work(4)
    io.println("${await handle}") // warning: 'work' runs with nothing alongside it
  }
  0
}

fun onExecutor(cpu: Executor): i64 throws Boom {
  scope(on: cpu) {
    val handle = async work(5)
    return await handle
  }
}

fun main() throws Boom {
  io.println("${try both()} ${try overlapped()} ${try branches(true)}")
}
