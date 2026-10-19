use utf8

/// TOML 1.0, read into a tree that remembers the line of every value, so a
/// program reading a configuration can say *where* it is wrong.
///
/// ```veles
/// use toml
///
/// val doc = try toml.parse(text)
/// val server = doc.get("server")?.asTable() ?: return
/// val port = server.get("port")?.asInt() ?: 8080
/// ```
///
/// `parse` reads the whole language: tables, array-of-tables, dotted keys,
/// basic and literal strings (single- and multi-line), integers (decimal,
/// hex, octal, binary), floats, booleans, dates and times, arrays and inline
/// tables. `parse(text, strict: true)` reads the subset a package manifest
/// uses — no floats, dates or `[[tables]]`, decimal integers, a few escapes —
/// and refuses the rest with a message, so a `veles.toml` cannot mean one thing
/// here and another elsewhere.
///
/// Dates and times are kept as the text that was written (`Value.str`), checked
/// for form and range; `Kind` says which of the four shapes TOML has.

/// What a value is.
public enum Kind {
  Str
  Int
  Float
  Bool
  OffsetDateTime
  LocalDateTime
  LocalDate
  LocalTime
  List
  Table
}

/// How a message names a kind of value.
public fun describe(kind: Kind): string => when (kind) {
  Kind.Str            => "a string"
  Kind.Int            => "a number"
  Kind.Float          => "a number"
  Kind.Bool           => "true or false"
  Kind.OffsetDateTime => "a date and time"
  Kind.LocalDateTime  => "a date and time"
  Kind.LocalDate      => "a date"
  Kind.LocalTime      => "a time"
  Kind.List           => "a list"
  Kind.Table          => "a table"
}

/// One value, with the line it was written on. Only the field of its kind
/// means anything; `as…` read it as the kind it must be.
public struct Value {
  public kind: Kind
  public line: i64
  /// A string's text; a date's or time's as written.
  public str:   string = ""
  public num:   i64 = 0
  public real:  f64 = 0.0
  public flag:  bool = false
  public items: List<Value> = []
  public table: (*Table)? = null
  /// A list made by `[[name]]` headers: it grows with the next header, an
  /// inline `[ … ]` does not.
  public tables: bool = false

  /// The text of a string value.
  public fun asString(): string? => if (this.kind == Kind.Str) this.str else null

  /// The value of an integer.
  public fun asInt(): i64? => if (this.kind == Kind.Int) this.num else null

  /// The value of a float; an integer reads as a float too.
  public fun asFloat(): f64? => when (this.kind) {
    Kind.Float => this.real
    Kind.Int   => this.num.toF64()
    else       => null
  }

  public fun asBool(): bool? => if (this.kind == Kind.Bool) this.flag else null

  public fun asList(): List<Value>? => if (this.kind == Kind.List) this.items else null

  public fun asTable(): (*Table)? => if (this.kind == Kind.Table) this.table else null
}

/// Where a table came from. A `[header]` or an inline table *defines* it; a
/// table only on the way to one (`[a.b]` makes `a`) or made by a dotted key
/// (`a.b = 1`) can still be defined later.
enum Origin {
  Implied
  Dotted
  Header
  Inline
}

/// A table of keys to values.
public struct Table {
  internal vals:       MutableMap<string, Value> = [:]
  public line:         i64
  internal var origin: Origin = Origin.Implied

  /// The value under `key`.
  public fun get(key: string): Value? => this.vals.get(key)

  /// True when `key` has a value.
  public fun has(key: string): bool => this.vals.containsKey(key)

  /// The keys, in a fixed (sorted) order.
  public fun keys(): List<string> => this.vals.keys().sorted()

  /// True when a `[header]` or an inline table named it, not when it is only
  /// the parent of one.
  public fun defined(): bool => this.origin == Origin.Header || this.origin == Origin.Inline
}

