# 9. Sealed types and `when`

A **sealed trait** is a type with a fixed set of variants, each a struct
(D12). Where a plain trait says "anything that can do X", a sealed trait
says "exactly one of these". The compiler knows the whole list, so
`when` can check that you handled every case (D13).

## Modelling a choice

```veles
use io

sealed trait Shape
struct Circle : Shape { r: f64 }
struct Rect : Shape { w: f64, h: f64 }
struct Point : Shape { }

fun area(s: Shape): f64 = when (s) {
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

## Guards and `else`

```veles
use io

sealed trait Event
struct Click : Event { x: i64, y: i64 }
struct Key : Event { code: i64 }
struct Quit : Event { }

fun handle(e: Event): string = when (e) {
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
  fun eval(): i64 = when (self) {
    is Num(v) => v
    is Add(l, r) => l.eval() + r.eval()
    is Mul(l, r) => l.eval() * r.eval()
  }
  fun show(): string = when (self) {
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

## `when` on other things

`when` also matches numbers and strings (chapter 3), nullable values
(`null =>` / `Some(x) =>`), tuples (`(0, y) => ...`), and `Result`
(`is Ok(v)` / `is Err(e)`, chapter 7). Patterns nest:

```veles
use io

sealed trait Tree
struct Leaf : Tree { n: i64 }
struct Branch : Tree { left: *Tree, right: *Tree }

fun sum(t: Tree): i64 = when (t) {
  is Leaf(n) => n
  is Branch(left, right) => sum(*left) + sum(*right)
}

fun classify(p: (i64, i64)): string = when (p) {
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

Next: [Closures and iterators](10-closures-and-iterators.md).
