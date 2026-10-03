// D120: what @packed, @align(n), @transparent and `extern union` apply to.
@packed // error: @packed applies to an extern struct or extern union
struct Plain {
  a: u8
}

@transparent // error: @transparent applies to a plain struct of one field
extern struct Handle {
  fd: i32
}

@transparent
struct Pair { // error: a @transparent struct has exactly one field, which C sees in its place; 'Pair' has 2
  a: i32
  b: i32
}

@align(3) // error: @align takes a power of two from 1 to 4096
struct Odd {
  a: u8
}

@align(8192) // error: @align takes a power of two from 1 to 4096
struct Huge {
  a: u8
}

@align(2)
struct Low { // error: @align(2) is below the natural alignment of 'Low', 8
  a: i64
}

extern struct Fields {
  @align(2)
  a: i32 // error: @align(2) is below the natural alignment of 'i32', 4
  @align(16)
  b: i32
}

struct NotC {
  @align(8) // error: @align on a field applies in an extern struct
  a: i32
}

@align(64)
struct Counter {
  hits: i64
}

extern union Mixed {
  fd: i32
  name: string // error: a field of an extern union is a C type
}

@inline // error: @inline applies to functions
extern union Tagged {
  fd: i32
}

@packed // error: @packed applies to a struct
fun notAStruct() {
}

@align(4) // error: @align applies to a struct, or to a field of an extern struct
fun alsoNot() {
}

@align // error: @align takes one argument, the alignment in bytes
struct Bare {
  a: u8
}

extern union Empty { // error: an extern union needs a field
}

fun main() {
}
