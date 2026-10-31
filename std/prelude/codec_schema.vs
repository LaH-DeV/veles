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
  public static fun leaf(kind: SchemaKind, what: string): Schema => Schema(kind, what)

  /// A type read as it is: the schema of a hand-written `decode`.
  public static fun opaque(): Schema => Schema(kind: SchemaKind.Opaque, what: "a value")

  /// A type no flat source can give; `why` names it (`a Map`).
  public static fun unsupported(why: string): Schema => Schema(kind: SchemaKind.Unsupported, what: why)

  /// Several of `of`.
  public static fun list(of: Schema): Schema =>
    Schema(kind: SchemaKind.List, what: "a list of ${of.what}, separated by commas", inner: [of])

  /// A struct: `what` is its name.
  public static fun object(what: string, fields: List<SchemaField>): Schema =>
    Schema(kind: SchemaKind.Object, what, fields)

  /// The element of a `List`.
  public fun element(): Schema? => this.inner.at(0)

  /// The same schema, marked as a secret.
  public fun asSecret(): Schema => Schema(kind: this.kind, what: this.what, secret: true, fields: this.fields, inner: this.inner)
}

/// A field's default as the text a source holds it as, for `describe`: what
/// `decode` would be given to produce it. `null` when it has no text form.
public fun defaultText<T: Encodable>(value: T, format: string, keys: Case): string? {
  val encoder = ValueEncoder.of(format: format, enums: EnumStyle.Name, keys: keys, durations: DurationStyle.Text)
  value.encode(encoder) catch (e) {
    return null
  }
  valueText(encoder.value())
}

fun valueText(v: Value): string? => when (v) {
  is VString => v.value
  is VInt    => "${v.value}"
  is VFloat  => "${v.value}"
  is VBool   => if (v.value) "true" else "false"
  is VList   => v.items.map(x => valueText(x) ?: "").join(",")
  else       => null
}
