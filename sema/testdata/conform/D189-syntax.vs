// D189–D194: the membership operator, the loop head that binds, the compound
// bit and wrapping assignments, and variants declared in a sealed trait. The
// file is also read by the selfhost parse gate, so both front ends must agree.
use io

sealed trait Shape {
  struct Circle {
    r: i64
  }
  struct Rect {
    w: i64
    h: i64
  }
}

sealed trait Option<T> {
  struct Some<T> : Option<T> {
    value: T
  }
  struct None<T> : Option<T> { }
}

fun kind(c: u8, xs: List<i64>): bool {
  if (c in '0'..'9') return true
  if (c !in " \t\r\n") return xs.len() in 1..3
  c in ['a', 'e'] || 3 !in xs
}

fun bits() {
  var a: u32 = 0b1010
  a |= 1
  a &= 6
  a ^= 3
  a <<= 2
  a >>= 1
  var w: u8 = 250
  w +%= 10
  w -%= 3
  w *%= 2
  io.println("$a $w")
}

fun next(i: i64): i64? => if (i < 3) i + 1 else null

fun drain() {
  var i: i64 = 0
  loop (val n = next(i) && n < 3) {
    i = n
  }
}

trait Named {
  fun label(): string
}

sealed trait Animal {
  struct Dog {
    name: string

    implement Named {
      fun label(): string => "dog"
    }
    fun sound(): string => "woof"
  }
  struct Cat {
    name: string

    implement Named {
      fun label(): string => "cat"
    }
    fun sound(): string => "meow"
  }
  fun sound(): string
}
