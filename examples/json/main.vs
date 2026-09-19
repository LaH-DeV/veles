// A JSON parser and printer: a sealed value type, a recursive-descent parser
// over the bytes of the text with positioned errors, a compact and a pretty
// printer, and a few queries on the parsed tree.
use io

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
  fun message(): string = "${self.message} at offset ${self.pos}"
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
  src: string
  pos: i64 = 0

  static fun of(text: string): Parser = Parser(src: text)

  /// Parses the whole text: one value, surrounded by whitespace only.
  mut fun parseDocument(): Json throws ParseError {
    val v = try self.parseValue()
    self.skipSpace()
    if (self.pos < self.src.len()) throw self.fail("trailing characters after the value")
    v
  }

  fun fail(message: string): ParseError = ParseError(message, pos: self.pos)

  fun peek(): u8? = if (self.pos < self.src.len()) self.src.byteAt(self.pos) else null

  mut fun skipSpace() {
    loop {
      val b = self.peek() ?: return
      if (b != SPACE && b != TAB && b != LF && b != CR) return
      self.pos += 1
    }
  }

  mut fun expect(b: u8, what: string) throws ParseError {
    self.skipSpace()
    if (self.peek() != b) throw self.fail("expected $what")
    self.pos += 1
  }

  mut fun parseValue(): Json throws ParseError {
    self.skipSpace()
    val b = self.peek() ?: throw self.fail("unexpected end of input")
    when {
      b == LBRACE => try self.parseObject()
      b == LBRACKET => try self.parseArray()
      b == QUOTE => JStr(value: try self.parseString())
      b == MINUS || (b >= ZERO && b <= NINE) => try self.parseNumber()
      self.src.substring(self.pos, self.pos + 4) == "true" => {
        self.pos += 4
        JBool(value: true)
      }
      self.src.substring(self.pos, self.pos + 5) == "false" => {
        self.pos += 5
        JBool(value: false)
      }
      self.src.substring(self.pos, self.pos + 4) == "null" => {
        self.pos += 4
        JNull()
      }
      else => throw self.fail("unexpected character")
    }
  }

  mut fun parseObject(): Json throws ParseError {
    try self.expect(LBRACE, "'{'")
    var fields: MutableMap<string, Json> = [:]
    self.skipSpace()
    if (self.peek() == RBRACE) {
      self.pos += 1
      return JObj(fields: fields.toMap())
    }
    loop {
      self.skipSpace()
      if (self.peek() != QUOTE) throw self.fail("expected a string key")
      val key = try self.parseString()
      try self.expect(COLON, "':'")
      fields.set(key, try self.parseValue())
      self.skipSpace()
      val b = self.peek() ?: throw self.fail("unterminated object")
      self.pos += 1
      if (b == RBRACE) break
      if (b != COMMA) throw self.fail("expected ',' or '}'")
    }
    JObj(fields: fields.toMap())
  }

  mut fun parseArray(): Json throws ParseError {
    try self.expect(LBRACKET, "'['")
    var items: MutableList<Json> = []
    self.skipSpace()
    if (self.peek() == RBRACKET) {
      self.pos += 1
      return JArr(items: items.toList())
    }
    loop {
      items.push(try self.parseValue())
      self.skipSpace()
      val b = self.peek() ?: throw self.fail("unterminated array")
      self.pos += 1
      if (b == RBRACKET) break
      if (b != COMMA) throw self.fail("expected ',' or ']'")
    }
    JArr(items: items.toList())
  }

  mut fun parseNumber(): Json throws ParseError {
    val start = self.pos
    if (self.peek() == MINUS) self.pos += 1
    self.digits()
    if (self.peek() == DOT) {
      self.pos += 1
      self.digits()
    }
    val e = self.peek()
    if (e == 'e' || e == 'E') {
      self.pos += 1
      val sign = self.peek()
      if (sign == PLUS || sign == MINUS) self.pos += 1
      self.digits()
    }
    val text = self.src.substring(start, self.pos) ?: ""
    val value = text.toF64() ?: throw ParseError(message: "malformed number '$text'", pos: start)
    JNum(value)
  }

  mut fun digits() {
    loop {
      val b = self.peek() ?: return
      if (b < ZERO || b > NINE) return
      self.pos += 1
    }
  }

  /// A string literal, with escapes decoded; the cursor is on the opening quote.
  mut fun parseString(): string throws ParseError {
    self.pos += 1
    var out: MutableList<u8> = []
    loop {
      val b = self.peek() ?: throw self.fail("unterminated string")
      self.pos += 1
      if (b == QUOTE) break
      if (b != BACKSLASH) {
        out.push(b)
        continue
      }
      val esc = self.peek() ?: throw self.fail("unterminated escape")
      self.pos += 1
      when {
        esc == QUOTE || esc == BACKSLASH || esc == '/' => out.push(esc)
        esc == 'n' => out.push(LF)
        esc == 't' => out.push(TAB)
        esc == 'r' => out.push(CR)
        esc == 'b' => out.push(8)
        esc == 'f' => out.push(12)
        esc == 'u' => {
          val hex = self.src.substring(self.pos, self.pos + 4) ?: throw self.fail("short \\u escape")
          val cp = parseHex(hex) ?: throw self.fail("bad \\u escape '$hex'")
          self.pos += 4
          encodeUtf8(cp, out)
        }
        else => throw self.fail("unknown escape")
      }
    }
    out.decodeUtf8() ?: throw self.fail("string is not valid UTF-8")
  }
}

