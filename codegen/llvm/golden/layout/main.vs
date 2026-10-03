// D120: C layout. A packed struct is LLVM's packed struct, read and written
// whole; a union is an integer as aligned as it and the bytes after; an
// aligned field gets explicit padding before it; an over-aligned struct
// gets padding after its fields and aligned allocas.
use io

extern union Data {
  var fd: i32
  var big: u64
}

@packed
extern struct Event {
  var events: u32
  var data: Data
}

extern struct Spaced {
  a: u8
  @align(16) b: i32
}

@align(64)
struct Counter {
  var hits: i64
}

struct Holder {
  tag: u8
  var c: Counter
}

fun main() {
  var ev = Event(events: 1, data: Data(fd: 7))
  ev.events = 2
  // SAFETY: data was built from fd
  unsafe { ev.data.fd = ev.data.fd + 1 }
  val s = Spaced(a: 1, b: 2)
  var h = Holder(tag: 3, c: Counter(hits: 4))
  h.c.hits = h.c.hits + 1
  // SAFETY: as above
  io.println("${ev.events} ${unsafe { ev.data.fd }} ${s.b} ${h.c.hits}")
}
