// `[lint]` of a manifest: reading it into `LintSettings`. What there is to
// set — the lints and the options — is declared in registry.vs; adding one
// needs nothing here.
use toml { Value }
use settings { Schema, Section, Setting, SettingsError }
use text { didYouMean, quote }

/// The level a value spells, or null.
fun levelOf(value: Value): LintLevel? {
  val text = value.asString() ?: return null
  lintLevelNamed(text)
}

/// The setting that gives `lint` a level: `unused_import = "deny"`.
fun levelSetting(lint: LintInfo): Setting<LintSettings> =>
  Setting(key: lint.name, doc: lint.doc, read: (section, value, settings) => {
    settings.levels.set(lint.name, try expectedLevel(section, lint.name, value))
  })

/// `[lint]` is every option and every lint of the registry; a key that is
/// neither is a rule of a dependency (`package.rule`) or a refusal.
fun lintSchema(): Schema<LintSettings> => Schema(
  table: "lint",
  settings: lintOptions().concat(lints().map(lint => levelSetting(lint))),
  fallback: (section, entry, settings) => {
    if (val table = entry.value.asTable()) {
      // `acme-lints.no_sleep = "warn"` reads as a table `acme-lints`
      loop (rule in section.within(table).entries()) {
        settings.library.set("${entry.key}.${rule.key}", try libraryLevel(section, "${entry.key}.${rule.key}", rule.value))
      }
    } else if (entry.key.contains(".")) {
      settings.library.set(entry.key, try libraryLevel(section, entry.key, entry.value))
    } else {
      throw section.failAt(entry.value, unknownLint(entry.key))
    }
  },
)

fun readLint(section: Section): LintSettings throws SettingsError {
  val settings = &LintSettings()
  try lintSchema().apply(section, settings)
  *settings
}

fun unknownLint(key: string): string {
  val known = lintOptions().map(option => option.key).concat(lintNames())
  val hint = didYouMean(key, known)
  val suggestion = if (hint.isEmpty()) "" else "; did you mean ${quote(hint)}?"
  "unknown [lint] key ${quote(key)}$suggestion (run 'veles lint --list' for the lints)"
}

fun expectedLevel(section: Section, key: string, value: Value): LintLevel throws SettingsError =>
  levelOf(value) ?: throw section.failAt(value, "[lint] $key is \"allow\" (silent), \"warn\" or \"deny\" (an error)")

/// The level of `package.rule`; the rule's existence is checked when it is loaded.
fun libraryLevel(section: Section, key: string, value: Value): LintLevel throws SettingsError {
  val level = try expectedLevel(section, key, value)
  if (key.startsWith(".") || key.endsWith(".")) {
    throw section.failAt(value, "[lint] $key: a rule of another package is written package.rule, e.g. acme-lints.no_sleep = \"warn\"")
  }
  level
}
