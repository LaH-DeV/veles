// D182: a /…/ literal needs `use regex` in the file.
use io

fun main() {
  val digits = /\d+/ // error: needs the 'regex' module
  io.println("${digits}")
}
