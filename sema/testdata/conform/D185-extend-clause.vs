// D185: the `extend` clause of a use is checked: the module must have
// extension functions, and each one named must be one of them.
use io
use path extend // error: module 'path' has no extension functions: drop 'extend'
use regex { Regex } extend { fnd } // error: module 'regex' has no extension function 'fnd'; did you mean 'find'?

fun main() {
  io.println("${Regex.escape("a.b")}")
}
