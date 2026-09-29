// D98: what is refused about the syntax of `do { ... } catch (e) { ... }`.
use io

fun noCatch() {
  do {
    io.println("x")
  } // error: a 'do' block needs a 'catch (e) { ... }' after it
  io.println("after")
}

fun doWhile() {
  var i = 0
  do {
    i += 1
  } while (i < 3) // error: there is no do-while
}

fun strayCatch() {
  catch (e) { // error: 'catch' follows an expression
    io.println("x")
  }
}

fun oldHandlers() {
  val a = g() ?? { e => 0 } // error: a handler that sees the error is 'catch (e) { ... }' now
  val b = g() else { e => return } // error: a handler that sees the error is 'catch (e) { ... }' now
}

fun oldCatchHead() {
  do {
    g()
  } catch { e => // error: the error is named in a head
    0
  }
}

fun main() {
  noCatch()
}
