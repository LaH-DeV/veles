// Prelude — Codable (D58). A type that travels to the outside world — JSON
// in and out, a database row, the environment — implements `Encodable`
// and `Decodable`, together `Codable`, and almost always asks the compiler
// to write them: `implement Codable` in a struct body (or `implement Codable for
// pkg.Type` at top level) derives both from the fields. The traits are
// format-agnostic: a format is one implementation of `Encoder` and
// `Decoder` (`std/json` is the first), and the same derived code drives
// every format. The traits are a flat event stream — begin/key/end and the
// primitives — and a value encodes *itself* (`self.id.encode(to)`), so
// nothing here is generic and nothing is boxed on the way out.

// ---------------------------------------------------------------------------
// errors

/// One thing wrong with a document being decoded: where (`user.address[2].zip`)
/// and what.
public struct Problem {
  public path:    string
  public message: string

  /// The path as an RFC 6901 JSON pointer: `/user/address/2/zip`.
  public fun pointer(): string {
    val out = stringBuilder()
    loop (part in splitPath(self.path)) {
      out.append("/")
      out.append(part.replace("~", "~0").replace("/", "~1"))
    }
    out.toString()
  }

  public fun toString(): string = if (self.path.isEmpty()) self.message else "${self.path}: ${self.message}"
}

/// The parts of a path: `a.b[2].c` is `a`, `b`, `2`, `c`; a quoted key
/// `a["x.y"]` is one part.
fun splitPath(path: string): List<string> {
  val parts: MutableList<string> = []
  val cur = stringBuilder()
  var i = 0
  val n = path.len()
  loop (i < n) {
    val b = path.byteAt(i)
    when {
      b == '.' => {
        if (!cur.isEmpty()) {
          parts.push(cur.toString())
          cur.clear()
        }
        i += 1
      }
      b == '[' => {
        if (!cur.isEmpty()) {
          parts.push(cur.toString())
          cur.clear()
        }
        i += 1
        if (i < n && path.byteAt(i) == '"') {
          i += 1
          loop (i < n && path.byteAt(i) != '"') {
            cur.appendByte(path.byteAt(i))
            i += 1
          }
          i += 1  // the closing quote
        } else {
          loop (i < n && path.byteAt(i) != ']') {
            cur.appendByte(path.byteAt(i))
            i += 1
          }
        }
        parts.push(cur.toString())
        cur.clear()
        i += 1  // the ]
      }
      else     => {
        cur.appendByte(b)
        i += 1
      }
    }
  }
  if (!cur.isEmpty()) parts.push(cur.toString())
  parts.toList()
}

/// Encoding failed: the sink refused a write, or a value cannot be
/// represented (a NaN in JSON, a nested object in a row).
public error EncodeError {
  public message: string
  public path:    string = ""
  fun message(): string = if (self.path.isEmpty()) self.message else "${self.path}: ${self.message}"
}

/// Decoding failed. Every problem found is listed — a wrong type here, a
/// missing field there — so a client fixes its document in one round trip;
/// a malformed document is one problem, since nothing follows it.
public error DecodeError {
  public problems: List<Problem>
  fun message(): string = self.problems.map(p => p.toString()).join("\n")
}

// ---------------------------------------------------------------------------
// what the derived code talks to

/// How an enum travels: its name (`"Active"`) or its number (`1`). A
/// property of the format, not of the enum: JSON wants names, a `smallint`
/// column wants numbers.
public enum EnumStyle {
  Name
  Number
}

/// How a `Duration` travels. Like `EnumStyle`, a property of the format
/// and the API it talks to, not of the type: the same struct may go to a
/// gRPC gateway as `"90.5s"` and to a Java service as `"PT1M30.5S"`.
/// Decoding is strict per style — a number where `Seconds` wants text is a
/// problem, as it is for enums.
public enum DurationStyle {
  /// `"90.5s"`: seconds with up to nine decimals and an `s` — protobuf's
  /// JSON form for `google.protobuf.Duration`, and what Go's
  /// `time.ParseDuration` reads. Exact. The default.
  Seconds
  /// `"PT1M30.5S"`, `"-PT0.25S"`: ISO 8601, for Java, .NET and JavaScript's
  /// Temporal. Days are read (as 24 hours); years and months are refused,
  /// because they have no fixed length.
  Iso8601
  /// `"1m30.5s"`: the `Display` text, which `Duration.parse` reads back.
  Text
  /// `90500000000`: whole nanoseconds, a number. Exact.
  Nanos
  /// `90500`: milliseconds, a number — whole when the duration is, with a
  /// fraction otherwise. What many JavaScript APIs expect.
  Millis
}

