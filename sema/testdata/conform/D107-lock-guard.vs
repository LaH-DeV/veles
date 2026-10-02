// D107: `with n = m.lock()` holds a Mutex to the end of a block. `n` is a
// pointer to the value; nothing in the held region may suspend, the
// pointer cannot leave it, and `lock()` is usable only as a `with` value.
use io

struct Notes {
  var count: i64 = 0

  fun add() {
    this.count += 1
  }
}

fun fetch(): i64 suspends {
  await sleep(Duration.millis(1))
  return 1
}

fun statementForm(notes: Mutex<Notes>, ch: Channel<i64>) {
  with n = notes.lock()
  n.add()
  val k = await ch.recv() // error: 'recv' suspends, and the lock on 'notes' is held until the end of this block
  io.println("$k")
}

fun blockForm(notes: Mutex<Notes>, ch: Channel<i64>) {
  with (n = notes.lock()) {
    n.add()
    await sleep(Duration.millis(1)) // error: 'sleep' suspends, and the lock on 'notes' is held until the end of its 'with' block
  }
  with (notes.lock()) {
    ch.send(1) // error: 'send' suspends, and the lock on 'notes' is held until the end of its 'with' block
  }
}

fun suspendingValue(notes: Mutex<Notes>, f: fun(): i64 suspends) {
  with notes.lock()
  io.println("${f()}") // error: this call suspends, and the lock on 'notes' is held until the end of this block
}

fun afterTheBlock(notes: Mutex<Notes>): i64 suspends {
  with (n = notes.lock()) {
    n.add()
  }
  // released: waiting here is fine
  return fetch()
}

fun escapes(notes: Mutex<Notes>): *Notes {
  with n = notes.lock()
  return n // error: 'n' cannot be returned: it points into a Mutex that is unlocked when its 'with' block ends
}

fun copied(notes: Mutex<Notes>): Notes {
  with n = notes.lock()
  n.add()
  return *n
}

fun outside(notes: Mutex<Notes>) {
  val n = notes.lock() // error: 'lock()' holds the lock to the end of a 'with' block, so it is usable only as a 'with' value
  io.println("${n}")
}

fun lambdaInside(notes: Mutex<Notes>) {
  with n = notes.lock()
  n.add()
  // a lambda defined in the region runs later, when it is called
  val later = () => fetch()
  io.println("${n.count}")
  val _ = later
}

fun main() {
  val notes = Mutex(value: Notes())
  statementForm(notes, Channel<i64>(capacity: 1))
  blockForm(notes, Channel<i64>(capacity: 1))
  suspendingValue(notes, () => fetch())
  io.println("${afterTheBlock(notes)} ${escapes(notes).count} ${copied(notes).count}")
  outside(notes)
  lambdaInside(notes)
}
