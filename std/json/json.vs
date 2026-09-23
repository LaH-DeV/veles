/// JSON for anything `Codable` (D58). `json.encode(user)` writes a value
/// straight to text, `json.decode<User>(text)` reads it back, and every
/// problem found on the way — a wrong type here, a missing field there —
/// comes back at once in one `DecodeError`, each with its path. `Options`
/// are the API-wide policies: key casing, how enums travel, whether nulls
/// are written, pretty printing, and the limits a hostile document meets.
///
/// ```veles
/// struct User {
///   id:    i64
///   name:  string
///   email: string?
///   implement Codable
/// }
///
/// val text = try json.encode(User(id: 1, name: "ann", email: null))
/// // {"id":1,"name":"ann","email":null}
/// val user = try json.decode<User>(text)
/// val tree = try json.parse(text)          // a Value, untyped
/// ```

use utf8

/// The policies of one encoder or decoder.
public struct Options {
  /// How field names are spelled as keys: `passwordHash`, `password_hash`
  /// or `passwordHash`; a `@key` the author wrote is never restyled.
  public keys: KeyStyle = KeyStyle.AsWritten
  /// Enums as their names (`"Active"`) or their numbers (`1`).
  public enums: EnumStyle = EnumStyle.Name
  /// Leave out object members whose value is null.
  public omitNulls: bool = false
  /// Newlines and indentation.
  public pretty: bool = false
  public indent: string = "  "
  /// Nesting deeper than this is refused: a `DecodeError` on the way in, an
  /// `EncodeError` on the way out. 64 is well past what documents in the
  /// wild nest to and well short of what the stack can take; the prelude's
  /// `maxRecursionDepth` is the general bound this is a policy against.
  public maxDepth: i64 = 64
  /// Problems recorded beyond this are dropped.
  public maxProblems: i64 = 100
}

// ---------------------------------------------------------------------------
// the entry points

/// `value` as JSON text.
public fun encode<T: Encodable>(value: T, options: Options = Options()): string throws EncodeError {
  val enc = JsonEncoder.of(options)
  try value.encode(enc)
  enc.text()
}

/// `value` as JSON text over several lines.
public fun pretty<T: Encodable>(value: T, options: Options = Options()): string throws EncodeError =
  try encode(value, Options(keys: options.keys, enums: options.enums, omitNulls: options.omitNulls, pretty: true, indent: options.indent))

/// A `T` read from JSON text. Every problem in the document is reported
/// together; a document that is not JSON at all is one problem.
public fun decode<T: Decodable>(text: string, options: Options = Options()): T throws DecodeError {
  val dec = JsonDecoder.of(text, options)
  val value = try T.decode(dec)
  try dec.end()
  try finish(dec, value)
}

/// The document as a `Value` tree, untyped.
public fun parse(text: string, options: Options = Options()): Value throws DecodeError = try decode<Value>(text, options)

/// `value` as a `Value` tree — what it would encode to, before it is text.
public fun toValue<T: Encodable>(value: T, options: Options = Options()): Value throws EncodeError {
  val enc = ValueEncoder.of("json", options.enums, options.keys, options.maxDepth)
  try value.encode(enc)
  enc.value()
}

/// A `T` read from a `Value` tree.
public fun fromValue<T: Decodable>(v: Value, options: Options = Options()): T throws DecodeError {
  val dec = ValueDecoder.of(v, "json", options.enums, options.keys, options.maxDepth)
  val value = try T.decode(dec)
  try finish(dec, value)
}

// ---------------------------------------------------------------------------
// writing

/// An open container while writing: whether a member has been written
/// (so the next one needs a comma).
struct Open {
  var members: i64 = 0
  isList:      bool
}

/// The `Encoder` that writes JSON text. Its state lives behind handles —
/// a builder and a list — so the copy a trait object makes (D9) writes
/// into the same text.
public struct JsonEncoder {
  private out:     StringBuilder = stringBuilder()
  private stack:   MutableList<Open> = []
  private pending: MutableList<string> = []  // the key waiting for its value, if any
  private options: Options