/// How an encoder spells the keys it is given: as written, `snake_case`
/// or `camelCase`. A policy for a whole API, never an attribute on a type.
public enum KeyStyle {
  AsWritten
  SnakeCase
  CamelCase
}

/// `name` spelled in `style`: `passwordHash` → `password_hash` → `passwordHash`.
public fun styleKey(name: string, style: KeyStyle): string = when (style) {
  KeyStyle.AsWritten => name
  KeyStyle.SnakeCase => {
    val out = stringBuilder()
    var i = 0
    loop (i < name.len()) {
      val b = name.byteAt(i)
      if (b >= 'A' && b <= 'Z') {
        if (i > 0) out.append("_")
        out.appendByte(b + 32)
      } else {
        out.appendByte(b)
      }
      i += 1
    }
    out.toString()
  }
  KeyStyle.CamelCase => {
    val out = stringBuilder()
    var up = false
    var i = 0
    loop (i < name.len()) {
      val b = name.byteAt(i)
      if (b == '_') {
        up = true
      } else if (up && b >= 'a' && b <= 'z') {
        out.appendByte(b - 32)
        up = false
      } else {
        out.appendByte(b)
        up = false
      }
      i += 1
    }
    out.toString()
  }
}

/// What the next value in a document is, before it is read.
public enum Kind {
  Null
  Bool
  Int
  Float
  String
  List
  Object
}

/// The sink a value writes itself into: objects and lists open and close
/// around their members, a key precedes each object member, and the
/// primitives write one value. A format implements this once.
public trait Encoder {
  /// The format's name (`"json"`, `"db"`, `"env"`): what `@key(json: ...)`
  /// selects on and what a library format chooses for itself.
  fun format(): string
  /// How durations are written; `Seconds` unless the format says otherwise.
  fun durations(): DurationStyle = DurationStyle.Seconds
  /// How enums are written; `Name` unless the format says otherwise.
  fun enums(): EnumStyle = EnumStyle.Name
  /// How field names are spelled as keys; derived code applies it to a
  /// field's own name, never to a `@key` the author wrote.
  fun keys(): KeyStyle = KeyStyle.AsWritten
  fun beginObject() throws EncodeError
  fun key(name: string) throws EncodeError
  fun endObject() throws EncodeError
  fun beginList() throws EncodeError
  fun endList() throws EncodeError
  fun writeI64(v: i64) throws EncodeError
  fun writeU64(v: u64) throws EncodeError
  fun writeF64(v: f64) throws EncodeError
  fun writeBool(v: bool) throws EncodeError
  fun writeString(v: string) throws EncodeError
  fun writeNull() throws EncodeError
}

/// The source a value reads itself from, mirroring `Encoder`. A value of
/// the wrong type is a *problem*, not an error: the decoder records it at
/// the current path, consumes the value and returns a zero, so decoding
/// goes on and every problem is found; `DecodeError` is thrown only for
/// what nothing can continue from (a malformed document) and, by derived
/// code, when a value cannot be built at all (a required field missing).
public trait Decoder {
  fun format(): string
  fun durations(): DurationStyle = DurationStyle.Seconds
  fun enums(): EnumStyle = EnumStyle.Name
  fun keys(): KeyStyle = KeyStyle.AsWritten
  /// What comes next, without consuming it.
  fun peek(): Kind throws DecodeError
  fun beginObject() throws DecodeError
  /// The next key of the open object, or `null` at its end.
  fun nextKey(): string? throws DecodeError
  fun endObject() throws DecodeError
  fun beginList() throws DecodeError
  /// Whether the open list has another element.
  fun hasNext(): bool throws DecodeError
  fun endList() throws DecodeError
  fun readI64(): i64 throws DecodeError
  fun readU64(): u64 throws DecodeError
  fun readF64(): f64 throws DecodeError
  fun readBool(): bool throws DecodeError
  fun readString(): string throws DecodeError
  fun readNull() throws DecodeError
  /// Skips the next value, whatever it is (an unknown key's).
  fun skip() throws DecodeError
  /// Where the decoder is: `user.address[2].zip`.
  fun path(): string
  /// Records a problem at `path`.
  fun problemAt(path: string, message: string)
  /// Records a problem at the current path.
  fun problem(message: string) = self.problemAt(self.path(), message)
  /// Every problem recorded so far.
  fun problems(): List<Problem>
}

