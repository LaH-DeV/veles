// D45: an error union can be written as a type argument — the `E` of a
// `Result` — as the checker prints it; names joined by '|' where a constant
// goes (D121) are still the constant expression.
use io

error Low {
}

error High {
}

const LOW: i64 = 4
const HIGH: i64 = 8

fun fails(n: i64): i64 throws Low | High {
  if (n < 0) throw Low()
  if (n > 9) throw High()
  n
}

fun main() {
  val results: MutableList<Result<i64, Low | High>> = []
  results.push(fails(3))
  val tasks: MutableList<Task<Result<i64, Low | High>>> = []
  scope {
    tasks.push(async fails(4))
  }
  val one: Result<i64, Low> = fails(5) // error: type mismatch: expected 'Result<i64, Low>', found 'Result<i64, High | Low>'
  val bits: Array<u8, LOW | HIGH> = Array.make(0)
  io.println("${results.len()} ${tasks.len()} ${bits.len()} ${one.ok}")
}
