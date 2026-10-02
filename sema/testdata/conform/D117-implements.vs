// D117 (compile time): `T implements Trait` asks about a type parameter,
// per instance; each instance checks only the branch it takes.
use io

struct Opaque {
  id: i64
}

fun show<T>(x: T): string {
  val unusedElsewhere = "only read where T is Display"
  if (T implements Display) return "${x.toString()} $unusedElsewhere"
  "<opaque>"
}

fun wrong<T>(x: T): string {
  if (x implements Display) return "x" // error: 'implements' asks about a type parameter of the function or type it is written in, and 'x' is not one
  if (T implements Opaque) return "o" // error: 'Opaque' is not a trait; 'implements' asks whether a type implements a trait
  "neither"
}

fun main() {
  io.println(show(Opaque(id: 1)))
  io.println(wrong(1))
}
