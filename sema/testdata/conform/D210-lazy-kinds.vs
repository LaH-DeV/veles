// D210: `lazy` goes on a `val`. A `lazy var` would change after it is
// built; a `lazy const` mixes a value computed while compiling with one
// built at run time.
use io

lazy var counter = 0  // error: it is a 'lazy val', not a 'lazy var'
lazy const LIMIT = 4  // error: a 'const' is computed while compiling; 'lazy' is for a value built at run time

fun main() {
  io.println("${counter} ${LIMIT}")
}
