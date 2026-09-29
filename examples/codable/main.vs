// codable: the compiler writes the wire code (D58). A struct asks for it
// with one line — `implement Codable` — and `std/json` reads and writes it; the
// same derived code would drive a database row or the environment, since
// the traits are format-agnostic. Every problem in a document comes back
// at once, with its path.
use codec, io { println }, json

enum Role {
  Admin
  Member
}

struct Address {
  street: string
  zip:    string
  implement Codable
}

struct User {
  @key("user_id") id: i64      // this key, in every format
  name:    string              // required: no default, not nullable
  email:   string?             // null or absent → null
  role:    Role = Role.Member  // absent → the default
  tags:    List<string> = []
  address: Address?
  @skip passwordHash: string = ""  // never on the wire
  implement Codable
}

/// A closed family travels with a discriminator: `{"kind": "circle", ...}`.
@tag("kind")
sealed trait Shape
@key("circle")
struct Circle : Shape {
  r: f64
}
@key("rect")
struct Rect : Shape {
  w: f64
  h: f64
}
implement Codable for Shape

/// Generic: `Page<T>` is Codable exactly when `T` is.
struct Page<T> {
  items: List<T>
  total: i64
  implement Codable
}

/// Half by hand: `encode` is written, `decode` is derived.
struct Money {
  amount:   i64
  currency: string
  implement Codable {
    fun encode(to: codec.Encoder) throws EncodeError = try to.writeString("${this.amount} ${this.currency}")
  }
}

/// Ordered by field, in declaration order.
struct Version {
  major: i64
  minor: i64
  implement Comparable
  implement Codable
}

fun main() throws EncodeError | DecodeError {
  val ann = User(id: 7, name: "ann", email: null, role: Role.Admin, tags: ["staff"], address: Address(street: "Main 1", zip: "00-001"), passwordHash: "hunter2")
  val text = try json.encode(ann)
  println(text)
  println(try json.pretty(ann, json.Options(omitNulls: true)))

  val back = try json.decode<User>(text)
  println("round trip: ${back.id == ann.id && back.address == ann.address}; the secret stayed off the wire: ${back.passwordHash.isEmpty()}")

  // a document with several things wrong: every one is reported
  val bad = "{\"user_id\": \"seven\", \"role\": \"Boss\", \"tags\": [1, \"ok\"], \"address\": {\"street\": 5}}"
  when (json.decode<User>(bad)) {
    is Ok(u)  => println("unexpected: $u")
    is Err(e) => {
      println("problems:")
      loop (p in e.problems) println("  ${p.path}: ${p.message}")
    }
  }

  // absent, null and default
  val sparse = try json.decode<User>("{\"user_id\": 1, \"name\": \"bo\"}")
  println("sparse: $sparse")

  // a sealed family, a generic page, snake_case keys
  val shapes: List<Shape> = [Circle(r: 1.5), Rect(w: 2.0, h: 3.0)]
  val page = Page(items: shapes, total: 2)
  val pageText = try json.encode(page)
  println(pageText)
  println("page back: ${try json.decode<Page<Shape>>(pageText)}")
  when (json.decode<Shape>("{\"kind\": \"blob\"}")) {
    is Ok(s)  => println("unexpected: $s")
    is Err(e) => println("shape: ${e.message()}")
  }
  println(try json.encode(Page(items: [Version(major: 1, minor: 2)], total: 1), json.Options(keys: codec.KeyStyle.SnakeCase)))

  // enums by name or by number: the format's choice
  println(try json.encode([Role.Admin, Role.Member]))
  println(try json.encode([Role.Admin, Role.Member], json.Options(enums: codec.EnumStyle.Number)))
  println("${try json.decode<List<Role>>("[1, 0]", json.Options(enums: codec.EnumStyle.Number))}")

  // hand-written encode, derived decode; derived ordering
  println(try json.encode(Money(amount: 5, currency: "EUR")))
  println("${try json.decode<Money>("{\"amount\": 3, \"currency\": \"PLN\"}")}")
  println("${Version(major: 1, minor: 9) < Version(major: 2, minor: 0)} ${[Version(major: 2, minor: 1), Version(major: 1, minor: 5)].sorted()}")

  // an untyped tree
  val tree = try json.parse("{\"a\": [1, 2.5, \"caf\\u00e9\", true, null]}")
  println("${tree.get("a")?.at(2)?.asString()} ${tree.get("a")?.at(1)?.asF64()}")
  println(try json.encode(tree))

  // malformed text is one problem
  when (json.parse("{\"a\": [1, 2,]}")) {
    is Ok(v)  => println("unexpected: $v")
    is Err(e) => println(e.message())
  }

  // nesting is bounded in every direction (§2 of the checklist): one limit
  // on the way in, the same one on the way out, and one sentence for both.
  // A tree deep enough to overflow the stack is refused before it is walked.
  when (json.parse("[".repeat(200) + "]".repeat(200))) {
    is Ok(v)  => println("unexpected: $v")
    is Err(e) => println("parse:   ${e.message()}")
  }
  var nested: codec.Value = codec.VInt(value: 1)
  loop (_ in 0..<200) nested = codec.VList(items: [nested])
  when (json.encode(nested, json.Options(maxDepth: 8))) {
    is Ok(t)  => println("unexpected: $t")
    is Err(e) => println("encode:  ${e.message()}")
  }
  when (json.toValue(nested, json.Options(maxDepth: 8))) {
    is Ok(_)  => println("unexpected")
    is Err(e) => println("toValue: ${e.message()}")
  }
  when (json.fromValue<codec.Value>(nested, json.Options(maxDepth: 8))) {
    is Ok(_)  => println("unexpected")
    is Err(e) => println("fromValue: ${e.message()}")
  }
  println("within the limit: ${try json.encode(codec.VList(items: [codec.VList(items: [codec.VInt(value: 1)])]), json.Options(maxDepth: 8))}")
  try durations()
}

