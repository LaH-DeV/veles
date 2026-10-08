// D115: a Closeable a function produces and never closes nor hands on is a
// warning; handing it on in any way is enough, so a hand-off is never
// reported. `val _ = …` discards on purpose.
use io

struct Res {
  n: i64
  implement Closeable { fun close() { io.println("closed ${this.n}") } }
  fun get(): i64 => this.n
}

struct Holder {
  res: Res
}

fun make(n: i64): Res => Res(n)
fun consume(r: Res) { r.close() }

fun leaks(): i64 {
  val r = make(1) // warning: 'r' is never closed: bind it with 'with r = …' so it is closed at the end of the block, close it, or hand it on (D115)
  r.get() + r.n
}

fun leaksAConstructed(): i64 {
  val r = Res(n: 2) // warning: 'r' is never closed
  r.get()
}

fun closes(): i64 {
  val r = make(3)
  val n = r.get()
  r.close()
  n
}

fun returns(): Res {
  val r = make(4)
  r
}

fun passes() {
  val r = make(5)
  consume(r)
}

fun stores(sink: MutableList<Res>): Holder {
  val r = make(6)
  sink.push(r)
  val s = make(7)
  Holder(res: s)
}

fun captures(): fun(): i64 {
  val r = make(8)
  () => r.get()
}

fun aliases() {
  val r = make(9)
  val again = r
  again.close()
}

fun borrowed(h: Holder): i64 {
  val r = h.res
  r.get()
}

fun withBound(): i64 {
  with r = make(10)
  r.get()
}

fun discards() {
  make(11) // warning: the 'Res' this returns is never closed: bind it with 'with', or discard it on purpose with 'val _ = …' (D115)
  val _ = make(12)
}

fun main() {
  io.println("${leaks()} ${leaksAConstructed()} ${closes()} ${returns().get()} ${withBound()}")
  passes()
  val h = stores([])
  io.println("${captures()()} ${borrowed(h)}")
  aliases()
  discards()
}
