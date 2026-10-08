// Rows into structs, with no server: a result as the text the server sends.
use crypto
use time

struct User {
  id:        i64
  fullName:  string
  active:    bool
  score:     f64
  nickname:  string?
  createdAt: time.Timestamp
  tags:      List<string>
  implement Decodable
}

struct Renamed {
  @key(db: "user_id")
  id: i64
  implement Decodable
}

test fun cols(names: List<string>, oids: List<i64>): List<Column> {
  val out: MutableList<Column> = []
  loop ((i, n) in names.enumerate()) {
    out.push(Column(name: n, oid: oids.at(i) ?: 25))
  }
  out.toList()
}

test fun one(columns: List<Column>, row: List<string?>): Rows => Rows(columns, rows: [row], tag: "SELECT 1")

test "a row fills a struct: snake_case columns, text parsed by the field's type" {
  val columns = cols(["id", "full_name", "active", "score", "nickname", "created_at", "tags"], [20, 25, 16, 701, 25, 1184, 1009])
  val rows = one(columns, ["7", "Ada Lovelace", "t", "1.5", null, "2026-10-05 12:34:56.789+02", "{a,\"b c\",d}"])
  val users = try decodeRows<User>(rows)
  val u = users.at(0) ?: fail("one user")
  expect(u.id == 7)
  expect(u.fullName == "Ada Lovelace")
  expect(u.active)
  expect(u.score == 1.5)
  expect(u.nickname == null)
  val want = time.parseRfc3339("2026-10-05T10:34:56.789Z") ?: fail("a timestamp")
  expect(u.createdAt.toMicros() == want.toMicros())
  expect(u.tags == ["a", "b c", "d"])
}

test "@key(db:) names the column outright" {
  val rows = one(cols(["user_id"], [23]), ["42"])
  val r = try decodeRows<Renamed>(rows)
  expect(r.at(0)?.id == 42)
}

test "a scalar target reads the first column" {
  val rows = one(cols(["n"], [20]), ["99"])
  expect(try decodeRows<i64>(rows) == [99])
  val text = one(cols(["s"], [25]), ["hi"])
  expect(try decodeRows<string>(text) == ["hi"])
}

test "a nullable scalar target is null for NULL" {
  val rows = one(cols(["n"], [20]), [null])
  val got = try decodeRows<i64?>(rows)
  expect(got.len() == 1)
  expect(got.filter(x => x == null).len() == 1)
}

test "problems are reported together, by column" {
  val columns = cols(["id", "full_name", "active", "score", "nickname", "created_at", "tags"], [20, 25, 16, 701, 25, 1184, 1009])
  val rows = one(columns, ["seven", "x", "maybe", "fast", null, "2026-10-05 12:00:00+00", "{}"])
  val found = when (decodeRows<User>(rows)) {
    is Ok(_)  => 0
    is Err(e) => e.problems.len()
  }
  expect(found == 3)
}

struct Blob {
  data: List<u8>
  nums: List<i64>
  implement Decodable
}

test "bytea and arrays of numbers" {
  val rows = one(cols(["data", "nums"], [17, 1016]), ["\\x00ff10", "{1,2,3}"])
  val b = (try decodeRows<Blob>(rows)).at(0) ?: fail("one row")
  expect(b.data == [0, 255, 16])
  expect(b.nums == [1, 2, 3])
}

test "timestamps from the server become RFC 3339" {
  expect(rfc3339("2026-10-05 12:34:56.789+02", true) == "2026-10-05T12:34:56.789+02:00")
  expect(rfc3339("2026-10-05 12:34:56+05:30", true) == "2026-10-05T12:34:56+05:30")
  expect(rfc3339("2026-10-05 12:34:56", false) == "2026-10-05T12:34:56Z")
  expect(rfc3339("2026-10-05 12:34:56-0800", true) == "2026-10-05T12:34:56-08:00")
  expect(rfc3339("infinity", true) == "infinity")
}

test "array text: quotes, escapes, NULL and the empty array" {
  expect(arrayItems("{}") == [])
  expect(arrayItems("{a,b}") == ["a", "b"])
  expect(arrayItems("{\"a,b\",\"say \\\"hi\\\"\"}") == ["a,b", "say \"hi\""])
  val nulls = arrayItems("{NULL,\"NULL\",x}") ?: fail("parsed")
  expect(nulls.len() == 3)
  expect(nulls.filter(x => x == null).len() == 1)
  expect(arrayItems("{{1,2},{3,4}}") == null)
  expect(arrayItems("1,2") == null)
}

struct Tagged {
  id: crypto.Uuid
  implement Decodable
}

test "a uuid column reads as a Uuid, and a bad one is a problem" {
  val id = crypto.uuidV4()
  val rows = one(cols(["id"], [2950]), ["$id"])
  expect((try decodeRows<Tagged>(rows)).at(0)?.id == id)
  val bad = one(cols(["id"], [2950]), ["not-a-uuid"])
  expect(decodeRows<Tagged>(bad) is Err)
}
