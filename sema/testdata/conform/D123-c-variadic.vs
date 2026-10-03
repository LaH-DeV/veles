// D123: `...` ends the parameter list of a function in an `extern "C"`
// block; the arguments after the named ones get C's promotions and must be
// numbers, raw pointers or C function pointers.
struct Point { x: i64 }

extern "C" {
  fun printf(format: *raw u8, ...): i32
}

fun show(format: *raw u8, none: (*raw u8)?) {
  val small: u8 = 7
  val f: f32 = 1.5
  // SAFETY: printf reads only what its format names; this is a checking case
  unsafe {
    printf(format, 1, -2, 2.5, small, f, true, format, none)
    printf(format, "text") // error: a 'string' cannot be passed to C's variadic arguments
    printf(format, Point(x: 1)) // error: a 'Point' cannot be passed to C's variadic arguments
    printf(format, n: 1) // error: C's variadic arguments are positional, after every named parameter
  }
}

fun main() {
}