/// The one thing wrong with a document: the line and what.
public error ParseError {
  public line:    i64
  public message: string
}

/// Reads `text` into its root table. `strict` reads the subset a package
/// manifest uses (see the module's description).
public fun parse(text: string, strict: bool = false): *Table throws ParseError {
  var src = text.replace("\r\n", "\n")
  if (src.startsWith("\u{FEFF}")) src = src.substring(3, src.len()) ?: ""  // a byte-order mark
  var reader = Reader(src, strict, line: 1)
  try reader.run()
}

/// The code point `code` as text.
fun runeText(code: i64): string {
  val bytes: MutableList<u8> = []
  utf8.encodeTo(bytes, code)
  bytes.decodeUtf8() ?: panic("runeText: encodeTo writes valid UTF-8")
}

fun isKeyByte(b: u8): bool =>
  (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'

fun isDigit(b: u8): bool => b >= '0' && b <= '9'

/// Where a value ends when it is none of the quoted kinds.
fun endsWord(b: u8): bool =>
  b == ' ' || b == '\t' || b == '\n' || b == ',' || b == ']' || b == '}' || b == '#'

/// Quotes `text` the way a message shows a key or a word: `"a b"`, with the
/// backslash, the quote and control characters escaped.
fun quoted(text: string): string {
  val out = StringBuilder()
  out.append("\"")
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i)
    when (b) {
      '"'  => out.append("\\\"")
      '\\' => out.append("\\\\")
      '\n' => out.append("\\n")
      '\t' => out.append("\\t")
      '\r' => out.append("\\r")
      else => {
        if (b < 0x20 || b == 0x7F) out.append("\\x${b.toI64().toString(radix: 16).padStart(2, "0")}") else out.appendByte(b)
      }
    }
  }
  out.append("\"")
  out.toString()
}

const INT_MIN: i64 = -9223372036854775807 - 1
fun infinity(): f64 => f64.fromBits(0x7FF0000000000000)

fun notANumber(): f64 => f64.fromBits(0x7FF8000000000000)

/// A decimal integer: an optional sign and digits, in the range of an i64.
fun parseDecimal(word: string): i64? {
  var digits = word
  var negative = false
  if (digits.startsWith("-")) {
    negative = true
    digits = digits.substring(1, digits.len()) ?: ""
  } else if (digits.startsWith("+")) {
    digits = digits.substring(1, digits.len()) ?: ""
  }
  if (digits.isEmpty()) return null
  var value: i64 = 0
  loop (i in 0..<digits.len()) {
    val b = digits.byteAt(i)
    if (b < '0' || b > '9') return null
    val digit = (b - '0').toI64()
    // accumulate downwards, so i64's lowest value fits
    if (value < (INT_MIN + digit) / 10) return null
    value = value * 10 - digit
  }
  if (negative) value else if (value == INT_MIN) null else -value
}

/// The value of a digit in `radix`, or -1.
fun digitValue(b: u8, radix: i64): i64 {
  val v: i64 = when {
    b >= '0' && b <= '9' => (b - '0').toI64()
    b >= 'a' && b <= 'f' => (b - 'a').toI64() + 10
    b >= 'A' && b <= 'F' => (b - 'A').toI64() + 10
    else                 => return -1
  }
  if (v < radix) v else -1
}

/// Digits in `radix` and nothing else, as a number up to 2^63 - 1.
fun parseRadix(text: string, radix: i64): i64? {
  if (text.isEmpty()) return null
  var value: i64 = 0
  loop (i in 0..<text.len()) {
    val digit = digitValue(text.byteAt(i), radix)
    if (digit < 0) return null
    if (value > (9223372036854775807 - digit) / radix) return null
    value = value * radix + digit
  }
  value
}

/// Four hex digits, as strconv.ParseUint(text, 16, 32) reads them: nothing else.
fun parseHex(text: string): i64? => if (text.isEmpty()) null else parseRadix(text, 16)

