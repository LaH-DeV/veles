// D100: the statement form `with x = e`, `with t = async f()`, and the rule
// that a resource does not outlive its block.
use io

struct Res {
  n: i64

  implement Closeable {
    fun close() { }
  }
}

struct NoClose {
  n: i64
}

struct Holder {
  r: Res
}

struct Cache {
  var last: Res? = null

  fun keep() {
    with r = open(1)
    this.last = r // error: 'r' cannot be stored in 'this.last', which outlives the block
  }
}

fun open(n: i64): Res => Res(n)

fun work(n: i64): i64 suspends {
  await sleep(Duration.millis(n))
  return n
}

fun fine(): i64 {
  with a = open(1)
  with b = open(2)
  // passing a resource, or a lambda capturing it, as an argument is allowed
  io.println("${useIt(a)} ${call(() => b.n)}")
  a.n + b.n
}

fun useIt(r: Res): i64 => r.n

fun call(f: fun(): i64): i64 => f()

fun lastStatement() {
  io.println("x")
  with r = open(1) // warning: 'r' is closed as soon as it is opened
}

fun notCloseable() {
  with r = NoClose(n: 1) // error: 'NoClose' is not Closeable
  io.println("${r.n}")
}

fun returned(): Res {
  with r = open(1)
  return r // error: 'r' cannot be returned: it is closed when its 'with' block ends
}

fun blockValue(): Res {
  with r = open(1)
  r // error: 'r' cannot be the value of its block
}

fun blockFormValue(): Res => with (r = open(1)) { r } // error: 'r' cannot be the value of its block

fun aliased(): Res {
  with r = open(1)
  val again = r
  return again // error: 'again' cannot be returned: it holds 'r'
}

fun storedOutside() {
  var outer = open(0)
  with r = open(1)
  outer = r // error: 'r' cannot be stored in 'outer', which outlives the block
  io.println("${outer.n}")
}

fun storedInside() {
  with r = open(1)
  var inner = open(0)
  inner = r // declared inside the block: it ends with the resource
  io.println("${inner.n}")
}

fun inStruct(): Holder {
  with r = open(1)
  return Holder(r) // error: 'r' cannot be returned
}

fun inCollection(list: MutableList<Res>) {
  with r = open(1)
  list.push(r) // error: 'r' cannot be stored in a collection or sent on a channel
}

fun lambdaKept(fns: MutableList<fun(): i64>) {
  with r = open(1)
  fns.push(() => r.n) // error: this lambda captures 'r', which is closed when its 'with' block ends, so the lambda cannot be stored in a collection
}

fun lambdaLent(xs: MutableList<i64>) {
  with r = open(1)
  // a method that only calls what it is given, for the call: a hand-off
  xs.sortWith((a, b) => if (a + r.n < b) Ordering.Less else Ordering.Greater)
  io.println("${xs.map(x => x + r.n)}")
}

fun lambdaReturned(): fun(): i64 {
  with r = open(1)
  return () => r.n // error: this lambda captures 'r', which is closed when its 'with' block ends, so the lambda cannot be returned
}

fun background(): i64 {
  with t = async work(1)
  with r = open(2)
  // awaiting the task gives its value; the close then has nothing to do
  return (await t) + r.n
}

fun taskReturned(): Task<i64> {
  with t = async work(1)
  return t // error: 't' cannot be returned: it is cancelled when its 'with' block ends
}

fun bareAsync() {
  with t = async work(1)
  val u = async work(2) // error: 'async' must be lexically inside a 'scope' or 'gather' block, or be the value of a 'with'
}

fun taskNotStartedHere() {
  scope {
    val t = async work(1)
    with (u = t) { } // error: a task is a 'with' resource only where 'with' starts it
  }
}

// D136: closing a `with` value by hand would close it twice
fun closedByHand() {
  with r = open(1)
  io.println("${r.n}")
  r.close() // error: 'r' is closed when its 'with' block ends; closing it here would close it twice
}

fun closedThroughAlias() {
  with (r = open(1)) {
    val same = r
    call(() => { same.close(); 0 }) // error: closing it here would close it twice
  }
}

fun closedPlain() {
  val r = open(1) // not a `with`: closing it is the program's own business
  r.close()
}

fun stoppedEarly() {
  with t = async work(1)
  t.cancel() // a with-task may be stopped early; the block's end joins it
}

fun inScope() {
  scope {
    with t = async work(1)
    // an `async` in the body belongs to the enclosing scope
    val u = async work(2)
    io.println("${await u}")
  }
}
