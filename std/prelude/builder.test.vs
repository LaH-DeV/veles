// D207: StringBuilder's in-place edits, and receiver blocks with `apply`
// and `edit`.

struct Server {
  var host: string = "localhost"
  var port: i64 = 80
}

struct Deploy {
  var server: Server = Server()
  var name:   string = "  Web Front  "
}

test fun slug(text: string): string => text.edit {
  trim()
  toLower()
  replace(" ", "-")
}

test fun built(body: fun on StringBuilder()): string {
  val builder = StringBuilder()
  body(builder)
  builder.toString()
}

test "trim drops ASCII whitespace at both ends only" {
  val builder = StringBuilder()
  builder.append(" \t a b \r\n")
  builder.trim()
  expect(builder.toString() == "a b")
  builder.clear()
  builder.append("   ")
  builder.trim()
  expect(builder.isEmpty())
}

test "toLower and toUpper change ASCII letters and nothing else" {
  val builder = StringBuilder()
  builder.append("MiXeD 123 é")
  builder.toUpper()
  expect(builder.toString() == "MIXED 123 é")
  builder.toLower()
  expect(builder.toString() == "mixed 123 é")
}

test "replace shrinks, keeps and grows the text" {
  val builder = StringBuilder()
  builder.append("a--b--c")
  builder.replace("--", "-")
  expect(builder.toString() == "a-b-c")
  builder.replace("-", "+")
  expect(builder.toString() == "a+b+c")
  builder.replace("+", " plus ")
  expect(builder.toString() == "a plus b plus c")
  builder.replace("", "x")
  builder.replace("absent", "x")
  expect(builder.toString() == "a plus b plus c")
}

test "insert puts text at a byte index on a character's edge" {
  val builder = StringBuilder()
  builder.append("held")
  builder.insert(0, "[")
  builder.insert(builder.len(), "]")
  builder.insert(3, "l")
  expect(builder.toString() == "[hell" + "d]")
}

test "edit gives a changed copy of a string and leaves it alone" {
  val name = "  Hello Big World "
  expect(slug(name) == "hello-big-world")
  expect(name == "  Hello Big World ")
  expect("".edit { trim() } == "")
}

test "a receiver block reaches its receiver's members bare" {
  expect(built { append("a"); appendLine("b"); append("c") } == "ab\nc")
}

test "edit on a List, a Map and a struct copies once and keeps the original" {
  val scores = [3, 9, 1, 7]
  val ranked = scores.edit {
    sort()
    removeAt(0)
  }
  expect(scores == [3, 9, 1, 7])
  expect(ranked == [3, 7, 9])
  val counts = ["a": 1].edit { set("b", 2) }
  expect(counts.len() == 2)
  val server = Server()
  val moved = server.edit { port += 1 }
  expect(server.port == 80 && moved.port == 81)
}

test "apply changes the place it is called on" {
  var server = Server()
  val same = server.apply { port = 8080 }
  expect(server.port == 8080 && same.port == 8080)
  val fresh = Server().apply { host = "example.org" }
  expect(fresh.host == "example.org" && fresh.port == 80)
  var deploy = Deploy()
  deploy.server.apply { port = 443 }
  deploy.name.apply { trim(); toUpper() }
  expect(deploy.server.port == 443 && deploy.name == "WEB FRONT")
  val tags: MutableList<string> = ["x"]
  tags.apply { push("y") }
  expect(tags.len() == 2)
}

test "a block may fill a plain fun()" {
  var runs = 0
  val twice = (f: fun()) => {
    f()
    f()
  }
  twice { runs += 1 }
  expect(runs == 2)
}
