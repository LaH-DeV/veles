// D12/D26/D40: traits, sealed traits, and what an `implement` must match.
use io

trait Show {
  fun show(x: i64): string
}

trait Dup {
  fun a(): i64
  fun a(): i64 // error: duplicate trait method 'a'
}

trait Bounded {
  type Item: i64 // error: bound 'i64' is not a trait
}

trait Counted {
  type Item
  fun count(): i64
}

trait Named {
  type Item
  fun name(): string
}

sealed trait NoAssoc { // error: sealed traits cannot declare associated types
  type X
}

sealed trait Marker

sealed trait Shape {
  fun area(): i64
}

struct Square : Shape {
  side: i64
  implement Shape {
    fun area(): i64 => this.side * this.side
  }
}

trait Plain {
  fun p(): i64
}

struct NotVariant : Plain { } // error: 'Plain' is not a sealed trait

sealed trait Pair<T>

struct Half<A, B> : Pair<A> { } // error: variant 'Half' must have the same type parameters as 'Pair'

struct P {
  n: i64
}

implement Marker for P { } // error: sealed trait 'Marker' declares no methods to implement
implement i64 for P { } // error: 'i64' is not a trait
implement Shape for P { // error: 'P' is not a variant of sealed trait 'Shape'
  fun area(): i64 => 0
}

implement Show for P { // warning: can be written inside the body of 'P'
  type Nope = i64 // error: trait 'Show' has no associated type 'Nope'
  fun show(x: i64, y: i64): string => "" // error: method 'show' takes 2 parameters but trait 'Show' declares 1
}

implement Counted for P { // error: must bind associated type 'Item' // warning: can be written inside the body of 'P'
  fun count(): i64 => 1
}

struct Q {
  n: i64
  implement Show {
    fun show(x: string): string => x // error: parameter 'x' has type 'string' but trait 'Show' declares 'i64'
  }
}

error Oops { }

struct R {
  n: i64
  implement Show {
    fun show(x: i64): string throws Oops => "" // error: method 'show' throws but trait 'Show' declares it as non-throwing
  }
}

trait Mapper {
  fun apply<T>(x: T): T
}

struct M {
  n: i64
  implement Mapper {
    fun apply(x: i64): i64 => x // error: method 'apply' must declare the same type parameters as in trait 'Mapper'
  }
}

struct Both {
  n: i64
  implement Counted {
    type Item = i64
    fun count(): i64 => 1
  }
  implement Named {
    type Item = string
    fun name(): string => "both"
  }
}

fun unbound<T>(x: T.Item) { } // error: type parameter 'T' has no bound declaring an associated type 'Item'
fun chained<I: Iterable>(x: I.Iter.Missing) { } // error: cannot resolve 'I.Iter.Missing'
fun main() {
  io.println("${Square(side: 2).area()}")
}
