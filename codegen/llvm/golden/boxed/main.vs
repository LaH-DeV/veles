// D198: `Expr` contains itself, so it is a tree and a value is a tag and a
// word ({ i32, [1 x i64] }): `Add`, larger than a word, is built on the heap
// and read through the pointer, and a write through a narrowed `var` copies
// its box first; `Num` fits the word and `Hole`, marked `inline`, stays in
// the value. `Event` is not a tree and stays inline, except `Upload`, marked
// `indirect`.
use io

sealed trait Expr {
  struct Num {
    value: i64
  }
  struct Add {
    left:      *Expr
    right:     *Expr
    var label: string = ""
  }
  inline struct Hole { }
}

fun eval(e: Expr): i64 => when (e) {
  is Num  => e.value
  is Add  => eval(*e.left) + eval(*e.right)
  is Hole => 0
}

sealed trait Event {
  struct Tick {
    at: i64
  }
  indirect struct Upload {
    name: string
    size: i64
  }
}

fun relabel(e: Expr): Expr {
  var copy = e
  if (copy is Add) copy.label = "b"
  copy
}

fun main() {
  val sum: Expr = Add(left: &Num(value: 2), right: &Num(value: 3), label: "a")
  val ev: Event = Upload(name: "a", size: 4)
  io.println("${eval(sum)} ${relabel(sum)} $sum $ev ${Hole()}")
}
