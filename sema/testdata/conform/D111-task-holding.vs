// D111: a value holding a task is received with `with` where it is made,
// or returned straight to the caller; `async` is a field argument of its
// constructor only then; a returned value's task gets nothing this
// function closes.
use io

struct Worker {
  job: Task<()>
}

struct Pair {
  left: Worker
  right: Worker
}

struct Res {
  implement Closeable { fun close() { } }
}

fun idle() {
  await sleep(Duration.millis(1))
}

fun hold(r: Res) { }

fun make(): Worker = Worker(job: async idle())

fun pair(): Pair {
  return Pair(left: make(), right: Worker(job: async idle()))
}

fun consume(w: Worker) { }

fun refused() {
  val w = make() // error: 'Worker' holds a running task, so it must be received with 'with' where it is made: 'with x = …' (its tasks belong to that block), or returned to the caller (D111)
  make() // error: 'Worker' holds a running task
  consume(make()) // error: 'Worker' holds a running task
  val ws = [make()] // error: 'Worker' holds a running task
  val loose = Worker(job: async idle()) // error: 'Worker' holds a running task
  scope {
    async make() // error: 'async' would start a task whose result, a 'Worker', holds a task of its own that no block could receive; call it and receive the value with 'with' (D111)
  }
  io.println("${ws.len()}")
  consume(w)
  consume(loose)
}

fun leaks(): Worker {
  with r = Res()
  Worker(job: async hold(r)) // error: 'r' cannot be handed to a task that outlives this function: it is closed when its 'with' block ends (D100)
}

fun main() {
  with w = make()
  with p = pair()
  with Worker(job: async idle())
  io.println("received")
  refused()
}

error Broken { }

fun breaks(): string throws Broken {
  throw Broken()
}

struct Fragile {
  job: Task<Result<string, Broken>>
}

fun insideDo(): string throws Broken {
  do {
    with f = Fragile(job: async breaks()) // error: a value holding a task that can fail cannot be received inside a 'do' block
    "unreachable"
  } catch (e) {
    "caught"
  }
}
