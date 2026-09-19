# 3. Functions and control flow

## Declaring functions

```veles
use io

fun square(x: i64): i64 {
  return x * x
}

fun cube(x: i64): i64 = x * x * x      // expression body

fun greet(name: string, punctuation: string = "!") {
  io.println("Hello, $name$punctuation")
}

fun main() {
  io.println("${square(4)} ${cube(3)}")
  greet("Ada")
  greet("Bob", punctuation: "?")
  greet(punctuation: ".", name: "Cy")
}
```

Output:
```text
16 27
Hello, Ada!
Hello, Bob?
Hello, Cy.
```

- Parameters are `name: Type`; the return type follows the parameter
  list. No return type means the function returns nothing (`()`).
- A parameter can have a **default**. Arguments can be passed by
  **name** (`punctuation: "?"`), in any order, which is the idiomatic way
  to call a function with several parameters of the same type (D28).
- `fun f(...) = expr` is a one-expression function. When its return type
  is obvious you may omit it: `fun twice(x: i64) = x * 2`.
- The last parameter can be **variadic**: `fun sum(xs: i64...): i64` takes
  `sum()`, `sum(1)` or `sum(1, 2, 3)`, and inside `xs` is a `List<i64>`.
  A list you already have is passed whole with `sum(numbers...)`.

### The last expression is the result

A block's final expression is its value, so `return` is only needed to
leave early:

```veles
use io

fun clamp(x: i64, lo: i64, hi: i64): i64 {
  if (x < lo) return lo
  if (x > hi) return hi
  x
}

fun main() {
  io.println("${clamp(5, 0, 10)} ${clamp(-3, 0, 10)} ${clamp(42, 0, 10)}")
}
```

Output:
```text
5 0 10
```

If you dislike the bare `x` at the end, `return x` is equally valid.

## `if` is an expression

```veles
use io

fun describe(n: i64): string {
  val parity = if (n % 2 == 0) "even" else "odd"
  if (n < 0) {
    return "negative and $parity"
  } else if (n == 0) {
    return "zero"
  }
  "positive and $parity"
}

fun main() {
  io.println("${describe(-4)}; ${describe(0)}; ${describe(7)}")
}
```

Output:
```text
negative and even; zero; positive and odd
```

The condition must be a `bool` in parentheses. Braces are optional for a
single statement, as in `if (x < lo) return lo`.

## One loop keyword

`loop` covers every kind of iteration:

```veles
use io

fun main() {
  // 1. forever, until break
  var n = 0
  loop {
    n += 1
    if (n == 3) break
  }

  // 2. while a condition holds
  var m = 10
  loop (m > 1) { m = m / 2 }

  // 3. over a range: `..` inclusive, `..<` exclusive
  var sum = 0
  loop (i in 1..4) { sum += i }
  var count = 0
  loop (_ in 0..<4) { count += 1 }   // `_` when the value is not needed

  // 4. over a collection
  var total = 0
  loop (x in [5, 10, 15]) { total += x }

  io.println("$n $m $sum $count $total")
}
```

Output:
```text
3 1 10 4 30
```

There is no C-style `for (init; cond; step)`. A loop that walks by a fixed
stride — every third element, counting down, the inner loop of a sieve — is
a range with `step()` or `reversed()`, which says *what* is visited rather
than *how*:

```veles
use io

fun main() {
  var marked: MutableList<i64> = []
  val p = 3
  loop (m in (p * p..30).step(p)) marked.push(m)   // multiples of 3 from 9
  var down: MutableList<i64> = []
  loop (i in (0..<5).reversed()) down.push(i)
  io.println("$marked $down")
}
```

Output:
```text
[9, 12, 15, 18, 21, 24, 27, 30] [4, 3, 2, 1, 0]
```

When the step is not arithmetic (`m /= 10`, `cur = cur.next`), write the
condition form and update at the end of the body — and remember that a
`continue` skips that update.

The loop variable is a copy of the element, like every read (`val`, so
`x.n += 1` is an error). To change the elements of a `MutableList` write
`loop (&x in xs)`: `x` is then a pointer to the element and `x.bump()` or
`*x += 1` updates the list; `loop ((k, &v) in m)` does the same for the
values of a `MutableMap`. [Chapter 4](04-collections.md#updating-elements-in-place)
has the full rule.

`break` leaves the loop and `continue` skips to the next iteration. To
target an outer loop, label it after the keyword:

```veles
use io

fun main() {
  loop :rows (r in 1..3) {
    loop (c in 1..3) {
      if (c == 2) continue rows      // next row
      if (r == 3) break rows         // stop everything
      io.println("$r,$c")
    }
  }
}
```

Output:
```text
1,1
2,1
```

## `when`: matching values

`when` is a multi-way branch. With a subject it compares against each
arm; without one each arm is a condition. Every `when` used as a value
needs an `else` (or must be exhaustive, which sealed types make possible
— see [chapter 9](09-sealed-types.md)).

```veles
use io

fun sign(n: i64): string = when {
  n < 0 => "negative"
  n == 0 => "zero"
  else => "positive"
}

fun weekday(d: i64): string = when (d) {
  1 => "Mon"
  2 => "Tue"
  3, 4, 5 => "midweek"
  else => "weekend"
}

fun main() {
  io.println("${sign(-2)} ${sign(0)} ${weekday(1)} ${weekday(4)} ${weekday(7)}")
}
```

Output:
```text
negative zero Mon midweek weekend
```

Arm bodies can be blocks: `1 => { io.println("one"); "one" }`. When
the subject is an expression the arms need to refer to, name it in the
head: `when (val n = text.toInt()) { null => "none"; is i64 => "got $n" }`
— the name lives only inside the `when`, and the type tests narrow it.

## Recursion and mutual recursion

Functions in a module may refer to each other in any order; there are no
forward declarations.

```veles
use io

fun isEven(n: i64): bool = if (n == 0) true else isOdd(n - 1)
fun isOdd(n: i64): bool = if (n == 0) false else isEven(n - 1)

fun fib(n: i64): i64 = if (n < 2) n as i64 else fib(n - 1) + fib(n - 2)

fun main() {
  io.println("${isEven(10)} ${isOdd(7)} ${fib(30)}")
}
```

Output:
```text
true true 832040
```

## Generic functions, briefly

A function can take a type parameter. The compiler generates one copy
per type it is used with (D8), so there is no boxing and no runtime
cost:

```veles
use io

fun <T> first(xs: List<T>, fallback: T): T = xs.at(0) ?: fallback

fun main() {
  io.println("${first([3, 4], 0)} ${first([], "none")}")
}
```

Output:
```text
3 none
```

Type parameters can carry bounds (`<T: Area>`), which is where traits
come in — [chapter 8](08-traits-and-generics.md).

Next: [Collections](04-collections.md).
