# 10. Closures and iterators

## Lambdas

A lambda is `params => body`. One parameter needs no parentheses; a
block body goes in braces. Parameter types are inferred from where the
lambda is used, so `x => x * 2` passed to `map` on a `List<i64>` knows `x`
is an `i64` (D32).

```veles
use io

fun main() {
  val nums = [3, 1, 2]
  val doubled = nums.map(x => x * 2)
  val total = nums.fold(0, (acc, x) => acc + x)
  val labelled = nums.map((x: i64) => "<$x>")          // explicit type
  val big = nums.filter(x => {
    val limit = 1
    x > limit
  })
  io.println("$doubled $total $labelled $big")
}
```

Output:
```text
[6, 2, 4] 6 [<3>, <1>, <2>] [3, 2]
```

## Functions as values

The type of a function is `fun(A, B): R`. Named functions and lambdas
both have such a type, and both can be passed, stored and returned:

```veles
use io

fun apply(f: fun(i64): i64, x: i64): i64 => f(x)
fun double(x: i64): i64 => x * 2

fun compose(f: fun(i64): i64, g: fun(i64): i64): fun(i64): i64 => x => g(f(x))

fun main() {
  io.println("${apply(double, 21)} ${apply(x => x + 1, 1)}")
  val addThenDouble = compose(x => x + 1, double)
  io.println("${addThenDouble(4)}")
  val ops: Map<string, fun(i64, i64): i64> = ["add": (a, b) => a + b, "mul": (a, b) => a * b]
  val mul = ops.get("mul") ?: ((a: i64, b: i64) => 0)
  io.println("${mul(6, 7)}")
}
```

Output:
```text
42 2
10
42
```

## Capturing variables

A lambda can use variables from the surrounding function. It captures
them **by reference**: reads see later writes, and writes are visible
outside (D37).

```veles
use io

fun counter(): fun(): i64 {
  var n = 0
  () => {
    n += 1
    n
  }
}

fun main() {
  val next = counter()
  next()
  next()
  io.println("third call: ${next()}")

  var log: MutableList<string> = []
  val record = (msg: string) => log.push(msg)
  record("a")
  record("b")
  io.println("$log")

  var captured = "before"
  val show = () => captured
  captured = "after"
  io.println(show())
}
```

Output:
```text
third call: 3
[a, b]
after
```

The variable `n` outlives `counter()` because the closure holds it; the
collector keeps it alive. There is no need to think about it.

A lambda that calls a `throws` function infers `throws` itself, and one
that suspends infers `suspends` — the same inference as for named
functions (chapters 7 and 12).

## What a function type says

The full shape of a function type is

```text
[sendable] fun(A, B): R [suspends] [throws E]
```

Each word is a separate promise, and each answers a different question.

- `fun(A, B): R` — what goes in and what comes out. Parameter names are
  not part of the type (a lambda may call them anything).
- `suspends` — *calling it may pause the task*: it awaits a channel, a
  timer or another task somewhere inside, so it can only be called from
  code that may itself suspend. A function without it is a plain call
  that runs to completion. Chapter 12 explains what suspending means.
- `throws E` — *calling it may fail with E*: the call yields a `Result`
  and the caller writes `try` or handles it with `when` (chapter 7). A
  function without it cannot fail.
