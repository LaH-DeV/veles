# 4. Collections

Veles has three built-in collections — `List`, `Map` and `Set` — and each
comes in an **immutable** and a **mutable** flavour. This split is the
one idea to absorb in this chapter: `List<T>` never changes after it is
built, `MutableList<T>` grows and shrinks, and converting between them
copies (D25). The payoff comes later: an immutable collection can be
shared between tasks without locks (D35).

## Lists

```veles
use io

fun main() {
  val primes = [2, 3, 5, 7]                 // List<i64>
  io.println("${primes.len()} ${primes[0]} ${primes.at(10) ?: -1} ${primes.contains(5)}")

  val names = mut ["ann", "bob"]            // MutableList<string>
  names.push("cy")
  val last = names.pop()                    // string?
  io.println("$names ${last ?: "-"} ${names.isEmpty()}")

  val frozen = names.toList()               // copy into an immutable List
  names.push("dee")                         // does not affect `frozen`
  io.println("$frozen $names")
}
```

Output:
```text
4 2 -1 true
[ann, bob] cy false
[ann, bob] [ann, bob, dee]
```

- `[a, b, c]` is a `List`; `mut [a, b, c]` is a `MutableList`. When the
  expected type is known — an annotated binding, a parameter, a field, a
  return — the literal takes it, mutability included, and `mut` is not
  written: `var xs: MutableList<i64> = []`, `fill([1, 2])`. An **empty**
  literal always needs such a type, because there is nothing to infer it
  from. A list literal where a `Set` is expected builds a set:
  `val seen: Set<i64> = [1, 2]`.
- `xs[i]` reads an element and **panics** if `i` is out of range.
  `xs.at(i)` returns `T?` — null instead of a panic — for when the index
  comes from data you do not control.
- `push`, `pop` and `clear` exist only on `MutableList`; calling them on a
  `List` is a compile-time error, not a runtime one.

### Transforming lists

The familiar higher-order operations are methods on both list kinds and
always return a fresh `List`:

```veles
use io

fun main() {
  val xs = [5, 3, 8, 1]
  val doubled = xs.map(x => x * 2)
  val big = xs.filter(x => x > 2)
  val sum = xs.fold(0, (acc, x) => acc + x)
  io.println("$doubled $big $sum")
  io.println("${xs.sorted()} ${xs.sortedBy(x => -x)} ${xs.reversed()}")
  io.println("${xs.any(x => x > 7)} ${xs.all(x => x > 0)} ${xs.find(x => x > 4) ?: 0} ${xs.indexOf(8)}")
  io.println("${xs.first() ?: 0} ${xs.last() ?: 0} ${xs.joinToString(", ")}")
  xs.forEach(x => io.print("$x "))
  io.println("")
}
```

Output:
```text
[10, 6, 16, 2] [5, 3, 8] 17
[1, 3, 5, 8] [8, 5, 3, 1] [1, 8, 3, 5]
true true 5 2
5 1 5, 3, 8, 1
5 3 8 1 
```

`x => x * 2` is a lambda; [chapter 10](10-closures-and-iterators.md)
covers them fully. These operations are *eager*: each builds its result
immediately. For long pipelines over big data, `xs.iter()` gives a lazy
iterator with the same names — also chapter 10.

## Maps

A `Map<K, V>` keeps its entries in **insertion order** — iteration is
deterministic, always (D25).

```veles
use io

fun main() {
  val ages = ["ann": 41, "bob": 29]              // Map<string, i64>
  io.println("${ages["ann"] ?: 0} ${ages["zed"] ?: 0} ${ages.containsKey("bob")} ${ages.len()}")

  var stock = mut ["apples": 3]                  // MutableMap<string, i64>
  stock["pears"] = 5
  stock["apples"] = (stock["apples"] ?: 0) + 1
  stock.remove("pears")
  io.println("$stock ${stock.keys()} ${stock.values()}")

  loop ((name, age) in ages) {
    io.println("$name is $age")
  }
}
```

Output:
```text
41 0 true 2
{apples: 4} [apples] [4]
ann is 41
bob is 29
```

`m[key]` returns `V?` — null when the key is absent — which is why the
reads above carry `?: 0`. This is the honest type: a lookup can fail.
`m.get(key)` is the same operation spelled as a method. Writing
`m[key] = value` requires a `MutableMap` and inserts or replaces.

Keys must be **hashable**: numbers, strings, booleans, tuples of those,
and structs whose fields are (structs get equality and hashing for
free).

## Sets

```veles
use io

fun main() {
  var seen = MutableSet<i64>()
  loop (x in [3, 1, 3, 2, 1]) {
    if (!seen.add(x)) io.println("duplicate $x")
  }
  io.println("$seen ${seen.contains(2)} ${seen.len()} ${seen.toList()}")
}
```

Output:
```text
duplicate 3
duplicate 1
{3, 1, 2} true 3 [3, 1, 2]
```

`add` returns whether the element was new. There is no set literal of its
own: construct with `MutableSet<T>()` or `Set<T>()`, convert a list, or
write a list literal where a set type is expected — `val s: Set<i64> =
[1, 2, 2]` has two elements.

## Ranges

`1..5` and `1..<5` are values of type `Range<i64>`; they can be looped,
stored, and turned into lists:

```veles
use io

fun main() {
  val r = 1..5
  io.println("${r.lo} ${r.hi} ${r.inclusive}")
  val squares = (1..<5).iter().map(i => i * i).toList()
  io.println("$squares")
}
```

Output:
```text
1 5 true
[1, 4, 9, 16]
```

## Choosing between them

| You need | Use |
|---|---|
| a fixed sequence, possibly shared | `List<T>` |
| to accumulate results | `MutableList<T>` then `.toList()` |
| lookup by key, ordered | `Map<K, V>` / `MutableMap<K, V>` |
| membership tests, no duplicates | `Set<T>` / `MutableSet<T>` |

Next: [Structs and methods](05-structs-and-methods.md).
