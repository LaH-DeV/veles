package sema

import "testing"

// D143: `scope(on: e)` and `gather(on: e)` take an Executor, which crosses
// into tasks (it is Sendable); anything else is refused, as is any other
// argument than `on:`.
func TestExecutorPlacement(t *testing.T) {
	expectClean(t, prelude+`fun double(n: i64): i64 => n * 2
fun onPool(cpu: Executor): i64 {
  val (a, b) = gather(on: cpu) {
    async double(1)
    async double(2)
  }
  io.println("${a} ${b}")
  scope(on: cpu) {
    async double(3)
  }
  cpu.run(() => double(4))
}
fun main() throws ThreadError {
  with cpu = try Executor.pool(threads: 2, name: "cpu")
  scope {
    async onPool(cpu)
  }
  io.println("${try blocking(() => double(5))}")
}`)
	expectError(t, prelude+`fun main() {
  scope(on: 3) { }
}`, "'scope(on: …)' takes an Executor, not 'i64'")
	expectError(t, prelude+`fun main() {
  val n = 1
  val r = gather(on: n) { }
}`, "'gather(on: …)' takes an Executor, not 'i64'")
	expectError(t, prelude+`fun main(cpu: Executor) {
  scope(cpu) { }
}`, "'scope(…)' takes one argument, the executor its tasks run on: 'scope(on: pool) { … }' (D143)")
}
