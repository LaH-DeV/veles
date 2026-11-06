# 4. Collections

Veles has three built-in collections — `List`, `Map` and `Set` — and all
three are **values**, like a struct (D231). That is the one idea to absorb
in this chapter: `val b = a` gives `b` a list of its own, passing a list to
a function gives the function its own, and changing one never changes
another. Copying is cheap — the elements are copied only when one of the
copies is first changed — and because nothing else can change a list you
hold, a list can be handed to another task without a lock (D35).

## Lists

::: code-group

```veles [Veles]
use io

fun main() {
  val primes = [2, 3, 5, 7]  // List<i64>
  io.println("${primes.len()} ${primes.get(0)} ${primes.get(10) ?: -1} ${primes.contains(5)}")

  val names = ["ann", "bob"]  // List<string>
  names.push("cy")
  val last = names.pop()  // string?
  io.println("$names ${last ?: "-"} ${names.isEmpty()}")

  val before = names  // a copy: lists are values
  names.push("dee")   // does not affect `before`
  io.println("$before $names")
}
```

```go [Go]
package main

import (
	"fmt"
	"slices"
)

func main() {
	primes := []int64{2, 3, 5, 7} // nothing stops a later primes[0] = 1
	tenth := int64(-1)
	if len(primes) > 10 {
		tenth = primes[10]
	}
	fmt.Println(len(primes), primes[0], tenth, slices.Contains(primes, 5))

	names := []string{"ann", "bob"}
	names = append(names, "cy")
	last := "-"
	if len(names) > 0 {
		last, names = names[len(names)-1], names[:len(names)-1]
	}
	fmt.Println(names, last, len(names) == 0)

	before := slices.Clone(names) // `before := names` would share the array
	names = append(names, "dee")
	fmt.Println(before, names)
}

// 4 2 -1 true
// [ann bob] cy false
// [ann bob] [ann bob dee]
```

```ts [TypeScript]
const primes: readonly number[] = [2, 3, 5, 7]; // readonly: no push, no index write
console.log(`${primes.length} ${primes[0]} ${primes.at(10) ?? -1} ${primes.includes(5)}`);

const names = ["ann", "bob"];
names.push("cy");
const last = names.pop(); // string | undefined
console.log(`[${names.join(", ")}] ${last ?? "-"} ${names.length === 0}`);

const before: readonly string[] = [...names]; // a copy
names.push("dee");
console.log(`[${before.join(", ")}] [${names.join(", ")}]`);
```

```swift [Swift]
let primes = [2, 3, 5, 7] // a `let` array cannot change
print("\(primes.count) \(primes[0]) \(primes.indices.contains(10) ? primes[10] : -1) \(primes.contains(5))")

var names = ["ann", "bob"]
names.append("cy")
let last = names.popLast() // String?
print("\(names) \(last ?? "-") \(names.isEmpty)")

let before = names // arrays are values: this is already a copy
names.append("dee")
print("\(before) \(names)")

// 4 2 -1 true
// ["ann", "bob"] cy false
// ["ann", "bob"] ["ann", "bob", "dee"]
```

```kotlin [Kotlin]
fun main() {
    // List is a read-only view, not immutable: a List passed
    // as a List can still change under its reader
    val primes = listOf(2L, 3L, 5L, 7L)
    println("${primes.size} ${primes[0]} ${primes.getOrNull(10) ?: -1L} ${primes.contains(5)}")

    val names = mutableListOf("ann", "bob")
    names.add("cy")
    val last = names.removeLastOrNull() // String?
    println("$names ${last ?: "-"} ${names.isEmpty()}")

    val before = names.toList() // a copy
    names.add("dee")
    println("$before $names")
}
```

```rust [Rust]
fn main() {
    let primes = vec![2_i64, 3, 5, 7]; // not `mut`: cannot change
    println!(
        "{} {} {} {}",
        primes.len(),
        primes[0],
        primes.get(10).copied().unwrap_or(-1),
        primes.contains(&5)
    );

    let mut names = vec!["ann", "bob"];
    names.push("cy");
    let last = names.pop(); // Option<&str>
    println!("{names:?} {} {}", last.unwrap_or("-"), names.is_empty());

    let before = names.clone(); // a copy
    names.push("dee");
    println!("{before:?} {names:?}");
}

// 4 2 -1 true
// ["ann", "bob"] cy false
// ["ann", "bob"] ["ann", "bob", "dee"]
```