/// A value that writes itself into an `Encoder`. Derived by `implement Encodable`
/// (or `implement Codable`) from a struct's fields, a sealed trait's variants;
/// enums and the built-in types have it already.
public trait Encodable {
  fun encode(to: Encoder) throws EncodeError
}

/// A value that reads itself from a `Decoder`; the inverse of `Encodable`.
public trait Decodable {
  static fun decode(from: Decoder): Self throws DecodeError
}

/// Both directions: the one line a DTO writes.
public trait Codable : Encodable + Decodable

/// The path of a member `key` under `base`.
public fun childPath(base: string, key: string): string {
  val plain = !key.isEmpty() && key.indexOf(".") < 0 && key.indexOf("[") < 0 && key.indexOf("]") < 0 && key.indexOf("\"") < 0
  when {
    plain && base.isEmpty() => key
    plain                   => "$base.$key"
    else                    => base + "[\"" + key.replace("\"", "\\\"") + "\"]"
  }
}

/// The path of element `i` under `base`.
public fun indexPath(base: string, i: i64): string = "$base[$i]"

/// A path found under `base`: `joinPath("user", "address[2]")` is
/// `user.address[2]`.
public fun joinPath(base: string, rel: string): string = when {
  rel.isEmpty()       => base
  base.isEmpty()      => rel
  rel.startsWith("[") => base + rel
  else                => "$base.$rel"
}

/// A problem list a decoder keeps: `record` adds one at a path, `list`
/// hands them out, and `cap` stops a hostile document growing it forever.
public struct Problems {
  private items: MutableList<Problem> = []
  public cap:    i64 = 100

  /// A list that keeps at most `cap` problems.
  public static fun capped(cap: i64): Problems = Problems(cap)

  public fun record(path: string, message: string) {
    if (self.items.len() < self.cap) self.items.push(Problem(path, message))
  }

  public fun isEmpty(): bool = self.items.isEmpty()
  public fun list(): List<Problem> = self.items.toList()
}

/// The decoded value, or the problems found: what `json.decode` and every
/// other format's entry point ends with.
public fun finish<T>(from: Decoder, value: T): T throws DecodeError {
  val problems = from.problems()
  if (!problems.isEmpty()) throw DecodeError(problems)
  value
}

// ---------------------------------------------------------------------------
// the built-in types

implement Encodable for i8 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(self as i64)
}
implement Encodable for i16 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(self as i64)
}
implement Encodable for i32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(self as i64)
}
implement Encodable for i64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(self)
}
implement Encodable for isize {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(self as i64)
}
implement Encodable for u8 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(self as u64)
}
implement Encodable for u16 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(self as u64)
}
implement Encodable for u32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(self as u64)
}
implement Encodable for u64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(self)
}
implement Encodable for usize {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(self as u64)
}
implement Encodable for f32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeF64(self as f64)
}
implement Encodable for f64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeF64(self)
}
implement Encodable for bool {
  fun encode(to: Encoder) throws EncodeError = try to.writeBool(self)
}
implement Encodable for string {
  fun encode(to: Encoder) throws EncodeError = try to.writeString(self)
}

