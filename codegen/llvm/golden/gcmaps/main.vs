// The collector's pointer maps of tagged values are exact (B32): a sealed
// value lists the union of its variants' pointer words, an error union its
// members', so `Number`'s two integers are never scanned as addresses.
use io

sealed trait Token { }
struct Number : Token {
  value: i64
  at:    i64
}
struct Word : Token {
  at:   i64
  text: string
}
struct Names : Token {
  names: List<string>
}

error Overflow {
  limit: i64
  got:   i64
}
error Malformed {
  reason: string
}
error ParseError = Overflow | Malformed
error Failure {
  cause: ParseError
}

fun weigh(t: Token): i64 => when (t) {
  is Number(value) => value
  is Word(_, text) => text.len()
  is Names(names)  => names.map(name => name.len()).sum()
}

fun limit(n: i64): i64 throws ParseError {
  if (n < 0) throw Malformed(reason: "negative")
  if (n > 1000) throw Overflow(limit: 1000, got: n)
  n
}

fun main() {
  val tokens: MutableList<Token> = []
  loop (i in 0..200) {
    tokens.push(Number(value: i, at: i))
    tokens.push(Word(at: i, text: "w${i}"))
    tokens.push(Names(names: ["ab", "c${i}"]))
  }
  io.println("${tokens.map(token => weigh(token)).sum()}")
  val failures: MutableList<Failure> = []
  loop (n in [5, -1, 5000]) {
    limit(n) catch (e) {
      when (e) {
        is Overflow  => io.println("overflow ${e.got}")
        is Malformed => io.println("malformed ${e.reason}")
      }
      failures.push(Failure(cause: e))
      continue
    }
    io.println("fine $n")
  }
  io.println("${failures.len()} failed")
}
