// D215: fields lie in declaration order; `compact struct` lets the compiler
// place them for the smallest size. Refused where C or an attribute fixes
// the layout (`@packed` is already C's only, so an extern one is refused
// first); a warning where there is nothing to place.
use io

compact struct Sample {
  flag: u8
  at:   i64
  kind: u8
}

public compact struct Pair<A, B> {
  first:  A
  second: B
}

sealed trait Event {
  compact struct Hit {
    tag:   u8
    stamp: i64
  }
  indirect compact struct Upload {
    kind:  u8
    bytes: List<u8>
    size:  i64
  }
  inline compact struct Mark {
    a: u8
    b: i64
  }
  struct Done { }
}

compact extern struct Wire { // error: an extern struct is laid out as C lays it out
  a: u8
  b: i64
}

@transparent
compact struct Handle { // error: a @transparent struct is its one field
  fd: i32
}

compact struct Lone { // warning: 'compact' changes nothing for 'Lone'
  only: i64
}

fun main() {
  val s = Sample(flag: 1, at: 2, kind: 3)
  val p = Pair<u8, i64>(first: 4, second: 5)
  val e: Event = Event.Hit(tag: 6, stamp: 7)
  io.println("$s ${p.first} ${p.second} $e ${Lone(only: 8).only}")
}
