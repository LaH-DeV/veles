// D9: a trait used as a type is a trait object; only the trait's own
// methods can be called through it, with the trait's parameters.
use io

trait Labelled {
  fun label(): string
  fun pad(n: i64): string
}

trait Waits {
  fun wait() suspends
}

fun misspelled(l: Labelled): string = l.lable() // error: trait 'Labelled' has no method 'lable'; did you mean 'label'?
fun short(l: Labelled): string = l.pad() // error: missing argument 'n' in call to 'pad'
fun waits(w: Waits) {
  w.wait() // error: suspending trait methods are not supported yet
}

fun main() {
  io.println("")
}
