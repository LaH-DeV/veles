package sema

import "testing"

// D146: RwLock's read/write are `with` values with a lock region; the
// waiting operations of Event, Watch and Subscription are awaited and can
// be race arms; the types are Sendable when what they hold is.
func TestSyncTypes(t *testing.T) {
	expectClean(t, prelude+`struct Config { var name: string }
val table = Lazy(init: () => [1, 2, 3])
fun follow(w: Watch<i64>, e: Event) {
  await w.changed()
  await e.wait()
}
fun main() throws Lagged {
  val config = RwLock(value: Config(name: "a"))
  with (c = config.read()) {
    io.println(c.name)
  }
  with (w = config.write()) {
    w.name = "b"
  }
  io.println(config.withRead(c => c.name) + config.withWrite(c => c.name))
  val e = Event()
  e.set()
  await e.wait()
  val news = Broadcast<string>(capacity: 4)
  with sub = news.subscribe()
  news.send("x")
  val first = try await sub.recv()
  val w = Watch(value: 1)
  val r = race {
    e.wait() => "event"
    w.changed() => "watch ${w.get()}"
    val v = try sub.recv() => "news $v"
    sleep(Duration.millis(1)) => "timeout"
  }
  io.println("$first $r ${table.get()}")
  scope {
    async follow(w, e)
  }
}`)
	expectError(t, prelude+`fun main() {
  val e = Event()
  e.wait()
}`, "'wait()' always suspends and must be awaited: 'await e.wait()' (D16, D146)")
	expectError(t, prelude+`fun main() {
  val w = Watch(value: 1)
  w.changed()
}`, "'changed()' always suspends and must be awaited")
	expectError(t, prelude+`fun make(): Event => Event()
fun main() {
  race {
    make().wait() => io.println("set")
    sleep(Duration.millis(1)) => io.println("no")
  }
}`, "a race arm on 'wait()' needs its receiver in a variable")
	expectError(t, prelude+`fun main() {
  val l = RwLock(value: 1)
  val x = l.read()
}`, "'read()' holds the lock to the end of a 'with' block, so it is usable only as a 'with' value: 'with n = l.read()'; for one expression, 'l.withRead(n => …)' (D107)")
	expectError(t, prelude+`fun main() {
  val l = RwLock(value: 1)
  with (w = l.write()) {
    await sleep(Duration.millis(1))
  }
}`, "the lock on 'l' is held until the end of its 'with' block")
	expectError(t, prelude+`fun main() {
  val l = RwLock(value: MutableList<i64>())
  scope {
    async hold(l)
  }
}
fun hold(l: RwLock<MutableList<i64>>) { }`, "is not Sendable")
}
