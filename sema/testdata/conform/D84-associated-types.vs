// D84: when two bounds of a type parameter declare an associated type of
// one name, `T.Item` does not pick one; `T.Trait.Item` names it.
use io

trait HasItem {
  type Item
}

trait AlsoItem {
  type Item
}

fun ambiguous<T: HasItem + AlsoItem>(x: T.Item) { } // error: 'T.Item' is ambiguous: bounds 'HasItem' and 'AlsoItem' both declare 'Item'; name the one meant: 'T.HasItem.Item' (D84)
fun named<T: HasItem + AlsoItem>(x: T.AlsoItem.Item) { }
fun noSuch<T: HasItem>(x: T.HasItem.Nope) { } // error: trait 'HasItem' has no associated type 'Nope'

fun main() {
  io.println("")
}