/// Underscores sit between digits only; a text without them is returned as is.
fun withoutUnderscores(text: string): string? {
  if (text.byteAt(0) == '_' || text.byteAt(text.len() - 1) == '_') return null
  var previous: u8 = 0
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i)
    if (b == '_' && !(isDigit(previous) || digitValue(previous, 16) >= 0)) return null
    if (previous == '_' && digitValue(b, 16) < 0) return null
    previous = b
  }
  text.replace("_", "")
}

fun daysIn(month: i64, year: i64): i64 {
  if (month == 2) return if ((year % 4 == 0 && year % 100 != 0) || year % 400 == 0) 29 else 28
  if (month == 4 || month == 6 || month == 9 || month == 11) 30 else 31
}

/// Two or four digits at `at`, as a number, or -1.
fun digitsAt(text: string, at: i64, count: i64): i64 {
  if (at + count > text.len()) return -1
  var value: i64 = 0
  loop (i in 0..<count) {
    val b = text.byteAt(at + i)
    if (!isDigit(b)) return -1
    value = value * 10 + (b - '0').toI64()
  }
  value
}

/// `HH:MM:SS[.frac]` at `at`: where it ends, or -1.
fun timeEnd(text: string, at: i64): i64 {
  val hour = digitsAt(text, at, 2)
  val minute = digitsAt(text, at + 3, 2)
  val second = digitsAt(text, at + 6, 2)
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 60) return -1
  if (at + 2 >= text.len() || text.byteAt(at + 2) != ':' || text.byteAt(at + 5) != ':') return -1
  var end = at + 8
  if (end < text.len() && text.byteAt(end) == '.') {
    end += 1
    val start = end
    loop (end < text.len() && isDigit(text.byteAt(end))) end += 1
    if (end == start) return -1
  }
  end
}

/// The kind of date or time `word` is, or null when it is none. Checked for
/// form and range.
fun dateKind(word: string): Kind? {
  val year = digitsAt(word, 0, 4)
  val isDate = year >= 0 && word.len() >= 10 && word.byteAt(4) == '-' && word.byteAt(7) == '-'
  if (!isDate) {
    // a bare time
    if (word.len() >= 8 && word.byteAt(2) == ':' && timeEnd(word, 0) == word.len()) return Kind.LocalTime
    return null
  }
  val month = digitsAt(word, 5, 2)
  val day = digitsAt(word, 8, 2)
  if (month < 1 || month > 12 || day < 1 || day > daysIn(month, year)) return null
  if (word.len() == 10) return Kind.LocalDate
  val sep = word.byteAt(10)
  if (sep != 'T' && sep != 't' && sep != ' ') return null
  val end = timeEnd(word, 11)
  if (end < 0) return null
  if (end == word.len()) return Kind.LocalDateTime
  val zone = word.byteAt(end)
  if ((zone == 'Z' || zone == 'z') && end + 1 == word.len()) return Kind.OffsetDateTime
  if ((zone == '+' || zone == '-') && end + 6 == word.len() && word.byteAt(end + 3) == ':') {
    val hours = digitsAt(word, end + 1, 2)
    val minutes = digitsAt(word, end + 4, 2)
    if (hours >= 0 && hours <= 23 && minutes >= 0 && minutes <= 59) return Kind.OffsetDateTime
  }
  null
}

