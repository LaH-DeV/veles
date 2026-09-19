use io

/// The text could not be read as a port number.
/// (`error` declares a struct that is an Error, D4: only these can be thrown.)
error ParseError {
  /// what we tried to parse
  text: string
  fun message(): string = "parsing problem in \"${self.text}\""
}
/// A number outside 0..65535. Its message() is the default: the show rendering.
error RangeError {
  value: i64
}
error Refused {
  message: string
}  // a `message` field is the message

error PortErrors = ParseError | RangeError | Refused  // a named error set (D45)

fun parsePort(text: string): i64 throws PortErrors {
  if (text == "22") throw Refused(message: "port 22 is reserved")
  val n = text.toInt() ?: throw ParseError(text)
  if (n < 0 || n > 65535) throw RangeError(value: n)
  n
}

/** Context wrapped around a cause: an error's field may hold an error set. */
error ConfigError {
  key:   string
  cause: PortErrors
  fun message(): string = "config '${self.key}': ${self.cause.message()}"
}

/** Codedoc */
fun loadConfig(key: string, text: string): i64 throws ConfigError {
  val port = parsePort(text)
  if (port is Err) throw ConfigError(key, cause: port)
  port * 2
}

fun main() throws {
  loop (setting in ["80", "22", "x", "70000"]) {
    val config = loadConfig("port", setting)
    // `is Ok` smart-casts config to the i64 inside; in the else branch it is the error
    if (config is Ok) {
      io.println("$setting -> $config")
    } else {
      io.println("$setting -> ${config.message()}")  // no need to know the fields
      when (config.cause) {                          // a field path narrows too (D5)
        is RangeError => io.println("  ${config.cause.value} is outside 0..65535")
        else          => io.println("  (not a range problem)")
      }
    }
  }
  val doubled = try loadConfig("port", "21")
  io.println("doubled $doubled")
  val fail = try loadConfig("port", "nope")
  io.println("unreachable $fail")
}