implement Decodable for i8 {
  static fun decode(from: Decoder): i8 throws DecodeError = try narrowI64(from, -128, 127) as i8
}
implement Decodable for i16 {
  static fun decode(from: Decoder): i16 throws DecodeError = try narrowI64(from, -32768, 32767) as i16
}
implement Decodable for i32 {
  static fun decode(from: Decoder): i32 throws DecodeError = try narrowI64(from, -2147483648, 2147483647) as i32
}
implement Decodable for i64 {
  static fun decode(from: Decoder): i64 throws DecodeError = try from.readI64()
}
implement Decodable for isize {
  static fun decode(from: Decoder): isize throws DecodeError = try from.readI64() as isize
}
implement Decodable for u8 {
  static fun decode(from: Decoder): u8 throws DecodeError = try narrowU64(from, 255) as u8
}
implement Decodable for u16 {
  static fun decode(from: Decoder): u16 throws DecodeError = try narrowU64(from, 65535) as u16
}
implement Decodable for u32 {
  static fun decode(from: Decoder): u32 throws DecodeError = try narrowU64(from, 4294967295) as u32
}
implement Decodable for u64 {
  static fun decode(from: Decoder): u64 throws DecodeError = try from.readU64()
}
implement Decodable for usize {
  static fun decode(from: Decoder): usize throws DecodeError = try from.readU64() as usize
}
implement Decodable for f32 {
  static fun decode(from: Decoder): f32 throws DecodeError = try from.readF64() as f32
}
implement Decodable for f64 {
  static fun decode(from: Decoder): f64 throws DecodeError = try from.readF64()
}
implement Decodable for bool {
  static fun decode(from: Decoder): bool throws DecodeError = try from.readBool()
}
implement Decodable for string {
  static fun decode(from: Decoder): string throws DecodeError = try from.readString()
}

/// An integer that must fit a narrower type: out of range is a problem.
fun narrowI64(from: Decoder, min: i64, max: i64): i64 throws DecodeError {
  val v = try from.readI64()
  if (v < min || v > max) {
    from.problem("$v is out of range ($min to $max)")
    return 0
  }
  v
}

fun narrowU64(from: Decoder, max: u64): u64 throws DecodeError {
  val v = try from.readU64()
  if (v > max) {
    from.problem("$v is out of range (0 to $max)")
    return 0
  }
  v
}

implement<T: Encodable> Encodable for T? {
  fun encode(to: Encoder) throws EncodeError {
    if (self == null) try to.writeNull() else try self.encode(to)
  }
}
implement<T: Decodable> Decodable for T? {
  static fun decode(from: Decoder): T? throws DecodeError {
    if (try from.peek() == Kind.Null) {
      try from.readNull()
      return null
    }
    try T.decode(from)
  }
}

implement<T: Encodable> Encodable for List<T> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginList()
    loop (x in self) try x.encode(to)
    try to.endList()
  }
}
implement<T: Encodable> Encodable for MutableList<T> {
  fun encode(to: Encoder) throws EncodeError = try self.toList().encode(to)
}
implement<T: Decodable> Decodable for List<T> {
  static fun decode(from: Decoder): List<T> throws DecodeError = (try MutableList<T>.decode(from)).toList()
}
implement<T: Decodable> Decodable for MutableList<T> {
  static fun decode(from: Decoder): MutableList<T> throws DecodeError {
    val out: MutableList<T> = []
    try from.beginList()
    loop (try from.hasNext()) out.push(try T.decode(from))
    try from.endList()
    out
  }
}

implement<V: Encodable> Encodable for Map<string, V> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginObject()
    loop ((k, v) in self.entries()) {
      try to.key(k)
      try v.encode(to)
    }
    try to.endObject()
  }
}
implement<V: Encodable> Encodable for MutableMap<string, V> {
  fun encode(to: Encoder) throws EncodeError = try self.toMap().encode(to)
}
implement<V: Decodable> Decodable for Map<string, V> {
  static fun decode(from: Decoder): Map<string, V> throws DecodeError = (try MutableMap<string, V>.decode(from)).toMap()
}
implement<V: Decodable> Decodable for MutableMap<string, V> {
  static fun decode(from: Decoder): MutableMap<string, V> throws DecodeError {
    val out: MutableMap<string, V> = [:]
    try from.beginObject()
    loop {
      val k = try from.nextKey() ?: break
      out.set(k, try V.decode(from))
    }
    try from.endObject()
    out
  }
}

// ---------------------------------------------------------------------------
// Value: a document as a tree

