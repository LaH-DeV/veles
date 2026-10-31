# 9. Sealed types, enums and `when`

A **sealed trait** is a type with a fixed set of variants, each a struct
(D12). Where a plain trait says "anything that can do X", a sealed trait
says "exactly one of these". The compiler knows the whole list, so
`when` can check that you handled every case (D13). An **enum** is the
same idea for plain values (D57): a fixed set of names, each a number.

## Modelling a choice

::: code-group

```veles [Veles]
use io

sealed trait Shape
struct Circle : Shape { r: f64 }
struct Rect : Shape { w: f64, h: f64 }
struct Point : Shape { }

fun area(s: Shape): f64 => when (s) {
  is Circle(r) => 3.0 * r * r
  is Rect(w, h) => w * h
  is Point => 0.0
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0), Point()]
  loop (s in shapes) { io.println("$s -> ${area(s)}") }
}
```

```go [Go]
package main

import "fmt"

// Go has no closed set of types: any type with isShape() is a Shape, and
// nothing checks that the switch below handles every one.
type Shape interface{ isShape() }

type Circle struct{ R float64 }
type Rect struct{ W, H float64 }
type Point struct{}

func (Circle) isShape() {}
func (Rect) isShape()   {}
func (Point) isShape()  {}

func area(s Shape) float64 {
	switch s := s.(type) {
	case Circle:
		return 3.0 * s.R * s.R
	case Rect:
		return s.W * s.H
	case Point:
		return 0.0
	}
	panic("unknown shape")
}

func main() {
	shapes := []Shape{Circle{R: 1.0}, Rect{W: 2.0, H: 3.0}, Point{}}
	for _, s := range shapes {
		fmt.Printf("%+v -> %.1f\n", s, area(s))
	}
}

// {R:1} -> 3.0
// {W:2 H:3} -> 6.0
// {} -> 0.0
```

```ts [TypeScript]
// a discriminated union: the `kind` tag tells the variants apart
type Shape =
  | { kind: "circle"; r: number }
  | { kind: "rect"; w: number; h: number }
  | { kind: "point" };

function area(s: Shape): number {
  switch (s.kind) {
    case "circle": return 3.0 * s.r * s.r;
    case "rect": return s.w * s.h;
    case "point": return 0.0;
  }
}

const shapes: Shape[] = [{ kind: "circle", r: 1.0 }, { kind: "rect", w: 2.0, h: 3.0 }, { kind: "point" }];
for (const s of shapes) console.log(`${JSON.stringify(s)} -> ${area(s).toFixed(1)}`);

// {"kind":"circle","r":1} -> 3.0
// {"kind":"rect","w":2,"h":3} -> 6.0
// {"kind":"point"} -> 0.0
```

```swift [Swift]
enum Shape {
    case circle(r: Double)
    case rect(w: Double, h: Double)
    case point
}

func area(_ s: Shape) -> Double {
    switch s {
    case .circle(let r): return 3.0 * r * r
    case .rect(let w, let h): return w * h
    case .point: return 0.0
    }
}

let shapes: [Shape] = [.circle(r: 1.0), .rect(w: 2.0, h: 3.0), .point]
for s in shapes { print("\(s) -> \(area(s))") }

// circle(r: 1.0) -> 3.0
// rect(w: 2.0, h: 3.0) -> 6.0
// point -> 0.0
```

```kotlin [Kotlin]
sealed interface Shape
data class Circle(val r: Double) : Shape
data class Rect(val w: Double, val h: Double) : Shape
data object Point : Shape

fun area(s: Shape): Double = when (s) {
    is Circle -> 3.0 * s.r * s.r
    is Rect -> s.w * s.h
    Point -> 0.0
}

fun main() {
    val shapes: List<Shape> = listOf(Circle(r = 1.0), Rect(w = 2.0, h = 3.0), Point)
    for (s in shapes) println("$s -> ${area(s)}")
}

// Circle(r=1.0) -> 3.0
// Rect(w=2.0, h=3.0) -> 6.0
// Point -> 0.0
```

