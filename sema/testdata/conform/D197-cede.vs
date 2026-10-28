// D197: `cede` is a statement that suspends: its callers suspend, and the
// places that may not suspend refuse it, directly or through a call.
use io

fun checkpoint() {
  cede
}

fun each(xs: List<i64>, f: fun(i64): () suspends) {
  loop (x in xs) f(x)
}

val ready = start()  // error: a global initializer cannot suspend

fun start(): i64 {
  checkpoint()
  1
}

fun main() {
  checkpoint()
  each([1, 2], x => checkpoint())
  val m = Mutex<i64>(value: 0)
  m.withLock(p => checkpoint())  // error: this lambda suspends, so its type must be a suspending function type
  with (n = m.lock()) {
    checkpoint()  // error: 'checkpoint' suspends, and the lock on 'm' is held
  }
  io.println("$ready")
}
