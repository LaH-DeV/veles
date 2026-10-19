// The text the harness compares with sema/toml_dump.go: the parsed tree, keys
// in order, or the one error (plan R1). The reader is std/toml, strict.
use toml { Kind, Table, Value }
use ast { quote }

/// The dump of `text`, or `error line N: message` when it is refused.
fun tomlDump(text: string): string {
  val out = StringBuilder()
  val root = toml.parse(text, strict: true) catch (e) {
    return "error line ${e.line}: ${e.message}\n"
  }
  out.appendLine("root @${root.line} ${definedText(root)}")
  dumpTable(out, root, "  ")
  out.toString()
}

fun definedText(table: *Table): string => if (table.defined()) "defined" else "implied"

fun dumpTable(out: StringBuilder, table: *Table, indent: string) {
  loop (key in table.keys()) {
    val value = table.get(key) ?: continue
    dumpValue(out, "$indent${quote(key)}", value, indent)
  }
}

fun dumpValue(out: StringBuilder, label: string, value: Value, indent: string) {
  when (value.kind) {
    Kind.Str   => out.appendLine("$label: str @${value.line} ${quote(value.str)}")
    Kind.Int   => out.appendLine("$label: int @${value.line} ${value.num}")
    Kind.Bool  => out.appendLine("$label: bool @${value.line} ${value.flag}")
    Kind.List  => {
      out.appendLine("$label: list @${value.line} ${value.items.len()}")
      loop (item in value.items) dumpValue(out, "$indent  -", item, "$indent  ")
    }
    Kind.Table => {
      val table = value.table ?: panic("dumpValue: a table value holds its table")
      out.appendLine("$label: table @${value.line} ${definedText(table)} @${table.line}")
      dumpTable(out, table, "$indent  ")
    }
    else       => out.appendLine("$label: ${toml.describe(value.kind)} @${value.line} (strict mode never makes it)")
  }
}
