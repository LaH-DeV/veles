// D44/D67/D69: what may cross the C boundary, and how it is declared.
use io

struct Managed {
  n: i64
}

extern struct Point {
  x: i32
  y: i32
}

extern "C" {
  fun printf(format: *raw u8, args: i64...): i32 // error: extern "C" functions cannot be variadic // error: cannot cross the C ABI
  fun keep(p: *Managed) // error: extern "C" functions cannot take GC-managed pointers; use '*raw T'
  fun take(m: Managed) // error: extern "C" functions can only pass 'extern struct' types by value
  fun place(p: Point): Point
}

fun register(cb: extern fun(i64): i64 suspends) { } // error: a C function pointer cannot suspend or throw

extern "C" fun twice(x: i64): i64 => x * 2

fun main() {
  io.println("${twice(2)}")
}
