use version { Version, VersionError }
use settings { Schema, Section, Setting, SettingsError }
use text { quote }

/// The oldest compiler a package names: `0.70` or `0.70.0`.
fun parseMinimumCompiler(written: string): Version throws VersionError {
  val full = if (written.split(".").len() == 2) "$written.0" else written
  try Version.parse(full)
}

/// `[package]`: who the package is.
struct Identity {
  var name: string = ""
  /// As written (`1.2.3`, `v1.2.3-rc.1`).
  var version:     string = ""
  var description: string = ""
  var license:     string = ""
  /// The oldest compiler that builds it, as written (`0.70`); empty when not stated.
  var minimumCompiler: string = ""
  /// `owner/name`: where `veles publish` publishes it; empty when not stated.
  var registry: string = ""
}

fun identitySchema(): Schema<Identity> {
  Schema(
    table: "package",
    settings: [
      Setting<Identity>.text("name", "the package's name, as `use` of a dependent names it", (section, value, text, identity) => {
        identity.name = text
      }),
      Setting<Identity>.text("version", "its version: major.minor.patch, with an optional -pre-release", (section, value, text, identity) => {
        Version.parse(text) catch (e) {
          throw section.failAt(value, "[package] version: ${e.message}")
        }
        identity.version = text
      }),
      Setting<Identity>.text("description", "what it is, in a line", (section, value, text, identity) => {
        identity.description = text
      }),
      Setting<Identity>.text("license", "the license it is published under", (section, value, text, identity) => {
        identity.license = text
      }),
      Setting<Identity>.text("veles", "the oldest compiler that builds it: 0.70 or 0.70.0", (section, value, text, identity) => {
        parseMinimumCompiler(text) catch (e) {
          throw section.failAt(value, "[package] veles: ${e.message}")
        }
        identity.minimumCompiler = text
      }),
      Setting<Identity>.text("registry", "owner/name: where `veles publish` publishes it", (section, value, text, identity) => {
        if (!isRegistryName(text)) {
          throw section.failAt(value, "[package] registry ${quote(text)} is owner/name, lowercase letters, digits, '-' and '_'")
        }
        identity.registry = text
      }),
    ],
    removed: [("exports", "'exports' was removed (D89): a package's surface is what its root module re-exports — write 'public use geometry' in lib.vs for each module listed here")],
  )
}

fun readIdentity(section: Section): Identity throws SettingsError {
  val identity = &Identity()
  try identitySchema().apply(section, identity)
  *identity
}
