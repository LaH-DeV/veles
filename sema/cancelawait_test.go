package sema

import "testing"

// D154: `await t` where `t.cancel()` was called on every path before it is
// refused; a cancel on one path only is left to the run-time panic.
func TestAwaitAfterCancel(t *testing.T) {
	const fetch = `fun fetch(id: i64): i64 {
  await sleep(Duration.millis(5))
  id * 2
}
`
	expectError(t, prelude+fetch+`fun main() {
  scope {
    val t = async fetch(21)
    t.cancel()
    io.println("${await t}")
  }
}`, "'t' was cancelled at line 9: awaiting it panics unless the task had already finished")
	// a race arm's await is the same wait
	expectError(t, prelude+fetch+`fun main() {
  scope {
    val t = async fetch(21)
    t.cancel()
    select {
      val v = t => io.println("$v")
      sleep(Duration.millis(1)) => io.println("late")
    }
  }
}`, "'t' was cancelled at line 9")
	// cancelled in both branches: still every path
	expectError(t, prelude+fetch+`fun run(args: List<string>) {
  scope {
    val t = async fetch(21)
    if (args.isEmpty()) {
      t.cancel()
    } else {
      t.cancel()
    }
    io.println("${await t}")
  }
}`, "'t' was cancelled at line")
	// one path only, inside a loop, or another handle: accepted
	expectClean(t, prelude+fetch+`fun run(args: List<string>) {
  scope {
    val t = async fetch(21)
    val u = async fetch(1)
    if (args.isEmpty()) t.cancel()
    loop (_ in args) {
      u.cancel()
    }
    io.println("${await t} ${await u}")
  }
  scope {
    val w = async fetch(2)
    io.println("${await w}")
    w.cancel()
  }
}
fun main() {
  run([])
}`)
}

// A fact the body of a loop establishes does not outlive the loop: the
// body may run no times, or be left by a `break` before it (D5).
func TestLoopFactsDoNotLeak(t *testing.T) {
	expectError(t, prelude+`fun main() {
  val xs: List<i64> = []
  var x: i64? = null
  loop (a in xs) {
    x = a
  }
  io.println("${x + 1}")
}`, "operator '+' is not defined for 'i64?'")
	expectError(t, prelude+`fun main() {
  var x: i64? = null
  var n: i64 = 0
  loop {
    n += 1
    if (n > 3) break
    x = n
  }
  io.println("${x + 1}")
}`, "operator '+' is not defined for 'i64?'")
	// what held before the loop still holds
	expectClean(t, prelude+`fun main() {
  var x: i64? = 1
  if (x == null) return
  loop (i in 0..<3) {
    io.println("$i")
  }
  io.println("${x + 1}")
}`)
}
