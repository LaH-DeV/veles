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

/// `name` spelled in `case`, as an encoder spells the keys it is given: as
/// written, `snake_case`, `camelCase`, `SCREAMING_SNAKE` (the environment's)
/// — a policy for a whole API, never an attribute on a type (D205: `Case`
/// replaced `Case`).
public fun styleKey(name: string, style: Case): string => caseSpelled(name, style)

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
  fun durations(): DurationStyle => DurationStyle.Seconds
  /// How enums are written; `Name` unless the format says otherwise.
  fun enums(): EnumStyle => EnumStyle.Name
  /// How field names are spelled as keys; derived code applies it to a
  /// field's own name, never to a `@key` the author wrote.
  fun keys(): Case => Case.AsWritten
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
  fun durations(): DurationStyle => DurationStyle.Seconds
  fun enums(): EnumStyle => EnumStyle.Name
  fun keys(): Case => Case.AsWritten
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
  fun problem(message: string) => this.problemAt(this.path(), message)
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
  static fun decode(from: Decoder): This throws DecodeError

  /// What the type reads, as a source that is not a document needs to know it
  /// (D125: `config` asks for each environment variable by name): the
  /// fields, with the keys `decode` would match for `format` and `keys`, which
  /// of them are required, and their defaults. `implement Decodable` derives
  /// it; written by hand, a type that does not override it is one `Opaque`
  /// value, read as it is.
  static fun schema(format: string, keys: Case): Schema => Schema.opaque()
}

/// Both directions: the one line a DTO writes.
public trait Codable : Encodable + Decodable

/// The path of a member `key` under `base`.
public fun childPath(base: string, key: string): string {
  val plain = !key.isEmpty() && !key.contains(".") && !key.contains("[") && !key.contains("]") && !key.contains("\"")
  when {
    plain && base.isEmpty() => key
    plain                   => "$base.$key"
    else                    => base + "[\"" + key.replace("\"", "\\\"") + "\"]"
  }
}

/// The path of element `i` under `base`.
public fun indexPath(base: string, i: i64): string => "$base[$i]"

/// A path found under `base`: `joinPath("user", "address[2]")` is
/// `user.address[2]`.
public fun joinPath(base: string, rel: string): string => when {
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
  public static fun capped(cap: i64): Problems => Problems(cap)

  public fun record(path: string, message: string) {
    if (this.items.len() < this.cap) this.items.push(Problem(path, message))
  }

  public fun isEmpty(): bool => this.items.isEmpty()
  public fun list(): List<Problem> => this.items.toList()
}

/// The decoded value, or the problems found: what `json.decode` and every
/// other format's entry point ends with.
public fun finish<T>(from: Decoder, value: T): T throws DecodeError {
  val problems = from.problems()
  if (!problems.isEmpty()) throw DecodeError(problems)
  value
}
