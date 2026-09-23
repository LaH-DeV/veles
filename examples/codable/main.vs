// codable: the compiler writes the wire code (D58). A struct asks for it
// with one line — `implement Codable` — and `std/json` reads and writes it; the
// same derived code would drive a database row or the environment, since
// the traits are format-agnostic. Every problem in a document comes back
// at once, with its path.
use io, json

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
    fun encode(to: Encoder) throws EncodeError = try to.writeString("${self.amount} ${self.currency}")
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
  io.println(text)
  io.println(try json.pretty(ann, json.Options(omitNulls: true)))

  val back = try json.decode<User>(text)
  io.println("round trip: ${back.id == ann.id && back.address == ann.address}; the secret stayed off the wire: ${back.passwordHash.isEmpty()}")

  // a document with several things wrong: every one is reported
  val bad = "{\"user_id\": \"seven\", \"role\": \"Boss\", \"tags\": [1, \"ok\"], \"address\": {\"street\": 5}}"
  when (json.decode<User>(bad)) {
    is Ok(u)  => io.println("unexpected: $u")
    is Err(e) => {
      io.println("problems:")
      loop (p in e.problems) io.println("  ${p.path}: ${p.message}")
    }
  }

  // absent, null and default
  val sparse = try json.decode<User>("{\"user_id\": 1, \"name\": \"bo\"}")
  io.println("sparse: $sparse")

  // a sealed family, a generic page, snake_case keys
  val shapes: List<Shape> = [Circle(r: 1.5), Rect(w: 2.0, h: 3.0)]
  val page = Page(items: shapes, total: 2)
  val pageText = try json.encode(page)
  io.println(pageText)
  io.println("page back: ${try json.decode<Page<Shape>>(pageText)}")
  when (json.decode<Shape>("{\"kind\": \"blob\"}")) {
    is Ok(s)  => io.println("unexpected: $s")
    is Err(e) => io.println("shape: ${e.message()}")
  }
  io.println(try json.encode(Page(items: [Version(major: 1, minor: 2)], total: 1), json.Options(keys: KeyStyle.SnakeCase)))

  // enums by name or by number: the format's choice
  io.println(try json.encode([Role.Admin, Role.Member]))
  io.println(try json.encode([Role.Admin, Role.Member], json.Options(enums: EnumStyle.Number)))
  io.println("${try json.decode<List<Role>>("[1, 0]", json.Options(enums: EnumStyle.Number))}")

  // hand-written encode, derived decode; derived ordering
  io.println(try json.encode(Money(amount: 5, currency: "EUR")))
  io.println("${try json.decode<Money>("{\"amount\": 3, \"currency\": \"PLN\"}")}")
  io.println("${Version(major: 1, minor: 9) < Version(major: 2, minor: 0)} ${[Version(major: 2, minor: 1), Version(major: 1, minor: 5)].sorted()}")

  // an untyped tree
  val tree = try json.parse("{\"a\": [1, 2.5, \"caf\\u00e9\", true, null]}")
  io.println("${tree.get("a")?.at(2)?.asString()} ${tree.get("a")?.at(1)?.asF64()}")
  io.println(try json.encode(tree))

  // malformed text is one problem
  when (json.parse("{\"a\": [1, 2,]}")) {
    is Ok(v)  => io.println("unexpected: $v")
    is Err(e) => io.println(e.message())
  }

  // nesting is bounded in every direction (§2 of the checklist): one limit
  // on the way in, the same one on the way out, and one sentence for both.
  // A tree deep enough to overflow the stack is refused before it is walked.
  when (json.parse("[".repeat(200) + "]".repeat(200))) {
    is Ok(v)  => io.println("unexpected: $v")
    is Err(e) => io.println("parse:   ${e.message()}")
  }
  var nested: Value = VInt(value: 1)
  loop (_ in 0..<200) nested = VList(items: [nested])
  when (json.encode(nested, json.Options(maxDepth: 8))) {
    is Ok(t)  => io.println("unexpected: $t")
    is Err(e) => io.println("encode:  ${e.message()}")
  }
  when (json.toValue(nested, json.Options(maxDepth: 8))) {
    is Ok(_)  => io.println("unexpected")
    is Err(e) => io.println("toValue: ${e.message()}")
  }
  when (json.fromValue<Value>(nested, json.Options(maxDepth: 8))) {
    is Ok(_)  => io.println("unexpected")
    is Err(e) => io.println("fromValue: ${e.message()}")
  }
  io.println("within the limit: ${try json.encode(VList(items: [VList(items: [VInt(value: 1)])]), json.Options(maxDepth: 8))}")
}
