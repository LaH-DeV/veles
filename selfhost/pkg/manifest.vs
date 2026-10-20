use formatter { FormatOptions, readFormat }
use linter { LintSettings, readLint }
use settings { Schema, Section, Setting, SettingsError }
// `veles.toml` (M1, D138): package identity lives here, never in source. The
// file is TOML, read by std/toml in its manifest dialect. Each table is a
// `Schema` of settings that reads into a typed value of its own, and the file
// itself is a schema whose settings are those tables, so a typo is an error
// that lists what there is rather than a setting that silently does nothing.
use toml

/// Everything `veles.toml` says. Filled in by `parseManifest`; read it, do not
/// assign it.
struct Manifest {
  var identity: Identity = Identity()
  /// `[dependencies]` and `[dev-dependencies]` (only for `veles test` and `*.test.vs`).
  var requires:    List<Dependency> = []
  var devRequires: List<Dependency> = []
  /// The member directories of `[workspace]`; null in a package that is not a
  /// monorepo root.
  var workspace: List<string>? = null
  var format:    FormatOptions = FormatOptions()
  var lint:      LintSettings = LintSettings()
  var native:    NativeLibraries = NativeLibraries()
  var policy:    Policy = Policy()
  var registry:  RegistrySettings = RegistrySettings()
  /// The default pool's thread count; null for one per core.
  var threads: i64? = null

  /// The path dependencies, by name.
  fun paths(): Map<string, string> {
    val out: MutableMap<string, string> = [:]
    loop (dependency in this.requires) {
      val source = dependency.source
      if (source is PathSource) out.set(dependency.name, source.path)
    }
    out
  }
}

/// A table of the manifest as a setting of the file: reads the table into
/// `manifest` with `read`.
fun tableSetting(name: string, doc: string, read: fun(Section, *Manifest) throws SettingsError): Setting<Manifest> =>
  Setting(key: name, doc, read: (file, value, manifest) => {
    val table = value.asTable() ?: throw file.failAt(value, outsideAnyTable(name, file))
    try read(file.named(name, table), manifest)
  })

fun outsideAnyTable(key: string, file: Section): string =>
  "'$key' is outside any table: put it under [package] (the tables are ${manifestSchema().keys().join(", ")})"

/// The tables a manifest has. A table whose settings belong to another part of
/// the compiler (`[format]` to the formatter, `[lint]` to the linter) is read by
/// that part's schema; this one only says where each table goes.
fun manifestSchema(): Schema<Manifest> => Schema(
  table: "",
  settings: [
    tableSetting("package", "who the package is", (section, manifest) => {
      manifest.identity = try readIdentity(section)
    }),
    tableSetting("dependencies", "the packages it needs", (section, manifest) => {
      manifest.requires = try readDependencies(section)
    }),
    tableSetting("dev-dependencies", "the packages only tests need", (section, manifest) => {
      manifest.devRequires = try readDependencies(section)
    }),
    tableSetting("workspace", "the member packages of a monorepo", (section, manifest) => {
      manifest.workspace = try readWorkspace(section)
    }),
    tableSetting("format", "how `veles fmt` writes code", (section, manifest) => {
      manifest.format = try readFormat(section)
    }),
    tableSetting("lint", "which lints are on, and how loud", (section, manifest) => {
      manifest.lint = try readLint(section)
    }),
    tableSetting("native", "the C libraries to link", (section, manifest) => {
      manifest.native = try readNative(section)
    }),
    tableSetting("policy", "what dependencies may do", (section, manifest) => {
      manifest.policy = try readPolicy(section)
    }),
    tableSetting("registry", "where registry packages come from", (section, manifest) => {
      manifest.registry = try readRegistry(section)
    }),
    tableSetting("runtime", "how the program runs", (section, manifest) => {
      manifest.threads = try readRuntime(section)
    }),
  ],
  fallback: (file, entry, manifest) => {
    if (entry.value.asTable() == null) throw file.failAt(entry.value, outsideAnyTable(entry.key, file))
    throw file.failAt(entry.value, "unknown table [${entry.key}] (${manifestSchema().keys().join(", ")})")
  },
)

/// Reads the text of the `veles.toml` at `path`.
fun parseManifest(path: string, text: string): Manifest throws SettingsError {
  val root = toml.parse(text, strict: true) catch (e) {
    throw SettingsError(message: "$path:${e.line}: ${e.message}")
  }
  val manifest = &Manifest()
  try manifestSchema().apply(Section(path, name: "", table: root), manifest)
  if (manifest.identity.name.isEmpty() && manifest.workspace == null) throw SettingsError(message: "$path: [package] name is required")
  *manifest
}
