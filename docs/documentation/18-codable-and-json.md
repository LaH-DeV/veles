# 18. Codable and JSON

A type that travels to the outside world — JSON in and out, a database
row, the environment — implements `Encodable` and `Decodable`, together
`Codable`. You almost never write them: one line asks the compiler to.
`json` reads and writes anything `Codable`, and the same derived code
would drive any other format, because the traits do not know about JSON.
`examples/codable` is the whole of this chapter as one program.

## One line

```veles
use io, json

enum Role { Admin, Member }

struct User {
  id:    i64
  name:  string
  email: string?
  role:  Role = Role.Member
  implement Codable
}

fun main() throws EncodeError | DecodeError {
  val ann = User(id: 1, name: "ann", email: null, role: Role.Admin)
  val text = try json.encode(ann)
  io.println(text)
  io.println("${try json.decode<User>(text) == ann}")
  io.println(try json.pretty(ann))
}
```

Output:
```text
{"id":1,"name":"ann","email":null,"role":"Admin"}
true
{
  "id": 1,
  "name": "ann",
  "email": null,
  "role": "Admin"
}
```

- `implement Codable` in a struct body — no braces, nothing inside — is the
  request: the compiler writes `encode` and `decode` from the fields. The
  same line at top level covers a type from elsewhere:
  `implement Codable for geo.Point`.
- `Codable` is `Encodable` and `Decodable` together (a trait can require
  others: `trait Codable : Encodable + Decodable`). Implement one of them
  alone when only one direction is needed.
- Enums travel as their names and need no line at all; the numbers,
  strings, `bool`, `List`, `Map<string, _>` and `T?` are all covered by
  the prelude.

Why a line at all, when `==` needs none? Equality is a property of the
value. A wire format is a contract with other programs: renaming a field
changes what they receive, and a type with a `passwordHash` should not
become JSON because someone forgot to say it must not. So it is declared,
where a reader sees it.

## Absent, null and default

```veles
use io, json

struct Settings {
  host:    string            // required: absent or null is a problem
  port:    i64 = 8080        // absent → 8080
  proxy:   string?           // absent or null → null
  @required region: string?  // the key must be there, its value may be null
  implement Codable
}

fun main() throws DecodeError {
  io.println("${try json.decode<Settings>("{\"host\": \"a\", \"region\": null}")}")
  when (json.decode<Settings>("{\"port\": \"eighty\"}")) {
    is Ok(s)  => io.println("$s")
    is Err(e) => io.println(e.message())
  }
}
```

Output:
```text
Settings(host: a, port: 8080, proxy: null, region: null)
port: expected an integer, found a string
host: missing
region: missing
```

A field's default is the value when its key is absent — there is no
second way to say it. A nullable field reads `null` and *absent* the same
way. And decoding reports **every** problem it finds, each with its path,
in one `DecodeError`: a client fixes its document in one round trip. A
wrong type is recorded and read past; a value that cannot be built at all
(a required field missing) ends the decode after the rest of its object
has been checked. Text that is not JSON is one problem, since nothing
follows it. `Problem.pointer()` gives the path as a JSON pointer
(`/address/2/zip`) for clients that want that.

## Keys, skips and styles

```veles
use codec, io, json

struct Account {
  @key("account_id")
  id: i64  // this key, in every format
  @key(json: "displayName", db: "display_name")
  name: string
  createdAt: i64
  @skip
  secret: string = ""  // never on the wire
  implement Codable
}

fun main() throws EncodeError {
  val a = Account(id: 7, name: "ann", createdAt: 1, secret: "hunter2")
  io.println(try json.encode(a))
  io.println(try json.encode(a, json.Options(keys: codec.KeyStyle.SnakeCase)))
}
```

Output:
```text
{"account_id":7,"displayName":"ann","createdAt":1}
{"account_id":7,"displayName":"ann","created_at":1}
```

- `@key("name")` names the key for every format; `@key(json: "a", db:
  "b")` by format, the others keeping the field's name. A format is a
  string (`"json"`, `"db"`), so a format written in a library takes part
  without the compiler knowing it.
- `@skip` leaves the field out both ways (it needs a default, or nothing
  could construct the value); `@skip(json)` for one format only.
