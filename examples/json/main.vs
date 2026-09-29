// A JSON parser and printer: a sealed value type, a recursive-descent parser
// over the bytes of the text with positioned errors, a compact and a pretty
// printer, and a few queries on the parsed tree.
use io { println }

sealed trait Json
struct JNull : Json { }
struct JBool : Json {
  value: bool
}
struct JNum : Json {
  value: f64
}
struct JStr : Json {
  value: string
}
struct JArr : Json {
  items: List<Json>
}
struct JObj : Json {
  fields: Map<string, Json>
}

error ParseError {
  message: string
  pos:     i64
  fun message(): string = "${this.message} at offset ${this.pos}"
}

// the bytes the parser looks at: a byte literal is the u8 of one ASCII character
const QUOTE: u8 = '"'
const BACKSLASH: u8 = '\\'
const LBRACE: u8 = '{'
const RBRACE: u8 = '}'
const LBRACKET: u8 = '['
const RBRACKET: u8 = ']'
const COLON: u8 = ':'
const COMMA: u8 = ','
const MINUS: u8 = '-'
const PLUS: u8 = '+'
const DOT: u8 = '.'
const ZERO: u8 = '0'
const NINE: u8 = '9'
const SPACE: u8 = ' '
const TAB: u8 = '\t'
const LF: u8 = '\n'
const CR: u8 = '\r'

struct Parser {
  src:     string
  var pos: i64 = 0

  static fun of(text: string): Parser = Parser(src: text)

  /// Parses the whole text: one value, surrounded by whitespace only.
  fun parseDocument(): Json throws ParseError {
    val v = try this.parseValue()
    this.skipSpace()
    if (this.pos < this.src.len()) throw this.fail("trailing characters after the value")
    v
  }

  fun fail(message: string): ParseError = ParseError(message, pos: this.pos)

  fun peek(): u8? = if (this.pos < this.src.len()) this.src.byteAt(this.pos) else null

  fun skipSpace() {
    loop {
      val b = this.peek() ?: return
      if (b != SPACE && b != TAB && b != LF && b != CR) return
      this.pos += 1
    }
  }

  fun expect(b: u8, what: string) throws ParseError {
    this.skipSpace()
    if (this.peek() != b) throw this.fail("expected $what")
    this.pos += 1
  }

  fun parseValue(): Json throws ParseError {
    this.skipSpace()
    val b = this.peek() ?: throw this.fail("unexpected end of input")
    when {
      b == LBRACE => try this.parseObject()
      b == LBRACKET => try this.parseArray()
      b == QUOTE => JStr(value: try this.parseString())
      b == MINUS || (b >= ZERO && b <= NINE) => try this.parseNumber()
      this.src.substring(this.pos, this.pos + 4) == "true" => {
        this.pos += 4
        JBool(value: true)
      }
      this.src.substring(this.pos, this.pos + 5) == "false" => {
        this.pos += 5
        JBool(value: false)
      }
      this.src.substring(this.pos, this.pos + 4) == "null" => {
        this.pos += 4
        JNull()
      }
      else => throw this.fail("unexpected character")
    }
  }

  fun parseObject(): Json throws ParseError {
    try this.expect(LBRACE, "'{'")
    var fields: MutableMap<string, Json> = [:]
    this.skipSpace()
    if (this.peek() == RBRACE) {
      this.pos += 1
      return JObj(fields: fields.toMap())
    }
    loop {
      this.skipSpace()
      if (this.peek() != QUOTE) throw this.fail("expected a string key")
      val key = try this.parseString()
      try this.expect(COLON, "':'")
      fields.set(key, try this.parseValue())
      this.skipSpace()
      val b = this.peek() ?: throw this.fail("unterminated object")
      this.pos += 1
      if (b == RBRACE) break
      if (b != COMMA) throw this.fail("expected ',' or '}'")
    }
    JObj(fields: fields.toMap())
  }

