// D237: a function value started with `async` may not capture a collection
// its maker changes while the task may run — after the `async`, or in a
// loop around it. A change before the launch, or a capture list's copy, is
// fine. (Checked after the receiver pass: this case holds no other error.)
use io

fun fill(into: *List<i64>) {
  into.push(7)
}

// a task started from a parameter and handed back in a value (D111) runs
// beside the caller: the caller's lambda is checked at the call, through
// a function that passes the parameter on as well
struct Running {
  job: Task<i64>
}

fun starting(work: sendable fun(): i64): Running => Running(job: async work())

fun wrapped(work: sendable fun(): i64): Running => starting(work)

// one that waits for its task before returning: the caller cannot run beside it
fun waits(work: sendable fun(): i64): i64 suspends {
  with t = async work()
  val alongside = 1
  alongside + await t
}

// a lambda kept in a field of a local struct is followed too
struct Handler {
  run: sendable fun(): i64
}

fun inField() suspends {
  val ys: List<i64> = [1]
  val h = Handler(run: () => ys.len())
  with t = async h.run()                       // error: 'h.run' captures 'ys', which inField changes
  ys.push(2)
  io.println("${await t}")
}

fun held() suspends {
  val ys: List<i64> = [1]
  with r = starting(() => ys.len())            // error: 'starting' starts a task from its parameter 'work' and hands it back, and the lambda passed here captures 'ys', which held changes at line
  ys.push(2)
  val zs: List<i64> = [1]
  with w = wrapped(() => zs.len())             // error: 'wrapped' starts a task from its parameter 'work' and hands it back
  zs.push(2)
  val fine: List<i64> = [1]
  val n = waits(() => fine.len())
  fine.push(3)
  io.println("$n ${await r.job} ${await w.job}")
}

fun main() suspends {
  val ys: List<i64> = [10]
  val job = () => ys.len()
  with u = async job()                         // error: 'job' captures 'ys', which main changes at line
  ys.push(1)
  io.println("${await u}")

  val zs: List<i64> = [1]
  val before = () => zs.len()
  zs.push(2)
  with v = async before()
  zs.len()
  io.println("${await v}")

  val copied: List<i64> = [1]
  val safe = [copied] () => copied.len()
  with w = async safe()
  copied.push(3)
  io.println("${await w}")

  val ps: List<i64> = []
  val viaPtr = () => ps.len()
  with x = async viaPtr()                      // error: 'viaPtr' captures 'ps', which main changes at line
  fill(&ps)
  io.println("${await x}")

  val rs: List<i64> = []
  val inLoop = () => rs.len()
  scope {
    loop (i in 0..<3) {
      rs.push(i)
      async inLoop()                           // error: 'inLoop' captures 'rs', which main changes at line
    }
  }
}
