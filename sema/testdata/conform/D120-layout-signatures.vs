// D120: a @packed struct crosses into C by pointer, not by value; an extern
// union has nothing to derive from.
extern union Data {
  ptr: *raw ()
  fd: i32
}

@packed
extern struct Event {
  events: u32
  data: Data
}

extern "C" {
  fun byValue(e: Event): i32 // error: a @packed struct cannot be passed to or from C by value yet
  fun byPointer(e: *raw Event): i32
}

extern "C" fun callback(e: Event): i32 { // error: a @packed struct cannot be passed to or from C by value yet
  return 0
}

implement Encodable for Data { } // error: cannot derive 'Encodable' for the extern union 'Data'

fun main() {
}
