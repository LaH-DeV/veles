// Prelude — Codable (D58). A type that travels to the outside world — JSON
// in and out, a database row, the environment — implements `Encodable`
// and `Decodable`, together `Codable`, and almost always asks the compiler
// to write them: `implement Codable` in a struct body (or `implement Codable for
// pkg.Type` at top level) derives both from the fields. The traits are
// format-agnostic: a format is one implementation of `Encoder` and
// `Decoder` (`std/json` is the first), and the same derived code drives
// every format. The traits are a flat event stream — begin/key/end and the
// primitives — and a value encodes *itself* (`this.id.encode(to)`), so
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
    val out = StringBuilder()
    loop (part in splitPath(this.path)) {
      out.append("/")
      out.append(part.replace("~", "~0").replace("/", "~1"))
    }
    out.toString()
  }

  public fun toString(): string = if (this.path.isEmpty()) this.message else "${this.path}: ${this.message}"
}

// What a panic on an empty frame stack says: derived bodies call `key`,
// `nextKey` and `hasNext` only between a `begin…` and its `end…`.
const openFrame = "codec: a derived body calls this only between a begin and its end, so a frame is open"

/// The parts of a path: `a.b[2].c` is `a`, `b`, `2`, `c`; a quoted key
/// `a["x.y"]` is one part.
fun splitPath(path: string): List<string> {
  val parts: MutableList<string> = []
  val cur = StringBuilder()
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
  fun message(): string = if (this.path.isEmpty()) this.message else "${this.path}: ${this.message}"
}

