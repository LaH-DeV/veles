// D176: `annotation Name(…) on fun = wrapper` names an ordinary generic function
// that wraps the functions it stands on: the body becomes a lambda handed to
// the wrapper, in one of two shapes — (annotation, body) or (annotation, args,
// body) — and several decorators nest, the first written outermost.
use io

error Flaky { n: i64 }

annotation Retry(times: i64) on fun = retryCall
annotation Logged(label: string) on fun = logged
annotation Twice on fun = twice

fun retryCall<R, E>(a: Retry, body: fun(): R suspends throws E): R suspends throws E =>
  try retry(a.times, () => try body(), delay: Duration.zero)

fun logged<A, R>(a: Logged, args: A, body: fun(A): R): R {
  io.println("[${a.label}] $args")
  body(args)
}

fun twice<R>(a: Twice, body: fun(): R): R {
  body()
  body()
}

@Retry(times: 2)
fun flaky(n: i64): i64 throws Flaky {
  if (n < 0) throw Flaky(n)
  n
}

@Logged(label: "add")
fun add(a: i64, b: i64): i64 => a + b

@Logged(label: "x")
@Twice
fun doubled(x: i64): i64 => x * 2

struct Counter {
  var n: i64 = 0

  @Twice
  fun bump() {
    this.n += 1
  }
}

// the declaration's own mistakes
annotation OnField(x: i64) on field = twice // error: stands on functions and nothing else
annotation Missing(x: i64) on fun = nothingHere // error: is not a function in scope
annotation NotFun(x: i64) on fun = Flaky // error: must be a function
annotation Arity(x: i64) on fun = io.println // error: takes the annotation and the body
annotation Wrong(x: i64) on fun = twice2 // error: must be the annotation 'Wrong'

fun twice2<R>(a: Twice, body: fun(): R): R => body()

// and where it is used
trait Sig {
  @Twice // error: wraps a body, and 'f' has none
  fun f()
}

@Logged(label: "v") // error: variadic parameter
fun variadic(xs: i64...): i64 => xs.len()

fun main() {
  io.println("${add(1, 2)} ${doubled(4)} ${flaky(3) catch (e) { -1 }}")
  var c = Counter()
  c.bump()
  io.println("${c.n}")
}
