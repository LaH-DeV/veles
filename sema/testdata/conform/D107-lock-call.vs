// D107: a call of a function that suspends, inside a held region, is
// refused once suspension inference knows it suspends.
use io

fun fetch(): i64 {
  await sleep(Duration.millis(1))
  return 1
}

fun calledSuspending(notes: Mutex<i64>) {
  with n = notes.lock()
  *n += fetch() // error: 'fetch' suspends, and the lock on 'notes' is held until the end of this block
}

fun calledAfter(notes: Mutex<i64>) {
  val k = fetch()
  with n = notes.lock()
  *n += k
}

fun main() {
  val notes = Mutex(value: 0)
  calledSuspending(notes)
  calledAfter(notes)
  io.println("${notes.get()}")
}