  public static fun of(options: Options = Options()): JsonEncoder = JsonEncoder(options)

  /// The text written so far.
  public fun text(): string = self.out.toString()

  /// Before a value: the separator, the newline and indent when pretty,
  /// and the key that was waiting.
  private fun beforeValue() {
    if (self.stack.isEmpty()) return
    val top = self.stack.refOrPanic(self.stack.len() - 1)
    if (top.members > 0) self.out.append(",")
    top.members += 1
    self.newline()
    if (!top.isList && !self.pending.isEmpty()) {
      val k = self.pending.removeAt(self.pending.len() - 1)
      writeQuoted(self.out, k)
      self.out.append(if (self.options.pretty) ": " else ":")
    }
  }

  private fun newline() {
    if (!self.options.pretty) return
    self.out.append("\n")
    loop (_ in 0..<self.stack.len()) self.out.append(self.options.indent)
  }

  /// A tree nests as deep as whoever built it wanted, and `Value.encode`
  /// walks it by recursion, so the limit belongs on the way out as much as
  /// on the way in — a document that was refused as too deep must not come
  /// back as a stack overflow when something re-encodes it.
  private fun checkDepth() throws EncodeError {
    if (self.stack.len() >= self.options.maxDepth) {
      throw EncodeError(message: tooDeepMessage(self.options.maxDepth))
    }
  }

  private fun close(what: string) {
    val top = self.stack.removeAt(self.stack.len() - 1)
    if (top.members > 0) self.newline()
    self.out.append(what)
  }

  implement Encoder {
    fun format(): string = "json"
    override fun enums(): EnumStyle = self.options.enums
    override fun keys(): KeyStyle = self.options.keys

    fun beginObject() throws EncodeError {
      try self.checkDepth()
      self.beforeValue()
      self.out.append("{")
      self.stack.push(Open(isList: false))
    }

    fun key(name: string) throws EncodeError {
      self.pending.push(name)
    }

    fun endObject() throws EncodeError = self.close("}")

    fun beginList() throws EncodeError {
      try self.checkDepth()
      self.beforeValue()
      self.out.append("[")
      self.stack.push(Open(isList: true))
    }

    fun endList() throws EncodeError = self.close("]")

    fun writeI64(v: i64) throws EncodeError {
      self.beforeValue()
      self.out.append("$v")
    }

    fun writeU64(v: u64) throws EncodeError {
      self.beforeValue()
      self.out.append("$v")
    }

    fun writeF64(v: f64) throws EncodeError {
      if (v.isNaN() || v.isInfinite()) throw EncodeError(message: "JSON has no representation for $v")
      self.beforeValue()
      self.out.append("$v")
    }

    fun writeBool(v: bool) throws EncodeError {
      self.beforeValue()
      self.out.append(if (v) "true" else "false")
    }

    fun writeString(v: string) throws EncodeError {
      self.beforeValue()
      writeQuoted(self.out, v)
    }

    fun writeNull() throws EncodeError {
      if (self.options.omitNulls && !self.pending.isEmpty()) {
        val _ = self.pending.removeAt(self.pending.len() - 1)
        return
      }
      self.beforeValue()
      self.out.append("null")
    }
  }
}

const HEX: string = "0123456789abcdef"

/// `s` as a JSON string literal, escapes and all.
fun writeQuoted(out: StringBuilder, s: string) {
  out.append("\"")
  loop (i in 0..<s.len()) {
    val b = s.byteAt(i)
    when {
      b == '"'  => out.append("\\\"")
      b == '\\' => out.append("\\\\")
      b == '\n' => out.append("\\n")
      b == '\r' => out.append("\\r")
      b == '\t' => out.append("\\t")
      b == 8    => out.append("\\b")
      b == 12   => out.append("\\f")
      b < 32    => {
        out.append("\\u00")
        out.appendByte(HEX.byteAt((b >> 4) as i64))
        out.appendByte(HEX.byteAt((b & 15) as i64))
      }
      else      => out.appendByte(b)
    }
  }
  out.append("\"")
}

