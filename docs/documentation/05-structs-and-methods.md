# 5. Structs and methods

## Defining a type

```veles
use io

struct Point {
  x: i64
  y: i64

  fun manhattan(): i64 = if (self.x < 0) -self.x else self.x + (if (self.y < 0) -self.y else self.y)
  fun moved(dx: i64, dy: i64): Point = Point(x: self.x + dx, y: self.y + dy)
}

fun main() {
  val p = Point(x: 3, y: -4)
  val q = p.moved(1, 1)
  io.println("$p ${p.manhattan()} $q ${p == Point(x: 3, y: -4)}")
}
```

Output:
```text
Point(x: 3, y: -4) 7 Point(x: 4, y: -3) true
```

- Fields are declared one per line (or comma-separated on one line:
  `struct Point { x: i64, y: i64 }`).
- Construction uses the field names: `Point(x: 3, y: -4)`. Order does
  not matter and nothing can be forgotten. A variable named like the
  field can stand alone: `Point(x, y)` is `Point(x: x, y: y)` (and
  writing `x: x` is a warning with a fix). Nothing else may be passed
  bare — `Point(3, -4)` is an error.
- Methods live in the struct body and refer to the receiver as `self`.
- Every struct can be printed with `$p`, compared with `==`, and used as
  a map key, automatically. To order structs with `<` or `sorted()`, or
  to change what `==` and `$p` mean, implement the operator traits of
  [chapter 8](08-traits-and-generics.md#the-operator-traits).

### Defaults

A field with a default may be omitted at construction:

```veles
use io

struct Config {
  host: string = "localhost"
  port: i64 = 8080
  debug: bool = false
}

fun main() {
  val a = Config()
  val b = Config(port: 9000, debug: true)
  io.println("$a")
  io.println("$b")
}
```

Output:
```text
Config(host: localhost, port: 8080, debug: false)
Config(host: localhost, port: 9000, debug: true)
```

## Structs are values

Assigning a struct **copies** it (D7). Two variables never share a
struct by accident:

```veles
use io

struct Counter { n: i64 }

fun main() {
  val a = Counter(n: 1)
  var b = a
  b = Counter(n: 2)
  io.println("$a $b")
}
```

Output:
```text
Counter(n: 1) Counter(n: 2)
```

## Mutating methods

A method that changes its receiver's fields must be declared `mut fun`
(D22), and can only be called on something mutable — a `var`, not a
`val`:

```veles
use io

struct Account {
  balance: i64

  mut fun deposit(amount: i64) { self.balance += amount }
  mut fun withdraw(amount: i64): bool {
    if (amount > self.balance) return false
    self.balance -= amount
    true
  }
}

fun main() {
  var acct = Account(balance: 100)
  acct.deposit(50)
  val ok = acct.withdraw(500)
  io.println("$acct $ok")
}
```

Output:
```text
Account(balance: 150) false
```

Try `val acct = ...` instead: the compiler refuses `acct.deposit(50)`
because a `val` cannot be mutated. This is how Veles makes "does this
call change my data?" visible at every call site.

## Static functions

A `static fun` has no `self`: it belongs to the type and is called on the
type name. That is where constructors with a story go — a default value,
a parse from text — next to the fields they build.

```veles
use io

struct Point {
  x: i64
  y: i64

  static fun origin(): Point = Point(x: 0, y: 0)

  static fun fromText(s: string): Point? {
    val parts = s.split(",")
    if (parts.len() != 2) return null
    val x = i64.parse(parts.atOrPanic(0)) ?: return null
    val y = i64.parse(parts.atOrPanic(1)) ?: return null
    Point(x, y)
  }

  fun shifted(dx: i64): Point = Point(x: self.x + dx, y: self.y)
}

fun main() {
  io.println("${Point.origin()} ${Point.origin().shifted(2)} ${Point.fromText("3,4")} ${Point.fromText("3")}")
}
```

Output:
```text
Point(x: 0, y: 0) Point(x: 2, y: 0) Point(x: 3, y: 4) null
```

`i64.parse(text)` is the same idea on a built-in type: it returns `i64?`,
null when the text is not a number. Calling a static function on a value
(`p.origin()`) or a method on the type (`Point.shifted(2)`) is an error
that says which one you meant. For a generic struct, name the instance:
`Stack<i64>.empty()`.

## Sharing with pointers

When two places must see the *same* struct, take its address. `&x`
gives a `*T`, a garbage-collected pointer (D10); there is nothing to free.

```veles
use io

struct Node {
  value: i64
  next: (*Node)?
}

fun length(n: (*Node)?): i64 {
  var count = 0
  var cur = n
  loop (cur != null) {
    count += 1
    cur = cur.next
  }
  count
}

fun main() {
  val third = Node(value: 3, next: null)
  val second = Node(value: 2, next: &third)
  val first = Node(value: 1, next: &second)
  io.println("${length(&first)} ${first.next?.value ?: 0}")

  var shared = Counter(n: 0)
  val p = &shared
  p.n += 1
  p.n += 1
  io.println("$shared")
}

struct Counter { n: i64 }
```

Output:
```text
3 2
Counter(n: 2)
```

Pointers dereference automatically for field access and method calls
(`p.n`, `cur.next`), so code reads the same whether it holds a value or a
pointer (D39). `*p` reads the whole value out. A pointer that may be
absent is written `(*Node)?` — the parentheses matter, because `*Node?`
would be a pointer to a nullable `Node`. `&third` in the example
takes the address of a local; the collector keeps it alive as long as
anything points at it, so returning it from a function is fine.

## Visibility

Declarations are private to their module unless marked `pub`; the same
goes for fields and methods (M5). Inside one directory everything sees
everything. More in [chapter 11](11-modules-and-packages.md).

## Generic structs

```veles
use io

struct Pair<A, B> {
  first: A
  second: B
  fun swap(): Pair<B, A> = Pair(first: self.second, second: self.first)
}

fun main() {
  val p = Pair(first: 1, second: "one")
  io.println("$p ${p.swap()} ${p.second.len()}")
}
```

Output:
```text
Pair(first: 1, second: one) Pair(first: one, second: 1) 3
```

Type arguments are inferred from the constructor's arguments. Each
distinct `Pair<A, B>` is compiled separately (D8).

## Methods outside the body: `extend`

A struct body can get long. `extend` adds inherent methods to a type from
anywhere in the same package — another file, or a block below the struct
that groups related operations:

```veles
use io

struct Pair<A, B> {
  first: A
  second: B
}

extend<A, B> Pair<A, B> {
  fun swap(): Pair<B, A> = Pair(first: self.second, second: self.first)
}

extend Pair<i64, i64> {
  fun sum(): i64 = self.first + self.second
}

fun main() {
  val p = Pair(first: 1, second: 2)
  io.println("${p.swap()} ${p.sum()}")
}
```

Output:
```text
Pair(first: 2, second: 1) 3
```

The second block applies to `Pair<i64, i64>` only; a bound like
`extend<T: Show> Box<T>` works the same way. You can only extend types your
own package declares — for anyone else's type, declare a trait and
implement it ([chapter 8](08-traits-and-generics.md)). The built-in types
belong to the standard library, which is where `"a,b".split(",")` and
`xs.take(3)` come from: they are `extend` blocks in the prelude, written in
Veles.

Next: [Nullable types](06-nullable-types.md).