/// A decoded document with no type in mind — what `json.parse` returns and
/// what a derived sealed-trait decoder buffers an object into before it
/// knows the variant. It is `Codable` itself, so it passes through any
/// format.
public sealed trait Value
public struct VNull : Value { }
public struct VBool : Value {
  public value: bool
}
public struct VInt : Value {
  public value: i64
}
public struct VFloat : Value {
  public value: f64
}
public struct VString : Value {
  public value: string
}
public struct VList : Value {
  public items: List<Value>
}
public struct VObject : Value {
  public fields: Map<string, Value>
}

extend Value {
  /// The member `key` of an object, or `null`.
  public fun get(key: string): Value? = when (self) {
    is VObject => self.fields.get(key)
    else       => null
  }

  /// Element `i` of a list, or `null`.
  public fun at(i: i64): Value? = when (self) {
    is VList => self.items.at(i)
    else     => null
  }

  public fun asString(): string? = when (self) {
    is VString => self.value
    else       => null
  }

  public fun asI64(): i64? = when (self) {
    is VInt   => self.value
    is VFloat => if (self.value == (self.value as i64) as f64) self.value as i64 else null
    else      => null
  }

  public fun asF64(): f64? = when (self) {
    is VInt   => self.value as f64
    is VFloat => self.value
    else      => null
  }

  public fun asBool(): bool? = when (self) {
    is VBool => self.value
    else     => null
  }

  public fun isNull(): bool = self is VNull
}

implement Encodable for Value {
  fun encode(to: Encoder) throws EncodeError = when (self) {
    is VNull   => try to.writeNull()
    is VBool   => try to.writeBool(self.value)
    is VInt    => try to.writeI64(self.value)
    is VFloat  => try to.writeF64(self.value)
    is VString => try to.writeString(self.value)
    is VList   => try self.items.encode(to)
    is VObject => try self.fields.encode(to)
  }
}

implement Decodable for Value {
  static fun decode(from: Decoder): Value throws DecodeError = when (try from.peek()) {
    Kind.Null   => {
      try from.readNull()
      VNull()
    }
    Kind.Bool   => VBool(value: try from.readBool())
    Kind.Int    => VInt(value: try from.readI64())
    Kind.Float  => VFloat(value: try from.readF64())
    Kind.String => VString(value: try from.readString())
    Kind.List   => VList(items: try List<Value>.decode(from))
    Kind.Object => VObject(fields: try Map<string, Value>.decode(from))
  }
}

// ---------------------------------------------------------------------------
// decoding from a Value

/// One open container while walking a tree: an object's remaining keys or
/// a list's remaining items, and the value the next `read` consumes.
struct Frame {
  var path:    string
  var keys:    List<string> = []
  var values:  List<Value> = []
  var next:    i64 = 0
  var pending: Value? = null  // the value of the key just handed out
  isList:      bool
}

/// A `Decoder` over a `Value`: what a derived decoder uses to read a
/// variant out of an object it buffered, and what `json.decodeValue<T>`
/// uses to type a tree after the fact.
public struct ValueDecoder {
  private root:          Value
  private stack:         MutableList<Frame> = []
  private recorded:      Problems = Problems()
  private started:       bool = false
  private name:          string = "json"
  private style:         EnumStyle = EnumStyle.Name
  private keyStyle:      KeyStyle = KeyStyle.AsWritten
  private durationStyle: DurationStyle = DurationStyle.Seconds
  private maxDepth:      i64 = maxRecursionDepth

  public static fun of(v: Value, format: string = "json", enums: EnumStyle = EnumStyle.Name, keys: KeyStyle = KeyStyle.AsWritten, durations: DurationStyle = DurationStyle.Seconds, maxDepth: i64 = maxRecursionDepth): ValueDecoder =
    ValueDecoder(root: v, name: format, style: enums, keyStyle: keys, durationStyle: durations, maxDepth)

  /// The value about to be read.
  private fun current(): Value {
    if (self.stack.isEmpty()) return self.root
    val top = self.stack.refOrPanic(self.stack.lastIndex())
    if (top.isList) return top.values.at(top.next) ?: VNull()
    top.pending ?: VNull()
  }

  /// A read consumed the current value.
  private fun advance() {
    if (self.stack.isEmpty()) return
    val top = self.stack.refOrPanic(self.stack.lastIndex())
    if (top.isList) top.next += 1 else top.pending = null
  }

