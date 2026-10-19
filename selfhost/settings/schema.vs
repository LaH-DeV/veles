// Settings registries. A table of a configuration file is a `Schema` of
// `Setting`s: each says its key, what it is for, and how its value is read into
// the typed settings of the part of the compiler that owns the table. The
// formatter, the linter and the package system each declare their own schema;
// the manifest only says which table goes to which. Nothing here knows what any
// of them accepts.
use toml { Value }
use text { quote }

/// One key of a `veles.toml` table: its name, what it is for, and how its value
/// is read into the settings being built (`T`).
///
/// Every table of the manifest is a `Schema` of these. The reading, the
/// refusal of an unknown key (with the keys there are), and the list of settings
/// a tool shows — in a reference, in completion, on hover — come from the same
/// declarations.
struct Setting<T> {
  key: string
  /// What the setting is for, in a line.
  doc:  string
  read: fun(Section, Value, *T) throws SettingsError

  /// A string setting. `set` stores the string, and may refuse it; when the
  /// value is not a string the refusal is "[table] key " and `expects`, which says
  /// `is a string, e.g. key = "..."` unless given.
  static fun text(key: string, doc: string, set: fun(Section, Value, string, *T) throws SettingsError, expects: string = ""): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val sentence = if (expects.isEmpty()) "is a string, e.g. $key = \"...\"" else expects
      val text = value.asString() ?: throw section.failAt(value, "[${section.name}] $key $sentence")
      try set(section, value, text, target)
    })

  /// A setting that is one of a fixed set of words; `expects` is the whole
  /// refusal, naming them.
  static fun choice(key: string, doc: string, words: List<string>, expects: string, set: fun(string, *T)): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val word = value.asString()
      if (word == null || !words.contains(word)) throw section.failAt(value, expects)
      set(word, target)
    })

  /// A list of strings, stored by `set`.
  static fun names(key: string, doc: string, example: string, set: fun(Section, Value, List<string>, *T) throws SettingsError): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val names = try stringList(section, value, "[${section.name}] $key", example)
      try set(section, value, names, target)
    })
}

/// The settings of one table, in the order a reference lists them.
struct Schema<T> {
  /// The table's name in the manifest: `format`, `native`, …
  table:    string
  settings: List<Setting<T>>
  /// Keys that were settings once: reading one is a refusal with the way out.
  removed: List<(string, string)> = []
  /// What reads a key no setting has — `[lint]` sets lints by name — or null to refuse it.
  fallback: (fun(Section, Entry, *T) throws SettingsError)? = null

  /// The keys, in the order declared.
  fun keys(): List<string> => this.settings.map(setting => setting.key)

  fun find(key: string): Setting<T>? {
    loop (setting in this.settings) {
      if (setting.key == key) return setting
    }
    null
  }

  /// Reads every key of `section` into `target`, in a fixed order.
  fun apply(section: Section, target: *T) throws SettingsError {
    loop (entry in section.entries()) {
      if (val setting = this.find(entry.key)) {
        try setting.read(section, entry.value, target)
        continue
      }
      loop (retired in this.removed) {
        if (retired.0 == entry.key) throw section.failAt(entry.value, retired.1)
      }
      val other = this.fallback ?: throw section.failAt(entry.value, "unknown [${this.table}] key ${quote(entry.key)} (${this.keys().join(", ")})")
      try other(section, entry, target)
    }
  }
}
