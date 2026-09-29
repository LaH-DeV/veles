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
/// val tree = try json.parse(text)          // a codec.Value, untyped
/// ```

use codec, recursion, utf8

/// The policies of one encoder or decoder.
public struct Options {
  /// How field names are spelled as keys: `passwordHash`, `password_hash`
  /// or `passwordHash`; a `@key` the author wrote is never restyled.
  public keys: codec.KeyStyle = codec.KeyStyle.AsWritten
  /// Enums as their names (`"Active"`) or their numbers (`1`).
  public enums: codec.EnumStyle = codec.EnumStyle.Name
  /// Durations as `"90.5s"` (the default), ISO 8601, the `Display` text, or
  /// a number of nanoseconds or milliseconds; see `codec.DurationStyle`.
  public durations: codec.DurationStyle = codec.DurationStyle.Seconds
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
  val enc = JsonEncoder(options)
  try value.encode(enc)
  enc.text()
}

/// `value` as JSON text over several lines.
public fun pretty<T: Encodable>(value: T, options: Options = Options()): string throws EncodeError =
  try encode(value, Options(keys: options.keys, enums: options.enums, durations: options.durations, omitNulls: options.omitNulls, pretty: true, indent: options.indent, maxDepth: options.maxDepth, maxProblems: options.maxProblems))

/// A `T` read from JSON text. Every problem in the document is reported
/// together; a document that is not JSON at all is one problem.
public fun decode<T: Decodable>(text: string, options: Options = Options()): T throws DecodeError {
  val dec = JsonDecoder.of(text, options)
  val value = try T.decode(dec)
  try dec.end()
  try codec.finish(dec, value)
}

/// The document as a `codec.Value` tree, untyped.
public fun parse(text: string, options: Options = Options()): codec.Value throws DecodeError = try decode<codec.Value>(text, options)

/// `value` as a `codec.Value` tree — what it would encode to, before it is text.
public fun toValue<T: Encodable>(value: T, options: Options = Options()): codec.Value throws EncodeError {
  val enc = codec.ValueEncoder.of("json", options.enums, options.keys, options.durations, options.maxDepth)
  try value.encode(enc)
  enc.value()
}

/// A `T` read from a `codec.Value` tree.
public fun fromValue<T: Decodable>(v: codec.Value, options: Options = Options()): T throws DecodeError {
  val dec = codec.ValueDecoder.of(v, "json", options.enums, options.keys, options.durations, options.maxDepth)
  val value = try T.decode(dec)
  try codec.finish(dec, value)
}

// ---------------------------------------------------------------------------
// writing

/// An open container while writing: whether a member has been written
/// (so the next one needs a comma).
struct Open {
  var members: i64 = 0
  isList:      bool
}

/// The `codec.Encoder` that writes JSON text. Its state lives behind handles —
/// a builder and a list — so the copy a trait object makes (D9) writes
/// into the same text. `json.JsonEncoder()`, or `JsonEncoder(options)`.
public struct JsonEncoder {
  public options:  Options = Options()
  private out:     StringBuilder = StringBuilder()
  private stack:   MutableList<Open> = []
  private pending: MutableList<string> = []  // the key waiting for its value, if any

  /// The text written so far.
  public fun text(): string = this.out.toString()

  /// Before a value: the separator, the newline and indent when pretty,
  /// and the key that was waiting.
  private fun beforeValue() {
    val top = this.stack.ref(-1) ?: return
    if (top.members > 0) this.out.append(",")
    top.members += 1
    this.newline()
    if (!top.isList && !this.pending.isEmpty()) {
      val k = this.pending.removeAt(this.pending.len() - 1)
      writeQuoted(this.out, k)
      this.out.append(if (this.options.pretty) ": " else ":")
    }
  }

  private fun newline() {
    if (!this.options.pretty) return
    this.out.append("\n")
    loop (_ in 0..<this.stack.len()) this.out.append(this.options.indent)
  }