- How *field names* are spelled as keys — `snake_case`, `camelCase` — is
  a policy of the encoder, `json.Options(keys:)`, never an attribute: a
  whole API changes style in one place, and a `@key` the author wrote is
  never restyled.
- Two fields on one key, or a field that has no wire form (a function, a
  `Mutex`), stop the derivation with a message naming the field.

## Sealed traits and generics

```veles
use io, json

@tag("kind") sealed trait Shape
@key("circle") struct Circle : Shape { r: f64 }
struct Rect : Shape { w: f64; h: f64 }
implement Codable for Shape

struct Page<T> {
  items: List<T>
  total: i64
  implement Codable
}

fun main() throws EncodeError | DecodeError {
  val shapes: List<Shape> = [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)]
  val text = try json.encode(Page(items: shapes, total: 2))
  io.println(text)
  io.println("${try json.decode<Page<Shape>>(text)}")
  when (json.decode<Shape>("{\"kind\": \"blob\"}")) {
    is Ok(s)  => io.println("$s")
    is Err(e) => io.println(e.message())
  }
}
```

Output:
```text
{"items":[{"kind":"circle","r":1.0},{"kind":"Rect","w":2.0,"h":3.0}],"total":2}
Page(items: [Circle(r: 1.0), Rect(w: 2.0, h: 3.0)], total: 2)
unknown variant "blob" of Shape (one of "circle", "Rect")
```

- A sealed family is *internally tagged*: the variant's name under
  `"type"`, renamed with `@tag("kind")` on the trait and `@key` on a
  variant; `@tag("type", content: "value")` puts the fields under a key of
  their own instead. `implement Codable for Shape` covers every variant; a
  variant may still write its own implement.
- `implement Codable` inside `Page<T>` is read as `implement<T: Codable> Codable for
  Page<T>`: the bound is inferred from the fields, and `Page<Shape>` is
  Codable exactly when `Shape` is.

## Enums by name or by number

```veles
use codec, io, json

enum Status { Active, Suspended }

fun main() throws EncodeError | DecodeError {
  io.println(try json.encode([Status.Active, Status.Suspended]))
  io.println(try json.encode([Status.Active, Status.Suspended], json.Options(enums: codec.EnumStyle.Number)))
  io.println("${try json.decode<List<Status>>("[1]", json.Options(enums: codec.EnumStyle.Number))}")
}
```

Output:
```text
["Active","Suspended"]
[0,1]
[Suspended]
```

Whether an enum is a name or a number is a property of where it is going
— JSON wants names, a `smallint` column wants numbers — so it is the
format's option, not a mark on the enum. `@key("active")` on a member
renames it.

## Durations

A `Duration` field is `"90.5s"` on the wire — seconds with their exact
fraction, the form protobuf's JSON mapping and Go's `time.ParseDuration`
use. When the other side expects something else, say so once, in the
options:

```veles
use codec, io, json

struct Job {
  name:    string
  timeout: Duration
  implement Codable
}

fun main() throws EncodeError | DecodeError {
  val job = Job(name: "backup", timeout: Duration.seconds(90) + Duration.millis(500))
  loop (style in codec.DurationStyle.values()) {
    io.println("$style: ${try json.encode(job, json.Options(durations: style))}")
  }
  val java = json.Options(durations: codec.DurationStyle.Iso8601)
  io.println("${try json.decode<Job>("{\"name\": \"x\", \"timeout\": \"PT2H\"}", java).timeout}")
}
```

Output:
```text
Seconds: {"name":"backup","timeout":"90.5s"}
Iso8601: {"name":"backup","timeout":"PT1M30.5S"}
Text: {"name":"backup","timeout":"1m30.5s"}
Nanos: {"name":"backup","timeout":90500000000}
Millis: {"name":"backup","timeout":90500}
2h
```

Every text style reads back to the same nanosecond; `Millis` does when the
duration is whole milliseconds (a finer one is written with a fraction and
rounded to the nanosecond on the way back). Decoding is strict per style,
as for enums. ISO 8601 refuses years, months and weeks — `"P1M"` is not a
length of time, since months differ — with a message that says so.

## Writing part by hand

