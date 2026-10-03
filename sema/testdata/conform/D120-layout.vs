// D120: an extern union is built from one field and touched in `unsafe`;
// it has no equality and no text; a field of a @packed struct is never
// pointed to.
use io

extern union Data {
  ptr: *raw ()
  var fd: i32
  var big: u64
}

@packed
extern struct Event {
  var events: u32
  var data: Data
}

struct Wrapper {
  inner: Data
}

fun main() {
  val d = Data(fd: 3)
  val both = Data(fd: 1, big: 2) // error: a union is built from one field: 'fd' and 'big' would lie on the same bytes
  val none = Data() // error: a union is built from one of its fields
  val read = d.fd // error: a field of the extern union 'Data' is read and written inside 'unsafe'
  // SAFETY: d was built from fd
  val fine = unsafe { d.fd }
  val same = d == d // error: cannot be compared
  io.println("$d") // error: cannot interpolate a value of type 'Data': it holds the extern union 'Data'
  io.println("${Wrapper(inner: d)}") // error: it holds the extern union 'Data'
  val keys: Set<Data> = [] // error: an extern union has no equality
  var ev = Event(events: 1, data: d)
  ev.events = 2
  val p = &ev.events // error: '&ev.events' would point into a @packed struct
  val whole = &ev
  io.println("${fine} ${ev.events} ${read} ${both} ${none} ${same} ${keys.len()} ${p} ${whole}")
}
