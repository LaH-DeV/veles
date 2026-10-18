// D174: `T.fields` describes a struct's fields while the program is compiled.
// A `const fun` reads it as data; `inline loop (f in T.fields)` is unrolled,
// once per field, with `f` bound to that field. Nothing of it exists when the
// program runs.
use io

annotation Short(letter: string) on field
annotation Hidden on field

struct Options {
  /// How many files to hash at once.
  @Short("w")
  workers: i64 = 4
  @Hidden
  secret: string = ""
  verbose: bool = false
}

struct Plain {
  a: i64
  b: string
}

sealed trait Shape
struct Circle : Shape { r: f64 }

// data: a `const fun` reads names, docs and annotations
const fun flags<T>(): List<string> {
  val out: MutableList<string> = []
  loop (f in T.fields) {
    if (!f.has<Hidden>()) out.push("${f.name} ${f.attr<Short>()?.letter ?: "-"} ${f.typeName} ${f.doc}")
  }
  out.toList()
}

const HELP: List<string> = flags<Options>()

// code: an inline loop gives each copy the field's own type
fun describe<T>(x: T): string {
  val out: MutableList<string> = []
  inline loop (f in T.fields) {
    out.push("${f.name}=${f.get(x)}")
  }
  out.join(", ")
}

fun count<T>(): i64 {
  var n = 0
  inline loop (f in T.fields) {
    if (f.has<Hidden>()) n += 1
    if (val s = f.attr<Short>()) {
      val letter: string = s.letter
      n += letter.len()
    }
  }
  n
}

trait Zero {
  static fun zero(): This => This.build(f => f.Type.default())
}

struct Pair { a: i64; b: string; implement Zero }

fun reads(f: FieldInfo): bool => f.has<Short>() // error: reads annotations, which exist while the program is compiled

fun oops(): i64 {
  inline loop (f in Options.fields) {
    break // error: cannot leave an inline loop
  }
  inline loop (g in Options.fields) {
    val x = g // error: stands for a field of the type being walked
    val y = g.nope // error: a field handle has name, doc, typeName
    val z = g.nope() // error: a field handle has the methods get, has and attr
    val w = g.get() // error: 'get' takes the value to read the field from
    val v = g.has<Plain>() // error: 'Plain' is not an annotation
    val u = g.has() // error: takes the annotation as its type argument
  }
  inline loop (h in 5) { } // error: an inline loop walks the fields of a type
  inline loop :outer (i in Options.fields) { } // error: it has no label
  inline loop (&j in Options.fields) { } // error: an inline loop names the field
  0
}

fun main() {
  io.println("${HELP} ${describe(Options())} ${count<Options>()} ${Pair.zero()}")
  val runtime = flags<Options>() // error: reads annotations, which exist only while the program is compiled
  io.println("${runtime.len()} ${oops()}")
  val bad = i64.fields // error: 'i64' has no fields to read
  val family = Shape.fields // error: 'Shape' is a sealed trait
}

// the fields that carry one annotation: a list for a const fun, a loop for code
const fun mins<T>(): List<string> {
  val out: MutableList<string> = []
  loop ((f, s) in T.annotated<Short>()) out.push("${f.name}=${s.letter}") // error: 'i64' has no fields to read
  out.toList()
}

fun shorts<T>(x: T): i64 {
  var n = 0
  inline loop ((f, s) in T.annotated<Short>()) {
    n += s.letter.len() + f.name.len()
  }
  n
}

const MINS: List<string> = mins<Options>()
const BAD: List<string> = mins<i64>()

fun more() {
  val a = Options.annotated<Plain>() // error: 'Plain' is not an annotation
  io.println("${a.len()}")
  io.println("${MINS} ${shorts(Options())}")
}
