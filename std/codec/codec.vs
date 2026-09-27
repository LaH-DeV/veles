/// The machinery behind `implement Codable` (D58, D75): the `Encoder` and
/// `Decoder` a format implements, the dynamic `Value` tree (`VNull`,
/// `VBool`, `VInt`, `VFloat`, `VString`, `VList`, `VObject`) with the
/// `ValueEncoder`/`ValueDecoder` that convert to and from it, the styles a
/// format may be asked for (`KeyStyle`, `EnumStyle`, `DurationStyle`) and
/// the `Problems` a decoder records.
///
/// Deriving needs none of it: `implement Codable` works without an import,
/// and `EncodeError`, `DecodeError`, `Encodable`, `Decodable`, `Codable`
/// are global. A program imports `codec` to walk a document whose shape it
/// does not know, to pick a style, or to write a format of its own:
///
/// ```veles
/// use codec, json
///
/// fun count(v: codec.Value): i64 = when (v) {
///   is codec.VList   => v.items.fold(1, (n, x) => n + count(x))
///   is codec.VObject => v.fields.values().fold(1, (n, x) => n + count(x))
///   else             => 1
/// }
/// ```
///
/// The declarations are written with the prelude, which implements
/// `Encodable` for the built-in types with them; this module is where a
/// program finds them.
