// The operator traits (prelude): a struct gets structural `==`, hashing
// and `Name(field: value)` text for free, and replaces any of them by
// implementing Equatable, Hashable, Comparable or Display.
use io

/// An impl for your own type may sit inside its body (D23); the
/// top-level `implement Trait for Type` below is the same thing written apart.
struct Version {
  major: i64
  minor: i64

  implement Comparable {
    fun compareTo(other: Version): Ordering =
      if (this.major != other.major) this.major.compareTo(other.major)
      else this.minor.compareTo(other.minor)
  }

  implement Display {
    fun toString(): string = "v${this.major}.${this.minor}"
  }
}

/// A case-insensitive name: equal values must hash alike, so Equatable
/// comes with Hashable.
public struct Name {
  text: string

  implement Hashable {
    fun hash(): i64 = this.text.toLower().len()
  }

  implement Equatable {
    fun equals(other: Name): bool = this.text.toLower() == other.text.toLower()
  }
}

struct Pair<T> {
  a: T
  b: T
}

implement<T: Display> Display for Pair<T> {
  fun toString(): string = "<${this.a} | ${this.b}>"
}

sealed trait Shape
struct Circle : Shape {
  r: f64
}
struct Square : Shape {
  side: f64
}

fun area(s: Shape): f64 = when (s) {
  is Circle => 3.0 * s.r * s.r
  is Square => s.side * s.side
}

implement Comparable for Shape {
  fun compareTo(other: Shape): Ordering = area(this).compareTo(area(other))
}

implement Display for Shape {
  fun toString(): string = when (this) {
    is Circle => "circle(${this.r})"
    is Square => "square(${this.side})"
  }
}

/// Generic code orders anything Comparable: numbers and strings implement
/// it in the prelude.
fun largest<T: Comparable>(xs: List<T>): T? {
  var best: T? = null
  loop (x in xs) {
    if (best == null || x.compareTo(best) > 0) best = x
  }
  best
}

fun main() {
  val a = Version(major: 1, minor: 10)
  val b = Version(major: 1, minor: 9)
  io.println("$a > $b: ${a > b}; sorted: ${[a, b, Version(major: 0, minor: 99)].sorted()}")
  io.println("min ${[a, b].min()}, max ${[a, b].max()}, descending ${[b, a].sortedDescending()}")

  val ann = Name(text: "Ann")
  io.println("${ann == Name(text: "ANN")} ${ann != Name(text: "Bob")} ${[ann].contains(Name(text: "ann"))}")
  var counts = mut [ann: 1]
  counts.set(Name(text: "ANN"), 2)
  io.println("${counts.len()} entry, value ${counts.get(Name(text: "aNN"))}")

  io.println("${Pair(a, b)} ${Pair(a: 1, b: 2)}")
  val shapes: List<Shape> = [Square(side: 2.0), Circle(r: 1.0), Square(side: 1.0)]
  io.println("${shapes.sorted()} largest ${shapes.max()}")
  io.println("${largest([3, 9, 4])} ${largest(["pear", "apple"])} ${largest(shapes)}")
  io.println("${(5).compareTo(7)} ${"b".compareTo("a")} ${(2.5).compareTo(2.5)}")
}