```rust [Rust]
// an enum's variants are not types of their own, as Circle is in Veles:
// a function cannot take just a Circle
#[derive(Debug)]
enum Shape {
    Circle { r: f64 },
    Rect { w: f64, h: f64 },
    Point,
}

fn area(s: &Shape) -> f64 {
    match s {
        Shape::Circle { r } => 3.0 * r * r,
        Shape::Rect { w, h } => w * h,
        Shape::Point => 0.0,
    }
}

fn main() {
    let shapes = [Shape::Circle { r: 1.0 }, Shape::Rect { w: 2.0, h: 3.0 }, Shape::Point];
    for s in &shapes {
        println!("{s:?} -> {:?}", area(s));
    }
}

// Circle { r: 1.0 } -> 3.0
// Rect { w: 2.0, h: 3.0 } -> 6.0
// Point -> 0.0
```

:::

Output:
```text
Circle(r: 1.0) -> 3.0
Rect(w: 2.0, h: 3.0) -> 6.0
Point -> 0.0
```

- `struct Circle : Shape` makes `Circle` a variant of `Shape`. Variants
  are ordinary structs: construct them, print them, compare them.
- A `Shape` value is stored **inline** as a tagged union — no allocation
  — sized for its largest variant (D12). A sealed type that contains
  itself is laid out differently: see "Trees" below.
- `is Circle(r)` both tests the variant and **destructures** its fields
  by position. `is Point` tests without binding. You can also write
  `is Circle` and then read `s.r`, because inside that arm `s` is
  smart-cast to `Circle`.

The variants may instead be declared **inside** the trait, which keeps the
family in one place. They are the same structs: top-level names, the
trait's visibility, and the `: Shape` clause implied (a generic trait
writes it out, `struct Some<T> : Option<T>`). The formatter keeps the
trait's own methods at the top and the variants after them:

```veles
use io

sealed trait Shape {
  struct Circle { r: f64 }
  struct Rect { w: f64, h: f64 }
}

fun area(s: Shape): f64 => when (s) {
  is Circle(r) => 3.0 * r * r
  is Rect(w, h) => w * h
}

fun main() {
  io.println("${area(Circle(r: 1.0))} ${area(Rect(w: 2.0, h: 3.0))}")
}
```

Output:
```text
3.0 6.0
```

Remove the `is Point` arm and the compiler reports that `when` is not
exhaustive and names the missing variant. Add a fourth variant next year
and every `when` over `Shape` that forgot it stops compiling — that is
the point of sealing.

A variant is also a type of its own. `Circle(r: 1.0)` is a `Shape` — so
`var s = Circle(r: 1.0)` may later hold a `Rect` — but where a `Circle` is
expected, the value stays a `Circle`, and its fields are read and assigned
without a `when`:

```veles
use io

sealed trait Shape
struct Circle : Shape {
  var r: f64
}
struct Rect : Shape {
  w: f64
  h: f64
}

fun scaled(c: Circle, by: f64): Circle {
  var out: Circle = c
  out.r = out.r * by
  out
}

fun main() {
  val c: Circle = Circle(r: 1.5)
  val shapes: List<Shape> = [scaled(c, 2.0), Rect(w: 1.0, h: 2.0)]
  io.println("${c.r} ${shapes.len()}")
}
```

Output:
```text
1.5 2
```

## Guards and `else`

```veles
use io

sealed trait Event
struct Click : Event { x: i64, y: i64 }
struct Key : Event { code: i64 }
struct Quit : Event { }

fun handle(e: Event): string => when (e) {
  is Click(x, y) if x < 0 || y < 0 => "click off-screen"
  is Click(x, y) => "click at $x,$y"
  is Key(code) if code == 27 => "escape"
  is Key => "key ${e.code}"
  is Quit => "bye"
}

fun main() {
  io.println("${handle(Click(x: 3, y: 4))} / ${handle(Click(x: -1, y: 0))} / ${handle(Key(code: 27))} / ${handle(Key(code: 65))} / ${handle(Quit())}")
}
```

Output:
```text
click at 3,4 / click off-screen / escape / key 65 / bye
```