/// Decoding failed. Every problem found is listed — a wrong type here, a
/// missing field there — so a client fixes its document in one round trip;
/// a malformed document is one problem, since nothing follows it.
public error DecodeError {
  public problems: List<Problem>
  fun message(): string = this.problems.map(p => p.toString()).join("\n")
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

/// How an encoder spells the keys it is given: as written, `snake_case`,
/// `camelCase` or `UPPER_SNAKE` (the environment's). A policy for a whole API,
/// never an attribute on a type.
public enum KeyStyle {
  AsWritten
  SnakeCase
  CamelCase
  UpperSnake
}

/// `name` spelled in `style`: `passwordHash` → `password_hash` → `passwordHash`.
public fun styleKey(name: string, style: KeyStyle): string = when (style) {
  KeyStyle.AsWritten  => name
  KeyStyle.UpperSnake => styleKey(name, KeyStyle.SnakeCase).toUpper()
  KeyStyle.SnakeCase  => {
    val out = StringBuilder()
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
  KeyStyle.CamelCase  => {
    val out = StringBuilder()
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
  fun problem(message: string) = this.problemAt(this.path(), message)
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

  /// What the type reads, as a source that is not a document needs to know it
  /// (D125: `config` asks for each environment variable by name): the
  /// fields, with the keys `decode` would match for `format` and `keys`, which
  /// of them are required, and their defaults. `implement Decodable` derives
  /// it; written by hand, a type that does not override it is one `Opaque`
  /// value, read as it is.
  static fun schema(format: string, keys: KeyStyle): Schema = Schema.opaque()
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
    if (this.items.len() < this.cap) this.items.push(Problem(path, message))
  }

  public fun isEmpty(): bool = this.items.isEmpty()
  public fun list(): List<Problem> = this.items.toList()
}

/// The decoded value, or the problems found: what `json.decode` and every
/// other format's entry point ends with.
public fun finish<T>(from: Decoder, value: T): T throws DecodeError {
  val problems = from.problems()
  if (!problems.isEmpty()) throw DecodeError(problems)
  value
}

// ---------------------------------------------------------------------------
// the shape of a type

/// What one node of a `Schema` holds.
public enum SchemaKind {
  Bool
  Int
  Float
  /// Text: a string, an enum member's name, a duration, a timestamp.
  Text
  /// Several of the element's schema.
  List
  /// Named fields.
  Object
  /// A type with a `decode` of its own: one value, read as it is.
  Opaque
  /// A type a flat source cannot give (a map, a sealed family).
  Unsupported
}

/// One field of an `Object` schema.
public struct SchemaField {
  /// The name it is declared with: `poolSize`.
  public name: string
  /// The key `decode` matches for the format and key style that were asked
  /// for: `POOL_SIZE` in the environment, or what a `@key` says.
  public key:    string
  public schema: Schema
  /// Whether decoding fails without it: no default, and not nullable.
  public required: bool
  public nullable: bool
  /// The default, as the text a source would hold it as; `null` when there is
  /// none or it has no text form.
  public fallback: string?
}

/// The shape of a `Decodable` type (see `Decodable.schema`): enough to ask a
/// source for each value by name instead of walking a document.
public struct Schema {
  public kind: SchemaKind
  /// What a value looks like, for a message: `an integer`, `one of "Debug", "Info"`.
  public what: string
  /// Whether the value is a `Secret`: it is never shown in a message.
  public secret: bool = false
  /// The fields of an `Object`.
  public fields: List<SchemaField> = []
  private inner: List<Schema> = []

  /// One value of `kind`.
  public static fun leaf(kind: SchemaKind, what: string): Schema = Schema(kind, what)

  /// A type read as it is: the schema of a hand-written `decode`.
  public static fun opaque(): Schema = Schema(kind: SchemaKind.Opaque, what: "a value")

  /// A type no flat source can give; `why` names it (`a Map`).
  public static fun unsupported(why: string): Schema = Schema(kind: SchemaKind.Unsupported, what: why)

  /// Several of `of`.
  public static fun list(of: Schema): Schema =
    Schema(kind: SchemaKind.List, what: "a list of ${of.what}, separated by commas", inner: [of])

  /// A struct: `what` is its name.
  public static fun object(what: string, fields: List<SchemaField>): Schema =
    Schema(kind: SchemaKind.Object, what, fields)

  /// The element of a `List`.
  public fun element(): Schema? = this.inner.at(0)

  /// The same schema, marked as a secret.
  public fun asSecret(): Schema = Schema(kind: this.kind, what: this.what, secret: true, fields: this.fields, inner: this.inner)
}

/// A field's default as the text a source holds it as, for `describe`: what
/// `decode` would be given to produce it. `null` when it has no text form.
public fun defaultText<T: Encodable>(value: T, format: string, keys: KeyStyle): string? {
  val encoder = ValueEncoder.of(format: format, enums: EnumStyle.Name, keys: keys, durations: DurationStyle.Text)
  value.encode(encoder) catch (e) {
    return null
  }
  valueText(encoder.value())
}

fun valueText(v: Value): string? = when (v) {
  is VString => v.value
  is VInt    => "${v.value}"
  is VFloat  => "${v.value}"
  is VBool   => if (v.value) "true" else "false"
  is VList   => v.items.map(x => valueText(x) ?: "").join(",")
  else       => null
}

// ---------------------------------------------------------------------------
// the built-in types

implement Encodable for i8 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(this.toI64())
}
implement Encodable for i16 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(this.toI64())
}
implement Encodable for i32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(this.toI64())
}
implement Encodable for i64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(this)
}
implement Encodable for isize {
  fun encode(to: Encoder) throws EncodeError = try to.writeI64(this.toI64())
}
implement Encodable for u8 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(this.toU64())
}
implement Encodable for u16 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(this.toU64())
}
implement Encodable for u32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(this.toU64())
}
implement Encodable for u64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(this)
}
implement Encodable for usize {
  fun encode(to: Encoder) throws EncodeError = try to.writeU64(this.toU64())
}
implement Encodable for f32 {
  fun encode(to: Encoder) throws EncodeError = try to.writeF64(this.toF64())
}
implement Encodable for f64 {
  fun encode(to: Encoder) throws EncodeError = try to.writeF64(this)
}
implement Encodable for bool {
  fun encode(to: Encoder) throws EncodeError = try to.writeBool(this)
}
implement Encodable for string {
  fun encode(to: Encoder) throws EncodeError = try to.writeString(this)
}

fun intSchema(what: string): Schema = Schema.leaf(SchemaKind.Int, what)