- `sendable` — *the value may be handed to another task*: it touches no
  shared mutable state, so two tasks may call it at once. Chapter 12's
  [What may cross a task boundary](12-concurrency.md#what-may-cross-a-task-boundary)
  has the rule; in short, a named function always is, and a lambda is
  when everything it captures is a `val` of a Sendable type.

The first three describe what happens when the function is *called*; the
last describes what the *value* is. That difference decides how they
adapt.

**A lambda written where a function is expected takes the expected
effects.** `xs.mapConcurrent(n => fetch(n))` expects
`sendable fun(T): R suspends throws E`; the lambda is compiled as a
suspending function because the parameter says so, its `E` is whatever
it throws (nothing, here), and it is checked to be sendable. You never
write the words on a lambda.

**A function value does not change its type.** A named function or a
stored closure has fixed effects, and it fits a parameter only when they
match: `needsSuspend(plain, 1)` is an error when `plain` does not suspend,
because the two are compiled differently. The fix is the one the message
suggests, a lambda that calls it: `needsSuspend(n => plain(n), 1)`. The one
exception is `sendable`: it is a property the value either has or lacks,
and a sendable function is accepted wherever a plain one is, never the
reverse.

**Generic code names the effects it passes through.** A function that
takes a callback and may fail *because the callback does* is written

```veles
// fragment
fun apply<T, R, E>(x: T, f: fun(T): R throws E): R throws E => try f(x)
```

`E` is inferred at each call from the lambda: `apply(2, n => n * 2)` is an
ordinary non-throwing call, `apply(2, n => try parse(n))` throws what `parse`
throws. Inside `apply`, `try f(x)` is correct either way. This is how
`mapConcurrent` in the prelude gets the same behaviour as the built-in
`map` without the compiler knowing about it, and it is what to write for
any helper of your own that runs a user's function.

## `try` inside a collection operation

A lambda may itself fail. Passed to an eager operation like `map`, that
makes the whole operation fallible: it stops at the first error and
yields it, otherwise the finished list. You `try` the operation as you
would any other call:

```veles
use io

error NotANumber {
  text: string
}

fun parseAll(words: List<string>): List<i64> throws NotANumber =>
  try words.map(w => w.toInt() ?: throw NotANumber(text: w))

fun main() {
  io.println("${parseAll(["1", "2", "3"])}")
  io.println("${parseAll(["1", "x", "3"])}")
  val r = ["4", "5"].map(w => w.toInt() ?: throw NotANumber(text: w))
  if (r.ok) io.println("parsed $r")
}
```

Output:
```text
Ok(value: [1, 2, 3])
Err(error: NotANumber(text: x))
parsed [4, 5]
```

The same holds for `filter`, `fold`, `forEach`, `find`, `any`/`all` and
the map operations. A sort key (`sortedBy`) cannot fail — it runs inside
the sort, where there is no way to stop — so handle the error inside it.

## Currying and tuples

`a => b => a + b` is a function returning a function. A lambda taking
one tuple parameter can be written with the tuple destructured — `(n, s)
=> ...` — where a `fun((i64, string)): R` is expected (D37's tupling
conversion):

```veles
use io

fun main() {
  val adder = (a: i64) => (b: i64) => a + b
  val pairs = [(1, "one"), (2, "two")]
  io.println("${adder(2)(3)} ${pairs.map((n, s) => "$n=$s")}")
  // a parameter may itself be a tuple pattern, as deep as the value goes
  val groups = [((3, 7), ["a"]), ((1, 2), ["b", "c"])]
  io.println("${groups.map(((size, hash), files) => size * files.len() + hash)}")
  val byName = groups.sortedWith((((sa, _), _), ((sb, _), _)) => sa.compareTo(sb))
  val [((size, _), _), ..] = byName else return
  io.println("$size")
}
```

Output:
```text
5 [1=one, 2=two]
[10, 4]
1
```

The same nesting works in `val`/`var` bindings and loop heads
(`loop (((size, hash), files) in groups)`). When positional access
starts to pile up — `e.0.0` — that is usually the moment to name the
tuple as a struct instead.

## Iterators

`loop (x in xs)` works on anything that implements `Iterable`, which
means: it can produce an `Iterator`, which is anything with a `next()`
that returns the next `Item?` or null at the end (D42). Writing your own
takes two small structs:

```veles
use io

struct Countdown {
  from: i64

  implement Iterable {
    type Iter = CountdownIter
    fun iterator(): CountdownIter => CountdownIter(current: this.from)
  }
}

struct CountdownIter {
  var current: i64

  implement Iterator {
    type Item = i64
    fun next(): i64? {
      if (this.current <= 0) return null
      val v = this.current
      this.current -= 1
      v
    }
  }
}

fun main() {
  loop (n in Countdown(from: 3)) { io.print("$n ") }
  io.println("liftoff")
  val squares = Countdown(from: 4).iter().map(n => n * n).toList()
  io.println("$squares")
}
```

Output:
```text
3 2 1 liftoff
[16, 9, 4, 1]
```

The `type Item = i64` line is an **associated type**: each iterator says
what it yields, and the compiler checks that `next()` agrees. `type Iter
= CountdownIter` likewise names which iterator an `Iterable` produces.
Because these are types, not runtime values, `loop` over a `Countdown`
compiles to a plain counting loop.

## Lazy pipelines

`Iterator` comes with adapters — `map`, `filter`, `take`, `skip`,
`enumerate`, `zip` — that build a new iterator without doing any work
until something pulls from it (D46). Terminal operations pull: `toList`,
`count`, `fold`, `forEach`, `any`, `all`, `find`, `last`.

```veles
use io

fun main() {
  val nums = [1, 2, 3, 4, 5, 6]
  val evensSquared = nums.iter().filter(x => x % 2 == 0).map(x => x * x).toList()
  io.println("$evensSquared")
  io.println("${nums.iter().map(x => x * 10).take(2).toList()} ${nums.iter().skip(4).toList()}")
  io.println("${nums.iter().enumerate().map((i, x) => "$i:$x").toList()}")
  io.println("${nums.iter().zip(["a", "b", "c"].iter()).toList()}")
  io.println("${nums.iter().count()} ${nums.iter().fold(0, (a, b) => a + b)} ${nums.iter().find(x => x > 3) ?: -1}")
  loop (x in (1..3).iter().map(x => x * 100)) { io.print("$x ") }
  io.println("")
}
```

Output:
```text
[4, 16, 36]
[10, 20] [5, 6]
[0:1, 1:2, 2:3, 3:4, 4:5, 5:6]
[(1, a), (2, b), (3, c)]
6 21 4
100 200 300 
```

When to use which: `xs.map(f)` on a list is eager and returns a `List`
— simplest when you want the list anyway. `xs.iter().map(f)` allocates
nothing until you ask for a result, and `take(n)` on it stops early —
better for long chains, large inputs, or infinite sources.

Next: [Modules, packages and tests](11-modules-and-packages.md).