/// A float written with a fraction or an exponent (or `inf`, `nan`), or null.
fun parseFloat(word: string): f64? {
  var body = word
  var negative = false
  if (body.startsWith("-")) {
    negative = true
    body = body.substring(1, body.len()) ?: ""
  } else if (body.startsWith("+")) {
    body = body.substring(1, body.len()) ?: ""
  }
  if (body == "inf") return if (negative) -infinity() else infinity()
  if (body == "nan") return notANumber()
  // digits [. digits] [e [sign] digits], no leading zeros, underscores between digits
  var i: i64 = 0
  val n = body.len()
  val intStart = i
  loop (i < n && (isDigit(body.byteAt(i)) || body.byteAt(i) == '_')) i += 1
  val intPart = body.substring(intStart, i) ?: ""
  if (intPart.isEmpty() || (intPart.len() > 1 && intPart.byteAt(0) == '0')) return null
  var fraction = ""
  if (i < n && body.byteAt(i) == '.') {
    i += 1
    val start = i
    loop (i < n && (isDigit(body.byteAt(i)) || body.byteAt(i) == '_')) i += 1
    fraction = body.substring(start, i) ?: ""
    if (fraction.isEmpty()) return null
  }
  var exponent = ""
  if (i < n && (body.byteAt(i) == 'e' || body.byteAt(i) == 'E')) {
    i += 1
    val start = i
    if (i < n && (body.byteAt(i) == '+' || body.byteAt(i) == '-')) i += 1
    val digitsFrom = i
    loop (i < n && (isDigit(body.byteAt(i)) || body.byteAt(i) == '_')) i += 1
    if (i == digitsFrom) return null
    exponent = body.substring(start, i) ?: ""
  }
  if (i != n || (fraction.isEmpty() && exponent.isEmpty())) return null
  val intClean = withoutUnderscores(intPart) ?: return null
  val fractionClean = if (fraction.isEmpty()) "" else (withoutUnderscores(fraction) ?: return null)
  val exponentClean = if (exponent.isEmpty()) "" else (withoutUnderscores(exponent) ?: return null)
  val text = intClean + (if (fractionClean.isEmpty()) "" else ".$fractionClean") + (if (exponentClean.isEmpty()) "" else "e$exponentClean")
  val value = text.toF64() ?: return null
  if (negative) -value else value
}

struct Reader {
  src:      string
  strict:   bool
  var pos:  i64 = 0
  var line: i64

  fun more(): bool => this.pos < this.src.len()

  fun fail(message: string): ParseError => ParseError(line: this.line, message)

  fun eat(c: u8): bool {
    if (this.more() && this.src.byteAt(this.pos) == c) {
      this.pos += 1
      return true
    }
    false
  }

  /// True when the text at `pos` begins with the byte `c` repeated `count` times.
  fun lookingAt(c: u8, count: i64): bool {
    if (this.pos + count > this.src.len()) return false
    loop (i in 0..<count) {
      if (this.src.byteAt(this.pos + i) != c) return false
    }
    true
  }

  /// Skips blanks and comments; newlines too when `newlines` is set.
  fun skipSpace(newlines: bool) {
    loop (this.more()) {
      val c = this.src.byteAt(this.pos)
      if (c == ' ' || c == '\t') {
        this.pos += 1
      } else if (c == '#') {
        loop (this.more() && this.src.byteAt(this.pos) != '\n') this.pos += 1
      } else if (c == '\n' && newlines) {
        this.pos += 1
        this.line += 1
      } else {
        return
      }
    }
  }

  /// Nothing but a comment may stand before the next newline.
  fun endOfLine() throws ParseError {
    this.skipSpace(false)
    if (this.more() && this.src.byteAt(this.pos) != '\n') throw this.fail("unexpected text after the value")
  }

