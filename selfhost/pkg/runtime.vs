use settings { Schema, Section, Setting, SettingsError }
/// `[runtime]` (D143): how the program runs.
struct RuntimeSettings {
  /// The default pool's thread count; null for one per core.
  var threads: i64? = null
}

fun runtimeSchema(): Schema<RuntimeSettings> => Schema(
  table: "runtime",
  settings: [
    Setting<RuntimeSettings>(key: "threads", doc: "threads in the default pool: \"auto\" (one per core) or 1 to 256", read: (section, value, runtime) => {
      val count = value.asInt()
      if (value.asString() == "auto") {
        runtime.threads = null
      } else if (count != null && count >= 1 && count <= 256) {
        runtime.threads = count
      } else {
        throw section.failAt(value, "[runtime] threads is \"auto\" (one per core, the default) or a number from 1 to 256, e.g. threads = 4")
      }
    }),
  ],
)

fun readRuntime(section: Section): i64? throws SettingsError {
  val runtime = &RuntimeSettings()
  try runtimeSchema().apply(section, runtime)
  runtime.threads
}
