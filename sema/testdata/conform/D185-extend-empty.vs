// D185: `extend { }` names nothing: write `extend` alone for all of them.
use io
use regex extend { } // error: 'extend { }' names nothing

fun main() {
  io.println("x")
}