A guarded arm (`if cond`) does not count towards exhaustiveness; there
must be an unguarded arm (or `else`) for each variant.

## Trees: where a variant is stored

A sealed type that contains itself — through a pointer, a list, a map or
another type's field — is a **tree**: an expression made of expressions,
a JSON value holding JSON values. Sizing every value of a tree for its
largest node would copy that much at every step, so a tree is laid out the
way Go and Kotlin lay out such values: each variant larger than a word
lives on the heap, and the value is its tag and a pointer, 16 bytes (D198).
Nothing in the program changes — you construct, match and read the
variants as before, and a value is still a value: changing a copy never
changes the original.

```veles
use io

sealed trait Expr {
  struct Num {
    value: f64
  }
  struct Bin {
    op:    string
    left:  *Expr
    right: *Expr
  }
  inline struct Missing { }
}

fun eval(e: Expr): f64 => when (e) {
  is Num     => e.value
  is Missing => 0.0
  is Bin     => {
    val (a, b) = (eval(*e.left), eval(*e.right))
    if (e.op == "+") a + b else a * b
  }
}

sealed trait Event {
  struct Tick {
    at: i64
  }
  indirect struct Upload {
    name:  string
    owner: string
    bytes: List<u8>
    sent:  string
  }
}

fun main() {
  val two: Expr = Num(value: 2.0)
  val sum: Expr = Bin(op: "+", left: &Num(value: 1.5), right: &two)
  val product: Expr = Bin(op: "*", left: &sum, right: &Num(value: 4.0))
  io.println("${eval(product)} ${eval(Missing())}")
  val events: List<Event> = [Tick(at: 1), Upload(name: "a.txt", owner: "ann", bytes: [1, 2], sent: "now"), Tick(at: 2)]
  io.println("${events.len()} events")
}
```

Output:
```text
14.0 0.0
3 events
```

- `Expr` is a tree (`Bin` holds `*Expr`), so `Bin` is stored behind a
  pointer and built on the heap; `Num` fits a word and stays in the value.
- `inline struct` keeps a variant in the value even in a tree — for a small
  node made very often, when the allocation costs more than the room.
- `indirect struct` stores a variant behind a pointer in a type that is not
  a tree: `Event` is mostly `Tick`s, and without it every `Tick` in the list
  would carry room for an `Upload`. `indirect sealed trait` does it for
  every variant.
- When one variant of a non-tree type is much larger than the others, the
  compiler says so (lint `large_variant`) and offers to add `indirect`.
## Methods on a sealed trait

Shared behaviour goes in the sealed trait's body and dispatches on the
tag, with no vtable:

```veles
use io

sealed trait Expr {
  fun eval(): i64 => when (this) {
    is Num(v) => v
    is Add(l, r) => l.eval() + r.eval()
    is Mul(l, r) => l.eval() * r.eval()
  }
  fun show(): string => when (this) {
    is Num(v) => "$v"
    is Add(l, r) => "(${l.show()} + ${r.show()})"
    is Mul(l, r) => "${l.show()} * ${r.show()}"
  }
}
struct Num : Expr { v: i64 }
struct Add : Expr { l: *Expr, r: *Expr }
struct Mul : Expr { l: *Expr, r: *Expr }

fun main() {
  val e = Add(l: &Num(v: 2), r: &Mul(l: &Num(v: 3), r: &Num(v: 4)))
  io.println("${e.show()} = ${e.eval()}")
}
```

Output:
```text
(2 + 3 * 4) = 14
```

Recursive variants hold pointers (`*Expr`) — a value cannot contain
itself, and the compiler reports an infinite-size type if you try (D31).

A method declared without a body is written by each variant instead, in
its own body; a method with a body is a default a variant may `override`.
A method of a variant that has the name of a method of its sealed trait
*is* that implementation (D195), so there is no `implement Shape { }` to
write around it (an `implement` block for another trait is still how a
variant takes that trait on):

