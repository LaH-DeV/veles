// D4: an `error` declaration implements the prelude's Error trait whatever
// the module calls its own types, so a module may declare its own `Error`
// (`config.Error` is one) without the declaration implementing itself.
use io

error Error {
  message: string
}

fun risky(n: i64): i64 throws Error {
  if (n < 0) throw Error(message: "negative")
  n
}

fun main() {
  io.println("${risky(1) catch (e) { 0 }}")
}
