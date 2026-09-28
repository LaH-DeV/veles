// D58 supertraits: a trait may require others, each once, never itself.
use io

trait Named {
  fun name(): string
}

struct NotTrait {
  n: i64
}

trait Shape : Named + Named { // error: supertrait 'Named' is listed twice
  fun area(): i64
}

trait Loop : Loop { } // error: trait 'Loop' cannot require itself

trait Odd : NotTrait { } // error: supertrait 'NotTrait' is not a trait

trait Shared : Sendable { } // error: Sendable is derived from a type's fields and cannot be required as a supertrait

sealed trait Kind : Named // error: a sealed trait has no supertraits

fun main() {
  io.println("supertraits")
}
