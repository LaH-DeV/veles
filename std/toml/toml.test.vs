test fun read(text: string): *Table => parse(text) catch (e) {
  panic("bad test document: line ${e.line}: ${e.message}")
}

/// The message of the error `text` gives, with its line, or "ok".
test fun why(text: string, strict: bool = false): string {
  parse(text, strict) catch (e) {
    return "line ${e.line}: ${e.message}"
  }
  "ok"
}

test fun str(table: *Table, key: string): string => table.get(key)?.asString() ?: "-"

test fun int(table: *Table, key: string): i64 => table.get(key)?.asInt() ?: -1

test scalars "strings, numbers and booleans" {
  val doc = read("title = \"TOML\"\nlit = 'C:\\path'\nport = 8_080\nneg = -17\nok = true\nno = false\n")
  expect(str(doc, "title") == "TOML")
  expect(str(doc, "lit") == "C:\\path")
  expect(int(doc, "port") == 8080)
  expect(int(doc, "neg") == -17)
  expect(doc.get("ok")?.asBool() == true)
  expect(doc.get("no")?.asBool() == false)
  expect(doc.get("title")?.line == 1)
  expect(doc.get("neg")?.line == 4)
}

test integers "hex, octal, binary and the limits" {
  val doc = read("h = 0xDEADBEEF\no = 0o755\nb = 0b1101\nz = 0\nmax = 9223372036854775807\nmin = -9223372036854775808\nu = 0xdead_beef\n")
  expect(int(doc, "h") == 3735928559)
  expect(int(doc, "o") == 493)
  expect(int(doc, "b") == 13)
  expect(int(doc, "z") == 0)
  expect(int(doc, "max") == 9223372036854775807)
  expect(int(doc, "u") == 3735928559)
  expect(why("a = 9223372036854775808") == "line 1: the number \"9223372036854775808\" is not in the range of a 64-bit integer")
  expect(why("a = 0x8000000000000000").startsWith("line 1: the number"))
  expect(why("a = 007").startsWith("line 1: cannot read"))
  expect(why("a = 1__0").startsWith("line 1: cannot read"))
  expect(why("a = _1").startsWith("line 1: cannot read"))
  expect(why("a = 1_").startsWith("line 1: cannot read"))
}

test floats "fractions, exponents, inf and nan" {
  val doc = read("a = 3.14\nb = -0.5\nc = 5e+22\nd = 1e06\ne = 6.626e-34\nf = 224_617.445_991\ng = inf\nh = -inf\ni = nan\nj = 7\n")
  expect(doc.get("a")?.asFloat() == 3.14)
  expect(doc.get("b")?.asFloat() == -0.5)
  expect(doc.get("c")?.asFloat() == 5e22)
  expect(doc.get("d")?.asFloat() == 1000000.0)
  expect(doc.get("f")?.asFloat() == 224617.445991)
  expect(doc.get("g")?.asFloat()?.isInfinite() == true)
  expect(doc.get("i")?.asFloat()?.isNaN() == true)
  expect(doc.get("j")?.kind == Kind.Int)
  expect(doc.get("j")?.asFloat() == 7.0)
  expect(why("a = 1.").startsWith("line 1: cannot read"))
  expect(why("a = .5").startsWith("line 1: cannot read"))
  expect(why("a = 1e").startsWith("line 1: cannot read"))
  expect(why("a = 03.5").startsWith("line 1: cannot read"))
}

test dates "the four shapes of dates and times" {
  val doc = read("a = 1979-05-27T07:32:00Z\nb = 1979-05-27T00:32:00.999999-07:00\nc = 1979-05-27 07:32:00\nd = 1979-05-27\ne = 07:32:00\nf = 07:32:00.5\n")
  expect(doc.get("a")?.kind == Kind.OffsetDateTime)
  expect(doc.get("b")?.kind == Kind.OffsetDateTime)
  expect(doc.get("c")?.kind == Kind.LocalDateTime)
  expect(str(doc, "c") == "-")
  expect(doc.get("c")?.str == "1979-05-27 07:32:00")
  expect(doc.get("d")?.kind == Kind.LocalDate)
  expect(doc.get("e")?.kind == Kind.LocalTime)
  expect(doc.get("f")?.str == "07:32:00.5")
  expect(why("a = 1979-13-01").startsWith("line 1: cannot read"))
  expect(why("a = 2023-02-29").startsWith("line 1: cannot read"))
  expect(why("a = 2024-02-29") == "ok")
  expect(why("a = 25:00:00").startsWith("line 1: cannot read"))
}

test escapes "basic string escapes" {
  val doc = read("a = \"tab\\there\"\nb = \"\\u00E9\"\nc = \"\\U0001F600\"\nd = \"\\b\\f\\r\\\"\\\\\"\n")
  expect(str(doc, "a") == "tab\there")
  expect(str(doc, "b") == "é")
  expect(str(doc, "c") == "\u{1F600}")
  expect(doc.get("d")?.str?.len() == 5)
  expect(why("a = \"\\q\"") == "line 1: unknown escape \\q")
  expect(why("a = \"\\ud800\"") == "line 1: bad \\u escape")
  expect(why("a = \"\\U00110000\"") == "line 1: bad \\u escape")
  expect(why("a = \"x\ty\"") == "ok")
  expect(why("a = \"x\u{1}y\"") == "line 1: a control character in a string must be escaped")
}