  fun parseArray(): Json throws ParseError {
    try this.expect(LBRACKET, "'['")
    var items: MutableList<Json> = []
    this.skipSpace()
    if (this.peek() == RBRACKET) {
      this.pos += 1
      return JArr(items: items.toList())
    }
    loop {
      items.push(try this.parseValue())
      this.skipSpace()
      val b = this.peek() ?: throw this.fail("unterminated array")
      this.pos += 1
      if (b == RBRACKET) break
      if (b != COMMA) throw this.fail("expected ',' or ']'")
    }
    JArr(items: items.toList())
  }

  fun parseNumber(): Json throws ParseError {
    val start = this.pos
    if (this.peek() == MINUS) this.pos += 1
    this.digits()
    if (this.peek() == DOT) {
      this.pos += 1
      this.digits()
    }
    val e = this.peek()
    if (e == 'e' || e == 'E') {
      this.pos += 1
      val sign = this.peek()
      if (sign == PLUS || sign == MINUS) this.pos += 1
      this.digits()
    }
    val text = this.src.substring(start, this.pos) ?: ""
    val value = text.toF64() ?: throw ParseError(message: "malformed number '$text'", pos: start)
    JNum(value)
  }

  fun digits() {
    loop {
      val b = this.peek() ?: return
      if (b < ZERO || b > NINE) return
      this.pos += 1
    }
  }

  /// A string literal, with escapes decoded; the cursor is on the opening quote.
  fun parseString(): string throws ParseError {
    this.pos += 1
    var out: MutableList<u8> = []
    loop {
      val b = this.peek() ?: throw this.fail("unterminated string")
      this.pos += 1
      if (b == QUOTE) break
      if (b != BACKSLASH) {
        out.push(b)
        continue
      }
      val esc = this.peek() ?: throw this.fail("unterminated escape")
      this.pos += 1
      when {
        esc == QUOTE || esc == BACKSLASH || esc == '/' => out.push(esc)
        esc == 'n' => out.push(LF)
        esc == 't' => out.push(TAB)
        esc == 'r' => out.push(CR)
        esc == 'b' => out.push(8)
        esc == 'f' => out.push(12)
        esc == 'u' => {
          val hex = this.src.substring(this.pos, this.pos + 4) ?: throw this.fail("short \\u escape")
          val cp = parseHex(hex) ?: throw this.fail("bad \\u escape '$hex'")
          this.pos += 4
          encodeUtf8(cp, out)
        }
        else => throw this.fail("unknown escape")
      }
    }
    out.decodeUtf8() ?: throw this.fail("string is not valid UTF-8")
  }
}

fun parseHex(text: string): i64? {
  var n: i64 = 0
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i).toI64()
    val d = when {
      b >= '0' && b <= '9' => b - '0'
      b >= 'a' && b <= 'f' => b - 'a' + 10
      b >= 'A' && b <= 'F' => b - 'A' + 10
      else                 => return null
    }
    n = n * 16 + d
  }
  n
}

fun encodeUtf8(cp: i64, out: MutableList<u8>) {
  when {
    cp < 0x80  => out.push(cp.wrapU8())
    cp < 0x800 => {
      out.push((0xC0 | (cp >> 6)).wrapU8())
      out.push((0x80 | (cp & 0x3F)).wrapU8())
    }
    else       => {
      out.push((0xE0 | (cp >> 12)).wrapU8())
      out.push((0x80 | ((cp >> 6) & 0x3F)).wrapU8())
      out.push((0x80 | (cp & 0x3F)).wrapU8())
    }
  }
}

fun parse(text: string): Json throws ParseError {
  var p = Parser.of(text)
  try p.parseDocument()
}

// ---------------------------------------------------------------------------
// printing

fun quote(s: string): string {
  val sb = StringBuilder()
  sb.append("\"")
  loop (i in 0..<s.len()) {
    val b = s.byteAt(i)
    when {
      b == QUOTE     => sb.append("\\\"")
      b == BACKSLASH => sb.append("\\\\")
      b == LF        => sb.append("\\n")
      b == TAB       => sb.append("\\t")
      b == CR        => sb.append("\\r")
      else           => sb.appendByte(b)
    }
  }
  sb.append("\"")
  sb.toString()
}

