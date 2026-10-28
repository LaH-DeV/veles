// D217: a function value whose type does not say `suspends` — a named
// function or a variable — is accepted for a `suspends` parameter of a
// conditionally suspending function: the call is plain, as with a plain
// lambda. Where the parameter is kept, where another argument makes the
// call suspend, or anywhere that is not such an argument, it is refused
// with the lambda that adapts it.
use io

fun double(n: i64): i64 => n * 2
fun isOdd(n: i64): bool => n % 2 == 1
fun add(a: i64, b: i64): i64 => a + b

fun slowDouble(n: i64): i64 {
  await sleep(Duration.millis(1))
  n * 2
}

struct Hooks {
  var onEach: (fun(i64): i64 suspends)? = null
}

fun register(hooks: *Hooks, f: fun(i64): i64 suspends) {
  hooks.onEach = f
}

fun both(xs: List<i64>, f: fun(i64): i64 suspends, g: fun(i64): i64 suspends): i64 {
  var total = 0
  loop (x in xs) total += f(x) + g(x)
  total
}

fun main() {
  val f = double
  val doubled = [1, 2].map(double).sum()
  val viaVar = [3, 4].map(f).sum()
  val odd = [1, 2, 3].filter(isOdd).len()
  val summed = [1, 2, 3].fold(0, add)
  val plainCall = both([1, 2], double, x => x + 1)
  var hooks = Hooks()
  register(&hooks, double)  // error: the function this is passed to keeps it
  val suspending = both([1, 2], double, x => slowDouble(x))  // error: another argument makes this call suspend
  val stored: fun(i64): i64 suspends = double  // error: which it may only do as an argument a function calls
  io.println("$doubled $viaVar $odd $summed $plainCall $suspending ${stored(1)}")
}
