// Tests of std/http's forms and query strings (D94): the fields, and reading
// them into a struct.

use codec

struct Profile {
  name:     string
  age:      i64
  height:   f64 = 0.0
  admin:    bool = false
  nickname: string? = null
  score:    i64? = null
  tags:     List<string> = []
  ids:      List<i64> = []
  implement Decodable
}

struct Sized {
  count: u64
  implement Decodable
}

enum Mood {
  Happy
  Sad
}

struct Feeling {
  mood: Mood
  implement Decodable
}

struct Snake {
  firstName: string
  implement Decodable
}

struct Nested {
  name:  string
  inner: Profile? = null
  implement Decodable
}

fun profile(text: string): Profile throws DecodeError = try decodeFields<Profile>(Fields.parse(text), codec.KeyStyle.AsWritten)

test "fields keep their order and repeats" {
  val f = Fields.parse("a=1&b=x%20y&a=3&flag&&c=+")
  expect(f.names() == ["a", "b", "flag", "c"])
  expect(f.all("a") == ["1", "3"])
  expect(f.get("a") == "1")
  expect(f.get("b") == "x y")
  expect(f.get("flag") == "")
  expect(f.get("c") == " ")
  expect(f.get("nope") == null)
  expect(f.all("nope").isEmpty())
  expect(Fields.parse("").names().isEmpty())
}

test "a form is read into a struct" {
  val p = require(profile("name=Ada+Lovelace&age=36&height=1.68&admin=on&nickname=ada&score=9&tags=a&tags=b&ids=1&ids=2"))
  expect(p.name == "Ada Lovelace")
  expect(p.age == 36)
  expect(p.height == 1.68)
  expect(p.admin)
  expect(p.nickname == "ada")
  expect(p.score == 9)
  expect(p.tags == ["a", "b"])
  expect(p.ids == [1, 2])
}

test "defaults apply to what is absent, and empty optional values are null" {
  val p = require(profile("name=Ada&age=36&nickname=&score=&tags="))
  expect(p.height == 0.0)
  expect(!p.admin)
  expect(p.nickname == null)
  expect(p.score == null)
  expect(p.tags.isEmpty())
}

test "a single value is a list of one" {
  val p = require(profile("name=Ada&age=1&tags=solo&ids=7"))
  expect(p.tags == ["solo"])
  expect(p.ids == [7])
}

test "booleans read as a checkbox sends them, and as people write them" {
  loop (yes in ["on", "true", "yes", "1", "TRUE", " On "]) {
    expect(require(profile("name=a&age=1&admin=$yes")).admin)
  }
  loop (no in ["off", "false", "no", "0"]) {
    expect(!require(profile("name=a&age=1&admin=$no")).admin)
  }
}

test "problems are reported together, each at its field" {
  val result = profile("age=x&height=tall&admin=maybe&ids=1&ids=two")
  val e = when (result) {
    is Ok       => fail("should not decode")
    is Err(err) => err
  }
  val paths = e.problems.map(p => p.path)
  expect(paths.contains("age"))
  expect(paths.contains("height"))
  expect(paths.contains("admin"))
  expect(paths.contains("ids[1]"))
  expect(paths.contains("name"))
  expect(e.message().contains("age: expected an integer, found \"x\""))
  expect(e.message().contains("name: missing"))
}

test "several values where one is asked for is a problem" {
  val result = profile("name=a&name=b&age=1")
  val e = when (result) {
    is Ok       => fail("should not decode")
    is Err(err) => err
  }
  expect(e.message().contains("name: expected one value, found 2"))
}

test "a u64 refuses a negative number" {
  val ok = require(decodeFields<Sized>(Fields.parse("count=5"), codec.KeyStyle.AsWritten))
  expect(ok.count == 5)
  val bad = decodeFields<Sized>(Fields.parse("count=-1"), codec.KeyStyle.AsWritten)
  expect(bad is Err)
}

test "an enum is read by name" {
  val f = require(decodeFields<Feeling>(Fields.parse("mood=Sad"), codec.KeyStyle.AsWritten))
  expect(f.mood == Mood.Sad)
  expect(decodeFields<Feeling>(Fields.parse("mood=Angry"), codec.KeyStyle.AsWritten) is Err)
}

test "nothing nested exists in a form" {
  expect(decodeFields<Nested>(Fields.parse("name=a&inner=b"), codec.KeyStyle.AsWritten) is Err)
}

test "a key style renames the fields the struct is read from" {
  val p = require(decodeFields<Profile>(Fields.parse("name=a&age=1&NICKNAME=x"), codec.KeyStyle.AsWritten))
  expect(p.nickname == null)
  val s = require(decodeFields<Snake>(Fields.parse("first_name=Ada"), codec.KeyStyle.SnakeCase))
  expect(s.firstName == "Ada")
}

test "form<T> answers 400 with the problems and 415 for another type" {
  val form = ["Content-Type": "application/x-www-form-urlencoded; charset=utf-8"]
  val ok = call(handler(req => Response.text((try req.form<Profile>()).name)), Method.post, "/", body: "name=Ada&age=1", headers: form)
  expect(ok.status == Status.ok)
  val bad = call(handler(req => Response.text((try req.form<Profile>()).name)), Method.post, "/", body: "age=x", headers: form)
  expect(bad.status == Status.badRequest)
  val text = bad.body.decodeUtf8() ?: ""
  expect(text.startsWith("invalid form:\n"))
  expect(text.contains("name: missing"))
  val json = call(handler(req => Response.text((try req.form<Profile>()).name)), Method.post, "/", body: "{}", headers: ["Content-Type": "application/json"])
  expect(json.status == Status.unsupportedMediaType)
  val none = call(handler(req => Response.text((try req.form<Profile>()).name)), Method.post, "/", body: "name=a&age=1")
  expect(none.status == Status.unsupportedMediaType)
}

test "query<T> reads the query string, repeats included" {
  val resp = call(handler(req => Response.text("${(try req.query<Profile>()).tags}")), Method.get, "/find?name=a&age=1&tags=x&tags=y")
  expect((resp.body.decodeUtf8() ?: "") == "[x, y]")
  val bad = call(handler(req => Response.text("${(try req.query<Profile>()).tags}")), Method.get, "/find?age=z")
  expect(bad.status == Status.badRequest)
  expect((bad.body.decodeUtf8() ?: "").startsWith("invalid query:\n"))
}

test "the untyped readers see every value" {
  val h = handler(req => Response.text("${try req.formValue("a")}/${try req.formValues("a")}/${req.queryFields().all("q")}/${req.query.get("q")}"))
  val resp = call(h, Method.post, "/?q=1&q=2", body: "a=x&a=y", headers: ["Content-Type": "application/x-www-form-urlencoded"])
  expect((resp.body.decodeUtf8() ?: "") == "x/[x, y]/[1, 2]/2")
}
