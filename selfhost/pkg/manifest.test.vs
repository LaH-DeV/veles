use formatter { formatSchema }
use linter { LintLevel, lintLevelName, lintLevelNamed, lintNames, lintSchema, lints }
use settings { Schema }

/// The text of the manifest at a made-up path, or the refusal.
test fun why(text: string): string {
  parseManifest("veles.toml", text) catch (e) {
    return e.message
  }
  "ok"
}

test fun keysOf<T>(schema: Schema<T>): List<string> => schema.keys()

test documented "every setting of every table says what it is for, and no key is declared twice" {
  expect(identitySchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(formatSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(nativeSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(policySchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(registrySchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(runtimeSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(workspaceSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(lintSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(manifestSchema().settings.all(setting => !setting.doc.isEmpty()))
  expect(keysOf(formatSchema()).toSet().len() == keysOf(formatSchema()).len())
  expect(keysOf(manifestSchema()).toSet().len() == keysOf(manifestSchema()).len())
}

test unknownKeys "an unknown key lists the keys there are, in the order declared" {
  expect(why("[package]\nname = \"a\"\nauthor = \"x\"\n") == "veles.toml:3: unknown [package] key \"author\" (name, version, description, license, veles, registry)")
  expect(why("[package]\nname = \"a\"\n[format]\nwidth = 3\n") == "veles.toml:4: unknown [format] key \"width\" (indent, max_blank_lines, imports, trailing_commas, line_endings)")
  expect(why("[package]\nname = \"a\"\n[runtime]\nworkers = 2\n") == "veles.toml:4: unknown [runtime] key \"workers\" (threads)")
  expect(why("[pakage]\nname = \"a\"\n").startsWith("veles.toml:1: unknown table [pakage] (package, dependencies, dev-dependencies, workspace, format, lint, native, policy, registry, runtime)"))
}

test removedKeys "a removed key says how to write it now" {
  expect(why("[package]\nname = \"a\"\nexports = [\"x\"]\n").contains("'exports' was removed (D89)"))
}

test lintNames "an unknown lint is suggested from the registry" {
  expect(why("[package]\nname = \"a\"\n[lint]\nlong_lin = \"warn\"\n").contains("did you mean \"long_line\"?"))
  expect(why("[package]\nname = \"a\"\n[lint]\nlong_line = \"warn\"\n") == "ok")
}

test lintRegistry "the lints are listed by name, once each, each with its documentation" {
  val names = lintNames()
  expect(names == names.sorted())
  expect(names.toSet().len() == names.len())
  expect(lints().all(lint => !lint.doc.isEmpty()))
  expect(lintLevelNamed("deny") == LintLevel.Deny && lintLevelNamed("Deny") == null && lintLevelNamed("loud") == null)
  expect(lintLevelName(LintLevel.Warn) == "warn")
}
