package sema

import "testing"

// D116, D231: a sendable lambda that captures a list is wrapped by the share
// pass in a block marking the capture shared; the call it is passed to is
// still plain when the lambda does not suspend. It was not: `main` below
// suspended, so `time.measure` ran its coroutine copy (bench/adapters) and
// a lock held around such a call was refused.
func TestCapturingLambdaStaysPlain(t *testing.T) {
	src := prelude + `use time

fun main() {
  val input: List<i64> = [1, 2, 3]
  val m = Mutex<i64>(value: 0)
  with (n = m.lock()) {
    val (total, took) = time.measure(() => input.map(x => x * 2).len())
    io.println("$total ${took.toNanos() >= 0} $n")
  }
}`
	expectClean(t, src)
	prog := checkProgram(t, src)
	for _, fn := range prog.Funcs {
		if fn.Name == "v_main.main" && fn.Suspends {
			t.Errorf("main suspends; a lambda capturing a list should leave it plain")
		}
	}
}
