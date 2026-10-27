// Decorators (D176): an annotation that names an ordinary generic function
// wrapping the functions it stands on. `@Retry(times: 3)` makes `flaky`'s body
// a lambda handed to `retryCall`; `throws` and suspension flow through the
// generic wrapper, so nothing here is special to the compiler but the rewrite.
// A wrapper that takes `args` also sees the call's arguments; several
// decorators nest, the first written outermost.
use io { println }

error Flaky {
  n: i64
}

annotation Retry(times: i64, delay: Duration = Duration.zero) on fun = retryCall
annotation Logged(label: string) on fun = logged
annotation Twice on fun = twice

fun retryCall<R, E>(a: Retry, body: fun(): R suspends throws E): R suspends throws E =>
  try retry(a.times, () => try body(), delay: a.delay)

fun logged<A, R>(a: Logged, args: A, body: fun(A): R): R {
  println("[${a.label}] called with $args")
  val result = body(args)
  println("[${a.label}] returned $result")
  result
}

fun twice<R>(a: Twice, body: fun(): R): R {
  body()
  body()
}

val attempts = Atomic(value: 0)

@Retry(times: 3)
fun flaky(): string throws Flaky {
  val n = attempts.add(1)
  if (n < 3) throw Flaky(n)
  "succeeded on attempt $n"
}

@Logged(label: "add")
fun add(a: i64, b: i64): i64 => a + b

@Logged(label: "doubled")
@Twice
fun doubled(x: i64): i64 {
  println("  body runs with $x")
  x * 2
}

struct Counter {
  var n: i64 = 0

  @Twice
  fun bump() {
    this.n += 1
  }
}

fun main() {
  println(flaky() catch (e) {
    "gave up"
  })
  println("${add(2, 3)}")
  println("${doubled(4)}")
  val c = Counter()
  c.bump()
  println("the counter bumped ${c.n} times")
}
