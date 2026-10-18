// D173: `extend T with fun f() …` (or `extend T { }`) on a type the package does
// not own is an extension function: a statically resolved call, visible where
// its module is imported. An inherent method always wins.
use io

extend string with fun shout(): string => this.toUpper() + "!"

public extend string {
  fun slug(): string => this.toLower().replace(" ", "-")
}

extend<T: Comparable> List<T> with fun isSorted(): bool => this.sorted() == this

extend string with fun trim(): string => this // warning: extension 'trim' on 'string' is never called
extend string with static fun make(): string => "" // error: an extension on the foreign type 'string' cannot declare a static function
extend string with private fun hid(): string => "" // error: 'private' belongs to the members of a struct
extend<T, E> Result<T, E> with fun f(): i64 => 1 // error: cannot extend 'Result<T, E>': it is built in
extend *i64 with fun g(): i64 => 1 // error: cannot extend '*i64': only named types can be extended

fun main() {
  io.println(" Hi There ".trim().slug().shout())
  io.println("${[1, 2].isSorted()}")
}