```veles
use io

sealed trait Shape {
  fun area(): f64
  fun describe(): string => "shape with area ${this.area()}"
}
struct Circle : Shape {
  r: f64

  fun area(): f64 => 3.0 * this.r * this.r
  override fun describe(): string => "circle r=${this.r}"
}
struct Rect : Shape {
  w: f64
  h: f64

  fun area(): f64 => this.w * this.h
}

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)]
  loop (s in shapes) {
    io.println("${s.describe()} -> ${s.area()}")
  }
}
```

Output:
```text
circle r=1.0 -> 3.0
shape with area 6.0 -> 6.0
```

A variant that leaves such a method out is an error at the variant, not
at some later call: `variant 'Rect' of 'Shape' does not implement 'area';
write 'fun area(): f64' in its body`. The old form, `implement Shape { … }`
inside the variant, still works; the `redundant_implement` lint offers to
unwrap it.

## Enums: a closed set of values

A sealed trait is a closed set of *shapes*; an **enum** is a closed set of
*values* (D57). Use one when the alternatives carry no data of their
own — a phase, a direction, a level:

```veles
use io

/// The phases of a traffic light.
enum Phase {
  Red
  Amber
  Green
}

enum Level : u8 {
  Low = 1
  Mid
  High = 10
}

fun next(p: Phase): Phase => when (p) {
  Phase.Red   => Phase.Green
  Phase.Green => Phase.Amber
  Phase.Amber => Phase.Red
}

fun main() {
  val p = Phase.Amber
  io.println("$p ${p.value} ${next(p)} ${Phase.values()}")
  io.println("${Level.Mid.value} ${Level.fromValue(10)} ${Level.fromValue(5)} ${Phase.parse("Red")} ${Phase.parse("Blue")}")
  io.println("${Level.Low < Level.High} ${[Level.High, Level.Low].sorted()} ${Level.High == 10} ${Level.Low.compareTo(Level.Mid)}")
}
```

Output:
```text
Amber 1 Red [Red, Amber, Green]
2 High null Red null
true [Low, High] true Less
```

Every member is a number of one integer type — `i64` unless the
declaration says otherwise with `: u8` and the like. A member without a
value is the previous one plus one, and the first counts from 0; two
members cannot share a value, and a value must fit the type. At run
time an enum *is* that number, so it costs nothing to store, compare
or hash.

What you get without writing anything:

| | |
|---|---|
| `Phase.Amber` | a member; `when` over a `Phase` must name every member (or use `else`) |
| `p.value` | the number |
| `p.toString()`, `"$p"` | the member's name, `"Amber"` |
| `Phase.values()` | every member in declaration order |
| `Phase.fromValue(n)` | the member with that number, or `null` |
| `Phase.parse("Amber")` | the member with that name, exactly, or `null` |
| `Phase.parse("amber", case: Case.Any)` | the member in any spelling (`AMBER`, `am-ber`), or `null` (D205) |
| `p.toString(case: Case.Kebab)` | the name in a `Case`: `Lower`, `Upper`, `Camel`, `Pascal`, `Snake`, `Kebab`, `ScreamingSnake` |
| `p.isAmber()` | true for that member alone, one predicate per member (D206) |
| `==`, `<`, `sorted()`, map keys | by the number; an enum is `Comparable` and `Hashable` |

An enum compares with its own base type — `level == 10`, `level < 5` —
but never converts to or from it: a `u8` is not a `Level` (say
`Level.fromValue(n)`), and a `Level` is not a `u8` (say `.value`). It has
no fields and cannot be generic: an enum is a set of values. Every enum is
`Enumerated` (`values`, `parse(text, case:)`, `toString(case:)`), so generic
code reads any of them; two members that are one name in another spelling
(`Ok` and `OK`) are refused, so a `Case.Any` parse is never ambiguous. The
one `implement` an enum may write is `Enumerated`'s, to give `parse` or
`toString(case:)` a meaning of its own:

```veles
use io

enum Arch { X64, Arm64 }

implement Enumerated for Arch {
  static fun parse(text: string, case: Case): Arch? => when (text) {
    "x86_64", "amd64" => Arch.X64
    "aarch64"         => Arch.Arm64
    else              => null
  }
}

fun main() {
  val arch = Arch.parse("x86_64") ?: Arch.Arm64
  io.println("${arch} ${arch.isX64()} ${Arch.Arm64.toString(case: Case.Lower)}")
}
```

