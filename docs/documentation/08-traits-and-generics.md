# 8. Traits and generics

A **trait** names a capability: a set of methods a type promises to
provide. There is no inheritance in Veles; traits are how unrelated
types share an interface, how generic code states what it needs, and how
you write one function that works on many types (D6).

## Declaring and implementing

```veles
use io

trait Shape {
  fun area(): f64
  fun name(): string
  fun describe(): string = "${self.name()} with area ${self.area()}"   // default body
}

struct Circle {
  r: f64

  impl Shape {
    fun area(): f64 = 3.0 * self.r * self.r
    fun name(): string = "circle"
  }
}

struct Square {
  side: f64

  impl Shape {
    fun area(): f64 = self.side * self.side
    fun name(): string = "square"
    override fun describe(): string = "a square of side ${self.side}"
  }
}

fun main() {
  io.println(Circle(r: 1.0).describe())
  io.println(Square(side: 2.0).describe())
}
```

Output:
```text
circle with area 3.0
a square of side 2.0
```

- A method with a body in the trait is a **default**; an `impl` may keep
  it or replace it with `override fun`.
- For a type you declare, the impl sits **inside the struct body** as
  `impl Shape { ... }` — the type is implied, and generic structs pass
  their type parameters along. The top-level `impl Shape for Square`
  form is the same thing written apart, and it is the form for a type
  you did not write (`impl Shape for i64`), or when the impl needs
  bounds the struct does not have (`impl<T: Display> Display for
  Pair<T>`). Writing it at top level for your own type in the same
  module is a lint: the compiler warns, and the editor's quick fix moves
  it into the body (`veles check --fix` does the same in bulk).
- A type may implement any number of traits, and you may implement
  *your* trait for a type you did not write — `impl Shape for i64` is
  legal. What is not legal is two impls of the same trait for the same
  type anywhere in the program (D17).
- Trait method calls need no import; if two traits in scope both define
  `describe` for the same type, the call is an error rather than a guess
  (D26).

## Generic functions with bounds

`<T: Shape>` says "any `T` that implements `Shape`", and inside the
function you may call exactly the methods `Shape` promises:

```veles
use io

trait Shape { fun area(): f64 }
struct Circle { r: f64; impl Shape { fun area(): f64 = 3.0 * self.r * self.r } }
struct Square { side: f64; impl Shape { fun area(): f64 = self.side * self.side } }

fun <T: Shape> largest(shapes: List<T>): T? {
  var best: T? = null
  loop (s in shapes) {
    if (best == null || s.area() > best.area()) best = s
  }
  best
}

fun main() {
  val big = largest([Square(side: 1.0), Square(side: 3.0), Square(side: 2.0)])
  io.println("${big?.area() ?: 0.0}")
}
```

Output:
```text
9.0
```

The compiler compiles `largest` once per `T` it is used with — a process
called stenciling (D8). The call `s.area()` becomes a direct call to
`Square.area`; there is no dynamic dispatch and nothing is boxed. This is
the form to prefer when all elements have the same type.

## Trait objects: mixing types at run time

Sometimes a list must hold *different* shapes. Use the trait itself as
the type: a `Shape` value carries a pointer to the data and a table of
its methods (D9):

```veles
use io

trait Shape {
  fun area(): f64
  fun name(): string
}
struct Circle {
  r: f64
  impl Shape {
    fun area(): f64 = 3.0 * self.r * self.r
    fun name(): string = "circle"
  }
}
struct Square {
  side: f64
  impl Shape {
    fun area(): f64 = self.side * self.side
    fun name(): string = "square"
  }
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  var total = 0.0
  loop (s in shapes) {
    io.println("${s.name()} ${s.area()}")
    total += s.area()
  }
  io.println("total $total")
}
```

Output:
```text
circle 3.0
square 4.0
total 7.0
```

A `Circle` converts to a `Shape` wherever a `Shape` is expected. Not
every trait can be used this way: one with generic methods, associated
types, or `Self` in a signature has no single method table to build, and
the compiler says so.

## Traits on built-in types, and generic structs

```veles
use io

trait Show { fun show(): string }

impl Show for i64 { fun show(): string = "#$self" }
impl Show for string { fun show(): string = "'$self'" }

struct Box<T: Show> {
  item: T
  fun label(): string = "[${self.item.show()}]"
}

fun <T: Show> showAll(xs: List<T>): string = xs.map(x => x.show()).join(" ")

fun main() {
  io.println("${Box(item: 7).label()} ${Box(item: "hi").label()}")
  io.println(showAll([1, 2, 3]))
}
```

Output:
```text
[#7] ['hi']
#1 #2 #3
```

## Static trait functions

A trait may declare a `static fun` — a function without `self`, called
on the type. Implementations write `static fun` too, and generic code
calls it on the type parameter: `T.parse(s)` picks the impl for whatever
`T` is at each call. The prelude's `Parsable` is the standard example:

```veles
use io

struct Celsius {
  degrees: f64

  impl Parsable {
    static fun parse(s: string): Celsius? {
      val n = f64.parse(s.trimEnd().replace("C", "")) ?: return null
      Celsius(degrees: n)
    }
  }
}

fun parseAll<T: Parsable>(xs: List<string>): List<T?> = xs.map(x => T.parse(x))

fun main() {
  val temps: List<Celsius?> = parseAll(["21.5C", "cold"])
  val ints: List<i64?> = parseAll(["1", "2", "x"])
  io.println("$temps $ints ${bool.parse("true")}")
}
```

Output:
```text
[Celsius(degrees: 21.5), null] [1, 2, null] true
```

A trait with a static function cannot be a trait object (there is no
value to dispatch on), and sealed traits cannot declare one.

## The operator traits

Every struct can be compared with `==`, used as a map key and printed
with `$x` without writing anything: equality is field by field, so is the
hash, and the text is `Name(field: value, ...)`. Four prelude traits let a
type replace that behaviour:

| Trait | Method | Replaces |
|---|---|---|
| `Comparable` | `compareTo(other: Self): i64` | `<`, `<=`, `>`, `>=`, `sorted()`, `min()`, `max()` |
| `Equatable` | `equals(other: Self): bool` | `==`, `!=`, `contains`, `indexOf` |
| `Hashable` | `hash(): i64` | map keys and set elements |
| `Display` | `toString(): string` | interpolation, `"$x"` |

`compareTo` returns a negative number, zero or a positive number. Numbers
and strings implement `Comparable` in the prelude, so a bound `T:
Comparable` accepts `i64`, `string` and your own types alike — this is
what `min()` and `sorted()` demand of their elements.

```veles
use io

struct Version {
  major: i64
  minor: i64

  impl Comparable {
    fun compareTo(other: Version): i64 =
      if (self.major != other.major) self.major.compareTo(other.major)
      else self.minor.compareTo(other.minor)
  }

  impl Display {
    fun toString(): string = "v${self.major}.${self.minor}"
  }
}

struct Name {
  text: string

  impl Equatable {
    fun equals(other: Name): bool = self.text.toLower() == other.text.toLower()
  }

  impl Hashable {
    fun hash(): i64 = self.text.toLower().len()
  }
}

fun largest<T: Comparable>(a: T, b: T): T = if (a.compareTo(b) >= 0) a else b

fun main() {
  val a = Version(major: 1, minor: 10)
  val b = Version(major: 1, minor: 9)
  io.println("${a > b} ${[a, b].sorted()} ${largest(a, b)} ${largest("x", "y")}")
  var seen = mut [Name(text: "Ann"): 1]
  seen.set(Name(text: "ANN"), 2)
  io.println("${Name(text: "ann") == Name(text: "ANN")} ${seen.len()}")
}
```

Output:
```text
true [v1.9, v1.10] v1.10 y
true 1
```

Two rules keep this honest. A type that implements `Equatable` must also
implement `Hashable` before it can be a map key, because values that are
equal must hash alike. And the built-in types keep their meaning: an
`impl Display for i64` in your package is accepted but interpolation of
an `i64` still prints the number.

## Comparing values

Three questions have three separate answers in Veles: *are these equal?*
(`==`), *which comes first?* (`<`, `sorted()`), and *can this be a key?*
(maps and sets). The table says what each type does by default:

| Type | `==` | `<`, `sorted()`, `min()` | Map key / set element |
|---|---|---|---|
| numbers, `bool`, `string` | by value | numbers and strings: natural order | yes |
| tuples | element by element | no (use a key or comparator) | when the elements are |
| structs | field by field | only with `impl Comparable` | when the fields are |
| sealed types | same variant, equal fields | only with `impl Comparable` | when the variants are |
| `T?` | both null, or both present and equal | no | when `T` is |
| `List`, `Map`, `Set` | by content (chapter 4) | no | immutable ones, when the elements are |
| `MutableList`, `MutableMap`, `MutableSet` | by content | no | never — they can change after being stored |
| pointers, channels, tasks | same object (identity) | no | pointers: by identity |
| functions | not comparable | no | no |

A struct changes any answer by implementing the matching trait —
`Equatable` for `==`, `Comparable` for ordering, `Hashable` for keys — as
`Version` and `Name` do above. A struct that defines `equals` must define
`hash` too before it can be a key, because equal values must hash alike.

### Comparing with a different strategy

The trait gives a type *one* natural meaning. Every other meaning is
passed at the call, as a **key** (a function from the element to something
`Comparable`) or a **comparator** (a function of two elements returning a
negative number, zero or a positive number):