  private fun here(): string {
    if (self.stack.isEmpty()) return ""
    val top = self.stack.refOrPanic(self.stack.lastIndex())
    if (top.isList) return indexPath(top.path, top.next)
    childPath(top.path, top.keys.at(top.next - 1) ?: "")
  }

  private fun wrong(expected: string): Value {
    val v = self.current()
    self.recorded.record(self.here(), "expected $expected, found ${kindName(v)}")
    self.advance()
    v
  }

  /// A tree nests as deep as whoever built it wanted, and walking it is
  /// recursion, so it is bounded like every other walk over input someone
  /// else produced (§2). Unlike a wrong type, this is not a problem to
  /// record and carry on from: the frames are already on the stack.
  private fun checkDepth() throws DecodeError {
    if (self.stack.len() < self.maxDepth) return
    throw DecodeError(problems: self.recorded.list().concat([Problem(path: self.here(), message: tooDeepMessage(self.maxDepth))]))
  }

  implement Decoder {
    fun format(): string = self.name
    override fun enums(): EnumStyle = self.style
    override fun durations(): DurationStyle = self.durationStyle
    override fun keys(): KeyStyle = self.keyStyle

    fun peek(): Kind throws DecodeError = when (self.current()) {
      is VNull   => Kind.Null
      is VBool   => Kind.Bool
      is VInt    => Kind.Int
      is VFloat  => Kind.Float
      is VString => Kind.String
      is VList   => Kind.List
      is VObject => Kind.Object
    }

    fun beginObject() throws DecodeError {
      try self.checkDepth()
      val v = self.current()
      when (v) {
        is VObject => {
          self.stack.push(Frame(path: self.here(), keys: v.fields.keys(), values: v.fields.values(), isList: false))
        }
        else       => {
          val _ = self.wrong("an object")
          self.stack.push(Frame(path: self.here(), isList: false))
        }
      }
    }

    fun nextKey(): string? throws DecodeError {
      val top = self.stack.refOrPanic(self.stack.lastIndex())
      if (top.next >= top.keys.len()) return null
      val k = top.keys.atOrPanic(top.next)
      top.pending = top.values.at(top.next)
      top.next += 1
      k
    }

    fun endObject() throws DecodeError {
      val _ = self.stack.removeAt(self.stack.lastIndex())
      self.advance()
    }

    fun beginList() throws DecodeError {
      try self.checkDepth()
      val v = self.current()
      when (v) {
        is VList => self.stack.push(Frame(path: self.here(), values: v.items, isList: true))
        else     => {
          val _ = self.wrong("a list")
          self.stack.push(Frame(path: self.here(), isList: true))
        }
      }
    }

    fun hasNext(): bool throws DecodeError {
      val top = self.stack.refOrPanic(self.stack.lastIndex())
      top.next < top.values.len()
    }

    fun endList() throws DecodeError {
      val _ = self.stack.removeAt(self.stack.lastIndex())
      self.advance()
    }

    fun readI64(): i64 throws DecodeError {
      val v = self.current()
      val n = v.asI64()
      if (n == null) {
        val _ = self.wrong("an integer")
        return 0
      }
      self.advance()
      n
    }

    fun readU64(): u64 throws DecodeError {
      val n = try self.readI64()
      if (n < 0) {
        self.recorded.record(self.here(), "$n is negative")
        return 0
      }
      n as u64
    }

    fun readF64(): f64 throws DecodeError {
      val v = self.current()
      val n = v.asF64()
      if (n == null) {
        val _ = self.wrong("a number")
        return 0.0
      }
      self.advance()
      n
    }

    fun readBool(): bool throws DecodeError {
      val v = self.current()
      val b = v.asBool()
      if (b == null) {
        val _ = self.wrong("a boolean")
        return false
      }
      self.advance()
      b
    }

    fun readString(): string throws DecodeError {
      val v = self.current()
      val s = v.asString()
      if (s == null) {
        val _ = self.wrong("a string")
        return ""
      }
      self.advance()
      s
    }

    fun readNull() throws DecodeError {
      if (!self.current().isNull()) {
        val _ = self.wrong("null")
        return
      }
      self.advance()
    }

    fun skip() throws DecodeError = self.advance()

    fun path(): string = self.here()

    fun problemAt(path: string, message: string) = self.recorded.record(path, message)

    fun problems(): List<Problem> = self.recorded.list()
  }
}

