use toml { Table, Value }

/// A configuration refusal, including the file and line of the value.
public error SettingsError {
  /// The complete diagnostic message.
  public message: string
}

/// One parsed key and the TOML value written for it.
public struct Entry {
  /// The key as written in the section.
  public key: string
  /// The parsed value, including its source line.
  public value: Value
}

/// A named TOML table with the path needed for source-located errors.
public struct Section {
  /// The configuration file path used in diagnostics.
  public path: string
  /// The table name, such as `format` or `tool`.
  public name: string
  /// The parsed table being read.
  public table: *Table

  /// Make a refusal at the given line of this file.
  public fun fail(line: i64, reason: string): SettingsError => SettingsError(message: "${this.path}:$line: $reason")

  /// Make a refusal at the line where `value` was written.
  public fun failAt(value: Value, reason: string): SettingsError => this.fail(value.line, reason)

  /// The table's keys and values in the TOML module's stable order.
  public fun entries(): List<Entry> {
    val out: MutableList<Entry> = []
    loop (key in this.table.keys()) {
      val value = this.table.get(key) ?: continue
      out.push(Entry(key, value))
    }
    out
  }

  /// A nested table in the same file and under the same section name.
  public fun within(table: *Table): Section => Section(path: this.path, name: this.name, table)

  /// A named table in the same file.
  public fun named(name: string, table: *Table): Section => Section(path: this.path, name, table)
}

fun stringList(section: Section, value: Value, what: string, example: string): List<string> throws SettingsError {
  val items = value.asList() ?: throw section.failAt(value, "$what is a list, e.g. $example")
  val out: MutableList<string> = []
  loop (item in items) {
    out.push(item.asString() ?: throw section.failAt(item, "$what holds strings only"))
  }
  out
}