:::

Output:
```text
4 2 -1 true
[ann, bob] cy false
[ann, bob] [ann, bob, dee]
```

- `[a, b, c]` is a `List`. When the expected type is known — an annotated
  binding, a parameter, a field, a return — the literal takes it:
  `val xs: List<i64> = []`, `fill([1, 2])`. An **empty** literal always
  needs such a type, because there is nothing to infer it from; when a
  later line decides it (`val xs = []` then `xs.push(1)`), the error comes
  with a fix that writes `val xs: List<i64> = []` for you (`veles check
  --fix`, or the editor's quick fix; D106). A list literal where a `Set` is
  expected builds a set: `val seen: Set<i64> = [1, 2]`.
- Reading is a method, never brackets — `[...]` only ever builds a literal.
  `xs.get(i)` returns a copy of the element, a `T?`: null when `i` is out
  of range. Where the compiler can **see** that `i` is in range, it
  returns `T` instead (see below). Where you know better than the
  compiler, say why: `xs.get(i) ?: panic("the header always has three
  parts")`. `xs.getOrDefault(i, d)` is `xs.get(i) ?: d`. A negative index
  counts from the end: `xs.get(-1)` is the last element. Its partner
  `xs.at(i)` reaches the element itself, to change it (see
  [Updating elements in place](#updating-elements-in-place)).
- `push`, `pop`, `set`, `clear` change the list where it lives. `val` only
  means the name is never bound to another list — as for a struct, whose
  methods may change it (D22) — so `val names = ["ann"]; names.push("bob")`
  is fine. What cannot be changed is a module-level `val`: a constant every
  task shares, and a change to it is a compile-time error.

### A list is a value

Everything that copies a struct copies a list: binding it, assigning it,
passing it, returning it, storing it in a field or another collection,
sending it on a channel. Afterwards the two are independent:

```veles
use io

struct Inventory {
  items: List<string> = []
}

fun withExtra(xs: List<string>): List<string> {
  xs.push("extra")  // the function's own copy: the caller's list is untouched
  xs
}

fun main() {
  val a = ["x", "y"]
  val b = a
  b.push("z")
  io.println("$a $b ${withExtra(a)} $a")

  val shop = Inventory(items: ["pen"])
  val copy = shop                // the struct's list is copied with it
  copy.items.push("ink")
  io.println("${shop.items} ${copy.items}")

  val grid: List<List<i64>> = [[1], [2]]
  val saved = grid
  grid.at(0)?.push(10)        // the inner list itself, in place
  io.println("$grid $saved")
}
```

Output:
```text
[x, y] [x, y, z] [x, y, extra] [x, y]
[pen] [pen, ink]
[[1, 10], [2]] [[1], [2]]
```

**What it costs.** A list is one pointer to a header (where its elements
are, how many, how many fit). Copying a list copies that pointer and marks
the header *shared*; the elements are not touched, so a copy is O(1) however
long the list is. The first change made through a shared header copies the
elements into storage of its own — O(n), once — and from then on that list
changes in place again. So the cost lands on the **first change after a
copy**, not where the copy is written:

```veles
use io

fun main() {
  val log: List<string> = []
  loop (i in 0..<3) log.push("line $i")  // no copy: nobody else holds `log`
  val snapshot = log                      // O(1): the header is marked shared
  log.push("line 3")                      // O(n) once: `log` gets its own elements
  log.push("line 4")                      // in place again
  io.println("${snapshot.len()} ${log.len()}")
}
```

Output:
```text
3 5
```

The compiler leaves the mark out where no second holder can exist: a
list that was just built (a literal, a call's result), a local returned at
the end of its function (it moves to the caller), and a local passed to a
parameter the function only reads (it is lent for the call). A copy
therefore costs something only where the program really keeps two lists
and then changes one. The mark is not undone: once two holders existed,
the first of them to change copies, even if the other is gone — at most one
copy per sharing. An empty `[]` is one shared constant per element type, so
it allocates nothing until its first `push`. `clear()` on a shared list (or
map, or set) copies none of the elements it would drop: the list takes that
empty constant, and the other holder keeps its elements.

**Seeing where it copies.** The copy happens at the first change, not
where the second holder was made, so the editor shows both ends. The
statement that made the second holder ends with what will be copied and
where: in the example above, `val snapshot = log` is followed by
`log copied at its first change, line 7`. A change two or more lines further
on gets a short marker of its own, `copies log`; inside a loop that repeats
the change but not the copy that shared the list, it says `copies log
once`, because only the first round copies and the others change the copy
in place:

```veles
// fragment
val more = list                     // « more copied at its first change, line 4 »
loop (x in list) {
  more.push(x * 2)                  // « copies more once »
}
```

A loop over a list its body changes says so on its first line
(`xs copied at its first change, line …`); the first change to a list
parameter says `copies xs (a parameter)`, and the first change in a lambda
to a capture-list copy says `copies xs (each call)` — every call starts from
the snapshot (chapter 10); a `clear()` of a shared list says
`copied as empty`. Making a changed copy this way costs one copy of the
elements, and the copy is made with room to grow, so the pushes that follow
do not copy them again. A share in another function is followed too: a
field a method returns (`fun all(): List<string> => this.items`) makes the
next change of it in any method say `copies this.items (shared in
Shelf.all)`, and a change through a pointer parameter whose caller had
shared the value says `copies into (shared by the caller, main)`. The hints
are on by default; the
`veles.inlayHints.copies` setting turns them off.

**Changing someone else's list** is spelled with a pointer, the same way as
for a struct (D10): the function takes `*List<T>`, the caller passes `&xs`.
The sharing is in the signature, where a reader sees it:

```veles
use io

fun fill(into: *List<i64>, n: i64) {
  loop (i in 0..<n) into.push(i * i)
}

fun main() {
  val squares: List<i64> = []
  fill(&squares, 4)
  io.println("$squares")
}
```

Output:
```text
[0, 1, 4, 9]
```

A function that changes a list parameter it took by value changes its own
copy, and the caller never sees the change — unless it returns the list,
as `withExtra` does. That is a warning (`param_copy`) naming the change and
the fix (take `*List<T>`, or return the value). So is the other half of the
same mistake: binding a copy of a field or an element and changing the
copy, `val edges = this.adj.get(a) ?: …; edges.push(b)` (`copy_changed`):
the change was meant for the element, which `at` reaches.

Two places copy where it is easy to miss:

- a **trait object** holds its own copy of the value boxed into it (D9).
  When the type's methods change `this`, boxing a variable, a parameter or
  a field is an error, since the changes would land in the copy: pass `&x`
  to box the variable itself, so its methods change `x`, or `x.copy()` for
  a copy on purpose (D232, chapter 8);
- an **element read** (`xs.get(i)`, `m.get(k)`, the variable of
  `loop (x in xs)`) is a copy of the element — for a list of lists, of the
  inner list; `at` and `loop (&x in xs)` reach the element itself (see
  [Updating elements in place](#updating-elements-in-place)).

### Copy or share

Every collection — `List`, `Map`, `Set`, `Array`, `Deque`,
`Heap`, `PriorityQueue` — is a value, so there are three ways to hand one on, and
the code says which:

| You want | Write | What happens |
|---|---|---|
| a copy | `val b = a`, or `a.copy()` to say so | O(1) now; the first change of either copies (D231, D236) |
| the same collection, changed by someone else | `&a`, a `*List<T>` parameter or field | one collection; changes are seen through the pointer (D10) |
| the same collection, across tasks | `Mutex(value: a)`, `withLock` | one collection behind a lock (D35) |

`copy()` is the same as binding — it exists so a copy made on purpose reads
as one, under the name a struct's copy has (D164). A lambda shares what
it captures; a capture list, `[a] () => …`, copies instead (see
[Closures](10-closures-and-iterators.md#copying-into-a-lambda)).

```veles
use io

fun main() {
  val draft: List<string> = ["intro"]
  val published = draft.copy()       // a copy, said so
  val editing = &draft               // the same list
  editing.push("body")
  io.println("$draft $published")
}
```

Output:
```text
[intro, body] [intro]
```

### When the index is known to be in range

The compiler tracks what your code has already checked, as it does for
null (D62), and a read it can prove is in range has type `T`:

```veles
use io

fun sum(xs: List<i64>): i64 {
  var total: i64 = 0
  loop (i in xs.indices()) {
    total += xs.get(i)  // i64: i is one of xs's indexes
  }
  total
}

fun ends(xs: List<i64>): i64 {
  if (xs.len() < 2) return 0
  xs.get(0) + xs.get(-1)  // i64: the length was checked above
}

fun build(n: i64): List<i64> {
  val out: List<i64> = []
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
- What ends it: assigning the list or the index. For a `List`,
  also any function call, because the call might shrink the list through
  another name. Index arithmetic such as `xs.get(i + 1)` is not tracked.
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

fun describe(args: List<string>): string => when (args) {
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
  `List` afterwards does not change them.

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
  io.println("done")
}
```

Output:
```text
[10, 6, 16, 2] [5, 3, 8] 17
[1, 3, 5, 8] [8, 5, 3, 1] [1, 8, 3, 5]
true true 5 2
5 1 5, 3, 8, 1
5 3 8 1 done
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
  // the first of the two 20s, and null for a value that is not there
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
1 null
1 3
3
1 null
```

None of them checks that the list is sorted, because checking is the linear
scan they exist to avoid: on a list that is not sorted the answer is simply
wrong, never a panic.

### Updating elements in place

Every read is a **value**: `xs.get(i)`, `xs.first()`, `xs.find(p)` and
the loop variable of `loop (x in xs)` all hand you a copy of a struct
element, exactly as `val t = xs.get(i)` does. So a change to what you
read never reaches the list — and the compiler says so rather than
letting it vanish: `xs.get(i)?.bump()` is an error ("a copy of the
element").

To change an element, reach it **at** its place. `xs.at(i)` is a
pointer to the element, `(*T)?`: null when `i` is out of range, and a
plain `*T` where the index is known to be in range. `loop (&x in xs)`
visits every element by reference. The pair is spelled the same on
`List`, `Array` and `Map` — `at` reaches the element, `get` copies it
(D231) — so a read and a write never look alike:

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
  val counters: List<Counter> = [Counter(name: "a"), Counter(name: "b")]
  var copy = counters.get(0) ?: panic("two counters")  // a copy: `var` because we change it
  copy.n = 100
  counters.at(0)?.bump()  // the element itself
  counters.at(1)?.n = 5   // in range: written; out of range: skipped
  counters.at(7)?.n = 5
  val p = counters.at(1) ?: panic("two counters")  // a pointer may sit in a `val`
  p.bump()
  loop (&c in counters) c.n *= 10  // every element, in place
  io.println("${copy.n} ${counters.map(c => c.n)}")

  val nums: List<i64> = [1, 2, 3]
  if (nums.len() > 0) *nums.at(0) += 10  // a primitive: write through the pointer
  loop (&n in nums) *n += 1
  nums.set(2, 0)  // or replace the element outright
  io.println("$nums")
}
```

Output:
```text
100 [10, 60]
[12, 3, 0]
```

On a map, `m.at(k)`, `m.atOrPut(k, make)` and `loop ((k, &v) in m)` reach
the stored values. A reference is a pointer into the
collection's own storage, so taking one first makes that storage the
list's alone — a list that was shared is copied then, once — and a copy of
the list taken while the pointer lives gets storage of its own, so the
pointer never reaches the copy (D231). Keep a reference short-lived: after
the list grows (`push`) an old reference points at the old buffer.

### Changing a collection while a loop walks it

A loop over a `List`, `Map` or `Set` walks the collection **as it was when
the loop began** (D231). The body may change it — push, remove, clear — and
the loop goes on over the original elements; the changes are there after
the loop:

```veles
use io

fun main() {
  val xs = [1, -2, 3]
  loop (x in xs) {
    if (x > 0) xs.push(-x)   // the loop still sees [1, -2, 3]
  }
  io.println("$xs")
}
```

Output:
```text
[1, -2, 3, -1, -3]
```

This is the value rule again: the loop holds a copy of the list. A copy
costs nothing until something changes: the loop marks the list as being
walked, and the first change made while it runs — in the body, or anywhere
else, through a pointer a called function holds — copies the elements
once, so the change lands in the variable and the loop keeps the original.
A loop during which nothing changes the list copies nothing (D102
addendum). Changing an element in place through `loop (&x in xs)` changes
the list itself — that loop walks the list's own storage, so a change to
its length from the body is refused, and one made elsewhere panics when
the loop next steps.

To grow a worklist and visit the new elements too, loop with an index, not
over the list:

```veles
use io

fun main() {
  val work: List<i64> = [3]
  var i = 0
  loop (i < work.len()) {  // reads the length every step: sees growth
    val n = work.get(i)
    if (n > 1) work.push(n - 1)
    i += 1
  }
  io.println("$work")
}
```

Output:
```text
[3, 2, 1]
```

A `Deque` is a value too (D234), and a loop over one walks it as it was
when the loop began, so the body may add to it or take from it.

### Comparing collections

Two lists are equal when they hold equal elements in the same order; two
maps when they hold the same keys with equal values; two sets when they
hold the same elements — order never matters for a map or a set. Anything
built from comparable parts is comparable: a `List<List<i64>>`, a struct with a
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
  var b: List<i64> = []
  b.push(1); b.push(2); b.push(3)
  io.println("${a == b} ${a == [3, 2, 1]} ${[[1], [2]] == [[1], [2]]}")
  io.println("${["x": 1, "y": 2] == ["y": 2, "x": 1]} ${[1, 2].contains(2)} ${[[1, 2], [3]].indexOf([3])}")
  val s1: Set<i64> = [1, 2]
  val s2: Set<i64> = [2, 1, 1]
  io.println("${s1 == s2} ${Row(name: "a", scores: [1]) == Row(name: "a", scores: [1])}")
  var byScores: Map<List<i64>, string> = [:]
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

## Tuples

A tuple groups a fixed number of values, each of its own type: `(i64,
string)`. Its elements are read by position — `t.0`, `t.1` — and, when
its type labels them, by name (D166):

```veles
use io

fun divide(a: i64, b: i64): (quotient: i64, remainder: i64) => (a / b, a % b)

fun main() {
  val r = divide(17, 5)
  io.println("${r.quotient} rest ${r.remainder}, or ${r.0} rest ${r.1}")
  io.println("$r")
  val (q, rest) = divide(9, 4)                         // destructuring is positional
  val plain: (i64, i64) = r                            // labels come off freely
  val again: (quotient: i64, remainder: i64) = plain   // and go back on
  io.println("$q $rest $plain ${again.quotient} ${plain == r}")

  val stock = ["apples": 3, "pears": 7]
  if (val most = stock.entries().sortedBy(e => -e.value).first()) {
    io.println("most: ${most.key} (${most.value})")
  }
  loop (e in ["a", "b"].enumerate()) io.println("${e.index} ${e.value}")
  if (val setting = "port=8080".splitOnce("=")) io.println("${setting.before} is ${setting.after}")
}
```

Output:
```text
3 rest 2, or 3 rest 2
(quotient: 3, remainder: 2)
2 1 (3, 2) 3 true
most: pears (7)
0 a
1 b
port is 8080
```

- A tuple type labels **every element or none**: `(key: K, value: V)`. A
  literal may be labelled the same way, `(key: "a", value: 1)`, or left
  unlabelled where the expected type gives the labels — as `divide`'s body
  does.
- Labels are for reading. A labelled tuple has the same layout, `==`,
  hashing and ordering as the unlabelled one, and a generic function
  compiles once for both; destructuring stays positional
  (`val (q, rest) = …`, `(k, v) => …`, `loop ((k, v) in m)`).
- An unlabelled tuple converts to a labelled one and back. Two **different**
  label lists do not: `(x: 1, y: 2)` where a `(y: i64, x: i64)` is
  expected is an error, because the elements would silently swap meaning.
  The fix drops the value's labels when the positions are what you meant.
- `"$t"` prints the labels: `(quotient: 3, remainder: 2)`.
- std labels its results where a name helps: `Map.entries()` gives
  `(key: K, value: V)`, `enumerate()` gives `(index: i64, value: T)`,
  `splitOnce` gives `(before: string, after: string)?`. `zip`, `gather`
  and `awaitAll` stay unlabelled.

**Tuple or struct?** A labelled tuple is for a short-lived grouping: a
function's several results, an entry, a row on its way to being printed. As
soon as the grouping has behaviour, an invariant, documentation per field,
or a wire form (tuples are never `Codable`), make it a struct (chapter 5).

One limit: a generic **type** of your own shares one instance among every
labelling of its arguments and keeps none, so writing
`Box<(lo: i64, hi: i64)>` is an error that says so — write
`Box<(i64, i64)>`, or make the pair a struct. Functions and the built-in
collections are not affected: `entries().sortedBy(…)` still gives labelled
entries, and `List<(key: K, value: V)>` keeps its labels.

## Maps

A `Map<K, V>` keeps its entries in **insertion order** — iteration is
deterministic, always (D25).

```veles
use io

fun main() {
  val ages = ["ann": 41, "bob": 29]  // Map<string, i64>
  io.println("${ages.get("ann") ?: 0} ${ages.get("zed") ?: 0} ${ages.containsKey("bob")} ${ages.len()}")

  var stock = ["apples": 3]  // Map<string, i64>
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
Writing `m.set(key, value)` requires a `Map` and inserts or
replaces. There is no bracket form: `[...]` only ever builds a literal.
A read is a copy of the value; to change the stored value ask for a
reference: `m.at(key)?.bump()` and `m.at(key)?.n += 1` update the
entry in the map, and `loop ((k, &v) in m)` visits every value by
reference (see [Updating elements in place](#updating-elements-in-place)
above; chapter 6 has the `?.` side).

Keys must be **hashable**: numbers, strings, booleans, tuples of those,
structs whose fields are (structs get equality and hashing for free),
and lists, maps and sets of hashable elements — `m.get([1, 2])`
finds a key stored as `[1, 2]`, because collections compare by content
(`[1, 2] == [1, 2]` is true; a set or map ignores insertion order). A key
is a copy, like any stored value, so changing the list you stored it from
does not change the key.

Grouping is `atOrPut`: the value stored under a key, as a pointer to
change it, made with the function you pass when there is none yet:

```veles
use io

fun main() {
  val words = ["apple", "avocado", "banana", "blueberry", "cherry"]
  val byLetter: Map<string, List<string>> = [:]
  loop (w in words) {
    byLetter.atOrPut(w.substring(0, 1) ?: "", () => []).push(w)
  }
  io.println("$byLetter")
}
```

Output:
```text
{a: [apple, avocado], b: [banana, blueberry], c: [cherry]}
```

## Sets

```veles
use io

fun main() {
  var seen = Set<i64>()
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

`add` returns whether the element was new; `toList()` gives the members as
a list, in the order they were added. There is no set literal of its
own: construct with `Set<T>()`, convert a list with `xs.toSet()`, or
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
  var state: Array<u32, 8> = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ]
}

fun main() {
  var a: Array<i64, 4> = [10, 20, 30, 40]
  a.set(1, 99)
  val b = a
  a.set(0, 7)
  io.println("$a $b ${a.len()}")
  val far = 9
  io.println("${a.get(1)} ${a.get(far)} ${a.fold(0, (s, x) => s + x)}")
  loop (&x in a) {
    *x = *x + 1
  }
  var sha = Sha()
  sha.state.set(0, 1)
  io.println("$a ${sha.state.get(0)} ${Array<i64, 3>.make(5)}")
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
called on, which must be a `var`. Reading is a list's — `a.get(i)` is a `T?`,
null when `i` is out of range, a negative `i` counts from the end, and
`a.at(i)` is a pointer to the element — with one difference: where the
index is a constant in range, `a.get(3)` is a plain `T` and compiles to a load, so a table needs no `?:`; a constant out of range
is an error. The bounds facts of [the earlier section](#when-the-index-is-known-to-be-in-range)
work too: `loop (i in a.indices())` and `loop (i in 0..<4)` make `a.get(i)`
a `T`. `loop (x in a)` walks a copy; `loop (&x in a)` walks the array itself
and lets the body change it. `a.len()` is the constant `N`.

An array never grows or shrinks, so it has none of a `List`'s
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
    total += SQUARES.get(i)
  }
  io.println("$total ${SQUARES.get(3)} ${[3, 1, 2].toArray<3>() == [3, 1, 2]}")
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

A `List` is already a stack (`push`, `pop`, `last`). For a queue,
the prelude has `Deque<T>`, a ring buffer with O(1) work at both ends; for
"smallest first", `Heap<T>`, a binary heap over the elements' own order
(`Comparable`), which compares them directly; and for an order of your
own, `PriorityQueue<T>(compare:)`, the same heap calling your comparator
(D239). All three are values like `List` (D234): a `val` binding can grow
them, a copy is O(1) and made at its first change, and a function that
should change the caller's queue takes `*Deque<T>`. A priority queue's
comparator is shared by its copies — a function cannot change — and is a
`sendable fun`, so a queue crosses into a task when its elements can.

```veles
use io

struct Job {
  name: string
  cost: i64

  implement Comparable {
    fun compareTo(other: Job): Ordering => this.cost.compareTo(other.cost)
  }
}

fun main() {
  val queue = Deque<string>()
  queue.addLast("b")
  queue.addLast("c")
  queue.addFirst("a")
  io.println("$queue ${queue.removeFirst()} ${queue.last()} ${queue.len()}")

  val jobs = Heap<Job>()               // Job's own order: by cost
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

`removeFirst`, `removeLast`, `first`, `last`, `get(i)` and `pop`, `peek`
return `T?`: null when the container is empty, never a panic. To build a
list of a known size up front, `List.repeat(false, n)` gives
`n` copies of a value (a `List<bool>`: the type comes from the
value, D137) and `List<List<i64>>.make(n, _ => [])`
calls the function once per slot. `xs.fill(x)` overwrites every element of
an existing list. On `List<i64>`, `counts.incrementAt(i)` increments
one element in place; negative indexes count from the end and an invalid
index panics. `repeat` and `fill` refuse an element type with shared
mutable state — a pointer, a closure — because every
slot would alias the one value; `make` is the form for those.

For a generic in-place transformation, use `updateAt`: it passes the current
element to a synchronous function, stores the returned value, and returns
that value itself. The callback runs once and should compute the replacement,
not mutate the same list:

```veles
use io

fun main() {
  val values: List<i64> = [3, 5]
  val updated = values.updateAt(-1, value => value * 2)
  io.println("$updated $values")
}
```

Output:
```text
10 [3, 10]
```

## Choosing between them

| You need | Use |
|---|---|
| a sequence: to build, read, pass around, share with tasks | `List<T>` |
| exactly `N` elements, inline, copied like a struct (a table, a hash state, a C buffer) | `Array<T, N>` |
| a function that fills the caller's list | a `*List<T>` parameter, `&xs` at the call |
| lookup by key, ordered | `Map<K, V>` |
| membership tests, no duplicates | `Set<T>` |
| a queue at both ends, a sliding window | `Deque<T>` |
| the smallest element next, by its own order | `Heap<T>` |
| the next element by an order of your own (the largest, by a field) | `PriorityQueue<T>(compare:)` |
| a few values together for a moment (several results, an entry) | a tuple, labelled: `(min: i64, max: i64)` |

Next: [Structs and methods](05-structs-and-methods.md).
