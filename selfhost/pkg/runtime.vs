use settings { Schema, Section, Setting, SettingsError }
use text { quote }
/// `[runtime]` (D143): how the program runs.
struct RuntimeSettings {
  /// The default pool's thread count; null for one per core.
  var threads: i64? = null
  /// `gc.minHeap` in bytes: no collection below it (D200); null for the default.
  var gcMinHeap: i64? = null
  /// `gc.goal`: the percent the heap grows over the live data (D200); null for the default.
  var gcGoal: i64? = null
}

fun runtimeSchema(): Schema<RuntimeSettings> => Schema(
  table: "runtime",
  settings: [
    Setting<RuntimeSettings>(key: "threads", doc: "threads in the default pool: \"auto\" (one per core) or 1 to 256", read: (section, value, runtime) => {
      val count = value.asInt()
      if (value.asString() == "auto") {
        runtime.threads = null
      } else if (count != null && count in 1..256) {
        runtime.threads = count
      } else {
        throw section.failAt(value, "[runtime] threads is \"auto\" (one per core, the default) or a number from 1 to 256, e.g. threads = 4")
      }
    }),
    Setting<RuntimeSettings>(key: "gc", doc: "the collector: gc.minHeap (no collection below it) and gc.goal (percent of growth)", read: (section, value, runtime) => {
      val table = value.asTable() ?: throw section.failAt(value, "[runtime] gc is a table: gc.minHeap = \"32MiB\", gc.goal = 100")
      try gcSchema().apply(section.within(table), runtime)
    }),
  ],
)

fun gcSchema(): Schema<RuntimeSettings> => Schema(
  table: "runtime",
  settings: [
    Setting<RuntimeSettings>(key: "minHeap", doc: "no collection before the heap holds this much: \"32MiB\"", read: (section, value, runtime) => {
      val size = heapSize(value.asString() ?: "")
      if (size == null || size !in (1 << 20)..(64 << 30)) {
        throw section.failAt(value, "[runtime] gc.minHeap is a size from \"1MiB\" to \"64GiB\": a whole number with B, KiB, MiB or GiB, e.g. gc.minHeap = \"32MiB\"")
      }
      runtime.gcMinHeap = size
    }),
    Setting<RuntimeSettings>(key: "goal", doc: "the percent the heap grows over the live data before the next collection", read: (section, value, runtime) => {
      val goal = value.asInt()
      if (goal == null || goal !in 10..1000) {
        throw section.failAt(value, "[runtime] gc.goal is the percent the heap grows over the live data before the next collection, a number from 10 to 1000 (default 100), e.g. gc.goal = 200")
      }
      runtime.gcGoal = goal
    }),
  ],
  fallback: (section, entry, runtime) => {
    throw section.failAt(entry.value, "unknown [runtime] key ${quote("gc.${entry.key}")} (gc.minHeap, gc.goal)")
  },
)

/// "32MiB", "512KiB", "1GiB" or "100B" in bytes; null for anything else.
fun heapSize(text: string): i64? {
  loop ((suffix, shift) in [("KiB", 10), ("MiB", 20), ("GiB", 30), ("B", 0)]) {
    if (!text.endsWith(suffix)) continue
    val digits = text.slice(0, text.len() - suffix.len())
    val n = digits.toInt() ?: return null
    if (n < 0 || n > (1 << 40) || "$n" != digits) return null
    return n << shift
  }
  null
}

fun readRuntime(section: Section): RuntimeSettings throws SettingsError {
  val runtime = &RuntimeSettings()
  try runtimeSchema().apply(section, runtime)
  *runtime
}
