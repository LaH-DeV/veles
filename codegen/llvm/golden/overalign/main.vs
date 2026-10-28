// `@align(n)` holds wherever the value lies (D120, B32): a tuple element,
// a nullable's value and a sealed or error payload start at the value's
// alignment, with padding before them; and a suspending function's frame
// is allocated as aligned as LLVM laid it out, which the release build
// relies on when it keeps an over-aligned local there across a suspension.
use io

@align(64)
struct Counter {
  var hits: i64
}

@align(64)
struct Wide {
  var a: i64
  b:     i64
  c:     i64
  d:     i64
  e:     i64
  f:     i64
  g:     i64
  h:     i64
  i:     i64
}

sealed trait Slot { }
struct Hot : Slot {
  counter: Counter
}
struct Cold : Slot {
  name: string
  tags: Array<string, 3>
}

error Gone {
  why: string
}

fun count(n: i64): Counter throws Gone {
  if (n < 0) throw Gone(why: "negative")
  Counter(hits: n)
}

fun both(x: i64, y: i64): i64 throws Gone => scope {
  val a = async count(x)
  val b = async count(y)
  (await a).hits + (await b).hits
}

@noinline
fun total(w: Wide): i64 => w.a + w.b + w.i

fun read(s: Slot): string => when (s) {
  is Hot(counter)  => "hot ${counter.hits}"
  is Cold(name, _) => "cold $name"
}

unsafe fun offset(base: *raw u8, at: *raw u8): i64 => at - base

fun main() {
  var pair: (u8, Counter) = (1, Counter(hits: 2))
  pair.1.hits += 1
  var some: Counter? = Counter(hits: 4)
  var slot: Slot = Hot(counter: Counter(hits: 5))
  // SAFETY: addresses inside one local each, only subtracted
  unsafe {
    val inPair = offset((&pair).cast<*raw u8>(), (&pair.1).cast<*raw u8>())
    val inSlot = if (slot is Hot) offset((&slot).cast<*raw u8>(), (&slot.counter).cast<*raw u8>()) else -1
    io.println("offsets $inPair $inSlot")
  }
  io.println("${pair} ${some?.hits} ${read(slot)} ${read(Cold(name: "c", tags: ["", "", ""]))}")

  var w = Wide(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9)
  await sleep(Duration.millis(1))
  w.a += total(w)
  await sleep(Duration.millis(1))
  io.println("frame ${w.a} ${total(w)}")
  io.println("tasks ${both(7, 8) ?? -1} ${both(7, -8) ?? -1}")
}
