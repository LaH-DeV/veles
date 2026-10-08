// catchblock: `do { ... } catch (e) { ... }` (D98) gives the failed `try`s and
// `throw`s of a block one place to go, in a function that carries on. `try`
// still marks every call that can fail; the block is not a lambda, so
// `return`, `break` and `continue` leave the function or loop around it.
use io

error Bad {
  message: string
}
error Worse {
  code: i64
}

fun parse(s: string): i64 throws Bad => try s.toInt() ?! Bad(message: "not a number: $s")

fun check(n: i64): i64 throws Worse {
  if (n < 0) throw Worse(code: n)
  n
}

fun slowParse(s: string): i64 suspends throws Bad {
  await sleep(Duration.millis(1))
  try parse(s)
}

struct Door {
  name: string

  implement Closeable {
    fun close() {
      io.println("close ${this.name}")
    }
  }
}

// the block's value, or what the handler yields; `e` is the union of the errors
fun sum(a: string, b: string): string {
  do {
    val x = try parse(a)
    val y = try check(try parse(b))
    "${x + y}"
  } catch (e) {
    "failed: ${e.message()}"
  }
}

fun which(a: string): string {
  do {
    try check(try parse(a))
    "fine"
  } catch (e) {
    when (e) {
      is Bad   => "bad input"
      is Worse => "negative"
    }
  }
}

// in a loop, `continue` and `break` are the program's own
fun total(lines: List<string>): i64 {
  var sum: i64 = 0
  loop (line in lines) {
    do {
      val a = try parse(line)
      val b = try check(a)
      if (b > 100) break
      sum += b
    } catch (e) {
      io.println("skip: ${e.message()}")
      continue
    }
  }
  sum
}

// a `throw` in the block goes to the handler, like a failed `try`
fun viaThrow(n: i64): string => do {
  if (n == 0) throw Bad(message: "zero")
  "ok $n"
} catch (e) {
  "caught ${e.message()}"
}

// the handler may leave the function
fun early(a: string): i64 {
  val n = do {
    try parse(a)
  } catch (e) {
    return -1
  }
  n * 2
}

// the handler is outside the block: what it throws is the function's
fun rethrow(a: string): i64 throws Bad {
  do {
    try parse(a)
  } catch (e) {
    throw Bad(message: "wrapped: ${e.message()}")
  }
}

// a `with` left by a failing `try` still closes
fun guarded(s: string): string {
  do {
    with d = Door(name: "d1")
    val n = try parse(s)
    "opened ${d.name} got $n"
  } catch (e) {
    "handled: ${e.message()}"
  }
}

// calls that suspend need nothing declared
fun waits(s: string): string {
  do {
    val a = try slowParse(s)
    val b = try slowParse("10")
    "sum ${a + b}"
  } catch (e) {
    "async failed: ${e.message()}"
  }
}

// `break outer` from inside the block leaves the outer loop
fun labelled(rows: List<List<string>>): i64 {
  var sum: i64 = 0
  loop :outer (row in rows) {
    loop (cell in row) {
      do {
        sum += try parse(cell)
      } catch (e) {
        if (cell == "stop") break outer
        continue
      }
    }
  }
  sum
}

// a lambda in the block has its own `try`; calling it is one more
fun withLambda(xs: List<string>): string {
  do {
    val f = (s: string) => try parse(s)
    var total: i64 = 0
    loop (x in xs) {
      total += try f(x)
    }
    "total $total"
  } catch (e) {
    "lambda: ${e.message()}"
  }
}

// a block inside a block, the inner handler failing to the outer one
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

// one fallible call and the methods after it: `try` marks the call that can
// fail, and `catch` covers the whole chain
fun width(s: string): i64 {
  try word(s).len() catch (e) {
    -1
  }
}

fun nullable(s: string): i64? {
  val r: i64? = do {
    try parse(s)
  } catch (e) {
    null
  }
  r
}

fun main() {
  io.println(sum("1", "2"))
  io.println(sum("x", "2"))
  io.println(sum("1", "-4"))
  io.println("${which("3")} ${which("x")} ${which("-1")}")
  io.println("${total(["1", "x", "-3", "5", "500", "7"])}")
  io.println(viaThrow(0))
  io.println(viaThrow(3))
  io.println("${early("21")} ${early("z")}")
  io.println("${rethrow("4") catch (e) { 0 }} ${rethrow("q") catch (e) { -9 }}")
  io.println(guarded("5"))
  io.println(guarded("x"))
  io.println(waits("1"))
  io.println(waits("no"))
  io.println("${labelled([["1", "2"], ["a", "3"], ["stop", "9"], ["100"]])}")
  io.println(withLambda(["1", "2"]))
  io.println(withLambda(["1", "z"]))
  io.println("${nested("5")} ${nested("q")}")
  io.println("${nullable("7")} ${nullable("q")}")
  io.println("${width("abc")} ${width("")}")
}
