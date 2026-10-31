use settings
use toml

struct ToolOptions {
  var name:       string = ""
  var mode:       string = "safe"
  var labels:     List<string> = []
  var extensions: MutableMap<string, string> = [:]
}

test fun toolSchema(removed: List<(string, string)> = []): settings.Schema<ToolOptions> => settings.Schema(
  table: "tool",
  settings: [
    settings.Setting<ToolOptions>.text("name", "the tool name", (section, value, text, options) => options.name = text),
    settings.Setting<ToolOptions>.choice(
      "mode", "the operating mode", ["safe", "strict"],
      "[tool] mode is \"safe\" or \"strict\"",
      (mode, options) => options.mode = mode,
    ),
    settings.Setting<ToolOptions>.names("labels", "labels to report", "labels = [\"syntax\"]", (section, value, labels, options) => options.labels = labels),
  ],
  removed,
)

test fun section(text: string): settings.Section {
  val document = toml.parse(text) catch (e) {
    panic(e.message)
  }
  val table = document.get("tool")?.asTable() ?: panic("the test document has [tool]")
  settings.Section(path: "tool.toml", name: "tool", table)
}

test fun refusal(text: string, schema: settings.Schema<ToolOptions>): string {
  val options = &ToolOptions()
  schema.apply(section(text), options) catch (e) {
    return e.message
  }
  "ok"
}

test schema "builds public settings and applies a typed configuration" {
  val schema = toolSchema()
  val options = &ToolOptions()
  try schema.apply(section("[tool]\nname = \"veles\"\nmode = \"strict\"\nlabels = [\"syntax\", \"systems\"]"), options)
  expect(options.name == "veles")
  expect(options.mode == "strict")
  expect(options.labels == ["syntax", "systems"])
  expect(schema.keys() == ["name", "mode", "labels"])
  expect(if (val setting = schema.find("mode")) setting.key == "mode" else false)
  expect(if (val setting = schema.find("missing")) setting.key != "missing" else true)
}

test errors "locates unknown and removed keys in the TOML source" {
  expect(refusal("[tool]\nextra = true", toolSchema()) == "tool.toml:2: unknown [tool] key \"extra\" (name, mode, labels)")
  val schema = toolSchema(removed: [("old_mode", "[tool] old_mode was removed; use mode")])
  expect(refusal("[tool]\nold_mode = \"safe\"", schema) == "tool.toml:2: [tool] old_mode was removed; use mode")
}

test extensionKeys "handles tool-owned extension keys" {
  val schema = settings.Schema<ToolOptions>(
    table: "tool",
    settings: toolSchema().settings,
    fallback: (section, entry, options) => {
      val value = entry.value.asString() ?: throw section.failAt(entry.value, "extension values are strings")
      options.extensions.set(entry.key, value)
    },
  )
  val options = &ToolOptions()
  try schema.apply(section("[tool]\ncustom_mode = \"fast\""), options)
  expect(if (val value = options.extensions.get("custom_mode")) value == "fast" else false)
}