// ---------------------------------------------------------------------------
// reading

/// An open container while reading: where the reader is inside it (for
/// the path) and whether it is real — a container that was asked for but
/// not there is a phantom that ends at once, after the problem is noted.
struct Frame {
  isList:      bool
  var index:   i64 = 0      // members read so far
  var key:     string = ""  // the current member's key (objects)
  var phantom: bool = false
}

/// The `Decoder` that reads JSON text. A wrong type is a problem at the
/// path, recorded and read past; malformed text is a `DecodeError`.
public struct JsonDecoder {
  private src:      List<u8>
  private cursor:   MutableList<i64> = [0]  // the position, behind a handle
  private stack:    MutableList<Frame> = []
  private recorded: Problems
  private options:  Options

  public static fun of(text: string, options: Options = Options()): JsonDecoder =
    JsonDecoder(src: text.bytes(), recorded: Problems.capped(options.maxProblems), options)

  /// After the value: nothing but whitespace may follow.
  public fun end() throws DecodeError {
    self.skipSpace()
    if (self.pos() < self.src.len()) throw self.malformed("text after the value")
  }

  private fun pos(): i64 = self.cursor.atOrPanic(0)
  private fun setPos(p: i64) = self.cursor.set(0, p)

  private fun malformed(what: string): DecodeError =
    DecodeError(problems: self.recorded.list().concat([Problem(path: "", message: "malformed JSON at byte ${self.pos()}: $what")]))

  private fun peekByte(): u8? = if (self.pos() < self.src.len()) self.src.atOrPanic(self.pos()) else null

  private fun skipSpace() {
    loop {
      val b = self.peekByte() ?: return
      if (b != ' ' && b != '\t' && b != '\n' && b != '\r') return
      self.setPos(self.pos() + 1)
    }
  }

  /// Describes the next value for a problem message.
  private fun found(): string throws DecodeError = when (try self.peek()) {
    Kind.Null   => "null"
    Kind.Bool   => "a boolean"
    Kind.Int    => "a number"
    Kind.Float  => "a number"
    Kind.String => "a string"
    Kind.List   => "a list"
    Kind.Object => "an object"
  }

  /// A value of the wrong type: recorded and read past.
  private fun wrong(expected: string) throws DecodeError {
    self.recorded.record(self.path(), "expected $expected, found ${try self.found()}")
    try self.skip()
  }

  /// The text of the number at the cursor, consumed.
  private fun number(): string throws DecodeError {
    val start = self.pos()
    if (self.peekByte() == '-') self.setPos(self.pos() + 1)
    self.digits()
    if (self.peekByte() == '.') {
      self.setPos(self.pos() + 1)
      self.digits()
    }
    val e = self.peekByte()
    if (e == 'e' || e == 'E') {
      self.setPos(self.pos() + 1)
      val sign = self.peekByte()
      if (sign == '+' || sign == '-') self.setPos(self.pos() + 1)
      self.digits()
    }
    if (self.pos() == start) throw self.malformed("expected a number")
    (self.src.slice(start, self.pos()).decodeUtf8() ?: "")
  }

  private fun digits() {
    loop {
      val b = self.peekByte() ?: return
      if (b < '0' || b > '9') return
      self.setPos(self.pos() + 1)
    }
  }

  /// Whether the number at the cursor has a fraction or an exponent.
  private fun numberIsFloat(): bool {
    var i = self.pos()
    loop (i < self.src.len()) {
      val b = self.src.atOrPanic(i)
      if (b == '.' || b == 'e' || b == 'E') return true
      if (b != '-' && (b < '0' || b > '9')) return false
      i += 1
    }
    false
  }

  private fun literal(word: string) throws DecodeError {
    loop (i in 0..<word.len()) {
      if (self.peekByte() != word.byteAt(i)) throw self.malformed("expected '$word'")
      self.setPos(self.pos() + 1)
    }
  }