  fun run(): *Table throws ParseError {
    val root = &Table(line: 1)
    var current = root
    loop {
      this.skipSpace(true)
      if (!this.more()) return root
      if (this.src.byteAt(this.pos) == '[') {
        val isArray = this.lookingAt('[', 2)
        if (isArray && this.strict) throw this.fail("arrays of tables ([[...]]) are not supported in veles.toml")
        this.pos += if (isArray) 2 else 1
        val path: MutableList<string> = []
        loop {
          this.skipSpace(false)
          path.push(try this.key())
          this.skipSpace(false)
          if (this.eat('.')) continue
          break
        }
        if (isArray) {
          if (!this.eat(']') || !this.eat(']')) throw this.fail("expected ']]' to close the array of tables name")
          current = try this.arrayTable(root, path.toList())
        } else {
          if (!this.eat(']')) throw this.fail("expected ']' to close the table name")
          current = try this.table(root, path.toList())
        }
        try this.endOfLine()
        continue
      }
      var key = try this.key()
      val line = this.line
      this.skipSpace(false)
      // a dotted key (`a.b = 1`) is the key of a table below the current one
      var dest = current
      loop (this.more() && this.src.byteAt(this.pos) == '.') {
        this.pos += 1
        this.skipSpace(false)
        dest = try this.dotted(dest, key, line)
        key = try this.key()
        this.skipSpace(false)
      }
      if (!this.eat('=')) throw this.fail("expected 'key = value'")
      this.skipSpace(false)
      val value = try this.value()
      if (dest.vals.containsKey(key)) throw ParseError(line, message: "key ${quoted(key)} is set twice")
      dest.vals.set(key, value)
      try this.endOfLine()
    }
  }

  /// The table below `parent` under `key`, made with `origin` when it is not
  /// there yet.
  fun child(parent: *Table, key: string, line: i64, origin: Origin): *Table throws ParseError {
    if (val existing = parent.vals.get(key)) {
      return existing.table ?: throw this.fail("${quoted(key)} is already ${describe(existing.kind)}, not a table")
    }
    val made = &Table(line, origin)
    parent.vals.set(key, Value(kind: Kind.Table, line, table: made))
    made
  }

  /// A table on the way of a dotted key. The full language lets a dotted key
  /// reach only tables that no header or inline table has defined.
  fun dotted(parent: *Table, key: string, line: i64): *Table throws ParseError {
    val below = try this.child(parent, key, line, Origin.Dotted)
    if (!this.strict && (below.origin == Origin.Header || below.origin == Origin.Inline)) {
      throw this.fail("cannot add to the table ${quoted(key)}: it is defined elsewhere")
    }
    below
  }

  /// Follows `path` below `root` up to its last key, through the last
  /// element of any array of tables on the way. Tables the path names are
  /// made when they are not there.
  fun walk(root: *Table, path: List<string>): *Table throws ParseError {
    var current = root
    var depth: i64 = 0
    loop (key in path.take(path.len() - 1)) {
      depth += 1
      if (val existing = current.vals.get(key)) {
        if (existing.kind == Kind.List && existing.tables) {
          val last = existing.items.last() ?: throw this.fail("${quoted(path.take(depth).join("."))} is an empty array of tables")
          current = last.table ?: throw this.fail("${quoted(path.take(depth).join("."))} is already ${describe(existing.kind)}, not a table")
          continue
        }
        val below = existing.table ?: throw this.fail("${quoted(path.take(depth).join("."))} is already ${describe(existing.kind)}, not a table")
        if (below.origin == Origin.Inline) throw this.fail("cannot add to the inline table ${quoted(path.take(depth).join("."))}")
        current = below
      } else {
        current = try this.child(current, key, this.line, Origin.Implied)
      }
    }
    current
  }

  /// Finds or makes the table at `path` below `root`. A table named twice is
  /// an error; one reached through a parent that is defined later is fine
  /// (`[a.b]` before `[a]`).
  fun table(root: *Table, path: List<string>): *Table throws ParseError {
    val parent = try this.walk(root, path)
    val key = path.last() ?: throw this.fail("expected a key")
    if (val existing = parent.vals.get(key)) {
      val found = existing.table ?: throw this.fail("${quoted(path.join("."))} is already ${describe(existing.kind)}, not a table")
      val redefines = found.origin == Origin.Header || found.origin == Origin.Inline || (!this.strict && found.origin == Origin.Dotted)
      if (redefines) throw this.fail("table [${path.join(".")}] is defined twice")
      found.origin = Origin.Header
      return found
    }
    try this.child(parent, key, this.line, Origin.Header)
  }

