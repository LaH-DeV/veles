// Integers and floats: pure functions from the text of a word to a value.

/// The word is not written as a number.
error Unreadable { }

/// The word is an integer that does not fit 64 bits.
error OutOfRange { }

error NumberProblem = Unreadable | OutOfRange

const INT_MIN: i64 = -9223372036854775807 - 1

fun infinity(): f64 => f64.fromBits(0x7FF0000000000000)

fun notANumber(): f64 => f64.fromBits(0x7FF8000000000000)

/// A decimal integer: an optional sign and digits, in the range of an i64.
fun parseDecimal(word: string): i64? {
  val negative = word.startsWith("-")
  val digits = if (negative || word.startsWith("+")) word.slice(1, word.len()) else word
  if (digits.isEmpty()) return null
  var value: i64 = 0
  loop (i in 0..<digits.len()) {
    val digit = digits.byteAt(i).digitValue() ?: return null
    // accumulate downwards, so the lowest i64 fits
    if (value < (INT_MIN + digit) / 10) return null
    value = value * 10 - digit
  }
  if (negative) value else if (value == INT_MIN) null else -value
}

/// Digits in `radix` and nothing else, up to 2^63 - 1.
fun parseRadix(digits: string, radix: i64): i64? {
  if (digits.isEmpty()) return null
  var value: i64 = 0
  loop (i in 0..<digits.len()) {
    val digit = digits.byteAt(i).digitValue(radix) ?: return null
    if (value > (9223372036854775807 - digit) / radix) return null
    value = value * radix + digit
  }
  value
}

/// `text` without its underscores, which may only stand between two digits;
/// null where one does not.
fun withoutUnderscores(text: string): string? {
  if (text.isEmpty() || text.startsWith("_") || text.endsWith("_")) return null
  var previous: u8 = 0
  loop (i in 0..<text.len()) {
    val byte = text.byteAt(i)
    val afterDigit = previous.isAsciiHexDigit()
    if ((byte == '_' && !afterDigit) || (previous == '_' && !byte.isAsciiHexDigit())) return null
    previous = byte
  }
  text.replace("_", "")
}

/// A float: digits, with a fraction or an exponent or both, or `inf`/`nan`;
/// null where the text is not one.
fun parseFloat(word: string): f64? {
  val negative = word.startsWith("-")
  val body = if (negative || word.startsWith("+")) word.slice(1, word.len()) else word
  if (body == "inf") return if (negative) -infinity() else infinity()
  if (body == "nan") return notANumber()
  val parts = FloatParts.of(body) ?: return null
  val clean = parts.cleaned() ?: return null
  val value = clean.toF64() ?: return null
  if (negative) -value else value
}

/// Walks digits and underscores in a word.
struct DigitScanner {
  text:   string
  var at: i64 = 0

  fun more(): bool => this.at < this.text.len()

  fun peek(): u8 => this.text.byteAt(this.at)

  /// The digits and underscores at the position.
  fun digits(): string {
    val start = this.at
    loop (this.more() && (this.peek().isAsciiDigit() || this.peek() == '_')) this.at += 1
    this.text.slice(start, this.at)
  }

  /// A `+` or `-` at the position, consumed; "" when there is none.
  fun sign(): string {
    if (!this.more() || (this.peek() != '+' && this.peek() != '-')) return ""
    this.at += 1
    this.text.slice(this.at - 1, this.at)
  }

  fun eat(byte: u8): bool {
    if (this.more() && this.peek() == byte) {
      this.at += 1
      return true
    }
    false
  }
}

/// The pieces of `digits[.digits][e[sign]digits]`, with no leading zero before
/// the point.
struct FloatParts {
  whole:    string
  fraction: string
  exponent: string

  /// Splits `body`, or null when it is not written that way.
  static fun of(body: string): FloatParts? {
    var scan = DigitScanner(text: body)
    val whole = scan.digits()
    if (whole.isEmpty() || (whole.len() > 1 && whole.startsWith("0"))) return null
    var fraction = ""
    if (scan.eat('.')) {
      fraction = scan.digits()
      if (fraction.isEmpty()) return null
    }
    var exponent = ""
    if (scan.eat('e') || scan.eat('E')) {
      val sign = scan.sign()
      val digits = scan.digits()
      if (digits.isEmpty()) return null
      exponent = sign + digits
    }
    if (scan.more() || (fraction.isEmpty() && exponent.isEmpty())) return null
    FloatParts(whole, fraction, exponent)
  }

  /// The number as `1234.5e6` for the machine reader, or null where an
  /// underscore sits wrong.
  fun cleaned(): string? {
    val whole = withoutUnderscores(this.whole) ?: return null
    val fraction = if (this.fraction.isEmpty()) "" else "." + (withoutUnderscores(this.fraction) ?: return null)
    val exponent = if (this.exponent.isEmpty()) "" else "e" + (withoutUnderscores(this.exponent) ?: return null)
    whole + fraction + exponent
  }
}

/// A number as TOML 1.0 writes it: decimal, `0x…`, `0o…`, `0b…`, a float, `inf`
/// or `nan`.
fun readNumber(word: string): Data throws NumberProblem {
  val sign: u8 = if (word.startsWith("+") || word.startsWith("-")) word.byteAt(0) else 0
  val body = if (sign == 0) word else word.slice(1, word.len())
  if (body in ["inf", "nan"]) return Real(value: parseFloat(word) ?: throw Unreadable())
  if (sign == 0 && body.len() > 2 && body.byteAt(0) == '0') {
    val radix: i64 = when (body.byteAt(1)) {
      'x'  => 16
      'o'  => 8
      'b'  => 2
      else => 0
    }
    if (radix != 0) {
      val digits = withoutUnderscores(body.slice(2, body.len())) ?: throw Unreadable()
      return Integer(value: parseRadix(digits, radix) ?: throw OutOfRange())
    }
  }
  if (body.isEmpty() || !body.byteAt(0).isAsciiDigit()) throw Unreadable()
  if (body.indexOf(".") >= 0 || body.indexOf("e") >= 0 || body.indexOf("E") >= 0) {
    return Real(value: parseFloat(word) ?: throw Unreadable())
  }
  val digits = withoutUnderscores(body) ?: throw Unreadable()
  loop (i in 0..<digits.len()) {
    if (!digits.byteAt(i).isAsciiDigit()) throw Unreadable()
  }
  if (digits.len() > 1 && digits.startsWith("0")) throw Unreadable()
  Integer(value: parseDecimal((if (sign == '-') "-" else "") + digits) ?: throw OutOfRange())
}
