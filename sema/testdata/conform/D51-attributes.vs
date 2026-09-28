// D51/D58: the compiler-known attributes and where each applies.
use io

@inline
@inline // error: duplicate attribute '@inline'
fun twice(x: i64): i64 = x * 2

@deprecated // error: @deprecated takes one argument: the reason
fun old1() { }

@deprecated(reason) // error: @deprecated reason must be a string literal
fun old2() { }

@inline(always) // error: @inline takes no arguments
fun fast() { }

@mustUse // error: @mustUse applies to functions
struct Result2 {
  n: i64
}

@skip // error: @skip applies to a struct field
fun skipped() { }

@required // error: @required applies to a struct field
fun required() { }

struct Wire {
  @required(true) // error: @required takes no arguments
  id: i64?
  @key // error: @key needs the name
  name: string
}

@tag // error: @tag needs the key that names the variant
sealed trait Event

struct Started : Event { }

@tag(1) // error: a @tag key is a string literal
sealed trait Signal

struct Raised : Signal { }

fun main() {
  io.println("${twice(2)}")
}
