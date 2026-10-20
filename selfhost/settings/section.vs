use toml { Table, Value }

/// A manifest the reader refuses: the whole message, `file:line: reason`.
error SettingsError {
  message: string
}

/// One key of a table and what it is set to.
struct Entry {
  key:   string
  value: Value
}

/// A table of `veles.toml` and where it is, so that every refusal names the
/// file and the line.
struct Section {
  path: string
  /// The table's name in the manifest (`package`, `format`, …); empty for the file itself.
  name:  string
  table: *Table

  /// A refusal at `line` of this file.
  fun fail(line: i64, reason: string): SettingsError => SettingsError(message: "${this.path}:$line: $reason")

  /// A refusal at the place `value` was written.
  fun failAt(value: Value, reason: string): SettingsError => this.fail(value.line, reason)

  /// The keys in a fixed order, each with its value.
  fun entries(): List<Entry> {
    val out: MutableList<Entry> = []
    loop (key in this.table.keys()) {
      val value = this.table.get(key) ?: continue
      out.push(Entry(key, value))
    }
    out
  }

  /// A table inside this one, read as part of the same section.
  fun within(table: *Table): Section => Section(path: this.path, name: this.name, table)

  /// The table called `name` of the file.
  fun named(name: string, table: *Table): Section => Section(path: this.path, name, table)
}

/// The strings of a list value, or a refusal naming `what` and an `example`.
fun stringList(section: Section, value: Value, what: string, example: string): List<string> throws SettingsError {
  val items = value.asList() ?: throw section.failAt(value, "$what is a list, e.g. $example")
  val out: MutableList<string> = []
  loop (item in items) {
    out.push(item.asString() ?: throw section.failAt(item, "$what holds strings only"))
  }
  out
}