  /// A tree nests as deep as whoever built it wanted, and `codec.Value.encode`
  /// walks it by recursion, so the limit belongs on the way out as much as
  /// on the way in — a document that was refused as too deep must not come
  /// back as a stack overflow when something re-encodes it.
  private fun checkDepth() throws EncodeError {
    if (this.stack.len() >= this.options.maxDepth) {
      throw EncodeError(message: recursion.tooDeepMessage(this.options.maxDepth))
    }
  }

  private fun close(what: string) {
    val top = this.stack.removeAt(this.stack.len() - 1)
    if (top.members > 0) this.newline()
    this.out.append(what)
  }

  implement codec.Encoder {
    fun format(): string = "json"
    override fun enums(): codec.EnumStyle = this.options.enums
    override fun durations(): codec.DurationStyle = this.options.durations
    override fun keys(): codec.KeyStyle = this.options.keys

    fun beginObject() throws EncodeError {
      try this.checkDepth()
      this.beforeValue()
      this.out.append("{")
      this.stack.push(Open(isList: false))
    }

    fun key(name: string) throws EncodeError {
      this.pending.push(name)
    }

    fun endObject() throws EncodeError = this.close("}")

    fun beginList() throws EncodeError {
      try this.checkDepth()
      this.beforeValue()
      this.out.append("[")
      this.stack.push(Open(isList: true))
    }

    fun endList() throws EncodeError = this.close("]")

    fun writeI64(v: i64) throws EncodeError {
      this.beforeValue()
      this.out.append("$v")
    }

    fun writeU64(v: u64) throws EncodeError {
      this.beforeValue()
      this.out.append("$v")
    }

    fun writeF64(v: f64) throws EncodeError {
      if (v.isNaN() || v.isInfinite()) throw EncodeError(message: "JSON has no representation for $v")
      this.beforeValue()
      this.out.append("$v")
    }

    fun writeBool(v: bool) throws EncodeError {
      this.beforeValue()
      this.out.append(if (v) "true" else "false")
    }

    fun writeString(v: string) throws EncodeError {
      this.beforeValue()
      writeQuoted(this.out, v)
    }

    fun writeNull() throws EncodeError {
      if (this.options.omitNulls && !this.pending.isEmpty()) {
        val _ = this.pending.removeAt(this.pending.len() - 1)
        return
      }
      this.beforeValue()
      this.out.append("null")
    }
  }
}

const HEX: string = "0123456789abcdef"

