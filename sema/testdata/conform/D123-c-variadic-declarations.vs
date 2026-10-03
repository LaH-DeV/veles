// D123: only a function in an `extern "C"` block may end its parameters with
// `...`, and (for now) not one passing a struct by value.
extern struct Pair {
  a: i32
  b: i32
}

extern "C" {
  fun takesPair(p: Pair, ...): i32 // error: a C function with variadic arguments cannot take or return the struct 'Pair' by value yet
}

fun notC(format: string, ...): i32 = 0 // error: only a function in an 'extern "C"' block takes C's variadic arguments

extern "C" fun callback(n: i32, ...): i32 { // error: only a function in an 'extern "C"' block takes C's variadic arguments
  return n
}

fun main() {
}