  /// `[[path]]`: a new table at the end of the array of tables there.
  fun arrayTable(root: *Table, path: List<string>): *Table throws ParseError {
    val parent = try this.walk(root, path)
    val key = path.last() ?: throw this.fail("expected a key")
    val made = &Table(line: this.line, origin: Origin.Header)
    val item = Value(kind: Kind.Table, line: this.line, table: made)
    if (val existing = parent.vals.get(key)) {
      if (existing.kind != Kind.List || !existing.tables) {
        throw this.fail("${quoted(path.join("."))} is already ${describe(existing.kind)}, not an array of tables")
      }
      val items: MutableList<Value> = []
      loop (old in existing.items) items.push(old)
      items.push(item)
      parent.vals.set(key, Value(kind: Kind.List, line: existing.line, items: items.toList(), tables: true))
    } else {
      parent.vals.set(key, Value(kind: Kind.List, line: this.line, items: [item], tables: true))
    }
    made
  }

  fun key(): string throws ParseError {
    if (this.more() && (this.src.byteAt(this.pos) == '"' || this.src.byteAt(this.pos) == '\'')) return try this.str()
    val start = this.pos
    loop (this.more() && isKeyByte(this.src.byteAt(this.pos))) this.pos += 1
    if (start == this.pos) throw this.fail("expected a key")
    this.src.substring(start, this.pos) ?: panic("key: the bytes of a key are ASCII")
  }

  /// A quoted string, the quote at `pos`; basic strings take escapes, literal
  /// ones do not. The full language also reads `"""…"""` and `'''…'''`.
  fun str(): string throws ParseError {
    val quoteByte = this.src.byteAt(this.pos)
    if (!this.strict && this.lookingAt(quoteByte, 3)) return try this.multiline(quoteByte)
    this.pos += 1
    val out = StringBuilder()
    loop (this.more()) {
      val c = this.src.byteAt(this.pos)
      if (c == '\n') throw this.fail("a string must close on its own line")
      if (c == quoteByte) {
        this.pos += 1
        return out.toString()
      }
      if (c == '\\' && quoteByte == '"') {
        this.pos += 1
        try this.escape(out)
        this.pos += 1
      } else {
        if (!this.strict && isControl(c) && c != '	') throw this.fail("a control character in a string must be escaped")
        out.appendByte(c)
        this.pos += 1
      }
    }
    throw this.fail("unterminated string")
  }

  /// The escape after a backslash, `pos` on its letter; leaves `pos` on its
  /// last byte.
  fun escape(out: StringBuilder) throws ParseError {
    if (!this.more()) throw this.fail("unfinished escape")
    val escape = this.src.byteAt(this.pos)
    when (escape) {
      'n'  => out.append("\n")
      't'  => out.append("\t")
      '"'  => out.append("\"")
      '\\' => out.append("\\")
      'u'  => out.append(try this.unicodeEscape(4))
      else => {
        if (!this.strict && escape == 'b') {
          out.append("\u{8}")
        } else if (!this.strict && escape == 'f') {
          out.append("\u{C}")
        } else if (!this.strict && escape == 'r') {
          out.append("\r")
        } else if (!this.strict && escape == 'e') {
          out.append("\u{1B}")
        } else if (!this.strict && escape == 'U') {
          out.append(try this.unicodeEscape(8))
        } else {
          throw this.fail("unknown escape \\${runeText(escape.toI64())}")
        }
      }
    }
  }

  /// `count` hex digits after the `u`/`U` at `pos`; leaves `pos` on the last.
  fun unicodeEscape(count: i64): string throws ParseError {
    if (this.pos + count >= this.src.len()) throw this.fail("unfinished \\u escape")
    val digits = this.src.substring(this.pos + 1, this.pos + 1 + count) ?: throw this.fail("bad \\u escape")
    val code = parseHex(digits) ?: throw this.fail("bad \\u escape")
    if (!utf8.isScalar(code)) throw this.fail("bad \\u escape")
    this.pos += count
    runeText(code)
  }

