# 6. Nothing, maybe: nullable types

A `string` is always a string. A `string?` is a string *or nothing*. The
question mark is part of the type, and the compiler will not let you use
a `T?` where a `T` is required until you have dealt with the nothing
case (D5). There is no null pointer exception in Veles; there is a
compile error instead.

## Producing and consuming `T?`

```veles
use io

fun findUser(id: i64): string? => when (id) {
  1 => "ann"
  2 => "bob"
  else => null
}

fun main() {
  val a = findUser(1)          // string?
  val z = findUser(9)          // string?
  io.println("${a ?: "nobody"} ${z ?: "nobody"}")
  io.println("${a?.len() ?: 0} ${z?.len() ?: 0}")
}
```

Output:
```text
ann nobody
3 0
```

Three operators do most of the work:

| Operator | Meaning |
|---|---|
| `x ?: fallback` | `x` if it is not null, otherwise `fallback` |
| `x?.member` | `null` if `x` is null, otherwise `x.member` — the result is nullable |
| `x?.method()` | same for calls |

A `null` skips the *rest of the chain*, not just the next step (D70):
`user?.address.city.len()` is an `i64?` — `null` when there is no user,
the length otherwise — with no `?.` needed after the first. Parentheses
end a chain: `(user?.address).city` reads `.city` on a nullable, which is
an error.

`?:` also accepts a `return` or `throw` on its right: `val n =
text.toInt() ?: return -1` bails out of the function when there is no
number. The same thing can be written as a binding, `val n = text.toInt()
else return -1` — the form that also works for a `Result` and for a pattern
(`val Circle(r) = shape else return 0.0`); see chapter 7, "Falling back and
bailing out".

## Smart casts

After a null check the compiler *knows* the value is present and lets
you use it as a plain `T` (D5). No unwrap syntax:

```veles
use io

fun describe(name: string?): string {
  if (name == null) return "anonymous"
  // from here on `name` is a `string`
  "${name.len()} letters"
}

fun shout(name: string?): string {
  if (name != null && name.len() > 0) {
    return name + "!"
  }
  "..."
}

fun main() {
  io.println("${describe(null)} ${describe("ann")} ${shout("hey")} ${shout(null)}")
  var maybe: string? = null
  if (maybe == null) maybe = "filled"
  io.println("${maybe.len()}")       // assignment narrowed it, too
}
```

Output:
```text
anonymous 3 letters hey! ...
6
```

Narrowing follows the control flow: `if (x == null) return` narrows
everything after it; `x != null && ...` narrows the right side of the
`&&`; assigning a non-null value narrows a `var`.

It also applies to a **field path** — a chain of plain struct fields from
a local variable or from `this`:

```veles
use io

struct Address { city: string }
struct User { name: string, var address: Address? }

fun main() {
  var u = User(name: "ann", address: Address(city: "Oslo"))
  if (u.address != null) io.println(u.address.city)   // u.address is an Address here
  u.address = null                                     // a write to the path forgets the fact
  io.println("${u.address?.city ?: "nowhere"}")
}
```

Output:
```text
Oslo
nowhere
```

The fact about `u.address` survives until something could change it: an
assignment to `u.address` or to `u` itself, or `&u` being taken. A method
call on `u` forgets the facts about its `var` fields — the method may
assign them — and keeps those about bare fields, which nothing can
assign (D22). Fields reached *through a pointer* (`p.address` with
`p: *User`) are never narrowed — another pointer to the same value could
change them in between; bind the field to a `val` first.

## Binding the value in the condition: `if (val x = e)`

A smart cast needs a place — a variable or a field path. When the nullable
is a call, or a `?.` chain, and you want to *do something if it is there*,
bind it in the condition:

```veles
use io

struct Cookie {
  name:   string
  maxAge: Duration? = null
  domain: string? = null
}

fun header(name: string): string? => if (name == "n") "42" else null

fun line(c: Cookie): string {
  val out = StringBuilder()
  out.append(c.name)
  if (val age = c.maxAge) out.append("; Max-Age=${age.toSeconds()}")
  if (val d = c.domain) out.append("; Domain=$d") else out.append("; no domain")
  out.toString()
}

fun main() {
  io.println(line(Cookie(name: "a")))
  io.println(line(Cookie(name: "a", maxAge: Duration.seconds(5), domain: "x.org")))
  // any nullable expression, and as an expression
  val label = if (val n = header("n")?.toInt()) "n=${n + 1}" else "none"
  io.println(label)
}
```

