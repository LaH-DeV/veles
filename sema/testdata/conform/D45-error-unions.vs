// D45: a method called on an error union is called on whichever member it
// holds, so every member must give the same type back.
use io

error Low {
  fun code(): i64 => 1
}

error High {
  fun code(): string => "h"
}

fun fails(n: i64): i64 throws Low | High {
  if (n < 0) throw Low()
  if (n > 9) throw High()
  n
}

fun main() {
  when (fails(3)) {
    is Ok(v)  => io.println("$v")
    is Err(e) => io.println("${e.code()}") // error: members of 'High | Low' disagree on the type of 'code'
  }
}
