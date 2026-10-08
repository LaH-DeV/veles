# 9. Sealed types, enums and `when`

A **sealed trait** is a type with a fixed set of variants, each a struct
(D12). Where a plain trait says "anything that can do X", a sealed trait
says "exactly one of these". The compiler knows the whole list, so
`when` can check that you handled every case (D13). An **enum** is the
same idea for plain values (D57): a fixed set of names, each a number.

## Modelling a choice

```veles
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

Output:
```text
Circle(r: 1.0) -> 3.0
Rect(w: 2.0, h: 3.0) -> 6.0
Point -> 0.0
```

- `struct Circle : Shape` makes `Circle` a variant of `Shape`. Variants
  are ordinary structs: construct them, print them, compare them.
- A `Shape` value is stored **inline** as a tagged union — no allocation
  — sized for its largest variant (D12).
- `is Circle(r)` both tests the variant and **destructures** its fields
  by position. `is Point` tests without binding. You can also write
  `is Circle` and then read `s.r`, because inside that arm `s` is
  smart-cast to `Circle`.

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
an `implement` block of its body; a method with a body is a default a
variant may `override`:

```veles
use io

sealed trait Shape {
  fun area(): f64
  fun describe(): string => "shape with area ${this.area()}"
}
struct Circle : Shape {
  r: f64

  implement Shape {
    fun area(): f64 => 3.0 * this.r * this.r
    override fun describe(): string => "circle r=${this.r}"
  }
}
struct Rect : Shape {
  w: f64
  h: f64

  implement Shape {
    fun area(): f64 => this.w * this.h
  }
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
add 'implement Shape { fun area(): f64 }' to its body`.

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
| `Phase.parse("Amber")` | the member with that name, or `null` |
| `==`, `<`, `sorted()`, map keys | by the number; an enum is `Comparable` and `Hashable` |

An enum compares with its own base type — `level == 10`, `level < 5` —
but never converts to or from it: a `u8` is not a `Level` (say
`Level.fromValue(n)`), and a `Level` is not a `u8` (say `.value`). It has
no methods, fields or `implement` blocks of its own, and it cannot be
generic: an enum is a set of values, and behaviour that needs one goes
in a function that takes it, like `next` above. When the alternatives
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
