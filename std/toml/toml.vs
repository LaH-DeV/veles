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
/// A value is a `Value`: the line it was written on and its `data`, a sealed
/// family (`Text`, `Integer`, `Real`, `Flag`, `Moment`, `Items`, `Nested`) to
/// match on, with `asString()`, `asInt()`, … for the common reads. Dates and
/// times are kept as the text written (`Moment.text`), checked for form and
/// range.

/// Reads `text` into its root table. `strict` reads the subset a package
/// manifest uses (see the module's description). Arrays and inline tables may
/// nest up to `recursion.maxRecursionDepth`; deeper input returns `ParseError`.
public fun parse(text: string, strict: bool = false): *Table throws ParseError {
  val withoutMark = if (text.startsWith("\u{FEFF}")) text.slice(3, text.len()) else text
  val rules = if (strict) Rules.manifest() else Rules.full()
  var reader = Reader.of(withoutMark.replace("\r\n", "\n"), rules)
  try reader.run()
}
