// D98: `do { ... } catch (e) { ... }` — several `try`s, one place that handles
// their failures, in a function that carries on.
use io

error Bad {
  message: string
}
error Worse {
  code: i64
}

fun parse(s: string): i64 throws Bad = try s.toInt() ?! Bad(message: "not a number: $s")

fun check(n: i64): i64 throws Worse {
  if (n < 0) throw Worse(code: n)
  n
}

// the block's value, or what the handler yields; the function is not `throws`
fun value(a: string): i64 {
  do {
    val n = try parse(a)
    try check(n)
  } catch (e) {
    -1
  }
}

// `e` is the union of the errors the block can raise
fun which(a: string): string {
  do {
    val n = try parse(a)
    try check(n)
    "fine"
  } catch (e) {
    when (e) {
      is Bad   => "bad: ${e.message()}"
      is Worse => "worse"
    }
  }
}

// as a statement, in a loop: `continue` and `break` are the program's own
fun total(lines: List<string>): i64 {
  var sum: i64 = 0
  loop (line in lines) {
    do {
      sum += try parse(line)
      if (sum > 100) break
    } catch (e) {
      io.println("skip: ${e.message()}")
      continue
    }
  }
  sum
}

// the handler may leave, and a `throw` in the block goes to it
fun leaves(a: string): i64 {
  val n = do {
    if (a.isEmpty()) throw Bad(message: "empty")
    try parse(a)
  } catch (e) {
    return -1
  }
  n * 2
}

// the handler is outside the block: what it throws is the function's
fun rethrows(a: string): i64 throws Bad {
  do {
    try parse(a)
  } catch (e) {
    throw Bad(message: "wrapped: ${e.message()}")
  }
}

// a block inside a handler, and the other way round
fun nested(a: string): i64 {
  do {
    val n = do {
      try parse(a)
    } catch (e) {
      throw Worse(code: 1)
    }
    n
  } catch (e) {
    0
  }
}

fun word(s: string): string throws Bad {
  if (s.isEmpty()) throw Bad(message: "empty")
  s
}

// one call: what `r ?? { e => }` was
fun oneCall(a: string): i64 {
  parse(a) catch (e) {
    -1
  }
}

// one call and the methods after it: `try` marks the call that can fail,
// `catch` covers the whole chain, so the handler yields the chain's type
fun chain(a: string): i64 {
  try word(a).len() catch (e) {
    0
  }
}

// the binding is optional, and the `catch` may start the next line
fun ignoring(a: string): i64 {
  parse(a) catch {
    0
  }
}

fun leavesFromCatch(a: string): i64 {
  val n = parse(a) catch (e) {
    return -1
  }
  n * 2
}

fun onNullable(): i64 {
  "5".toInt() catch (e) {  // error: 'catch' handles the error of a Result, and a nullable has none
    0
  }
}

fun onPlainValue(): i64 {
  5 catch {  // error: 'catch' needs a Result before it
    0
  }
}

fun nothingFails() {
  do {  // error: nothing in this 'do' block can fail
    io.println("x")
  } catch (e) {
    io.println("y")
  }
}

fun wrongHandlerType() {
  val n: i64 = do {
    try parse("1")
  } catch (e) {
    "text"  // error: expected 'i64', found 'string'
  }
  io.println("$n")
}

fun outsideStillNeedsThrows() {
  val n = try parse("1")  // error: 'try' propagates an error, but the enclosing function is not declared 'throws'
  io.println("$n")
}

fun handlerThrowLeavesTheFunction() {
  do {
    io.println("${try parse("1")}")
  } catch (e) {
    throw Bad(message: "x")  // error: 'throw' fails the function, but it is not declared 'throws'
  }
}

fun scopeInside() {
  do {
    scope {
      async parse("1")  // error: a child that can fail cannot be launched in a 'scope' inside a 'do' block
    }
  } catch (e) {
    io.println("failed")
  }
}

fun main() {
  io.println("${value("3")} ${which("x")} ${total(["1", "2"])} ${leaves("5")} ${nested("q")}")
}
