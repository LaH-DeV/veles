// Tests of the query type: what the text and the values look like, and that
// nothing a value holds can reach the text.

test "values become placeholders and keep their order" {
  val name = "O'Brien"
  val q = sql"select id from users where name = ${name} and age > ${18}"
  expect(q.text() == "select id from users where name = $1 and age > $2")
  expect(q.valueCount() == 2)
  expect(q.args.at(0)?.text == "O'Brien")
  expect(q.args.at(1)?.text == "18")
  expect(q.args.at(1)?.oid == 20)
}

test "a value that looks like SQL is still only a value" {
  val evil = "x'; drop table users; --"
  val q = sql"select * from t where a = ${evil}"
  expect(q.text() == "select * from t where a = $1")
  expect(q.args.at(0)?.text == evil)
}

test "a query with no values is its text" {
  val q = sql"select 1"
  expect(q.text() == "select 1")
  expect(q.valueCount() == 0)
}

test "a Sql among the values is spliced and renumbered" {
  val min = 18
  val adult = sql"age >= ${min}"
  val name = "ann"
  val q = sql"select * from users where name = ${name} and ${adult} and active = ${true}"
  expect(q.text() == "select * from users where name = $1 and age >= $2 and active = $3")
  expect(q.valueCount() == 3)
  expect(q.args.at(1)?.text == "18")
  expect(q.args.at(2)?.text == "t")
}

test "a fragment nests and can be used twice" {
  val a = sql"x = ${1}"
  val both = sql"${a} or ${a}"
  val q = sql"where ${both} and y = ${2}"
  expect(q.text() == "where x = $1 or x = $2 and y = $3")
  expect(q.valueCount() == 3)
}

test "a dollar sign in the text is left alone" {
  val q = sql"select '\$1', \$\$a\$\$, ${5}"
  expect(q.text() == "select '\$1', \$\$a\$\$, $1")
}

test "ident quotes a plain name and refuses the rest" {
  expect(ident("created_at")?.text() == "\"created_at\"")
  expect(ident("public.users")?.text() == "\"public\".\"users\"")
  expect(ident("a b") == null)
  expect(ident("a\"; drop") == null)
  expect(ident("") == null)
  expect(ident("a..b") == null)
  expect(ident("é") == null)
  val order = ident("name") ?: fail("a plain name")
  val q = sql"select * from users order by ${order} desc"
  expect(q.text() == "select * from users order by \"name\" desc")
  expect(q.valueCount() == 0)
}

test "the standard types say how they travel" {
  expect(5.toArg().oid == 20)
  expect(true.toArg().text == "t")
  expect("a".toArg().oid == 0)
  expect(1.5.toArg().text == "1.5")
  val bytes: List<u8> = [1, 2, 3]
  expect(bytes.toArg().bytes == bytes)
  expect(Duration.millis(2).toArg().text == "2000 microseconds")
  val none: string? = null
  expect(none.toArg().text == null)
  expect(none.toArg().bytes == null)
  val some: i64? = 7
  expect(some.toArg().text == "7")
  val token = Secret.of("hunter2")
  expect(token.toArg().secret)
  expect("$token" == "[redacted]")
}

test "dangerouslyRaw is the text as given" {
  val q = Sql.dangerouslyRaw("create table t (id int)")
  expect(q.text() == "create table t (id int)")
  expect(q.valueCount() == 0)
}
