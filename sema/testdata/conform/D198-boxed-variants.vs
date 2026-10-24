// D198: a sealed type that contains itself — a tree — stores its variants
// behind a pointer; `indirect struct` boxes a variant of any sealed type,
// `inline struct` keeps one in the value. A sealed type that is not a tree
// and is sized for one much larger variant is pointed out.
use io

sealed trait Json {                     // a tree: Arr holds Json values
  struct Num {
    value: f64
  }
  struct Arr {
    items: List<Json>
  }
  inline struct Null { }
}

sealed trait Event {
  struct Tick {
    at: i64
  }
  struct Upload { // warning: variant 'Upload' is 9 words, the others at most 1 word: mark it 'indirect' so values of 'Event' are not sized for it (D198)
    name:  string
    body:  string
    owner: string
    sizes: (i64, i64, i64)
  }
}

sealed trait Message {
  struct Ping {
    at: i64
  }
  indirect struct Send {                // the author chose: no warning
    name:  string
    body:  string
    owner: string
    sizes: (i64, i64, i64)
  }
}

indirect sealed trait Shape {
  struct Circle {
    r: f64
  }
  struct Rect {
    w: f64
    h: f64
  }
}

fun main() {
  val doc: Json = Arr(items: [Num(value: 1.0), Null()])
  val e: Event = Tick(at: 1)
  val m: Message = Ping(at: 2)
  val s: Shape = Circle(r: 1.0)
  io.println("$doc $e $m $s")
}
