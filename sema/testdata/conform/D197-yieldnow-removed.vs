// D197: the prelude's yieldNow() is gone; `cede` says it. A program's own
// yieldNow would be its own (sema/cede_test.go).
fun main() {
  yieldNow()          // error: 'yieldNow()' was removed (D197): write the statement 'cede'
  val x = yieldNow()  // error: 'cede' is a statement and has no value
}
