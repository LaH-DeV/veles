/// Reads a document: a sequence of `[headers]`, `[[array headers]]` and
/// `key = value` lines, each key a path.
struct Reader {
  cursor: *Cursor
  tree:   Tree

  static fun of(text: string, rules: Rules): Reader =>
    Reader(cursor: &Cursor(text, rules), tree: Tree.of(rules))

  fun run(): *Table throws ParseError {
    var current = this.tree.root
    loop {
      this.cursor.skipSpace(true)
      if (!this.cursor.more()) return this.tree.root
      if (this.cursor.peek() == '[') {
        current = try this.header()
      } else {
        try this.assignment(current)
      }
    }
  }

  /// `[a.b]` or `[[a.b]]`: the table the lines below it fill.
  fun header(): *Table throws ParseError {
    val cursor = this.cursor
    val isArray = cursor.lookingAt('[', 2)
    if (isArray && !cursor.rules.arrayTables) {
      throw cursor.fail("arrays of tables ([[...]]) are not supported in veles.toml")
    }
    val line = cursor.line
    cursor.pos += if (isArray) 2 else 1
    val path = try this.keyPath()
    val table = if (isArray) {
      if (!cursor.eat(']') || !cursor.eat(']')) throw cursor.fail("expected ']]' to close the array of tables name")
      try this.tree.arrayHeader(path, line)
    } else {
      if (!cursor.eat(']')) throw cursor.fail("expected ']' to close the table name")
      try this.tree.header(path, line)
    }
    try cursor.endOfLine()
    table
  }

  /// `a`, `a.b`, `"a b".c`: the keys of a header, blanks allowed around the dots.
  fun keyPath(): List<string> throws ParseError {
    val path: MutableList<string> = []
    loop {
      this.cursor.skipSpace(false)
      path.push(try this.cursor.key())
      this.cursor.skipSpace(false)
      if (!this.cursor.eat('.')) return path.toList()
    }
  }

  /// `key = value`, or `a.b.c = value`: a dotted key lands in tables below `table`.
  fun assignment(table: *Table) throws ParseError {
    val cursor = this.cursor
    var key = try cursor.key()
    val line = cursor.line
    cursor.skipSpace(false)
    var target = table
    loop (cursor.more() && cursor.peek() == '.') {
      cursor.pos += 1
      cursor.skipSpace(false)
      target = try this.tree.dotted(target, key, line)
      key = try cursor.key()
      cursor.skipSpace(false)
    }
    if (!cursor.eat('=')) throw cursor.fail("expected 'key = value'")
    cursor.skipSpace(false)
    val value = try this.value()
    if (target.has(key)) throw ParseError(line, message: "key ${quoted(key)} is set twice")
    target.entries.set(key, value)
    try cursor.endOfLine()
  }

  fun value(): Value throws ParseError {
    val cursor = this.cursor
    if (!cursor.more()) throw cursor.fail("expected a value")
    val line = cursor.line
    val first = cursor.peek()
    if (first in ['"', '\'']) return Value(data: Text(value: try cursor.string()), line)
    if (first == '[') return try this.list(line)
    if (first == '{') return try this.inlineTable(line)
    Value(data: try this.word(), line)
  }

  /// A value that is not quoted, nested or bracketed: a boolean, a number, a
  /// date or a time.
  fun word(): Data throws ParseError {
    val cursor = this.cursor
    val start = cursor.pos
    loop (cursor.more() && !endsWord(cursor.peek())) cursor.pos += 1
    var word = cursor.since(start)
    if (word in ["true", "false"]) return Flag(value: word == "true")
    if (word.isEmpty()) throw cursor.fail("expected a value")
    if (!cursor.rules.richValues) return try this.plainInteger(word)
    // a date and a time may be separated by a space
    if (this.dateThenTime(word)) {
      val dateEnd = cursor.pos
      cursor.pos += 1
      loop (cursor.more() && !endsWord(cursor.peek())) cursor.pos += 1
      val joined = cursor.since(start)
      if (momentShape(joined) != null) word = joined else cursor.pos = dateEnd
    }
    if (val shape = momentShape(word)) return Moment(shape, text: word)
    try this.number(word)
  }

  /// True when `word` is a whole date and a time follows after one blank.
  fun dateThenTime(word: string): bool =>
    word.len() == 10 && momentShape(word) == MomentShape.LocalDate && this.cursor.peekAt(0) == ' ' && this.cursor.peekAt(1).isAsciiDigit()

  /// A manifest's number: decimal digits, underscores anywhere.
  fun plainInteger(word: string): Data throws ParseError {
    val n = parseDecimal(word.replace("_", "")) ?: throw this.unreadable(word)
    Integer(value: n)
  }

  fun number(word: string): Data throws ParseError {
    readNumber(word) catch (problem) {
      when (problem) {
        is OutOfRange => throw this.cursor.fail("the number ${quoted(word)} is not in the range of a 64-bit integer")
        is Unreadable => throw this.unreadable(word)
      }
    }
  }

  fun unreadable(word: string): ParseError =>
    this.cursor.fail("cannot read ${quoted(word)}: strings are written in quotes (\"$word\")")

  fun list(line: i64): Value throws ParseError {
    val cursor = this.cursor
    cursor.pos += 1
    val items: MutableList<Value> = []
    loop {
      cursor.skipSpace(true)
      if (cursor.eat(']')) break
      items.push(try this.value())
      cursor.skipSpace(true)
      if (cursor.eat(',')) continue
      if (!cursor.eat(']')) throw cursor.fail("expected ',' or ']' in the list")
      break
    }
    Value(data: Items(items), line)
  }

  fun inlineTable(line: i64): Value throws ParseError {
    val cursor = this.cursor
    cursor.pos += 1
    val table = &Table(line, origin: Origin.Inline)
    val result = Value(data: Nested(table), line)
    cursor.skipSpace(false)
    if (cursor.eat('}')) return result
    loop {
      cursor.skipSpace(false)
      if (cursor.more() && cursor.peek() == '\n') {
        throw cursor.fail("an inline table is one line: close it with '}' or use a [table] header")
      }
      var key = try cursor.key()
      cursor.skipSpace(false)
      var target = table
      // `{ a.b = 1 }`: a dotted key below the inline table
      loop (cursor.rules.closedTables && cursor.more() && cursor.peek() == '.') {
        cursor.pos += 1
        cursor.skipSpace(false)
        target = try this.tree.child(target, key, line, Origin.Inline)
        key = try cursor.key()
        cursor.skipSpace(false)
      }
      if (!cursor.eat('=')) throw cursor.fail("expected '=' in the inline table")
      cursor.skipSpace(false)
      val item = try this.value()
      if (target.has(key)) throw cursor.fail("key ${quoted(key)} is set twice")
      target.entries.set(key, item)
      cursor.skipSpace(false)
      if (cursor.eat(',')) continue
      if (!cursor.eat('}')) throw cursor.fail("expected ',' or '}' in the inline table (an inline table is one line)")
      return result
    }
  }
}

/// Where an unquoted value ends.
fun endsWord(byte: u8): bool =>
  byte in [' ', '\t', '\n', ',', ']', '}', '#']
