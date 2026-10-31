// The formatter: what `[format]` of a manifest may say (D177), and — as the
// formatter is ported — the code that formats. This module owns its options; the
// package system only asks it to read the table.
//
// To add an option: a field with its default in `FormatOptions`, one `Setting`
// in `formatSchema()` (its key, a line of documentation, how the value is read),
// and the formatter reads the field. The key is accepted, documented, listed in
// the unknown-key refusal and offered by tools from that one entry.
use settings { Schema, Section, Setting, SettingsError }
use toml { Value }

/// How a line is indented.
sealed trait Indentation {
  /// The formatter's own choice.
  struct DefaultIndent { }
  struct TabIndent { }
  struct SpacesIndent {
    width: i64
  }
}

/// `imports = "lines"` (one `use` per module) or `"merged"` (one `use a, b`
/// per origin).
enum ImportStyle {
  Lines
  Merged
}

/// `trailing_commas = "multiline"` (a comma after the last element of a list
/// broken over several lines) or `"never"`.
enum CommaStyle {
  Multiline
  Never
}

enum LineEndings {
  /// Not stated: the formatter's default.
  Unset
  Lf
  Crlf
  /// Keep the file's own.
  Auto
}

/// `[format]`, for `veles fmt`.
struct FormatOptions {
  var indent: Indentation = DefaultIndent()
  /// null for the default.
  var maxBlankLines: i64? = null
  var imports:       ImportStyle = ImportStyle.Lines
  var commas:        CommaStyle = CommaStyle.Multiline
  var lineEndings:   LineEndings = LineEndings.Unset
}

fun formatSchema(): Schema<FormatOptions> => Schema(
  table: "format",
  settings: [
    Setting<FormatOptions>(key: "indent", doc: "spaces per level (1-8) or \"tab\"", read: (section, value, options) => {
      options.indent = try indentation(section, value)
    }),
    Setting<FormatOptions>(key: "max_blank_lines", doc: "the most blank lines kept in a row", read: (section, value, options) => {
      val limit = value.asInt() ?: (value.asString()?.leadingInteger() ?: 0)
      if (limit < 1) throw section.failAt(value, "[format] max_blank_lines must be a positive number")
      options.maxBlankLines = limit
    }),
    Setting<FormatOptions>.choice(
      "imports", "\"lines\": one `use` per module; \"merged\": one `use a, b` per origin", ImportStyle.values().map(member => member.toString().toLower()),
      "[format] imports is \"lines\" (one 'use' per module, the default) or \"merged\" (one 'use a, b' per origin)",
      (word, options) => {
        options.imports = ImportStyle.values().find(member => member.toString().toLower() == word) ?:
          panic("formatter: imports choice was validated against ImportStyle")
      },
    ),
    Setting<FormatOptions>.choice(
      "trailing_commas", "\"multiline\": a comma after the last element of a broken list; \"never\"", CommaStyle.values().map(member => member.toString().toLower()),
      "[format] trailing_commas is \"multiline\" (a comma after the last element of a list broken over several lines, the default) or \"never\"",
      (word, options) => {
        options.commas = CommaStyle.values().find(member => member.toString().toLower() == word) ?:
          panic("formatter: trailing_commas choice was validated against CommaStyle")
      },
    ),
    Setting<FormatOptions>.choice(
      "line_endings", "\"lf\" (the default), \"crlf\", or \"auto\" to keep the file's own", LineEndings.values().filter(member => member != LineEndings.Unset).map(member => member.toString().toLower()),
      "[format] line_endings is \"lf\" (the default), \"crlf\" or \"auto\" (keep the file's own)",
      (word, options) => {
        options.lineEndings = LineEndings.values().find(member => member.toString().toLower() == word) ?:
          panic("formatter: line_endings choice was validated against LineEndings")
      },
    ),
  ],
)

fun readFormat(section: Section): FormatOptions throws SettingsError {
  val options = &FormatOptions()
  try formatSchema().apply(section, options)
  *options
}

/// `indent = 4`, `"4"` or `"tab"`: 1 to 8 spaces, or a tab.
fun indentation(section: Section, value: Value): Indentation throws SettingsError {
  val refusal = section.failAt(value, "[format] indent must be a number of spaces (1-8) or \"tab\"")
  val written = if (value.asInt() != null) "${value.asInt() ?: 0}" else (value.asString() ?: throw refusal)
  if (written == "tab") return TabIndent()
  // the canonical spelling of a width only: not "04", not " 4"
  val width = written.toInt() ?: throw refusal
  if (width < 1 || width > 8 || "$width" != written) throw refusal
  SpacesIndent(width)
}

extend string {
  /// The number a text starts with: blanks, an optional sign and digits (an
  /// underscore ends them); 0 when there are none or they do not fit.
  fun leadingInteger(): i64 {
    var at: i64 = 0
    loop (at < this.len() && (this.byteAt(at) == ' ' || this.byteAt(at) == '\t')) at += 1
    val negative = at < this.len() && this.byteAt(at) == '-'
    if (at < this.len() && (this.byteAt(at) == '+' || this.byteAt(at) == '-')) at += 1
    val start = at
    loop (at < this.len() && this.byteAt(at).isAsciiDigit()) at += 1
    val magnitude = this.slice(start, at).toInt() ?: return 0
    if (negative) -magnitude else magnitude
  }
}