implement Decodable for i8 {
  static fun decode(from: Decoder): i8 throws DecodeError = try narrowI64(from, -128, 127).wrapI8()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from -128 to 127")
}
implement Decodable for i16 {
  static fun decode(from: Decoder): i16 throws DecodeError = try narrowI64(from, -32768, 32767).wrapI16()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from -32768 to 32767")
}
implement Decodable for i32 {
  static fun decode(from: Decoder): i32 throws DecodeError = try narrowI64(from, -2147483648, 2147483647).wrapI32()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from -2147483648 to 2147483647")
}
implement Decodable for i64 {
  static fun decode(from: Decoder): i64 throws DecodeError = try from.readI64()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer")
}
implement Decodable for isize {
  static fun decode(from: Decoder): isize throws DecodeError = try from.readI64().toIsize()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer")
}
implement Decodable for u8 {
  static fun decode(from: Decoder): u8 throws DecodeError = try narrowU64(from, 255).wrapU8()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from 0 to 255")
}
implement Decodable for u16 {
  static fun decode(from: Decoder): u16 throws DecodeError = try narrowU64(from, 65535).wrapU16()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from 0 to 65535")
}
implement Decodable for u32 {
  static fun decode(from: Decoder): u32 throws DecodeError = try narrowU64(from, 4294967295).wrapU32()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("an integer from 0 to 4294967295")
}
implement Decodable for u64 {
  static fun decode(from: Decoder): u64 throws DecodeError = try from.readU64()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("a non-negative integer")
}
implement Decodable for usize {
  static fun decode(from: Decoder): usize throws DecodeError = try from.readU64().toUsize()
  override static fun schema(format: string, keys: KeyStyle): Schema = intSchema("a non-negative integer")
}
implement Decodable for f32 {
  static fun decode(from: Decoder): f32 throws DecodeError = try from.readF64().toF32()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.leaf(SchemaKind.Float, "a number")
}
implement Decodable for f64 {
  static fun decode(from: Decoder): f64 throws DecodeError = try from.readF64()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.leaf(SchemaKind.Float, "a number")
}
implement Decodable for bool {
  static fun decode(from: Decoder): bool throws DecodeError = try from.readBool()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.leaf(SchemaKind.Bool, "true or false")
}
implement Decodable for string {
  static fun decode(from: Decoder): string throws DecodeError = try from.readString()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.leaf(SchemaKind.Text, "text")
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
    if (this == null) try to.writeNull() else try this.encode(to)
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
  override static fun schema(format: string, keys: KeyStyle): Schema = T.schema(format, keys)
}

implement<T: Encodable> Encodable for List<T> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginList()
    loop (x in this) try x.encode(to)
    try to.endList()
  }
}
implement<T: Encodable> Encodable for MutableList<T> {
  fun encode(to: Encoder) throws EncodeError = try this.toList().encode(to)
}
implement<T: Decodable> Decodable for List<T> {
  static fun decode(from: Decoder): List<T> throws DecodeError = try MutableList<T>.decode(from).toList()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.list(T.schema(format, keys))
}
implement<T: Decodable> Decodable for MutableList<T> {
  static fun decode(from: Decoder): MutableList<T> throws DecodeError {
    val out: MutableList<T> = []
    try from.beginList()
    loop (try from.hasNext()) out.push(try T.decode(from))
    try from.endList()
    out
  }
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.list(T.schema(format, keys))
}

// An array travels as a list; reading one needs exactly N elements (D121).
implement<T: Encodable, const N: i64> Encodable for Array<T, N> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginList()
    loop (x in this) try x.encode(to)
    try to.endList()
  }
}
implement<T: Decodable, const N: i64> Decodable for Array<T, N> {
  static fun decode(from: Decoder): Array<T, N> throws DecodeError {
    val items = try List<T>.decode(from)
    val a = items.toArray<N>()
    if (a != null) return a
    from.problem("expected $N elements, found ${items.len()}")
    throw DecodeError(problems: from.problems())
  }
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.list(T.schema(format, keys))
}