fun kindName(v: Value): string = when (v) {
  is VNull   => "null"
  is VBool   => "a boolean"
  is VInt    => "a number"
  is VFloat  => "a number"
  is VString => "a string"
  is VList   => "a list"
  is VObject => "an object"
}

// ---------------------------------------------------------------------------
// encoding into a Value

/// An open container being built.
struct Building {
  var keys:   MutableList<string> = []
  var values: MutableList<Value> = []
  var key:    string = ""
  isList:     bool
}

/// The state of a ValueEncoder, behind a pointer: an encoder is handed
/// around as a trait object, which boxes a copy of the struct (D9), so
/// what it builds must live where every copy can reach it.
struct ValueEncoderState {
  var result: Value? = null
}

/// An `Encoder` that builds a `Value`: `json.toValue(x)`, and the way a
/// value is re-encoded through another format without a document in
/// between.
public struct ValueEncoder {
  private stack:         MutableList<Building> = []
  private state:         *ValueEncoderState
  private name:          string = "json"
  private style:         EnumStyle = EnumStyle.Name
  private keyStyle:      KeyStyle = KeyStyle.AsWritten
  private durationStyle: DurationStyle = DurationStyle.Seconds
  private maxDepth:      i64 = maxRecursionDepth

  public static fun of(format: string = "json", enums: EnumStyle = EnumStyle.Name, keys: KeyStyle = KeyStyle.AsWritten, durations: DurationStyle = DurationStyle.Seconds, maxDepth: i64 = maxRecursionDepth): ValueEncoder =
    ValueEncoder(state: &ValueEncoderState(), name: format, style: enums, keyStyle: keys, durationStyle: durations, maxDepth)

  /// The value built, once one whole value has been written.
  public fun value(): Value = self.state.result ?: VNull()

  /// The same bound the text encoders keep (§2): a value nested deeper than
  /// this is refused rather than walked, whichever direction it is going.
  private fun checkDepth() throws EncodeError {
    if (self.stack.len() >= self.maxDepth) throw EncodeError(message: tooDeepMessage(self.maxDepth))
  }

  private fun put(v: Value) {
    if (self.stack.isEmpty()) {
      self.state.result = v
      return
    }
    val top = self.stack.refOrPanic(self.stack.lastIndex())
    if (!top.isList) top.keys.push(top.key)
    top.values.push(v)
  }

  implement Encoder {
    fun format(): string = self.name
    override fun enums(): EnumStyle = self.style
    override fun durations(): DurationStyle = self.durationStyle
    override fun keys(): KeyStyle = self.keyStyle

    fun beginObject() throws EncodeError {
      try self.checkDepth()
      self.stack.push(Building(isList: false))
    }

    fun key(name: string) throws EncodeError {
      self.stack.refOrPanic(self.stack.lastIndex()).key = name
    }

    fun endObject() throws EncodeError {
      val top = self.stack.removeAt(self.stack.lastIndex())
      val fields: MutableMap<string, Value> = [:]
      var i = 0
      loop (i < top.keys.len()) {
        fields.set(top.keys.atOrPanic(i), top.values.atOrPanic(i))
        i += 1
      }
      self.put(VObject(fields: fields.toMap()))
    }

    fun beginList() throws EncodeError {
      try self.checkDepth()
      self.stack.push(Building(isList: true))
    }

    fun endList() throws EncodeError {
      val top = self.stack.removeAt(self.stack.lastIndex())
      self.put(VList(items: top.values.toList()))
    }

    fun writeI64(v: i64) throws EncodeError = self.put(VInt(value: v))
    fun writeU64(v: u64) throws EncodeError = self.put(VInt(value: v as i64))
    fun writeF64(v: f64) throws EncodeError = self.put(VFloat(value: v))
    fun writeBool(v: bool) throws EncodeError = self.put(VBool(value: v))
    fun writeString(v: string) throws EncodeError = self.put(VString(value: v))
    fun writeNull() throws EncodeError = self.put(VNull())
  }
}
