// D28/D23: constructors take fields by name, `init` assigns before it
// reads, and a method is reached through a value — `this` inside the type.
use io

struct Plain {
  x: i64
  fun twice(): i64 => this.x * 2
  fun bare(): i64 => twice() // error: unknown function 'twice'; a member is reached through the receiver: 'this.twice'
  fun asValue(): i64 {
    val f = twice // error: unknown name 'twice'; a member is reached through the receiver: 'this.twice'
    1
  }
}

struct Holder<T> {
  items: List<T> = []
}

fun main() {
  val tw = Plain.twice // error: 'twice' is a method of 'Plain'; call it on a value, 'x.twice()', or wrap that call in a lambda to pass it
  val p = Plain(x: 1, 2) // error: construct 'Plain' by field name (D28)
  val h = Holder() // error: cannot infer type parameter 'T' of 'Holder'; write 'Holder<...>(...)' or annotate the binding
  io.println("$p")
}