fun number(x: f64): string = if (x == x.trunc() && x.abs() < 1.0e15) "${x.toI64()}" else "$x"

fun compact(v: Json): string = when (v) {
  is JNull        => "null"
  is JBool(value) => "$value"
  is JNum(value)  => number(value)
  is JStr(value)  => quote(value)
  is JArr(items)  => "[" + items.map(x => compact(x)).join(",") + "]"
  is JObj(fields) => "{" + fields.entries().map(e => quote(e.0) + ":" + compact(e.1)).join(",") + "}"
}

fun pretty(v: Json, indent: i64 = 0): string {
  val pad = "  ".repeat(indent + 1)
  val close = "  ".repeat(indent)
  when (v) {
    is JArr(items) if !items.isEmpty() =>
      "[\n" + items.map(x => pad + pretty(x, indent + 1)).join(",\n") + "\n$close]"
    is JObj(fields) if !fields.isEmpty() =>
      "{\n" + fields.entries().map(e => pad + quote(e.0) + ": " + pretty(e.1, indent + 1)).join(",\n") + "\n$close}"
    else => compact(v)
  }
}

// ---------------------------------------------------------------------------
// queries

extend Json {
  /// The field `name` of an object, or null.
  public fun field(name: string): Json? = when (this) {
    is JObj(fields) => fields.get(name)
    else            => null
  }

  /// The element `i` of an array, or null.
  public fun item(i: i64): Json? = when (this) {
    is JArr(items) => items.at(i)
    else           => null
  }

  public fun asString(): string? = when (this) {
    is JStr(value) => value
    else           => null
  }

  public fun asNumber(): f64? = when (this) {
    is JNum(value) => value
    else           => null
  }

  /// Walks a dotted path: "users.1.name".
  public fun path(p: string): Json? {
    var cur: Json? = this
    loop (part in p.split(".")) {
      val here = cur ?: return null
      cur = when (val n = part.toInt()) {
        null   => here.field(part)
        is i64 => here.item(n)
      }
    }
    cur
  }
}

fun depth(v: Json): i64 = when (v) {
  is JArr(items)  => 1 + (items.map(x => depth(x)).max() ?: 0)
  is JObj(fields) => 1 + (fields.values().map(x => depth(x)).max() ?: 0)
  else            => 0
}

fun main() {
  val text = "{\"name\": \"Veles\", \"version\": 0.24, \"tags\": [\"fast\", \"safe\", \"gc\"],\n" +
    "  \"author\": {\"name\": \"Lah\", \"langs\": [\"pl\", \"en\"]}, \"stars\": 1e3,\n" +
    "  \"escaped\": \"line\\nbreak \\\"quoted\\\" \\u0041\\u00e9\", \"nothing\": null, \"ok\": true}"
  val doc = parse(text) catch (e) {
    println("parse failed: ${e.message()}")
    return
  }
  println(compact(doc))
  println(pretty(doc))
  println("name=${doc.path("name")?.asString()} second tag=${doc.path("tags.1")?.asString()} author=${doc.path("author.name")?.asString()}")
  println("stars=${doc.path("stars")?.asNumber()} missing=${doc.path("author.age")?.asNumber()} depth=${depth(doc)}")
  println("escaped=${doc.path("escaped")?.asString()} chars=${doc.path("escaped")?.asString()?.charCount()}")

  // round trip: printing and parsing again gives the same compact text
  val again = parse(compact(doc))
  println("round trip ${if (again.ok) compact(again) == compact(doc) else false}")

  // errors carry a position
  loop (bad in ["{\"a\": }", "[1, 2", "\"open", "{\"a\": 1} x", "[1, 2,]", "tru", "\"\\q\""]) {
    val r = parse(bad)
    if (r.err) println("$bad -> ${r.message()}")
  }
}