| Want | Write |
|---|---|
| natural order | `xs.sorted()`, `xs.min()`, `xs.max()`, `xs.sortedDescending()` |
| order by a field or derived value | `xs.sortedBy(x => x.age)`, `xs.sortedByDescending(key)`, `xs.minBy(key)`, `xs.maxBy(key)` |
| any order at all | `xs.sortedWith((a, b) => ...)`, `xs.minWith(compare)`, `xs.maxWith(compare)`, `ml.sortWith(compare)` |
| uniqueness by a key | `xs.distinctBy(x => x.email.toLower())` |
| a queue in a custom order | `priorityQueueBy<T>((a, b) => ...)` |
| equality with a different meaning | `xs.any(x => sameName(x, y))`, or a wrapper struct with its own `Equatable` |

Sorting is stable: elements the strategy cannot tell apart keep their
order, so sorting by one key and then by another gives a two-level order.

```veles
use io

struct Employee {
  name: string
  dept: string
  age:  i64
}

fun main() {
  val staff = [
    Employee(name: "Ola", dept: "ops", age: 41),
    Employee(name: "ann", dept: "dev", age: 29),
    Employee(name: "Bob", dept: "dev", age: 35),
    Employee(name: "Ann", dept: "ops", age: 29),
  ]
  val names = (xs: List<Employee>) => xs.map(e => e.name)
  // a key: youngest first; ties keep the list's order
  io.println("${names(staff.sortedBy(e => e.age))}")
  // a comparator: by department, then oldest first within it
  val byDeptThenAge = (a: Employee, b: Employee) =>
    if (a.dept != b.dept) a.dept.compareTo(b.dept) else b.age.compareTo(a.age)
  io.println("${names(staff.sortedWith(byDeptThenAge))}")
  // extremes by key or comparator
  io.println("${staff.maxBy(e => e.age)?.name} ${staff.minWith((a, b) => a.name.toLower().compareTo(b.name.toLower()))?.name}")
  // one entry per name, case-insensitively; and case-insensitive membership
  io.println("${names(staff.distinctBy(e => e.name.toLower()))} ${staff.any(e => e.name.toLower() == "bob")}")
}
```

Output:
```text
[ann, Ann, Bob, Ola]
[Bob, ann, Ola, Ann]
Ola ann
[Ola, ann, Bob] true
```

When a *different equality* is needed everywhere a type appears — not
just in one call — give the type an `Equatable` impl (with `Hashable`), as
`Name` does above. When two meanings are needed for one type, wrap it: a
`struct CaseInsensitive { text: string }` with its own `Equatable` and
`Hashable` is a key that treats `"Ann"` and `"ann"` as one, while plain
`string` keys stay exact.

## Associated types

A trait can declare a *type* its implementors choose, not just methods.
The standard `Iterator` is the canonical example:

```veles
// fragment — this is how the prelude declares it
pub trait Iterator {
  type Item
  mut fun next(): Item?
}
```

Each iterator names its `Item`; `next()` returns `Item?`, null at the
end. Inside the trait the associated type is used by its bare name;
outside, as `I.Item`. [Chapter 10](10-closures-and-iterators.md) shows
an implementation and what it buys you.

## Effects on trait methods

A trait method that may fail or suspend must say so, because callers
dispatch on the trait, not on any one body (D40). For errors there are
two spellings:

- `throws FetchError` fixes the error type: every implementation throws
  `FetchError` (or nothing), and callers see `FetchError` wherever they
  call it.
- a bare `throws` leaves the error to each implementation. The
  implementation's `fetch` declares its own, or just writes `throws` and
  lets the compiler infer it from the body as for any function; one that
  cannot fail costs its callers nothing. Generic code names it `F.Error`:

```veles
use io

error HttpError { status: i64 }
error Missing { name: string }

trait Fetcher {
  fun fetch(url: string): string throws
}

struct Http {
  impl Fetcher {
    fun fetch(url: string): string throws {
      if (url.startsWith("bad")) throw HttpError(status: 500)
      "http:$url"
    }
  }
}

struct Memory {
  data: Map<string, string>

  impl Fetcher {
    fun fetch(url: string): string throws Missing = self.data.get(url) ?: throw Missing(name: url)
  }
}

fun load<F: Fetcher>(f: F, url: string): string throws F.Error {
  val body = try f.fetch(url)
  "[$body]"
}

fun main() {
  io.println("${load(Http(), "x")}")
  val r = load(Http(), "bad")
  if (r.err) io.println("http: ${r.message()}")
  val m = load(Memory(data: ["k": "v"]), "zz")
  if (m.err) io.println("memory: ${m.name}")
}
```

Output:
```text
Ok(value: [http:x])
http: HttpError(status: 500)
memory: zz
```

`load` with `Memory` throws exactly `Missing` — the error is the
implementation's, not a union of everything any implementation might
throw. A trait whose error is left to the implementation cannot be used
as a trait object yet (`Fetcher` as a value); give it a fixed error type
for that.

Next: [Sealed types and `when`](09-sealed-types.md).