fun parseHex(text: string): i64? {
  var n: i64 = 0
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i) as i64
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
    cp < 0x80  => out.push(cp as u8)
    cp < 0x800 => {
      out.push((0xC0 | (cp >> 6)) as u8)
      out.push((0x80 | (cp & 0x3F)) as u8)
    }
    else       => {
      out.push((0xE0 | (cp >> 12)) as u8)
      out.push((0x80 | ((cp >> 6) & 0x3F)) as u8)
      out.push((0x80 | (cp & 0x3F)) as u8)
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
  val sb = stringBuilder()
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

fun number(x: f64): string = if (x == x.trunc() && x.abs() < 1.0e15) "${x as i64}" else "$x"

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
  public fun field(name: string): Json? = when (self) {
    is JObj(fields) => fields.get(name)
    else            => null
  }

  /// The element `i` of an array, or null.
  public fun item(i: i64): Json? = when (self) {
    is JArr(items) => items.at(i)
    else           => null
  }

  public fun asString(): string? = when (self) {
    is JStr(value) => value
    else           => null
  }

  public fun asNumber(): f64? = when (self) {
    is JNum(value) => value
    else           => null
  }

  /// Walks a dotted path: "users.1.name".
  public fun path(p: string): Json? {
    var cur: Json? = self
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
  val doc = when (val r = parse(text)) {
    is Ok  => r
    is Err => {
      io.println("parse failed: ${r.message()}")
      return
    }
  }
  io.println(compact(doc))
  io.println(pretty(doc))
  io.println("name=${doc.path("name")?.asString()} second tag=${doc.path("tags.1")?.asString()} author=${doc.path("author.name")?.asString()}")
  io.println("stars=${doc.path("stars")?.asNumber()} missing=${doc.path("author.age")?.asNumber()} depth=${depth(doc)}")
  io.println("escaped=${doc.path("escaped")?.asString()} chars=${doc.path("escaped")?.asString()?.charCount()}")

  // round trip: printing and parsing again gives the same compact text
  val again = parse(compact(doc))
  io.println("round trip ${if (again.ok) compact(again) == compact(doc) else false}")

  // errors carry a position
  loop (bad in ["{\"a\": }", "[1, 2", "\"open", "{\"a\": 1} x", "[1, 2,]", "tru", "\"\\q\""]) {
    val r = parse(bad)
    if (r.err) io.println("$bad -> ${r.message()}")
  }
}
