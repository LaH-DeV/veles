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
  val primes = [2, 3, 5, 7]  // List<i64>
  io.println("${primes.len()} ${primes.at(0)} ${primes.at(10) ?: -1} ${primes.contains(5)}")

  val names = mut ["ann", "bob"]  // MutableList<string>
  names.push("cy")
  val last = names.pop()  // string?
  io.println("$names ${last ?: "-"} ${names.isEmpty()}")

  val frozen = names.toList()  // copy into an immutable List
  names.push("dee")            // does not affect `frozen`
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
  from; when a later line decides it (`var xs = []` then `xs.push(1)`),
  the error comes with a fix that writes `var xs: MutableList<i64> = []`
  for you (`veles check --fix`, or the editor's quick fix; D106). A list literal where a `Set` is expected builds a set:
  `val seen: Set<i64> = [1, 2]`.
- Reading is a method, never brackets — `[...]` only ever builds a literal.
  `xs.at(i)` returns `T?`: null when `i` is out of range. Where the
  compiler can **see** that `i` is in range, it returns `T` instead (see
  below). Where you know better than the compiler, say why:
  `xs.at(i) ?: panic("the header always has three parts")`.
  `xs.atOrDefault(i, d)` is `xs.at(i) ?: d`. A negative index counts from
  the end: `xs.at(-1)` is the last element.
- `push`, `pop` and `clear` exist only on `MutableList`; calling them on a
  `List` is a compile-time error, not a runtime one.
- A `List` never changes, so a `MutableList` does not quietly become one.
  `xs.toList()` copies it. The exception is a list you just built and
  nobody else has seen: returning it, or passing it on as its last use,
  hands it over without a copy (D63).

### When the index is known to be in range

The compiler tracks what your code has already checked, as it does for
null (D62), and a read it can prove is in range has type `T`:

```veles
use io

fun sum(xs: List<i64>): i64 {
  var total: i64 = 0
  loop (i in xs.indices()) {
    total += xs.at(i)  // i64: i is one of xs's indexes
  }
  total
}

fun ends(xs: List<i64>): i64 {
  if (xs.len() < 2) return 0
  xs.at(0) + xs.at(-1)  // i64: the length was checked above
}

fun build(n: i64): List<i64> {
  val out: MutableList<i64> = []
  out.reserve(n)  // room for all n: the pushes below never grow it
  loop (i in 0..<n) out.push(i * i)
  out  // handed over as a List, not copied
}

fun main() {
  io.println("${sum([1, 2, 3])} ${ends([4, 5, 6])} ${build(4)}")
}
```

Output:
```text
6 10 [0, 1, 4, 9]
```

- What counts: `loop (i in 0..<xs.len())` or `loop (i in xs.indices())`;
  a check `i < xs.len()` when `i` cannot be negative (a `var i = 0` that
  only counts up, or `i >= 0`); a constant index after `xs.len() >= n`,
  `xs.len() == n` or `!xs.isEmpty()`. `first()` and `last()` are `T` once
  the list is known not to be empty.
- What ends it: assigning the list or the index. For a `MutableList`,
  also any function call, because the call might shrink the list through
  another name. Index arithmetic such as `xs.at(i + 1)` is not tracked.
- When a proof is lost, for example because a refactor moved a call
  between the check and the read, the read goes back to `T?` and the
  compiler reports it. A `?:` after a read that became `T` is a warning
  with a fix, not an error.

### Matching a list's shape

When code needs particular elements, such as the three parts of a token
or the command and its arguments, match the list's **shape** instead of
checking its length and then reading indexes. A list pattern does both in
one step, so no read can be out of range (D62):

```veles
use io

fun describe(args: List<string>): string = when (args) {
  []                  => "no arguments"
  ["help"]            => "help"
  [cmd]               => "just $cmd"
  ["run", target, ..] => "run $target"
  [cmd, ..rest]       => "$cmd and ${rest.len()} more"
}

fun ends(xs: List<i64>): string {
  val [first, .., last] = xs else return "too short"
  "$first to $last"
}

fun main() {
  io.println(describe(["run", "tests", "-v"]))
  io.println(describe(["fmt", "a", "b"]))
  io.println(ends([3, 1, 4, 1, 5]))
  io.println(ends([9]))

  val [user, host] = "ann@example.com".split("@") else return
  io.println("$user at $host")
}
```

Output:
```text
run tests
fmt and 2 more
3 to 5
too short
ann at example.com
```

- `[a, b]` matches **exactly** two elements. A `..` matches any number of
  elements (none included), in any position, at most once per pattern.
  `..rest` binds those elements as a new `List`.
- Elements are patterns themselves: names, `_`, literals such as
  `"help"`, and nested tuple, variant or list patterns.
- In `val`, a list pattern can fail to match, so it needs `else`, and the
  `else` has to leave (`return`, `throw`, `break`, `continue`), as with
  every val-else.
- A `when` is exhaustive over lengths: `[]`, `[x]` and `[x, ..]` together
  cover every list, and the compiler names any lengths you missed. An arm
  with a literal element, such as `["help"]`, covers nothing on its own.
- The names are copies taken when the match runs, so changing a
  `MutableList` afterwards does not change them.

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
  io.println("${xs.first() ?: 0} ${xs.last() ?: 0} ${xs.join(", ")}")
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

`indexOf` scans, which is the honest answer on a list in no particular
order. On a list that *is* sorted, `binarySearch` answers the same question
in log *n* comparisons, and `lowerBound`/`upperBound`/`partitionPoint`
answer the neighbouring ones — where a value would be inserted, where a run
ends:

```veles
use io

struct Row {
  id:   i64
  name: string
}

fun main() {
  val sorted = [10, 20, 20, 30]
  // the first of the two 20s, and -1 for a value that is not there
  io.println("${sorted.binarySearch(20)} ${sorted.binarySearch(25)}")
  // where 20 begins and where it ends: it occurs 3 - 1 = 2 times
  io.println("${sorted.lowerBound(20)} ${sorted.upperBound(20)}")
  // the general form: where the run of smaller values ends
  io.println("${sorted.partitionPoint(x => x < 25)}")
  // and by a key, on a list sorted by that key
  val rows = [Row(id: 3, name: "ann"), Row(id: 7, name: "bo")]
  io.println("${rows.binarySearchBy(r => r.id, 7)} ${rows.binarySearchBy(r => r.id, 4)}")
}
```

Output:
```text
1 -1
1 3
3
1 -1
```

None of them checks that the list is sorted, because checking is the linear
scan they exist to avoid: on a list that is not sorted the answer is simply
wrong, never a panic.

### Updating elements in place

Every read is a **value**: `xs.at(i)`, `xs.first()`, `xs.find(p)` and
the loop variable of `loop (x in xs)` all hand you a copy of a struct
element, exactly as `val t = xs.at(i)` does. So a change to what you
read never reaches the list — and the compiler says so rather than
letting it vanish: `xs.at(i)?.bump()` is an error ("a copy of the
element").

To change an element, ask for a **reference** to it. `ref(i)` is a
pointer to the element, `(*T)?`: null when `i` is out of range, and a
plain `*T` where the index is known to be in range. `loop (&x in xs)`
visits every element by reference. A read and a write then never look alike:

```veles
use io

struct Counter {
  name:  string
  var n: i64 = 0

  fun bump() {
    this.n += 1
  }
}

fun main() {
  val counters: MutableList<Counter> = [Counter(name: "a"), Counter(name: "b")]
  var copy = counters.at(0) ?: panic("two counters")  // a copy: `var` because we change it
  copy.n = 100
  counters.ref(0)?.bump()            // the element itself
  counters.ref(1)?.n = 5             // in range: written; out of range: skipped
  counters.ref(7)?.n = 5
  val p = counters.ref(1) ?: panic("two counters")  // a pointer may sit in a `val`
  p.bump()
  loop (&c in counters) c.n *= 10    // every element, in place
  io.println("${copy.n} ${counters.map(c => c.n)}")

  val nums: MutableList<i64> = [1, 2, 3]
  if (nums.len() > 0) *nums.ref(0) += 10  // a primitive: write through the pointer
  loop (&n in nums) *n += 1
  nums.set(2, 0)                     // or replace the element outright
  io.println("$nums")
}
```

Output:
```text
100 [10, 60]
[12, 3, 0]
```

`ref` exists only on `MutableList` and `MutableMap`; `m.ref(k)` and
`loop ((k, &v) in m)` reach map values. Two things to know about a reference: it is a pointer
into the collection's storage, so keep it short-lived — after the list
grows (`push`) an old reference points at the old buffer; and `&` in a
loop head means the same as `ref`, so `loop (&x in xs)` on a read-only
`List` is an error.

### Changing a collection while a loop walks it

A loop over a `MutableList`, `MutableMap`, `MutableSet` or `Deque` walks
the collection as it is, so the body may not change its length or order
(D102): `loop (x in xs) { if (x < 3) xs.push(x) }` is an error naming the
call and the loop. Replacing an element in place is fine — `xs.set(i, v)`,
`*x = v` in `loop (&x in xs)`, `m.set(k, v)` for the key being visited.
To change the collection, loop over a copy (`xs.toList()`, the quick fix)
or collect the changes and apply them after the loop; to grow a worklist
while walking it, loop over its indices or a range, which walks no
collection:

```veles
use io

fun main() {
  val work: MutableList<i64> = [3]
  var i = 0
  loop (i < work.len()) {          // a condition loop: the list may grow
    val n = work.at(i) ?: 0
    if (n > 1) work.push(n - 1)
    i += 1
  }
  val evens: MutableList<i64> = []
  loop (n in work) {
    if (n % 2 == 0) evens.push(n)  // another list: fine
  }
  io.println("$work $evens")
}
```

Output:
```text
[3, 2, 1] [2]
```

A change the compiler cannot see — a function handed the same list —
is caught when the loop next steps: the program panics with
`'xs' changed while a loop walked it` at the loop's line, in debug and
release builds alike.

### Comparing collections

Two lists are equal when they hold equal elements in the same order; two
maps when they hold the same keys with equal values; two sets when they
hold the same elements — order never matters for a map or a set. The
mutable and immutable kinds compare with each other, and anything built
from comparable parts is comparable: a `List<List<i64>>`, a struct with a
`List` field, a `Map<string, Set<i64>>`. What is *not* comparable is a
value with no meaningful equality — a function.

```veles
use io

struct Row {
  name:   string
  scores: List<i64>
}

fun main() {
  val a = [1, 2, 3]
  var b: MutableList<i64> = []
  b.push(1); b.push(2); b.push(3)
  io.println("${a == b} ${a == [3, 2, 1]} ${[[1], [2]] == [[1], [2]]}")
  io.println("${["x": 1, "y": 2] == ["y": 2, "x": 1]} ${[1, 2].contains(2)} ${[[1, 2], [3]].indexOf([3])}")
  val s1: Set<i64> = [1, 2]
  val s2: Set<i64> = [2, 1, 1]
  io.println("${s1 == s2} ${Row(name: "a", scores: [1]) == Row(name: "a", scores: [1])}")
  var byScores: MutableMap<List<i64>, string> = [:]
  byScores.set([9, 9], "perfect")
  io.println("${byScores.get([9, 9])} ${byScores.get([9, 8])}")
}
```

Output:
```text
true false true
true true 1
true true
perfect null
```

`==` is one strategy; `sorted()` and `min()` use another (`Comparable`),
and both can be replaced per call — `sortedBy(key)`, `sortedWith(compare)`,
`distinctBy(key)`. [Chapter 8](08-traits-and-generics.md#comparing-values)
lays out the whole picture.

## Maps

A `Map<K, V>` keeps its entries in **insertion order** — iteration is
deterministic, always (D25).

```veles
use io

fun main() {
  val ages = ["ann": 41, "bob": 29]  // Map<string, i64>
  io.println("${ages.get("ann") ?: 0} ${ages.get("zed") ?: 0} ${ages.containsKey("bob")} ${ages.len()}")

  var stock = mut ["apples": 3]  // MutableMap<string, i64>
  stock.set("pears", 5)
  stock.set("apples", (stock.get("apples") ?: 0) + 1)
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

`m.get(key)` returns `V?` — null when the key is absent — which is why the
reads above carry `?: 0`. This is the honest type: a lookup can fail.
For a key you know is there, say why: `m.get(key) ?: panic("…")`;
`m.getOrDefault(key, d)` is `m.get(key) ?: d`.
Writing `m.set(key, value)` requires a `MutableMap` and inserts or
replaces. There is no bracket form: `[...]` only ever builds a literal.
A read is a copy of the value; to change the stored value ask for a
reference: `m.ref(key)?.bump()` and `m.ref(key)?.n += 1` update the
entry in the map, and `loop ((k, &v) in m)` visits every value by
reference (see [Updating elements in place](#updating-elements-in-place)
above; chapter 6 has the `?.` side).

Keys must be **hashable**: numbers, strings, booleans, tuples of those,
structs whose fields are (structs get equality and hashing for free),
and immutable lists, maps and sets of hashable elements — `m.get([1, 2])`
finds a key stored as `[1, 2]`, because collections compare by content
(`[1, 2] == [1, 2]` is true; a set or map ignores insertion order). A
`MutableList` cannot be a key: it could change after it was stored.

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

## Arrays

A `List<T>` is a heap object that can grow; an `Array<T, N>` is exactly `N`
elements stored *inline* — in a local, in a struct's field, in another
array — with no allocation and a length that is part of its type (D121). It
is what a lookup table, a hash state, a fixed buffer or a C `uint8_t
name[16]` is:

```veles
use io

struct Sha {
  var state: Array<u32, 8> = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                              0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19]
}

fun main() {
  var a: Array<i64, 4> = [10, 20, 30, 40]
  a.set(1, 99)
  val b = a
  a.set(0, 7)
  io.println("$a $b ${a.len()}")
  val far = 9
  io.println("${a.at(1)} ${a.at(far)} ${a.fold(0, (s, x) => s + x)}")
  loop (&x in a) {
    *x = *x + 1
  }
  var sha = Sha()
  sha.state.set(0, 1)
  io.println("$a ${sha.state.at(0)} ${Array<i64, 3>.make(5)}")
}
```

Output:
```text
[7, 99, 30, 40] [10, 99, 30, 40] 4
99 null 176
[8, 100, 31, 41] 1 [5, 5, 5]
```

A `[…]` literal is an array where an array type is expected, and its length
must be `N`. An array is a **value**, like a struct: `val b = a` and passing
`a` to a function copy it, and `a.set(i, v)` changes the variable it is
called on, which must be a `var`. Reading is a list's — `a.at(i)` is a `T?`,
null when `i` is out of range, and a negative `i` counts from the end — with
one difference: where the index is a constant in range, `a.at(3)` is a plain
`T` and compiles to a load, so a table needs no `?:`; a constant out of range
is an error. The bounds facts of [the earlier section](#when-the-index-is-known-to-be-in-range)
work too: `loop (i in a.indices())` and `loop (i in 0..<4)` make `a.at(i)` a
`T`. `loop (x in a)` walks a copy; `loop (&x in a)` walks the array itself
and lets the body change it. `a.len()` is the constant `N`.

An array never grows or shrinks, so it has none of a `MutableList`'s
`push` and `pop`; every read-only `List` method does work on it. `map`,
`filter`, `forEach`, `fold`, `any`, `all` and `find` read the array where it
lies; the others — `contains`, `sorted`, `join`, `sum`, `zip`, … — work over
a copy of its elements, and `a.toList()` makes the list itself. The other way is
`xs.toArray<4>()`, an `Array<T, 4>?`: null unless the list holds exactly four.

Two arrays of one type are `==`, hash and print as lists do. An array is
`Default` when its element is (`N` copies of it), `Codable` as a list —
reading one insists on exactly `N` elements — and `Sendable` when its
element is. A `const` may be an array: a read-only table in the binary,
read where it lies, and a constant index into it is the element itself at
compile time.

```veles
use io

const SQUARES: Array<i64, 5> = [0, 1, 4, 9, 16]

fun main() {
  var total = 0
  loop (i in SQUARES.indices()) {
    total += SQUARES.at(i)
  }
  io.println("$total ${SQUARES.at(3)} ${[3, 1, 2].toArray<3>() == [3, 1, 2]}")
}
```

Output:
```text
30 9 true
```

### Large arrays live in memory

A value of 128 bytes or more that holds an array — a 4 KiB buffer, a struct
with one, a `T?` around one, a `Result` carrying one — is never loaded into
registers. It is copied with `memmove`, passed to a function as the address of
a copy the caller makes, and returned through a pointer the caller supplies,
so a megabyte `var page: Array<u8, 1048576>` compiles and copies like the
`memcpy` it is. Nothing changes in how it is written; what to keep in mind is
that a copy is a copy: pass a pointer (`&page`) to share one array, as with
any large struct.

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

## Queues and priority queues

A `MutableList` is already a stack (`push`, `pop`, `last`). For a queue,
the prelude has `Deque<T>`, a ring buffer with O(1) work at both ends; for
"smallest first", `PriorityQueue<T>`, a binary heap. Both are reference
types like `MutableList`: a `val` binding can grow them, and a function
that receives one shares it with the caller.

```veles
use io

struct Job {
  name: string
  cost: i64

  implement Comparable {
    fun compareTo(other: Job): Ordering = this.cost.compareTo(other.cost)
  }
}

fun main() {
  val queue = Deque<string>()
  queue.addLast("b")
  queue.addLast("c")
  queue.addFirst("a")
  io.println("$queue ${queue.removeFirst()} ${queue.last()} ${queue.len()}")

  val jobs = PriorityQueue<Job>.natural()
  jobs.push(Job(name: "deploy", cost: 5))
  jobs.push(Job(name: "lint", cost: 1))
  jobs.push(Job(name: "test", cost: 3))
  loop {
    val job = jobs.pop() ?: break
    io.println("${job.name} (${job.cost})")
  }

  // a comparator instead of the natural order: largest first
  val biggest = PriorityQueue<i64>(compare: (a, b) => b.compareTo(a))
  loop (x in [4, 9, 2]) biggest.push(x)
  io.println("${biggest.pop()} ${biggest.peek()}")
}
```

Output:
```text
[a, b, c] a c 2
lint (1)
test (3)
deploy (5)
9 4
```

`removeFirst`, `removeLast`, `first`, `last`, `at(i)` and `pop`, `peek`
return `T?`: null when the container is empty, never a panic. To build a
list of a known size up front, `MutableList.repeat(false, n)` gives
`n` copies of a value (a `MutableList<bool>`: the type comes from the
value, D137) and `MutableList<MutableList<i64>>.make(n, _ => [])`
calls the function once per slot. `xs.fill(x)` overwrites every element of
an existing list. `repeat` and `fill` refuse an element type with shared
mutable state — a mutable collection, a pointer, a closure — because every
slot would alias the one value; `make` is the form for those.

## Choosing between them

| You need | Use |
|---|---|
| a fixed sequence, possibly shared | `List<T>` |
| exactly `N` elements, inline, copied like a struct (a table, a hash state, a C buffer) | `Array<T, N>` |
| to accumulate results | `MutableList<T>` then `.toList()` |
| lookup by key, ordered | `Map<K, V>` / `MutableMap<K, V>` |
| membership tests, no duplicates | `Set<T>` / `MutableSet<T>` |
| a queue, a sliding window, a stack shared by reference | `Deque<T>` |
| the smallest (or largest) element next | `PriorityQueue<T>` |

Next: [Structs and methods](05-structs-and-methods.md).
