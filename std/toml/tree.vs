/// Builds the tree of tables a document describes and enforces who may define
/// what: a table is defined once, a dotted key makes tables without defining
/// them, an inline table is closed, `[[name]]` extends an array of tables.
struct Tree {
  rules: Rules
  root:  *Table

  static fun of(rules: Rules): Tree => Tree(rules, root: &Table(line: 1))

  fun problem(line: i64, message: string): ParseError => ParseError(line, message)

  /// The table below `parent` under `key`, made with `origin` when it is not
  /// there yet.
  fun child(parent: *Table, key: string, line: i64, origin: Origin): *Table throws ParseError {
    if (val existing = parent.get(key)) {
      return existing.asTable() ?: throw this.problem(line, "${quoted(key)} is already ${existing.data.describe()}, not a table")
    }
    val made = &Table(line, origin)
    parent.entries.set(key, Value(data: Nested(table: made), line))
    made
  }

  /// The table a dotted key `key.` passes through. The full language lets one
  /// reach only tables that no header or inline table has defined.
  fun dotted(parent: *Table, key: string, line: i64): *Table throws ParseError {
    val below = try this.child(parent, key, line, Origin.Dotted)
    val final = below.origin in [Origin.Header, Origin.Inline]
    if (this.rules.closedTables && final) {
      throw this.problem(line, "cannot add to the table ${quoted(key)}: it is defined elsewhere")
    }
    below
  }

  /// Follows `path` below the root up to, not including, its last key: through
  /// the last element of any array of tables on the way, making the tables it
  /// names that are not there yet.
  fun walk(path: List<string>, line: i64): *Table throws ParseError {
    var current = this.root
    val parents = path.take(path.len() - 1)
    loop (depth in 0..<parents.len()) {
      val key = parents.at(depth)
      val here = path.take(depth + 1).join(".")
      val existing = current.get(key)
      if (existing == null) {
        current = try this.child(current, key, line, Origin.Implied)
        continue
      }
      if (val array = existing.asArrayOfTables()) {
        val last = array.last() ?: throw this.problem(line, "${quoted(here)} is an empty array of tables")
        current = last.asTable() ?: throw this.problem(line, "${quoted(here)} is already ${last.data.describe()}, not a table")
        continue
      }
      val below = existing.asTable() ?: throw this.problem(line, "${quoted(here)} is already ${existing.data.describe()}, not a table")
      if (below.origin == Origin.Inline) throw this.problem(line, "cannot add to the inline table ${quoted(here)}")
      current = below
    }
    current
  }

  /// `[path]`: finds or makes the table. A table named twice is an error; one
  /// reached through a parent defined later is fine (`[a.b]` before `[a]`).
  fun header(path: List<string>, line: i64): *Table throws ParseError {
    val parent = try this.walk(path, line)
    val key = path.last() ?: throw this.problem(line, "expected a key")
    val dotted = path.join(".")
    if (val existing = parent.get(key)) {
      val found = existing.asTable() ?: throw this.problem(line, "${quoted(dotted)} is already ${existing.data.describe()}, not a table")
      val reopened = found.origin in [Origin.Header, Origin.Inline] || (this.rules.closedTables && found.origin == Origin.Dotted)
      if (reopened) throw this.problem(line, "table [$dotted] is defined twice")
      found.origin = Origin.Header
      return found
    }
    try this.child(parent, key, line, Origin.Header)
  }

  /// `[[path]]`: a new table at the end of the array of tables there.
  fun arrayHeader(path: List<string>, line: i64): *Table throws ParseError {
    val parent = try this.walk(path, line)
    val key = path.last() ?: throw this.problem(line, "expected a key")
    val made = &Table(line, origin: Origin.Header)
    val element = Value(data: Nested(table: made), line)
    if (val existing = parent.get(key)) {
      val grown = existing.asArrayOfTables() ?: throw this.problem(line, "${quoted(path.join("."))} is already ${existing.data.describe()}, not an array of tables")
      parent.entries.set(key, Value(data: Items(items: grown.concat([element]), ofTables: true), line: existing.line))
    } else {
      parent.entries.set(key, Value(data: Items(items: [element], ofTables: true), line))
    }
    made
  }
}
