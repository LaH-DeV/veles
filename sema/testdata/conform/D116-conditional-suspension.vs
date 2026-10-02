// D116: a call of a function that suspends only through its `suspends`
// parameter suspends exactly when what it is given does. A parameter whose
// value is kept (stored, returned, handed to a function that keeps it)
// makes the function suspend whenever it is called.
use io

fun each<T>(xs: List<T>, f: fun(T): () suspends) {
  loop (x in xs) {
    f(x)
  }
}

struct Later {
  run: fun(): () suspends
}

fun keep(f: fun(): () suspends): Later {
  each([1], x => f())
  Later(run: f)
}

fun slow() {
  await sleep(Duration.millis(1))
}

fun main() {
  val m = Mutex<i64>(value: 0)
  m.withLock(p => each([1, 2], x => io.println("$x"))) // fine: nothing passed suspends
  m.withLock(p => each([1, 2], x => slow())) // error: this lambda suspends, so its type must be a suspending function type
  m.withLock(p => keep(() => io.println("kept"))) // error: this lambda suspends, so its type must be a suspending function type
  with (n = m.lock()) {
    each([1], x => io.println("${*n}"))
    each([1], x => slow()) // error: 'each' suspends, and the lock on 'm' is held until the end of its 'with' block
  }
}
