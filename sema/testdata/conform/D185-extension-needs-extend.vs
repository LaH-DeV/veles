// D185: a use brings the extension functions of its module (D173) only with
// an `extend` clause; without it the method is missing, and the error says how
// to import it.
use io
use regex { Regex }

fun main() {
  val re = Regex.compile("a+") ?? return
  io.println("${"banana".findAll(re)}") // error: 'findAll' is an extension function in module 'regex': import it with 'use regex extend { findAll }'
}
