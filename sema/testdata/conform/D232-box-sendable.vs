// D232: a pointer cannot be boxed as a trait that requires Sendable —
// another task would change the value through it.
use io

trait Job : Sendable {
  fun run(): i64
}

struct Count {
  n: i64 = 0

  implement Job {
    fun run(): i64 => this.n
  }
}

fun main() {
  var c = Count()
  val job: Job = &c                            // error: a pointer cannot be a 'Job': the trait requires Sendable, and another task would change '&c' through it
  val copied: Job = c
  io.println("${job.run()} ${copied.run()}")
}