// A Duration travels in the format's codec.DurationStyle: "90.5s" unless the
// options say otherwise; every style reads back to the same nanosecond.
struct Job {
  name:    string
  timeout: Duration
  retry:   Duration? = null
  implement Codable
}

fun durations() throws EncodeError | DecodeError {
  val ds = [Duration.seconds(90) + Duration.millis(500), Duration.zero, Duration.nanos(-1), Duration.hours(49) + Duration.nanos(1), Duration.millis(250)]
  loop (style in codec.DurationStyle.values()) {
    val o = json.Options(durations: style)
    var line = "$style:"
    loop (d in ds) {
      val text = try json.encode(Job(name: "x", timeout: d), o)
      val back = try json.decode<Job>(text, o)
      line += " ${text.substring(text.indexOf("\"timeout\":") + 10, text.len() - 1) ?: "?"}${if (back.timeout == d) "" else " (read back as ${back.timeout})"}"
    }
    println(line)
  }
  // refusals, each named with its path
  loop ((style, text) in [(codec.DurationStyle.Seconds, "{\"name\": \"x\", \"timeout\": \"1m30s\"}"), (codec.DurationStyle.Iso8601, "{\"name\": \"x\", \"timeout\": \"P1M\"}"), (codec.DurationStyle.Iso8601, "{\"name\": \"x\", \"timeout\": \"PT1.5M30S\"}"), (codec.DurationStyle.Iso8601, "{\"name\": \"x\", \"timeout\": \"PT\"}"), (codec.DurationStyle.Seconds, "{\"name\": \"x\", \"timeout\": 90}"), (codec.DurationStyle.Millis, "{\"name\": \"x\", \"timeout\": 1e300}")]) {
    when (val r = json.decode<Job>(text, json.Options(durations: style))) {
      is Ok  => println("unexpectedly read ${r.timeout}")
      is Err => println("$style refuses: ${r.message()}")
    }
  }
  println(try json.encode(Job(name: "y", timeout: Duration.seconds(5), retry: Duration.millis(1500))))
  println("${(try json.decode<Job>("{\"name\": \"z\", \"timeout\": \"P1DT2H\"}", json.Options(durations: codec.DurationStyle.Iso8601))).timeout}")
}