  /// `"""…"""` or `'''…'''`: a newline right after the opening is dropped; in
  /// a basic string a backslash at the end of a line drops the white space up
  /// to the next text.
  fun multiline(quoteByte: u8): string throws ParseError {
    this.pos += 3
    if (this.more() && this.src.byteAt(this.pos) == '\n') {
      this.pos += 1
      this.line += 1
    }
    val out = StringBuilder()
    loop (this.more()) {
      val c = this.src.byteAt(this.pos)
      if (c == quoteByte && this.lookingAt(quoteByte, 3)) {
        // up to two more quotes belong to the text
        var extra: i64 = 0
        loop (extra < 2 && this.pos + 3 + extra < this.src.len() && this.src.byteAt(this.pos + 3 + extra) == quoteByte) extra += 1
        loop (_ in 0..<extra) out.appendByte(quoteByte)
        this.pos += 3 + extra
        return out.toString()
      }
      if (c == '\\' && quoteByte == '"') {
        this.pos += 1
        if (this.more() && this.skipsLineEnding()) {
          loop (this.more()) {
            val w = this.src.byteAt(this.pos)
            if (w == '\n') {
              this.line += 1
            } else if (w != ' ' && w != '\t') {
              break
            }
            this.pos += 1
          }
          continue
        }
        try this.escape(out)
        this.pos += 1
      } else {
        if (c == '\n') this.line += 1
        if (isControl(c) && c != '\n' && c != '\t') throw this.fail("a control character in a string must be escaped")
        out.appendByte(c)
        this.pos += 1
      }
    }
    throw this.fail("unterminated string")
  }

  /// True when the bytes at `pos` are blanks and then a newline: a line-ending backslash.
  fun skipsLineEnding(): bool {
    var i = this.pos
    loop (i < this.src.len() && (this.src.byteAt(i) == ' ' || this.src.byteAt(i) == '\t')) i += 1
    i < this.src.len() && this.src.byteAt(i) == '\n'
  }

  fun value(): Value throws ParseError {
    if (!this.more()) throw this.fail("expected a value")
    val line = this.line
    val c = this.src.byteAt(this.pos)
    if (c == '"' || c == '\'') return Value(kind: Kind.Str, line, str: try this.str())
    if (c == '[') return try this.listValue(line)
    if (c == '{') return try this.inlineTable(line)
    val start = this.pos
    loop (this.more() && !endsWord(this.src.byteAt(this.pos))) this.pos += 1
    var word = this.src.substring(start, this.pos) ?: panic("value: a word stops at ASCII bytes")
    if (word == "true" || word == "false") return Value(kind: Kind.Bool, line, flag: word == "true")
    if (word.isEmpty()) throw this.fail("expected a value")
    if (this.strict) {
      if (val n = parseDecimal(word.replace("_", ""))) return Value(kind: Kind.Int, line, num: n)
      throw this.fail("cannot read ${quoted(word)}: strings are written in quotes (\"$word\")")
    }
    // a date and a time may be separated by a space
    if (word.len() == 10 && dateKind(word) == Kind.LocalDate && this.pos + 3 < this.src.len() && this.src.byteAt(this.pos) == ' ' && isDigit(this.src.byteAt(this.pos + 1))) {
      val timeStart = this.pos
      this.pos += 1
      loop (this.more() && !endsWord(this.src.byteAt(this.pos))) this.pos += 1
      val joined = this.src.substring(start, this.pos) ?: word
      if (dateKind(joined) != null) {
        word = joined
      } else {
        this.pos = timeStart
      }
    }
    if (val kind = dateKind(word)) return Value(kind, line, str: word)
    return try this.number(word, line)
  }

