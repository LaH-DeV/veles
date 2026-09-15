use io

struct ParseError { text: string }
struct RangeError { value: i64 }

fun parsePort(text: string): i64 throws ParseError | RangeError {
  val n = text.toInt() ?: return Err(ParseError(text: text))
  if (n < 0 || n > 65535) return Err(RangeError(value: n))
  n
}

// inferred error union: ParseError | RangeError
fun loadConfig(text: string): i64 throws {
  val port = try parsePort(text)
  port * 2
}

fun main() throws {
  loop (t in ["80", "x", "70000"]) {
    val r = loadConfig(t)
    when (r) {
      is Ok(value) => io.println("$t -> $value")
      is Err(error) => when (error) {
        is ParseError(text) => io.println("$t -> parse error on '$text'")
        is RangeError(value) => io.println("$t -> $value out of range")
      }
    }
  }
  val doubled = try loadConfig("21")
  io.println("doubled $doubled")
  val fail = try loadConfig("nope")
  io.println("unreachable $fail")
}
