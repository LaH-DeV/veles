// D237: a function value started with `async` may not capture a collection
// its maker changes while the task may run — after the `async`, or in a
// loop around it. A change before the launch, or a capture list's copy, is
// fine. (Checked after the receiver pass: this case holds no other error.)
use io

fun fill(into: *List<i64>) {
  into.push(7)
}

fun main() suspends {
  val ys: List<i64> = [10]
  val job = () => ys.len()
  with u = async job()                         // error: 'job' captures 'ys', which main changes at line 15 while the task may run
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
  with x = async viaPtr()                      // error: 'viaPtr' captures 'ps', which main changes at line 34 while the task may run
  fill(&ps)
  io.println("${await x}")

  val rs: List<i64> = []
  val inLoop = () => rs.len()
  scope {
    loop (i in 0..<3) {
      rs.push(i)
      async inLoop()                           // error: 'inLoop' captures 'rs', which main changes at line 41 while the task may run
    }
  }
}
