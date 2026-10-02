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
  fun describe(): string = "${this.name()} with area ${this.area()}"   // default body
}

struct Circle {
  r: f64

  implement Shape {
    fun area(): f64 = 3.0 * this.r * this.r
    fun name(): string = "circle"
  }
}

struct Square {
  side: f64

  implement Shape {
    fun area(): f64 = this.side * this.side
    fun name(): string = "square"
    override fun describe(): string = "a square of side ${this.side}"
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

- A method with a body in the trait is a **default**; an `implement` may keep
  it or replace it with `override fun`.
- For a type you declare, the implement sits **inside the struct body** as
  `implement Shape { ... }` — the type is implied, and generic structs pass
  their type parameters along. The top-level `implement Shape for Square`
  form is the same thing written apart, and it is the form for a type
  you did not write (`implement Shape for i64`), or when the implement needs
  bounds the struct does not have (`implement<T: Display> Display for
  Pair<T>`). Writing it at top level for your own type in the same
  module is a lint: the compiler warns, and the editor's quick fix moves
  it into the body (`veles check --fix` does the same in bulk).
- An implement must write every method the trait declares without a
  body; one that leaves some out is an error per method, and the
  editor's **Add the missing methods** writes them for you as stubs
  (`fun name(): string = panic("'name' is not written yet")`) to fill in.
- A type may implement any number of traits, and you may implement
  *your* trait for a type you did not write — `implement Shape for i64` is
  legal. What is not legal is two impls of the same trait for the same
  type anywhere in the program (D17).
- Trait method calls need no import; if two traits in scope both define
  `describe` for the same type, the call is an error rather than a guess
  (D26).

### If you know interfaces from another language

A trait is close to an interface, with these differences — each one a
deliberate choice, so it is worth knowing which is which.

| | TypeScript / Go interface | Kotlin interface | Veles trait |
|---|---|---|---|
| How a type gets it | structurally: having the methods is enough | `class C : I` at the class | `implement I for C` (or `implement I { }` in the body): explicit, by name |
| For a type you did not write | Go: yes (structural); TS: yes | no (extension functions do not implement interfaces) | yes: `implement Display for i64` in your package |
| Default method bodies | TS: no (abstract only); Go: no | yes | yes; an implement that replaces one writes `override` |
| Static / constructor-like members | TS: no; Go: no | companion objects | `static fun` on the trait: `T.parse(s)` in generic code |
| Associated types | generics on the interface | generics | `type Item` inside the trait (`Iterator.Item`) |
| A closed set of implementors | discriminated union (TS) | `sealed interface` | `sealed trait` + variant structs (chapter 9) |
| Used as a runtime value | yes | yes | yes, as a trait object `val s: Shape = ...` (boxed), or statically through a bound `<T: Shape>` |

The structural-vs-explicit difference is the one that changes how you
write code: in Veles a type never satisfies a trait by accident, and the
compiler tells you *at the implement* what is missing, not at the first call.
The price is one `implement` line per type; the return is that `Comparable`,
`Display` and your own traits can carry meaning ("this type promises
its `compareTo` is a total order") rather than only a method shape.

## Generic functions with bounds

`<T: Shape>` says "any `T` that implements `Shape`", and inside the
function you may call exactly the methods `Shape` promises:

```veles
use io

trait Shape { fun area(): f64 }
struct Circle { r: f64; implement Shape { fun area(): f64 = 3.0 * this.r * this.r } }
struct Square { side: f64; implement Shape { fun area(): f64 = this.side * this.side } }

fun largest<T: Shape>(shapes: List<T>): T? {
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
  implement Shape {
    fun area(): f64 = 3.0 * this.r * this.r
    fun name(): string = "circle"
  }
}
struct Square {
  side: f64
  implement Shape {
    fun area(): f64 = this.side * this.side
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
types, static functions, or `Self` in a signature has no single method
table to build, and the compiler says so.

### Objects of a trait that requires another

A trait may require others — `trait Shape : Named` means every `Shape` is
also a `Named`. The object carries one table with both traits' methods in
it, the required ones first, so an inherited method is called like any
other and a default body a super declares is in the table too:

```veles
use io

trait Named {
  fun name(): string
  fun shout(): string = "${this.name()}!"
}
trait Shape : Named {
  fun area(): f64
}
struct Square {
  side: f64
  implement Named { fun name(): string = "square" }
  implement Shape { fun area(): f64 = this.side * this.side }
}

fun main() {
  val s: Shape = Square(side: 2.0)
  io.println("${s.shout()} ${s.name()} ${s.area()}")
}
```

Output:
```text
square! square 4.0
```

A trait whose whole content is its requirements — `trait Codable :
Encodable + Decodable { }` — is an object too, built from the implements
of its parts; nothing implements it directly. Two required traits that
declare the same method name are an ambiguity, and the compiler refuses
the object rather than pick one. What a trait object still cannot do is
become *another* trait's object: `val n: Named = s` needs the concrete
type back, and a trait object has forgotten it.

## Traits on built-in types, and generic structs

```veles
use io

trait Show { fun show(): string }

implement Show for i64 { fun show(): string = "#$this" }
implement Show for string { fun show(): string = "'$this'" }

struct Box<T: Show> {
  item: T
  fun label(): string = "[${this.item.show()}]"
}

fun showAll<T: Show>(xs: List<T>): string = xs.map(x => x.show()).join(" ")

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

`Box(item: 7)` is a `Box<i64>`: a constructor takes its type arguments
from what it is given. So does a static function of a generic type (D137):
`Pair.of(1, "a")` below is a `Pair<i64, string>`, and the prelude's
`MutableList.repeat(false, 3)` a `MutableList<bool>`. When the arguments
say nothing — `Pair.empty()` — the expected type can
(`val p: Pair<i64, string>? = Pair.empty()`), or the type arguments are
written: `Pair<i64, string>.empty()`.

```veles
use io

struct Pair<A, B> {
  first: A
  second: B
  public static fun of(a: A, b: B): Pair<A, B> = Pair(first: a, second: b)
  public static fun empty(): Pair<A, B>? = null
}

fun main() {
  val p = Pair.of(1, "a")
  val none: Pair<i64, string>? = Pair.empty()
  val flags = MutableList.repeat(false, 3)
  io.println("${p.first} ${p.second} ${none == null} $flags")
}
```

Output:
```text
1 a true [false, false, false]
```

## Static trait functions

A trait may declare a `static fun` — a function without `this`, called
on the type. Implementations write `static fun` too, and generic code
calls it on the type parameter: `T.parse(s)` picks the implement for whatever
`T` is at each call. The prelude's `Parsable` is the standard example:

```veles
use io

struct Celsius {
  degrees: f64

  implement Parsable {
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
| `Comparable` | `compareTo(other: Self): Ordering` | `<`, `<=`, `>`, `>=`, `sorted()`, `min()`, `max()` |
| `Equatable` | `equals(other: Self): bool` | `==`, `!=`, `contains`, `indexOf` |
| `Hashable` | `hash(): i64` | map keys and set elements |
| `Display` | `toString(): string` | interpolation, `"$x"` |

`compareTo` returns `Ordering.Less`, `Ordering.Equal` or `Ordering.Greater` — an
enum (chapter 9) whose values are -1, 0 and 1, so `a.compareTo(b) < 0` reads
as it always has. Numbers
and strings implement `Comparable` in the prelude, so a bound `T:
Comparable` accepts `i64`, `string` and your own types alike — this is
what `min()` and `sorted()` demand of their elements.

```veles
use io

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

struct Name {
  text: string

  implement Equatable {
    fun equals(other: Name): bool = this.text.toLower() == other.text.toLower()
  }

  implement Hashable {
    fun hash(): i64 = this.text.toLower().len()
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
`implement Display for i64` in your package is accepted but interpolation of
an `i64` still prints the number.

### Arithmetic: `+ - * /` and unary `-`

Five more traits give a type the arithmetic operators (D71). Each has
associated types for the right operand and the result, read off the method
you write, so an implement is just the method — and the operand and result
may be other types:

| Trait | Method | Operator |
|---|---|---|
| `Addable` | `plus(other: Rhs): Out` | `a + b`, `a += b` |
| `Subtractable` | `minus(other: Rhs): Out` | `a - b`, `a -= b` |
| `Multipliable` | `times(other: Rhs): Out` | `a * b`, `a *= b` |
| `Divisible` | `dividedBy(other: Rhs): Out` | `a / b`, `a /= b` |
| `Negatable` | `negate(): Out` | `-a` |

```veles
use io

struct Money {
  cents: i64

  implement Addable {
    fun plus(other: Money): Money = Money(cents: this.cents + other.cents)
  }
  implement Multipliable {
    fun times(other: i64): Money = Money(cents: this.cents * other)
  }
  implement Display {
    fun toString(): string = "€${this.cents / 100}.${"${this.cents % 100}".padStart(2, "0")}"
  }
}

fun total<T: Addable>(items: List<T>, zero: T): T {
  var sum = zero
  loop (x in items) {
    sum += x
  }
  sum
}

fun main() {
  val coffee = Money(cents: 350)
  io.println("${coffee * 2 + Money(cents: 120)} ${total([coffee, coffee], Money(cents: 0))}")
  io.println("${Duration.seconds(90) + Duration.millis(500)} ${-Duration.seconds(1)}")
}
```

Output:
```text
€8.20 €7.00
1m30.5s -1s
```

`Duration` and `time.Timestamp` implement them (`t + Duration.days(1)`);
the difference of two timestamps is `t.since(earlier)`, since an
operator means one thing per type. Numbers keep their own arithmetic
(D21); an operator a type does not implement is an error that names the
trait to implement.

## Comparing values

Three questions have three separate answers in Veles: *are these equal?*
(`==`), *which comes first?* (`<`, `sorted()`), and *can this be a key?*
(maps and sets). The table says what each type does by default:

| Type | `==` | `<`, `sorted()`, `min()` | Map key / set element |
|---|---|---|---|
| numbers, `bool`, `string` | by value | numbers and strings: natural order | yes |
| tuples | element by element | no (use a key or comparator) | when the elements are |
| structs | field by field | only with `implement Comparable` | when the fields are |
| sealed types | same variant, equal fields | only with `implement Comparable` | when the variants are |
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
| a queue in a custom order | `PriorityQueue<T>(compare: (a, b) => ...)` |
| several keys at once | a tuple key: `xs.sortedBy(e => (-e.size, e.name))` — tuples of `Comparable` elements compare lexicographically, so this is "largest first, then by name" |
| equality with a different meaning | `xs.any(x => sameName(x, y))`, or a wrapper struct with its own `Equatable` |

Sorting is stable: elements the strategy cannot tell apart keep their
order, so sorting by one key and then by another gives a two-level order.
A tuple orders element by element (`(1, "b") < (2, "a")`), which makes
`sortedBy` with a tuple key the way to sort on several fields; negate a
number to flip its direction, and write a comparator when a *string* must
run backwards.

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
just in one call — give the type an `Equatable` implement (with `Hashable`), as
`Name` does above. When two meanings are needed for one type, wrap it: a
`struct CaseInsensitive { text: string }` with its own `Equatable` and
`Hashable` is a key that treats `"Ann"` and `"ann"` as one, while plain
`string` keys stay exact.

## Associated types

A trait can declare a *type* its implementors choose, not just methods.
The standard `Iterator` is the canonical example:

```veles
// fragment — this is how the prelude declares it
public trait Iterator {
  type Item
  fun next(): Item?
}
```

Each iterator names its `Item`; `next()` returns `Item?`, null at the
end. Inside the trait the associated type is used by its bare name;
outside, as `I.Item`. [Chapter 10](10-closures-and-iterators.md) shows
an implementation and what it buys you.

When two bounds of one type parameter both declare an associated type of
the same name, `T.Item` would have to guess, so it is an error; name the
trait whose `Item` you mean (D84):

```veles
// fragment
fun keys<T: Keyed + Iterable>(x: T): List<T.Keyed.Item> { ... }
```

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
  implement Fetcher {
    fun fetch(url: string): string throws {
      if (url.startsWith("bad")) throw HttpError(status: 500)
      "http:$url"
    }
  }
}

struct Memory {
  data: Map<string, string>

  implement Fetcher {
    fun fetch(url: string): string throws Missing = this.data.get(url) ?: throw Missing(name: url)
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

Next: [Sealed types, enums and `when`](09-sealed-types.md).