/// `s` as a JSON string literal, escapes and all.
fun writeQuoted(out: StringBuilder, s: string) {
  out.append("\"")
  // most strings need no escape: one scan, then one copy instead of a
  // push per byte
  var plain = true
  loop (i in 0..<s.len()) {
    val b = s.byteAt(i)
    if (b < 32 || b == '"' || b == '\\') {
      plain = false
      break
    }
  }
  if (plain) {
    out.append(s)
    out.append("\"")
    return
  }
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
        out.appendByte(HEX.byteAt((b >> 4).toI64()))
        out.appendByte(HEX.byteAt((b & 15).toI64()))
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

// What a panic on an empty container stack says: `more`, `nextKey` and
// `hasNext` run only inside a container `open` pushed.
const openContainer = "json: this runs only inside an open container"

/// The `codec.Decoder` that reads JSON text. A wrong type is a problem at the
/// path, recorded and read past; malformed text is a `DecodeError`.
// Where a JsonDecoder is reading. A decoder is a value that trait objects
// copy (D9), so the position lives behind a pointer every copy shares.
struct Cursor {
  var at: i64 = 0
}

public struct JsonDecoder {
  private src:      List<u8>
  private cursor:   *Cursor = &Cursor()  // the position, behind a handle
  private stack:    MutableList<Frame> = []
  private recorded: codec.Problems
  private options:  Options

  public static fun of(text: string, options: Options = Options()): JsonDecoder =
    JsonDecoder(src: text.bytes(), recorded: codec.Problems.capped(options.maxProblems), options)

  /// After the value: nothing but whitespace may follow.
  public fun end() throws DecodeError {
    this.skipSpace()
    if (this.pos() < this.src.len()) throw this.malformed("text after the value")
  }

  private fun pos(): i64 = this.cursor.at
  private fun setPos(p: i64) {
    this.cursor.at = p
  }

  private fun malformed(what: string): DecodeError =
    DecodeError(problems: this.recorded.list().concat([codec.Problem(path: "", message: "malformed JSON at byte ${this.pos()}: $what")]))

  private fun peekByte(): u8? = this.src.at(this.pos())

  private fun skipSpace() {
    loop {
      val b = this.peekByte() ?: return
      if (b != ' ' && b != '\t' && b != '\n' && b != '\r') return
      this.setPos(this.pos() + 1)
    }
  }

  /// Describes the next value for a problem message.
  private fun found(): string throws DecodeError = when (try this.peek()) {
    codec.Kind.Null   => "null"
    codec.Kind.Bool   => "a boolean"
    codec.Kind.Int    => "a number"
    codec.Kind.Float  => "a number"
    codec.Kind.String => "a string"
    codec.Kind.List   => "a list"
    codec.Kind.Object => "an object"
  }

  /// A value of the wrong type: recorded and read past.
  private fun wrong(expected: string) throws DecodeError {
    this.recorded.record(this.path(), "expected $expected, found ${try this.found()}")
    try this.skip()
  }

  /// The text of the number at the cursor, consumed.
  private fun number(): string throws DecodeError {
    val start = this.pos()
    if (this.peekByte() == '-') this.setPos(this.pos() + 1)
    this.digits()
    if (this.peekByte() == '.') {
      this.setPos(this.pos() + 1)
      this.digits()
    }
    val e = this.peekByte()
    if (e == 'e' || e == 'E') {
      this.setPos(this.pos() + 1)
      val sign = this.peekByte()
      if (sign == '+' || sign == '-') this.setPos(this.pos() + 1)
      this.digits()
    }
    if (this.pos() == start) throw this.malformed("expected a number")
    (listDecodeUtf8Range(this.src, start, this.pos()) ?: "")
  }

  private fun digits() {
    loop {
      val b = this.peekByte() ?: return
      if (b < '0' || b > '9') return
      this.setPos(this.pos() + 1)
    }
  }

  /// Whether the number at the cursor has a fraction or an exponent.
  private fun numberIsFloat(): bool {
    var i = this.pos()
    loop (i < this.src.len()) {
      val b = this.src.at(i) ?: break
      if (b == '.' || b == 'e' || b == 'E') return true
      if (b != '-' && (b < '0' || b > '9')) return false
      i += 1
    }
    false
  }

  private fun literal(word: string) throws DecodeError {
    loop (i in 0..<word.len()) {
      if (this.peekByte() != word.byteAt(i)) throw this.malformed("expected '$word'")
      this.setPos(this.pos() + 1)
    }
  }

  /// A string literal at the cursor, escapes decoded.
  private fun quoted(): string throws DecodeError {
    if (this.peekByte() != '"') throw this.malformed("expected a string")
    this.setPos(this.pos() + 1)
    // most strings have no escape: find the closing quote and copy once
    val first = this.pos()
    var end = first
    loop (end < this.src.len()) {
      val b = this.src.at(end) ?: break
      if (b == '"') {
        this.setPos(end + 1)
        return listDecodeUtf8Range(this.src, first, end) ?: throw this.malformed("a string that is not valid UTF-8")
      }
      if (b == '\\' || b < 32) break
      end += 1
    }
    val out: MutableList<u8> = []
    loop {
      val b = this.peekByte() ?: throw this.malformed("unterminated string")
      this.setPos(this.pos() + 1)
      if (b == '"') break
      if (b < 32) throw this.malformed("a control character inside a string")
      if (b != '\\') {
        out.push(b)
        continue
      }
      val esc = this.peekByte() ?: throw this.malformed("unterminated escape")
      this.setPos(this.pos() + 1)
      when {
        esc == '"' || esc == '\\' || esc == '/' => out.push(esc)
        esc == 'n' => out.push('\n')
        esc == 't' => out.push('\t')
        esc == 'r' => out.push('\r')
        esc == 'b' => out.push(8)
        esc == 'f' => out.push(12)
        esc == 'u' => {
          var cp = try this.hex4()
          if (utf8.isSurrogate(cp)) {
            // a surrogate pair: the low half follows as another \u escape
            if (this.peekByte() != '\\' || this.src.at(this.pos() + 1) != 'u') {
              throw this.malformed("a lone surrogate in a \\u escape")
            }
            this.setPos(this.pos() + 2)
            val low = try this.hex4()
            cp = utf8.combineSurrogates(cp, low) ?: throw this.malformed("a lone surrogate in a \\u escape")
          }
          val _ = utf8.encodeTo(out, cp)
        }
        else => throw this.malformed("unknown escape '\\${[esc].decodeUtf8() ?: "?"}'")
      }
    }
    out.decodeUtf8() ?: throw this.malformed("a string that is not valid UTF-8")
  }

  private fun hex4(): i64 throws DecodeError {
    var n: i64 = 0
    loop (_ in 0..<4) {
      val b = (this.peekByte() ?: throw this.malformed("short \\u escape")).toI64()
      val d = when {
        b >= '0' && b <= '9' => b - '0'
        b >= 'a' && b <= 'f' => b - 'a' + 10
        b >= 'A' && b <= 'F' => b - 'A' + 10
        else                 => throw this.malformed("bad \\u escape")
      }
      n = n * 16 + d
      this.setPos(this.pos() + 1)
    }
    n
  }

  /// Between members of the open container: the separator, or its end.
  /// True when another member follows.
  private fun more(closer: u8): bool throws DecodeError {
    val top = this.stack.ref(-1) ?: panic(openContainer)
    if (top.phantom) return false
    this.skipSpace()
    val b = this.peekByte() ?: throw this.malformed("unterminated container")
    if (b == closer) return false
    if (top.index == 0) return true
    if (b != ',') throw this.malformed("expected ',' or '${[closer].decodeUtf8() ?: "?"}'")
    this.setPos(this.pos() + 1)
    this.skipSpace()
    if (this.peekByte() == closer) throw this.malformed("a trailing comma")
    true
  }

  private fun open(isList: bool, opener: u8, expected: string) throws DecodeError {
    if (this.stack.len() >= this.options.maxDepth) throw this.malformed(recursion.tooDeepMessage(this.options.maxDepth))
    this.skipSpace()
    if (this.peekByte() == opener) {
      this.setPos(this.pos() + 1)
      this.stack.push(Frame(isList))
      return
    }
    try this.wrong(expected)
    this.stack.push(Frame(isList, phantom: true))
  }

  private fun closeContainer(closer: u8) throws DecodeError {
    val top = this.stack.removeAt(this.stack.len() - 1)
    if (top.phantom) return
    this.skipSpace()
    if (this.peekByte() != closer) throw this.malformed("expected '${[closer].decodeUtf8() ?: "?"}'")
    this.setPos(this.pos() + 1)
  }

  implement codec.Decoder {
    fun format(): string = "json"
    override fun enums(): codec.EnumStyle = this.options.enums
    override fun durations(): codec.DurationStyle = this.options.durations
    override fun keys(): codec.KeyStyle = this.options.keys

    fun peek(): codec.Kind throws DecodeError {
      this.skipSpace()
      val b = this.peekByte() ?: throw this.malformed("unexpected end of input")
      when {
        b == '{' => codec.Kind.Object
        b == '[' => codec.Kind.List
        b == '"' => codec.Kind.String
        b == 't' || b == 'f' => codec.Kind.Bool
        b == 'n' => codec.Kind.Null
        b == '-' || (b >= '0' && b <= '9') => if (this.numberIsFloat()) codec.Kind.Float else codec.Kind.Int
        else => throw this.malformed("unexpected character")
      }
    }

    fun beginObject() throws DecodeError = try this.open(false, '{', "an object")

    fun nextKey(): string? throws DecodeError {
      if (!(try this.more('}'))) return null
      val top = this.stack.ref(-1) ?: panic(openContainer)
      this.skipSpace()
      val k = try this.quoted()
      this.skipSpace()
      if (this.peekByte() != ':') throw this.malformed("expected ':' after a key")
      this.setPos(this.pos() + 1)
      top.key = k
      top.index += 1
      k
    }

    fun endObject() throws DecodeError = try this.closeContainer('}')

    fun beginList() throws DecodeError = try this.open(true, '[', "a list")

    fun hasNext(): bool throws DecodeError {
      val has = try this.more(']')
      if (has) {
        val top = this.stack.ref(-1) ?: panic(openContainer)
        top.index += 1
      }
      has
    }

    fun endList() throws DecodeError = try this.closeContainer(']')

    fun readI64(): i64 throws DecodeError {
      if (try this.peek() != codec.Kind.Int) {
        try this.wrong("an integer")
        return 0
      }
      // the digits straight into the value; the text only for a number
      // too big to hold
      val start = this.pos()
      val negative = this.peekByte() == '-'
      if (negative) this.setPos(start + 1)
      var n: i64 = 0
      var fits = true
      loop {
        val b = this.peekByte() ?: break
        if (b < '0' || b > '9') break
        val d = (b - '0').toI64()
        // accumulate negatively: -9223372036854775808 has no positive twin
        val next = n.checkedMul(10)?.checkedSub(d)
        if (next == null) fits = false else n = next
        this.setPos(this.pos() + 1)
      }
      if (this.pos() == start + (if (negative) 1 else 0)) throw this.malformed("expected a number")
      if (!fits || !negative && n == -9223372036854775807 - 1) {
        val text = listDecodeUtf8Range(this.src, start, this.pos()) ?: ""
        this.recorded.record(this.path(), "$text does not fit an integer")
        return 0
      }
      if (negative) n else -n
    }

    fun readU64(): u64 throws DecodeError {
      val n = try this.readI64()
      if (n < 0) {
        this.recorded.record(this.path(), "$n is negative")
        return 0
      }
      n.wrapU64()
    }

    fun readF64(): f64 throws DecodeError {
      val k = try this.peek()
      if (k != codec.Kind.Int && k != codec.Kind.Float) {
        try this.wrong("a number")
        return 0.0
      }
      val text = try this.number()
      val n = text.toF64()
      if (n == null) {
        this.recorded.record(this.path(), "$text is not a number")
        return 0.0
      }
      if (n.isInfinite()) {
        // `1e400` is valid JSON grammar, but no f64 holds it; accepting it
        // as infinity would read a document that cannot be written back
        // (RFC 8259 §6 lets a parser limit range; Go refuses it too)
        this.recorded.record(this.path(), "$text is out of range for a 64-bit float")
        return 0.0
      }
      n
    }

    fun readBool(): bool throws DecodeError {
      if (try this.peek() != codec.Kind.Bool) {
        try this.wrong("a boolean")
        return false
      }
      if (this.peekByte() == 't') {
        try this.literal("true")
        return true
      }
      try this.literal("false")
      false
    }

    fun readString(): string throws DecodeError {
      if (try this.peek() != codec.Kind.String) {
        try this.wrong("a string")
        return ""
      }
      try this.quoted()
    }

    fun readNull() throws DecodeError {
      if (try this.peek() != codec.Kind.Null) {
        return try this.wrong("null")
      }
      try this.literal("null")
    }

    fun skip() throws DecodeError {
      when (try this.peek()) {
        codec.Kind.Null   => try this.literal("null")
        codec.Kind.Bool   => {
          val _ = try this.readBool()
        }
        codec.Kind.Int    => {
          val _ = try this.number()
        }
        codec.Kind.Float  => {
          val _ = try this.number()
        }
        codec.Kind.String => {
          val _ = try this.quoted()
        }
        codec.Kind.List   => {
          try this.beginList()
          loop (try this.hasNext()) try this.skip()
          try this.endList()
        }
        codec.Kind.Object => {
          try this.beginObject()
          loop {
            val _ = try this.nextKey() ?: break
            try this.skip()
          }
          try this.endObject()
        }
      }
    }

    fun path(): string {
      var out = ""
      loop (fr in this.stack) {
        out = if (fr.isList) codec.indexPath(out, fr.index - 1) else codec.childPath(out, fr.key)
      }
      out
    }

    fun problemAt(path: string, message: string) = this.recorded.record(path, message)

    fun problems(): List<codec.Problem> = this.recorded.list()
  }
}
