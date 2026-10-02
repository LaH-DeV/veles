// D119: an empty `implement Default` builds the value from the fields'
// defaults and their types' `default()`; what has neither is named, and
// the types with no obvious start are written by hand.
use io

struct Handle {
  fd: i32
}

struct Conn {
  handle: Handle
  retries: i64
  implement Default // error: cannot derive 'Default' for 'Conn': field 'handle' has no default, and its type 'Handle' is not Default
}

struct Opened {
  path: string
  init(mode: string) { }
  implement Default // error: its 'init' takes 'mode', which nothing supplies
}

sealed trait Shape { }
struct Dot : Shape { }

implement Default for Shape { } // error: a sealed trait has no one variant to start from

enum Level { Low, High }

implement Default for Level { } // error: cannot implement 'Default' for enum 'Level': an enum is a set of values with no methods of its own

struct Worker {
  job: Task<()>
  implement Default // error: it holds a running task, which starts where the value is built (D111)
}

struct Fine {
  count: i64
  handle: Handle = Handle(fd: -1)
  implement Default
}

fun main() {
  io.println("${Fine.default()}")
}