  /// A string literal at the cursor, escapes decoded.
  private fun quoted(): string throws DecodeError {
    if (self.peekByte() != '"') throw self.malformed("expected a string")
    self.setPos(self.pos() + 1)
    val out: MutableList<u8> = []
    loop {
      val b = self.peekByte() ?: throw self.malformed("unterminated string")
      self.setPos(self.pos() + 1)
      if (b == '"') break
      if (b < 32) throw self.malformed("a control character inside a string")
      if (b != '\\') {
        out.push(b)
        continue
      }
      val esc = self.peekByte() ?: throw self.malformed("unterminated escape")
      self.setPos(self.pos() + 1)
      when {
        esc == '"' || esc == '\\' || esc == '/' => out.push(esc)
        esc == 'n' => out.push('\n')
        esc == 't' => out.push('\t')
        esc == 'r' => out.push('\r')
        esc == 'b' => out.push(8)
        esc == 'f' => out.push(12)
        esc == 'u' => {
          var cp = try self.hex4()
          if (utf8.isSurrogate(cp)) {
            // a surrogate pair: the low half follows as another \u escape
            if (self.peekByte() != '\\' || self.pos() + 1 >= self.src.len() || self.src.atOrPanic(self.pos() + 1) != 'u') {
              throw self.malformed("a lone surrogate in a \\u escape")
            }
            self.setPos(self.pos() + 2)
            val low = try self.hex4()
            cp = utf8.combineSurrogates(cp, low) ?: throw self.malformed("a lone surrogate in a \\u escape")
          }
          val _ = utf8.encodeTo(out, cp)
        }
        else => throw self.malformed("unknown escape '\\${[esc].decodeUtf8() ?: "?"}'")
      }
    }
    out.decodeUtf8() ?: throw self.malformed("a string that is not valid UTF-8")
  }

  private fun hex4(): i64 throws DecodeError {
    var n: i64 = 0
    loop (_ in 0..<4) {
      val b = (self.peekByte() ?: throw self.malformed("short \\u escape")) as i64
      val d = when {
        b >= '0' && b <= '9' => b - '0'
        b >= 'a' && b <= 'f' => b - 'a' + 10
        b >= 'A' && b <= 'F' => b - 'A' + 10
        else                 => throw self.malformed("bad \\u escape")
      }
      n = n * 16 + d
      self.setPos(self.pos() + 1)
    }
    n
  }

  /// Between members of the open container: the separator, or its end.
  /// True when another member follows.
  private fun more(closer: u8): bool throws DecodeError {
    val top = self.stack.refOrPanic(self.stack.len() - 1)
    if (top.phantom) return false
    self.skipSpace()
    val b = self.peekByte() ?: throw self.malformed("unterminated container")
    if (b == closer) return false
    if (top.index == 0) return true
    if (b != ',') throw self.malformed("expected ',' or '${[closer].decodeUtf8() ?: "?"}'")
    self.setPos(self.pos() + 1)
    self.skipSpace()
    if (self.peekByte() == closer) throw self.malformed("a trailing comma")
    true
  }

  private fun open(isList: bool, opener: u8, expected: string) throws DecodeError {
    if (self.stack.len() >= self.options.maxDepth) throw self.malformed(tooDeepMessage(self.options.maxDepth))
    self.skipSpace()
    if (self.peekByte() == opener) {
      self.setPos(self.pos() + 1)
      self.stack.push(Frame(isList))
      return
    }
    try self.wrong(expected)
    self.stack.push(Frame(isList, phantom: true))
  }

  private fun closeContainer(closer: u8) throws DecodeError {
    val top = self.stack.removeAt(self.stack.len() - 1)
    if (top.phantom) return
    self.skipSpace()
    if (self.peekByte() != closer) throw self.malformed("expected '${[closer].decodeUtf8() ?: "?"}'")
    self.setPos(self.pos() + 1)
  }