test multiline "multi-line strings" {
  val doc = read("a = \"\"\"\nRoses are red\nViolets are blue\"\"\"\nb = \"\"\"\nThe quick brown \\\n\n   fox\"\"\"\nc = '''\nraw \\n stays\n'''\nd = \"\"\"say \"\"hi\"\"\"\"\"\ne = 5\n")
  expect(str(doc, "a") == "Roses are red\nViolets are blue")
  expect(str(doc, "b") == "The quick brown fox")
  expect(str(doc, "c") == "raw \\n stays\n")
  expect(str(doc, "d") == "say \"\"hi\"\"")
  expect(int(doc, "e") == 5)
  expect(doc.get("e")?.line == 12)
  expect(why("a = \"\"\"open") == "line 1: unterminated string")
}

test arrays "nested and mixed arrays, several lines" {
  val doc = read("a = [1, 2, 3]\nb = [ \"x\", 'y', [1.5, true] ]\nc = [\n  1, # one\n  2,\n]\nd = []\n")
  expect(doc.get("a")?.asList()?.len() == 3)
  expect(doc.get("b")?.asList()?.len() == 3)
  expect(doc.get("c")?.asList()?.len() == 2)
  expect(doc.get("d")?.asList()?.isEmpty() == true)
  expect(why("a = [1 2]") == "line 1: expected ',' or ']' in the list")
}

test tables "headers, dotted keys and inline tables" {
  val doc = read("[server]\nhost = \"x\"\n[server.tls]\non = true\n[owner]\nname.first = \"Tom\"\nname.last = \"P\"\npoint = { x = 1, y = 2 }\n")
  val server = doc.get("server")?.asTable() ?: panic("server")
  expect(str(server, "host") == "x")
  expect(server.get("tls")?.asTable()?.get("on")?.asBool() == true)
  val owner = doc.get("owner")?.asTable() ?: panic("owner")
  expect(owner.get("name")?.asTable()?.get("last")?.asString() == "P")
  expect(owner.get("point")?.asTable()?.get("y")?.asInt() == 2)
  expect(server.defined())
  expect(owner.get("name")?.asTable()?.defined() == false)
  expect(owner.get("point")?.asTable()?.defined() == true)
}

test arrayOfTables "[[array of tables]]" {
  val doc = read("[[fruit]]\nname = \"apple\"\n[fruit.physical]\ncolor = \"red\"\n[[fruit.variety]]\nname = \"red delicious\"\n[[fruit.variety]]\nname = \"granny smith\"\n[[fruit]]\nname = \"banana\"\n")
  val fruit = doc.get("fruit")?.asList() ?: panic("fruit")
  expect(fruit.len() == 2)
  expect(fruit.at(0)?.asTable()?.get("physical")?.asTable()?.get("color")?.asString() == "red")
  expect(fruit.at(0)?.asTable()?.get("variety")?.asList()?.len() == 2)
  expect(fruit.at(1)?.asTable()?.get("name")?.asString() == "banana")
  expect(why("a = [1]\n[[a]]\n") == "line 2: \"a\" is already a list, not an array of tables")
  expect(why("[[a]]\n[a]\n").startsWith("line 2:"))
}

test refusals "documents the full language refuses" {
  expect(why("[a]\n[a]\n") == "line 2: table [a] is defined twice")
  expect(why("a.b = 1\n[a]\n") == "line 2: table [a] is defined twice")
  expect(why("a = { x = 1 }\na.y = 2\n") == "line 2: cannot add to the table \"a\": it is defined elsewhere")
  expect(why("a = { x = 1 }\n[a.b]\n") == "line 2: cannot add to the inline table \"a\"")
  expect(why("a = 1\na = 2\n") == "line 2: key \"a\" is set twice")
  expect(why("a = {\n x = 1 }\n").startsWith("line 1: an inline table is one line"))
}

test strict "strict mode reads a manifest's subset only" {
  expect(why("a = 1", strict: true) == "ok")
  expect(why("a = 1.5", strict: true).startsWith("line 1: cannot read"))
  expect(why("a = 1979-05-27", strict: true).startsWith("line 1: cannot read"))
  expect(why("a = 0xff", strict: true).startsWith("line 1: cannot read"))
  expect(why("[[a]]", strict: true) == "line 1: arrays of tables ([[...]]) are not supported in veles.toml")
  expect(why("a = \"\\b\"", strict: true) == "line 1: unknown escape \\b")
  expect(why("a.b = 1\n[a]\n", strict: true) == "ok")
}

test lines "values remember their line" {
  val doc = read("# c\n\na = 1\n[t]\n\nb = [\n  1,\n]\nc = 3\n")
  val t = doc.get("t")?.asTable() ?: panic("t")
  expect(t.line == 4)
  expect(t.get("b")?.line == 6)
  expect(t.get("c")?.line == 9)
  expect(why("a = 1\n\n\n# c\nb = [\n  1,\n  oops,\n]\n") == "line 7: cannot read \"oops\": strings are written in quotes (\"oops\")")
}
