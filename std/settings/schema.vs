/// TOML-backed schemas for configuration a tool reads itself (D226). Each
/// setting declares its key, documentation and conversion into a typed value;
/// a schema applies those declarations to a `Section` and rejects unknown keys.
/// This API does not register keys in compiler-owned `veles.toml` tables.
use fmt { quote }
use toml { Value }

/// One key of a TOML section and how its value is read into `T`.
public struct Setting<T> {
  /// The key in the section.
  public key: string
  /// What the setting is for, in one line.
  public doc: string
  /// Converts a parsed value into the caller's typed settings.
  public read: fun(Section, Value, *T) throws SettingsError

  /// A string setting, converted by `set`; `expects` customizes the type error.
  public static fun text(key: string, doc: string, set: fun(Section, Value, string, *T) throws SettingsError, expects: string = ""): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val sentence = if (expects.isEmpty()) "is a string, e.g. $key = \"...\"" else expects
      val text = value.asString() ?: throw section.failAt(value, "[${section.name}] $key $sentence")
      try set(section, value, text, target)
    })

  /// A setting whose value is one of `words`; `expects` is the full diagnostic.
  public static fun choice(key: string, doc: string, words: List<string>, expects: string, set: fun(string, *T)): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val word = value.asString()
      if (word == null || !words.contains(word)) throw section.failAt(value, expects)
      set(word, target)
    })

  /// A list of strings converted by `set`.
  public static fun names(key: string, doc: string, example: string, set: fun(Section, Value, List<string>, *T) throws SettingsError): Setting<T> =>
    Setting(key, doc, read: (section, value, target) => {
      val names = try stringList(section, value, "[${section.name}] $key", example)
      try set(section, value, names, target)
    })
}

/// The typed settings defined for one tool-owned TOML table.
public struct Schema<T> {
  /// The table name used in unknown-key diagnostics.
  public table: string
  /// The settings in the order tools list and document them.
  public settings: List<Setting<T>>
  /// Keys removed from this schema and the diagnostic that explains the change.
  public removed: List<(string, string)> = []
  /// Optional handling for keys outside this schema; absent means refuse them.
  public fallback: (fun(Section, Entry, *T) throws SettingsError)? = null

  /// The keys, in declaration order.
  public fun keys(): List<string> => this.settings.map(setting => setting.key)

  /// Find a setting by key.
  public fun find(key: string): Setting<T>? {
    loop (setting in this.settings) {
      if (setting.key == key) return setting
    }
    null
  }

  /// Read every entry in `section` into `target`, refusing unknown or removed keys.
  public fun apply(section: Section, target: *T) throws SettingsError {
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
