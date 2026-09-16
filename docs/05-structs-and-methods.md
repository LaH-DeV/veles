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
  not matter and nothing can be forgotten.
- Methods live in the struct body and refer to the receiver as `self`.
- Every struct can be printed with `$p`, compared with `==`, and used as
  a map key, automatically.

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

Next: [Nullable types](06-nullable-types.md).
