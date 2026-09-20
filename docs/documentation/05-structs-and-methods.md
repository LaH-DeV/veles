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

struct Counter { var n: i64 }

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

### If you know classes from another language

A struct is where a Kotlin, TypeScript or Java programmer would reach
for a class, so the differences are the things to unlearn:

| | class (Kotlin / TS / Java) | Veles struct |
|---|---|---|
| Assignment | copies a *reference*; two names, one object | copies the *value*; two names, two structs (D7) |
| Sharing one instance | the default | explicit: a pointer `*T` (`&x`, [Sharing with pointers](#sharing-with-pointers) below), or a reference type such as `MutableList` |
| Inheritance | `class Dog : Animal` | none. Shared behaviour is a trait (chapter 8); a closed family is a `sealed trait` with variant structs (chapter 9) |
| Constructor | written by hand | implicit, by field name: `Point(x: 1, y: 2)`; `static fun` for anything with logic |
| Mutation | any method may assign fields | only the fields declared `var`; a bare field never changes (D22) |
| Equality, printing, hashing | `equals`/`hashCode`/`toString` by hand (or `data class`) | structural by default; replaced with the operator traits |
| Interfaces | `implements I` | `impl I for T` — see chapter 8 |
| Private state | `private` fields | no `public` on the field; the implicit constructor then works only inside the module |

The value semantics are the one that surprises people: `var b = a` then
`b.n = 5` leaves `a` alone. When you want the class behaviour — one
object, many names — say so with a pointer or a `Mutex`, and the reader
sees where sharing happens.

## Mutable fields: `var`

Whether a field can change is written on the field. A bare field is set
once, by the constructor call, and never assigned again — by anyone,
whatever holds the struct. A field declared `var` can be assigned: from a
method through `self`, from outside through any binding, through a
pointer (D22).

```veles
use io

struct Account {
  owner:       string      // fixed for the life of the account
  var balance: i64         // the part that changes

  fun deposit(amount: i64) { self.balance += amount }
  fun withdraw(amount: i64): bool {
    if (amount > self.balance) return false
    self.balance -= amount
    true
  }
}

fun main() {
  val acct = Account(owner: "ann", balance: 100)
  acct.deposit(50)
  val ok = acct.withdraw(500)
  io.println("$acct $ok")
}
```

Output:
```text
Account(owner: ann, balance: 150) false
```

Two things to notice. There is no marker on `deposit`: a method may
assign the `var` fields of the struct it was called on, and the reader
learns what can change from the type, not from every method signature.
And `acct` is a `val`, yet `deposit` changed it: `val` and `var` on a
binding say whether the *name* can be rebound (`acct = Account(...)`),
exactly as they do for a `MutableList`. Immutability is a property of
the type — try `acct.owner = "bob"`: the compiler refuses, and no method
could do it either. A struct with no `var` fields and no collections
inside cannot change at all, whoever holds it.

Because a struct is a value, a method changes the copy it was called on.
`var b = acct; b.deposit(1)` leaves `acct` alone, and a method called
on a temporary — `accounts.at(0)?.deposit(1)`, a copy of the element —
is an error, since the change would be thrown away with the copy
(chapter 4 shows `ref` and `loop (&x in xs)`, which reach the element
itself). A struct passed to a function is a copy too: to let the
callee change yours, pass `&acct` ([Sharing with pointers](#sharing-with-pointers)).

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

### Constants on a type: `static val`

A `static val` is a value that lives in the type's namespace — the
well-known instances of a type that is otherwise open, such as HTTP status
codes, or a table the type's own functions consult. It is initialised
once, with the module's other globals, and read as `Type.name`:

```veles
use io

struct Status {
  code:   i64
  reason: string = ""

  static val ok       = Status(code: 200, reason: "OK")
  static val notFound = Status(code: 404, reason: "Not Found")
  static val known    = [Status.ok, Status.notFound]

  static fun of(code: i64): Status = Status.known.find(s => s.code == code) ?: Status(code)
  fun isError(): bool = self.code >= 400
}

fun main() {
  io.println("${Status.ok.code} ${Status.of(404).reason} ${Status.of(299).isError()} ${Status.notFound == Status(code: 404, reason: "Not Found")}")
}
```

Output:
```text
200 Not Found false true
```

A static is always a `val`; it is `public` when the type's users may read
it, private to the module otherwise, like any declaration. A generic
struct cannot have one (there would have to be a value per
instantiation) — use a `static fun` there. This is the tool for a set of
named values that stays *open*: anyone may still write `Status(code: 599)`.
A set that is closed is a different thing, and will be an `enum`.

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

struct Counter { var n: i64 }
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

Declarations are private to their module unless marked `public`; the same
goes for fields and methods (M5). Inside one directory everything sees
everything — which is right for a helper struct and wrong for an
invariant: in a one-file program, nothing would stop a handler from
poking a counter that only the type should touch. A member marked
`private` is visible only inside the type's own declarations — its
methods, its `impl` and `extend` blocks in the same module, and its
`static val` initializers:

```veles
use io

struct Note {
  id:   i64
  text: string
}

struct Notes {
  private var next: i64 = 1
  private items:    MutableList<Note> = []

  fun add(text: string): Note {
    val n = Note(id: self.next, text)
    self.items.push(n)
    self.next += 1
    n
  }
  fun all(): List<Note> = self.items.toList()
  fun find(id: i64): Note? = self.items.find(n => n.id == id)
}

fun main() {
  val notes = Notes()
  notes.add("buy milk")
  notes.add("call mum")
  io.println("${notes.all().len()} ${notes.find(2)?.text} ${notes.find(9)}")
}
```

Output:
```text
2 call mum null
```

Outside `Notes`, `notes.next` and `Notes(next: 5)` are errors, and so is
matching the field in a pattern. A private field must therefore have a
default, or the type must offer a `static fun` that builds it; `Notes()`
above works because both fields do. `private fun` marks a helper method
the same way. Read the two declarations together: `next` is the only
thing in `Notes` that is ever assigned, and only `Notes` may do it —
`items` is a handle whose *contents* change but which is never replaced.
More on modules in [chapter 11](11-modules-and-packages.md).

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


## Naming a type: `type`

`type Name = Type` gives a type another name. It is an *alias*, never a
new type: `Index` below **is** `i64`, so an `Index` goes wherever an
`i64` goes and back, with no conversion. What you gain is a name in
your code, in error messages and in hover — `Key` instead of `(i64, u64)`,
`Handler` instead of a function type nobody wants to read twice:

```veles
use io

type Index = i64
type Hash = u64
type Key = (Index, Hash)
type Groups = MutableMap<Key, MutableList<string>>
type Handler = fun(string): string
type StrMap<V> = Map<string, V>

fun apply(h: Handler, s: string): string = h(s)

fun main() {
  val groups: Groups = [:]
  val k: Key = (1, 2)
  groups.getOrPut(k, () => []).push("first")
  val n: Index = 40
  val plain: i64 = n + 2                 // an Index is an i64
  val names: StrMap<i64> = ["ann": 41]
  io.println("${groups.get(k)} $plain ${names.get("ann")} ${apply(s => s + "!", "hi")}")
}
```

Output:
```text
[first] 42 41 hi!
```

The rules are short. An alias lives at module level (`public type` exports
it, and exporting an alias of a private type is fine — that is how a
library presents a facade). A generic alias takes plain parameters, no
bounds: state `T: Comparable` where the alias is used. `type Point =
geo.Point` shortens a qualified name, and the alias then constructs
(`Point(x: 1.0, y: 2.0)`), calls statics and matches with `is` exactly
like the struct, because it is the struct.

### When you want something an alias is not

- **A distinct type** — an `i64` that cannot be confused with another
  `i64` (`UserId` vs `OrderId`): a one-field struct, `struct UserId { value: i64 }`.
  An alias is a name, never a wall.
- **A union** (`Circle | Square`): a `sealed trait` with variant structs
  ([chapter 9](09-sealed-types.md)); for errors, `error Set = A | B`
  ([chapter 7](07-errors.md)). `type` never spells `|`.
- **A recursive type** (`Json = Map<string, Json>`): a sealed trait or a
  struct — `examples/json` is the pattern — which also names the cases.
  A `type` that mentions itself is an error.
- **A constrained alias** (`Sorted<T: Comparable>`): put the bound on
  the function or struct that uses it; if the constraint is part of the
  meaning, it is a struct.
- **A type computed from a type** (TypeScript's `ReturnType<F>`,
  `keyof`): an associated type on a trait — `Iterator.Item` is exactly
  that ([chapter 8](08-traits-and-generics.md#associated-types)).

And one thing not to do: aliases for the built-in numbers (`type int =
i64`). Two spellings for one type give every reader two things to learn
and every codebase a style argument; if a short name were better it would
be the name.

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
