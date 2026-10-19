// The text the harness compares with the bootstrap compiler's: the tree std/toml reads
// from a file, keys in order, or the one error (plan R1).
use toml { Flag, Integer, Items, Nested, Table, Text, Value }
use text { quote }

/// The dump of `text` read in the manifest dialect, or `error line N: message`.
fun tomlDump(text: string): string {
  val root = toml.parse(text, strict: true) catch (e) {
    return "error line ${e.line}: ${e.message}\n"
  }
  val out = StringBuilder()
  out.appendLine("root @${root.line} ${origin(root)}")
  dumpTable(out, root, "  ")
  out.toString()
}

fun origin(table: *Table): string => if (table.defined()) "defined" else "implied"

fun dumpTable(out: StringBuilder, table: *Table, indent: string) {
  loop (key in table.keys()) {
    val value = table.get(key) ?: continue
    dumpValue(out, "$indent${quote(key)}", value, indent)
  }
}

fun dumpValue(out: StringBuilder, label: string, value: Value, indent: string) {
  val at = "@${value.line}"
  when (value.data) {
    is Text(text)      => out.appendLine("$label: str $at ${quote(text)}")
    is Integer(n)      => out.appendLine("$label: int $at $n")
    is Flag(flag)      => out.appendLine("$label: bool $at $flag")
    is Items(items, _) => {
      out.appendLine("$label: list $at ${items.len()}")
      loop (item in items) dumpValue(out, "$indent  -", item, "$indent  ")
    }
    is Nested(table)   => {
      out.appendLine("$label: table $at ${origin(table)} @${table.line}")
      dumpTable(out, table, "$indent  ")
    }
    else               => out.appendLine("$label: ${value.data.describe()} $at (the manifest dialect never makes it)")
  }
}
