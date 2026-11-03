// D35 addendum: a value that is not Sendable cannot become an object of a
// trait that requires it (a generic implementor passes the declaration check,
// its instance does not).
trait Sink : Sendable {
  fun put(n: i64)
}

struct Wrap<T> {
  value: T

  implement Sink {
    fun put(n: i64) { }
  }
}

fun forward(s: Sink) { }

fun main() {
  val items: List<i64> = []
  val bad: Sink = Wrap<*List<i64>>(value: &items)  // error: cannot be a 'Sink': the trait requires Sendable
  forward(bad)
}
