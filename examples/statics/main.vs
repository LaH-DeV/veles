// Static functions (D23): a `static fun` has no receiver and is called on
// the type. Struct bodies use them for constructors, traits for
// construction from data, and generic code calls them on a type parameter.
use io

struct Point {
  x: i64
  y: i64

  static fun origin(): Point = Point(x: 0, y: 0)

  /// "x,y" → Point; the prelude's `i64.parse` is a static function too.
  static fun fromText(s: string): Point? {
    val parts = s.split(",")
    if (parts.len() != 2) return null
    val x = i64.parse(parts.atOrPanic(0).trim()) ?: return null
    val y = i64.parse(parts.atOrPanic(1).trim()) ?: return null
    Point(x, y)
  }

  fun shifted(dx: i64): Point = Point(x: self.x + dx, y: self.y)

  impl Parsable {
    static fun parse(s: string): Point? = Point.fromText(s)
  }
}

struct Stack<T> {
  items: MutableList<T> = []

  static fun of(x: T): Stack<T> {
    val s = Stack<T>()
    s.items.push(x)
    s
  }
}

/// Parses every element; `T.parse` dispatches on the type parameter.
fun parseAll<T: Parsable>(xs: List<string>): List<T?> = xs.map(x => T.parse(x))

fun main() {
  io.println("${Point.origin()} ${Point.origin().shifted(3)} ${Point.fromText("1, 2")} ${Point.fromText("1")}")
  val p: Point? = Point.parse("5,6")
  io.println("$p ${i64.parse("42")} ${f64.parse("2.5")} ${bool.parse("yes")}")
  val ns: List<i64?> = parseAll(["1", "two", "3"])
  val ps: List<Point?> = parseAll(["0,0", "x"])
  io.println("$ns $ps ${Stack<string>.of("top").items}")
}