implement<V: Encodable> Encodable for Map<string, V> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginObject()
    loop ((k, v) in this.entries()) {
      try to.key(k)
      try v.encode(to)
    }
    try to.endObject()
  }
}
implement<V: Encodable> Encodable for MutableMap<string, V> {
  fun encode(to: Encoder) throws EncodeError = try this.toMap().encode(to)
}
implement<V: Decodable> Decodable for Map<string, V> {
  static fun decode(from: Decoder): Map<string, V> throws DecodeError = try MutableMap<string, V>.decode(from).toMap()
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.unsupported("a Map")
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
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.unsupported("a Map")
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
  public fun get(key: string): Value? = when (this) {
    is VObject => this.fields.get(key)
    else       => null
  }

  /// Element `i` of a list, or `null`.
  public fun at(i: i64): Value? = when (this) {
    is VList => this.items.at(i)
    else     => null
  }

  public fun asString(): string? = when (this) {
    is VString => this.value
    else       => null
  }

  public fun asI64(): i64? = when (this) {
    is VInt   => this.value
    is VFloat => if (this.value == this.value.trunc()) this.value.toI64() else null
    else      => null
  }

  public fun asF64(): f64? = when (this) {
    is VInt   => this.value.toF64()
    is VFloat => this.value
    else      => null
  }

  public fun asBool(): bool? = when (this) {
    is VBool => this.value
    else     => null
  }

  public fun isNull(): bool = this is VNull
}

