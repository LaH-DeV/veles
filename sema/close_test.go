package sema

import "testing"

// D147: a close() may suspend when it says so; a `with` on the type then
// suspends, and the places that cannot suspend refuse it.
func TestSuspendingClose(t *testing.T) {
	const conn = `struct Conn {
  name: string
  implement Closeable {
    fun close() suspends {
      await sleep(Duration.millis(1))
    }
  }
}
`
	expectClean(t, prelude+conn+`fun open(): i64 {
  with c = Conn(name: "a")
  1
}
fun closeIt<T: Closeable>(x: T) {
  with y = x
}
fun main() {
  io.println("${open()}")
  closeIt(Conn(name: "b"))
}`)
	expectError(t, prelude+`struct Conn {
  implement Closeable {
    fun close() {
      await sleep(Duration.millis(1))
    }
  }
}
fun main() { }`, "this 'close()' suspends, so it says so: 'fun close() suspends'")
	expectError(t, prelude+conn+`fun main() {
  val c: Closeable = Conn(name: "a")
}`, "'Conn' cannot be a 'Closeable' object: its 'close()' suspends")
	expectError(t, prelude+conn+`fun main() {
  val m = Mutex(value: 0)
  with (n = m.lock()) {
    with c = Conn(name: "a")
  }
}`, "'close' suspends, and the lock on 'm' is held until the end of its 'with' block")
	expectError(t, prelude+conn+`fun main() {
  val m = Mutex(value: 0)
  m.withLock(p => {
    with c = Conn(name: "a")
    *p += 1
  })
}`, "suspends")
}