  implement Decoder {
    fun format(): string = "json"
    override fun enums(): EnumStyle = self.options.enums
    override fun keys(): KeyStyle = self.options.keys

    fun peek(): Kind throws DecodeError {
      self.skipSpace()
      val b = self.peekByte() ?: throw self.malformed("unexpected end of input")
      when {
        b == '{' => Kind.Object
        b == '[' => Kind.List
        b == '"' => Kind.String
        b == 't' || b == 'f' => Kind.Bool
        b == 'n' => Kind.Null
        b == '-' || (b >= '0' && b <= '9') => if (self.numberIsFloat()) Kind.Float else Kind.Int
        else => throw self.malformed("unexpected character")
      }
    }

    fun beginObject() throws DecodeError = try self.open(false, '{', "an object")

    fun nextKey(): string? throws DecodeError {
      if (!(try self.more('}'))) return null
      val top = self.stack.refOrPanic(self.stack.len() - 1)
      self.skipSpace()
      val k = try self.quoted()
      self.skipSpace()
      if (self.peekByte() != ':') throw self.malformed("expected ':' after a key")
      self.setPos(self.pos() + 1)
      top.key = k
      top.index += 1
      k
    }

    fun endObject() throws DecodeError = try self.closeContainer('}')

    fun beginList() throws DecodeError = try self.open(true, '[', "a list")

    fun hasNext(): bool throws DecodeError {
      val has = try self.more(']')
      if (has) {
        self.stack.refOrPanic(self.stack.len() - 1).index += 1
      }
      has
    }

    fun endList() throws DecodeError = try self.closeContainer(']')

    fun readI64(): i64 throws DecodeError {
      if (try self.peek() != Kind.Int) {
        try self.wrong("an integer")
        return 0
      }
      val text = try self.number()
      val n = text.toInt()
      if (n == null) {
        self.recorded.record(self.path(), "$text does not fit an integer")
        return 0
      }
      n
    }

    fun readU64(): u64 throws DecodeError {
      val n = try self.readI64()
      if (n < 0) {
        self.recorded.record(self.path(), "$n is negative")
        return 0
      }
      n as u64
    }

    fun readF64(): f64 throws DecodeError {
      val k = try self.peek()
      if (k != Kind.Int && k != Kind.Float) {
        try self.wrong("a number")
        return 0.0
      }
      val text = try self.number()
      val n = text.toF64()
      if (n == null) {
        self.recorded.record(self.path(), "$text is not a number")
        return 0.0
      }
      n
    }

    fun readBool(): bool throws DecodeError {
      if (try self.peek() != Kind.Bool) {
        try self.wrong("a boolean")
        return false
      }
      if (self.peekByte() == 't') {
        try self.literal("true")
        return true
      }
      try self.literal("false")
      false
    }

    fun readString(): string throws DecodeError {
      if (try self.peek() != Kind.String) {
        try self.wrong("a string")
        return ""
      }
      try self.quoted()
    }

    fun readNull() throws DecodeError {
      if (try self.peek() != Kind.Null) {
        return try self.wrong("null")
      }
      try self.literal("null")
    }

    fun skip() throws DecodeError {
      when (try self.peek()) {
        Kind.Null   => try self.literal("null")
        Kind.Bool   => {
          val _ = try self.readBool()
        }
        Kind.Int    => {
          val _ = try self.number()
        }
        Kind.Float  => {
          val _ = try self.number()
        }
        Kind.String => {
          val _ = try self.quoted()
        }
        Kind.List   => {
          try self.beginList()
          loop (try self.hasNext()) try self.skip()
          try self.endList()
        }
        Kind.Object => {
          try self.beginObject()
          loop {
            val _ = try self.nextKey() ?: break
            try self.skip()
          }
          try self.endObject()
        }
      }
    }

    fun path(): string {
      var out = ""
      loop (fr in self.stack) {
        out = if (fr.isList) indexPath(out, fr.index - 1) else childPath(out, fr.key)
      }
      out
    }

    fun problemAt(path: string, message: string) = self.recorded.record(path, message)

    fun problems(): List<Problem> = self.recorded.list()
  }
}
