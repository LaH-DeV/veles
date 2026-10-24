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
  io.println("${a.toString()} ${b.toString()}")
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
}`, "'scope(…)' takes 'on:', the executor its tasks run on (D143), and 'workers:', how many of them run at once (D156)")
	// a generic method launched with async: its receiver crosses as a copy,
	// as a plain method's does (its instance is checked later, 2026-10-11)
	expectClean(t, prelude+`fun double(n: i64): i64 => n * 2
fun main() throws ThreadError {
  with cpu = try Executor.pool(threads: 2, name: "cpu")
  scope {
    val t = async cpu.run(() => double(21))
    io.println("${await t}")
  }
}`)
	expectError(t, prelude+`struct Bag {
  items: MutableList<i64>
  fun first<T>(x: T): T => x
}
fun main() {
  val b = Bag(items: [])
  scope {
    async b.first(1)
  }
}`, "argument of type 'Bag' is not Sendable")
}

// D152: runBlocking blocks its thread, so a function that suspends — a
// task's code — waits with run instead; plain code may call it.
func TestRunBlockingOnlyInPlainCode(t *testing.T) {
	expectError(t, prelude+`fun one(): i64 {
  await sleep(Duration.millis(1))
  1
}
fun waits(): i64 {
  await sleep(Duration.millis(1))
  Executor.default.runBlocking(() => one())
}
fun main() {
  io.println("${waits()}")
}`, "'runBlocking' blocks its thread until the task is done, and this function suspends")
	expectClean(t, prelude+`fun one(): i64 {
  await sleep(Duration.millis(1))
  1
}
fun plain(): i64 => Executor.default.runBlocking(() => one())
fun main() throws ThreadError {
  with t = try Thread.start(name: "t", f: () => io.println("${plain()}"))
  io.println("started")
}`)
}
