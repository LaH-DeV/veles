// D45: error types after `throws` are joined with '|'; a comma is caught
// with a fix, and the declaration still means the union.
use io

error Low {
}

error High {
}

fun fails(n: i64): i64 throws Low, High { // error: error types are joined with '|', not ','
  if (n < 0) throw Low()
  if (n > 9) throw High()
  n
}

fun main() {
  when (fails(3)) {
    is Ok(v)        => io.println("$v")
    is Err(e: Low)  => io.println("low")
    is Err(e: High) => io.println("high")
  }
}