Output:
```text
a; no domain
a; Max-Age=5; Domain=x.org
n=43
```

`val x = e` is true when `e` is not null, and `x` is then the plain value —
a `T` where `e` is a `T?` (D95). Its scope is the rest of the condition and
the `then` branch: not the `else`, and not the code after the `if`.

Bindings chain with `&&`, left to right, and each is in scope for what
follows it:

```veles
// fragment
if (val a = header("n") && val b = a.toInt() && b > 1) io.println("a=$a b=$b")
if (ready && val v = expensive()) use(v)   // expensive() runs only when ready is true
```

The chain stops at the first operand that fails, so a value further along
is not even computed. A binding is only allowed in the `&&` chain of an
`if` condition — not under `||` or `!`, not in a `while`/`loop` head (a
`loop { val x = next() ?: break ... }` says the same) — and the value must
be able to be null: `if (val n = 5)` says so, since a plain `val` binds it.
A binding that is never read is the usual unused-name warning.

It is a block, not a lambda, so `return`, `break`, `continue`, `throw` and
`try` inside it leave the enclosing function or loop, and a call that
suspends needs nothing declared — which is why Veles has this rather than
Kotlin's `x?.let { ... }`. To leave when the value is missing, `val x = e
else return` (chapter 7) is the other half: use `if (val ...)` when the
present case is the short one.

## Updating through `?.`

A narrowed value is not a copy: after the null test you can write to its
`var` fields and call methods on it, and the change lands in the variable or
field it came from. `?.` does the same in one step — on a call, and on an
assignment, which happens only when the left side is present and is
skipped otherwise. A nullable *pointer* is the common receiver: `m.ref(k)`
is `(*V)?`, a pointer to the value stored in the map or null, so `?.`
through it updates the map entry itself:

```veles
use io

struct Counter {
  var n: i64 = 0

  fun bump() {
    this.n += 1
  }
}

fun main() {
  var maybe: Counter? = Counter()
  if (maybe != null) {
    maybe.n = 5
    maybe.bump()
  }
  maybe?.n += 1

  val tally: MutableMap<string, Counter> = ["hits": Counter()]
  tally.ref("hits")?.bump()
  tally.ref("hits")?.n += 10
  tally.ref("misses")?.bump()     // no entry: nothing happens
  io.println("$maybe $tally")
}
```

Output:
```text
Counter(n: 7) {hits: Counter(n: 11)}
```

This is why get-and-update code rarely needs `get(k) ?: panic(…)`: `?.`
*is* the presence test. What stays closed to it is anything that is a copy — the
value `get(k)` or `at(i)` returns: `tally.get("hits")?.n += 10`
is an error, because the change could never be observed ([chapter 4](04-collections.md#updating-elements-in-place)
has the reading/writing rule).

## Where `T?` shows up

- `map.get(key)`, `list.at(i)`, `list.first()`, `list.pop()`,
  `text.toInt()`, `text.substring(a, b)` — every operation that can come
  up empty returns `T?` rather than a sentinel or a panic.
- Struct fields: `next: (*Node)?` for the end of a linked list.
- Your own functions, whenever "not found" is a normal outcome. When
  something went *wrong*, use errors ([chapter 7](07-errors.md)) instead.

```veles
use io

fun main() {
  val words = ["7", "x", "42"]
  var total = 0
  loop (w in words) {
    val n = w.toInt() ?: continue
    total += n
  }
  io.println("$total ${words.at(5) ?: "none"} ${words.first()?.len() ?: 0}")
}
```

Output:
```text
49 none 1
```

## Nullable of nullable

Because `T?` is a real type and not a flag on the value, it nests:
`i64??` distinguishes "absent" from "present but null". This matters for
generic containers — a `Map<string, i64?>` lookup must be able to say
which one it found. You will rarely write such a type by hand; when you
do, the inner value is wrapped with `Some`:

```veles
use io

fun main() {
  val settings: Map<string, i64?> = ["timeout": 30, "retries": null]
  val t = settings.get("timeout")  // i64??
  val r = settings.get("retries")
  val m = settings.get("missing")
  io.println("${t != null} ${r != null} ${m != null}")
  when (r) {
    Some(null)    => io.println("retries is explicitly unset")
    Some(Some(n)) => io.println("retries $n")
    null          => io.println("retries not configured")
  }
}
```

Output:
```text
true true false
retries is explicitly unset
```

Next: [Errors](07-errors.md).