```veles
use codec, io, json

struct Money {
  amount:   i64
  currency: string
  implement Codable {
    fun encode(to: codec.Encoder) throws EncodeError = try to.writeString("${this.amount} ${this.currency}")
  }
}

struct Version {
  major: i64
  minor: i64
  implement Comparable
}

fun main() throws EncodeError | DecodeError {
  io.println(try json.encode(Money(amount: 5, currency: "EUR")))
  io.println("${try json.decode<Money>("{\"amount\": 3, \"currency\": \"PLN\"}")}")
  io.println("${Version(major: 1, minor: 9) < Version(major: 2, minor: 0)}")
}
```

Output:
```text
"5 EUR"
Money(amount: 3, currency: PLN)
true
```

A method written in the body is kept and the rest is derived, so one
direction can be by hand. (Chapter 22's `config` also asks a type for its
`schema`, the list of its fields; that is derived only along with `decode`,
because the fields say nothing about what a hand-written `decode` reads — such
a type is read from one variable, as text.) `Comparable` is derived the same way: field by
field, in declaration order.

## Seeing what was derived

An empty `implement` writes code you never read, which is fine until a
key on the wire is not the one you expected. Hover the trait name in the
editor and the language server answers with what the compiler made of
that line — the implements it produced, with the bounds it inferred for a
generic target, the signatures, and the shape the value takes on the
wire:

```text
implement Decodable for Note   // derived (D58)
  static fun decode(from: Decoder): Note throws DecodeError

implement Encodable for Note   // derived (D58)
  fun encode(to: Encoder) throws EncodeError

keys      id · author → "userName" (json) · body · tags
optional  tags (field default)
skipped   cached (@skip)
```

A field with `@key(json: ...)` is shown as the mapping it is: the key is
chosen at run time from the encoder's `format()`, so the same type can be
`userName` in JSON and `author` everywhere else.

The bodies themselves are a command away:

```bash
veles explain path/to/package --derive Note
```

which prints the synthesized implements as ordinary Veles — the code you
would have had to write. A couple of forms in it have no spelling of
their own and are named between angle brackets, such as `<default of
Note.tags>` for the value a missing key falls back to.

## An untyped document

```veles
use io, json

fun main() throws EncodeError | DecodeError {
  val v = try json.parse("{\"a\": [1, 2.5, \"x\"], \"b\": {\"c\": true}}")
  io.println("${v.get("a")?.at(1)?.asF64()} ${v.get("b")?.get("c")?.asBool()} ${v.get("zzz") == null}")
  io.println(try json.encode(v))
}
```

Output:
```text
2.5 true true
{"a":[1,2.5,"x"],"b":{"c":true}}
```

`Value` is the tree — `VNull`, `VBool`, `VInt`, `VFloat`, `VString`,
`VList`, `VObject` — with `get`, `at`, `asString`, `asI64`, `asF64`,
`asBool`. It is `Codable` itself, so `json.parse` is `json.decode<Value>`
and `json.toValue(x)` / `json.fromValue<T>(v)` move between a typed value
and the tree.

The tree, the styles (`KeyStyle`, `EnumStyle`, `DurationStyle`) and the
`Encoder`/`Decoder` traits live in module `codec`: `use codec`, then
`codec.Value`, `codec.KeyStyle.SnakeCase`. Deriving needs none of it —
`implement Codable`, `EncodeError` and `DecodeError` are global — so only
a program that walks a document whose shape it does not know, picks a
style, or writes a format of its own imports it.

## Under the hood: `Encoder` and `Decoder`

A format is one implementation of each trait. They are a flat stream of
events — `beginObject`, `key`, `endObject`, `beginList`, `endList`, and
`writeI64` … `writeNull` (`readI64` … `readNull`, `peek`, `skip` on the
way in) — and a value encodes *itself* (`this.id.encode(to)`), so the
traits carry no generics and nothing is boxed. A decoder also keeps the
problems (`problem`, `problemAt`, `problems`) and the current `path`.
`std/json` is ~500 lines of Veles; a row decoder for a database, or one
over environment variables, is the same shape, and every `implement Codable`
in the program works with it unchanged.

`json.Options` holds the limits a hostile document meets: `maxDepth`
(64) and `maxProblems` (100); a NaN or infinity cannot be encoded and is
an `EncodeError`.

Next: [Secrets, hashes and tokens](19-secrets-and-crypto.md).