Output:
```text
X64 true arm64
``` Behaviour that needs one goes in a function that takes it, like
`next` above, or in an **extension function** (`extend Level { … }`, or
`extend Level with fun next(): Level? => …`), which reads as a method:

```veles
use io

enum Level { Debug, Info, Warn }

extend Level {
  fun label(): string => when (this) {
    Level.Debug => "dbg"
    Level.Info => "inf"
    Level.Warn => "wrn"
  }
  fun isSevere(): bool => this >= Level.Warn
}

fun main() {
  io.println("${Level.Warn.label()} ${Level.Warn.isSevere()} ${Level.Debug.isSevere()}")
}
```

Output:
```text
wrn true false
```

The extension cannot reuse a built-in name (`value`, `toString`, `values`,
`fromValue`, `parse`, a member's `is…`), and from another module it is brought in with the
`use … extend` clause like any extension function. When the alternatives
start to carry data, you have outgrown the enum and want a sealed trait.

The prelude's `Ordering` is an enum — `Less = -1`, `Equal`, `Greater` —
and it is what every `compareTo` returns (chapter 8), so a comparison
reads either way:

```veles
// fragment
when (guess.compareTo(secret)) {
  Ordering.Less    => io.println("Too small!")
  Ordering.Greater => io.println("Too big!")
  Ordering.Equal   => io.println("You win!")
}
if (a.compareTo(b) < 0) io.println("a first")   // -1 < 0: the base type
```

## `when` on other things

`when` also matches numbers and strings (chapter 3), nullable values
(`null =>` / `Some(x) =>`), tuples (`(0, y) => ...`), and `Result`
(`is Ok(v)` / `is Err(e)`, chapter 7). Patterns nest:

```veles
use io

sealed trait Tree
struct Leaf : Tree { n: i64 }
struct Branch : Tree { left: *Tree, right: *Tree }

fun sum(t: Tree): i64 => when (t) {
  is Leaf(n) => n
  is Branch(left, right) => sum(*left) + sum(*right)
}

fun classify(p: (i64, i64)): string => when (p) {
  (0, 0) => "origin"
  (0, _) => "on y axis"
  (_, 0) => "on x axis"
  else => "elsewhere"
}

fun main() {
  val t = Branch(left: &Leaf(n: 1), right: &Branch(left: &Leaf(n: 2), right: &Leaf(n: 3)))
  io.println("${sum(t)} ${classify((0, 5))} ${classify((2, 0))} ${classify((1, 1))}")
}
```

Output:
```text
6 on y axis on x axis elsewhere
```

Generic sealed types work too: `sealed trait Tree<T>` with
`struct Leaf<T> : Tree<T> { value: T }`. `Option<T>` (spelled `T?`) and
`Result<T, E>` are exactly such types in the prelude.

## Picking one variant out of a list

`shapes.filter(s => s is Circle)` gives back a `List<Shape>`: a predicate
is just a function returning `bool`, and nothing it learned survives the
call. When you want the *narrowed* list, say the variant as a type
argument instead:

```veles
use io

sealed trait Shape
struct Circle : Shape { r: f64 }
struct Square : Shape { side: f64 }

fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0), Circle(r: 3.0)]
  val circles = shapes.filterIs<Circle>()        // List<Circle>
  io.println("${circles.map(c => c.r)} ${shapes.filterIs<Square>().len()}")
  val maybe: List<i64?> = [1, null, 3]
  io.println("${maybe.filterNotNull()}")          // List<i64>
}
```

Output:
```text
[1.0, 3.0] 1
[1, 3]
```

`filterNotNull()` is the same idea for `T?`, and `oks()`/`errors()` for a
list of `Result` (chapter 7). There is deliberately no way to write a
predicate that narrows (TypeScript's `x is Circle` return type): the
compiler cannot check that such a function tells the truth, so the
promise would rest on the author. A type argument the compiler fills in
itself costs nothing to trust.

Next: [Closures and iterators](10-closures-and-iterators.md).