  /// An integer or a float in the full language.
  fun number(word: string, line: i64): Value throws ParseError {
    val unreadable = this.fail("cannot read ${quoted(word)}: strings are written in quotes (\"$word\")")
    var sign: u8 = 0
    var body = word
    if (word.startsWith("+") || word.startsWith("-")) {
      sign = word.byteAt(0)
      body = word.substring(1, word.len()) ?: ""
    }
    if (body == "inf" || body == "nan") {
      return Value(kind: Kind.Float, line, real: parseFloat(word) ?: throw unreadable)
    }
    if (sign == 0 && body.len() > 2 && body.byteAt(0) == '0') {
      val radix: i64 = when (body.byteAt(1)) {
        'x'  => 16
        'o'  => 8
        'b'  => 2
        else => 0
      }
      if (radix != 0) {
        val digits = withoutUnderscores(body.substring(2, body.len()) ?: "") ?: throw unreadable
        val n = parseRadix(digits, radix) ?: throw this.fail("the number ${quoted(word)} is not in the range of a 64-bit integer")
        return Value(kind: Kind.Int, line, num: n)
      }
    }
    if (body.isEmpty() || !isDigit(body.byteAt(0))) throw unreadable
    val isFloat = body.indexOf(".") >= 0 || body.indexOf("e") >= 0 || body.indexOf("E") >= 0
    if (isFloat) return Value(kind: Kind.Float, line, real: parseFloat(word) ?: throw unreadable)
    val digits = withoutUnderscores(body) ?: throw unreadable
    loop (i in 0..<digits.len()) {
      if (!isDigit(digits.byteAt(i))) throw unreadable
    }
    if (digits.len() > 1 && digits.byteAt(0) == '0') throw unreadable
    val n = parseDecimal((if (sign == '-') "-" else "") + digits) ?: throw this.fail("the number ${quoted(word)} is not in the range of a 64-bit integer")
    Value(kind: Kind.Int, line, num: n)
  }

  fun listValue(line: i64): Value throws ParseError {
    this.pos += 1
    val items: MutableList<Value> = []
    loop {
      this.skipSpace(true)
      if (this.eat(']')) return Value(kind: Kind.List, line, items: items.toList())
      items.push(try this.value())
      this.skipSpace(true)
      if (this.eat(',')) continue
      if (!this.eat(']')) throw this.fail("expected ',' or ']' in the list")
      return Value(kind: Kind.List, line, items: items.toList())
    }
  }

  fun inlineTable(line: i64): Value throws ParseError {
    this.pos += 1
    val table = &Table(line, origin: Origin.Inline)
    val result = Value(kind: Kind.Table, line, table)
    this.skipSpace(false)
    if (this.eat('}')) return result
    loop {
      this.skipSpace(false)
      if (this.more() && this.src.byteAt(this.pos) == '\n') {
        throw this.fail("an inline table is one line: close it with '}' or use a [table] header")
      }
      var key = try this.key()
      this.skipSpace(false)
      var dest = table
      // `{ a.b = 1 }`: a dotted key below the inline table
      loop (!this.strict && this.more() && this.src.byteAt(this.pos) == '.') {
        this.pos += 1
        this.skipSpace(false)
        dest = try this.child(dest, key, line, Origin.Inline)
        key = try this.key()
        this.skipSpace(false)
      }
      if (!this.eat('=')) throw this.fail("expected '=' in the inline table")
      this.skipSpace(false)
      val item = try this.value()
      if (dest.vals.containsKey(key)) throw this.fail("key ${quoted(key)} is set twice")
      dest.vals.set(key, item)
      this.skipSpace(false)
      if (this.eat(',')) continue
      if (!this.eat('}')) throw this.fail("expected ',' or '}' in the inline table (an inline table is one line)")
      return result
    }
  }
}

fun isControl(b: u8): bool => b < 0x20 || b == 0x7F