implement Encodable for Value {
  fun encode(to: Encoder) throws EncodeError = when (this) {
    is VNull   => try to.writeNull()
    is VBool   => try to.writeBool(this.value)
    is VInt    => try to.writeI64(this.value)
    is VFloat  => try to.writeF64(this.value)
    is VString => try to.writeString(this.value)
    is VList   => try this.items.encode(to)
    is VObject => try this.fields.encode(to)
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
  override static fun schema(format: string, keys: KeyStyle): Schema = Schema.unsupported("a free-form Value")
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
    val top = this.stack.ref(-1) ?: return this.root
    if (top.isList) return top.values.at(top.next) ?: VNull()
    top.pending ?: VNull()
  }

  /// A read consumed the current value.
  private fun advance() {
    val top = this.stack.ref(-1) ?: return
    if (top.isList) top.next += 1 else top.pending = null
  }

  private fun here(): string {
    val top = this.stack.ref(-1) ?: return ""
    if (top.isList) return indexPath(top.path, top.next)
    childPath(top.path, top.keys.at(top.next - 1) ?: "")
  }

  private fun wrong(expected: string): Value {
    val v = this.current()
    this.recorded.record(this.here(), "expected $expected, found ${kindName(v)}")
    this.advance()
    v
  }

  /// A tree nests as deep as whoever built it wanted, and walking it is
  /// recursion, so it is bounded like every other walk over input someone
  /// else produced (§2). Unlike a wrong type, this is not a problem to
  /// record and carry on from: the frames are already on the stack.
  private fun checkDepth() throws DecodeError {
    if (this.stack.len() < this.maxDepth) return
    throw DecodeError(problems: this.recorded.list().concat([Problem(path: this.here(), message: tooDeepMessage(this.maxDepth))]))
  }

  implement Decoder {
    fun format(): string = this.name
    override fun enums(): EnumStyle = this.style
    override fun durations(): DurationStyle = this.durationStyle
    override fun keys(): KeyStyle = this.keyStyle

    fun peek(): Kind throws DecodeError = when (this.current()) {
      is VNull   => Kind.Null
      is VBool   => Kind.Bool
      is VInt    => Kind.Int
      is VFloat  => Kind.Float
      is VString => Kind.String
      is VList   => Kind.List
      is VObject => Kind.Object
    }

    fun beginObject() throws DecodeError {
      try this.checkDepth()
      val v = this.current()
      when (v) {
        is VObject => {
          this.stack.push(Frame(path: this.here(), keys: v.fields.keys(), values: v.fields.values(), isList: false))
        }
        else       => {
          val _ = this.wrong("an object")
          this.stack.push(Frame(path: this.here(), isList: false))
        }
      }
    }

    fun nextKey(): string? throws DecodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      val k = top.keys.at(top.next) ?: return null
      top.pending = top.values.at(top.next)
      top.next += 1
      k
    }

    fun endObject() throws DecodeError {
      val _ = this.stack.removeAt(this.stack.lastIndex())
      this.advance()
    }

    fun beginList() throws DecodeError {
      try this.checkDepth()
      val v = this.current()
      when (v) {
        is VList => this.stack.push(Frame(path: this.here(), values: v.items, isList: true))
        else     => {
          val _ = this.wrong("a list")
          this.stack.push(Frame(path: this.here(), isList: true))
        }
      }
    }

    fun hasNext(): bool throws DecodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      top.next < top.values.len()
    }

    fun endList() throws DecodeError {
      val _ = this.stack.removeAt(this.stack.lastIndex())
      this.advance()
    }

    fun readI64(): i64 throws DecodeError {
      val v = this.current()
      val n = v.asI64()
      if (n == null) {
        val _ = this.wrong("an integer")
        return 0
      }
      this.advance()
      n
    }

    fun readU64(): u64 throws DecodeError {
      val n = try this.readI64()
      if (n < 0) {
        this.recorded.record(this.here(), "$n is negative")
        return 0
      }
      n.wrapU64()
    }

    fun readF64(): f64 throws DecodeError {
      val v = this.current()
      val n = v.asF64()
      if (n == null) {
        val _ = this.wrong("a number")
        return 0.0
      }
      this.advance()
      n
    }

    fun readBool(): bool throws DecodeError {
      val v = this.current()
      val b = v.asBool()
      if (b == null) {
        val _ = this.wrong("a boolean")
        return false
      }
      this.advance()
      b
    }

    fun readString(): string throws DecodeError {
      val v = this.current()
      val s = v.asString()
      if (s == null) {
        val _ = this.wrong("a string")
        return ""
      }
      this.advance()
      s
    }

    fun readNull() throws DecodeError {
      if (!this.current().isNull()) {
        val _ = this.wrong("null")
        return
      }
      this.advance()
    }

    fun skip() throws DecodeError = this.advance()

    fun path(): string = this.here()

    fun problemAt(path: string, message: string) = this.recorded.record(path, message)

    fun problems(): List<Problem> = this.recorded.list()
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
  public fun value(): Value = this.state.result ?: VNull()

  /// The same bound the text encoders keep (§2): a value nested deeper than
  /// this is refused rather than walked, whichever direction it is going.
  private fun checkDepth() throws EncodeError {
    if (this.stack.len() >= this.maxDepth) throw EncodeError(message: tooDeepMessage(this.maxDepth))
  }

  private fun put(v: Value) {
    val top = this.stack.ref(-1)
    if (top == null) {
      this.state.result = v
      return
    }
    if (!top.isList) top.keys.push(top.key)
    top.values.push(v)
  }

  implement Encoder {
    fun format(): string = this.name
    override fun enums(): EnumStyle = this.style
    override fun durations(): DurationStyle = this.durationStyle
    override fun keys(): KeyStyle = this.keyStyle

    fun beginObject() throws EncodeError {
      try this.checkDepth()
      this.stack.push(Building(isList: false))
    }

    fun key(name: string) throws EncodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      top.key = name
    }

    fun endObject() throws EncodeError {
      val top = this.stack.removeAt(this.stack.lastIndex())
      val fields: MutableMap<string, Value> = [:]
      var i: i64 = 0
      loop (k in top.keys) {
        fields.set(k, top.values.at(i) ?: panic("codec: an object's keys and values are pushed in pairs"))
        i += 1
      }
      this.put(VObject(fields: fields.toMap()))
    }

    fun beginList() throws EncodeError {
      try this.checkDepth()
      this.stack.push(Building(isList: true))
    }

    fun endList() throws EncodeError {
      val top = this.stack.removeAt(this.stack.lastIndex())
      this.put(VList(items: top.values.toList()))
    }

    fun writeI64(v: i64) throws EncodeError = this.put(VInt(value: v))
    fun writeU64(v: u64) throws EncodeError = this.put(VInt(value: v.wrapI64()))
    fun writeF64(v: f64) throws EncodeError = this.put(VFloat(value: v))
    fun writeBool(v: bool) throws EncodeError = this.put(VBool(value: v))
    fun writeString(v: string) throws EncodeError = this.put(VString(value: v))
    fun writeNull() throws EncodeError = this.put(VNull())
  }
}
