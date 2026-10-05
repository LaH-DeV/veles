// The query type and the values it carries (D129). A `Sql` is made by the
// template `sql"…"` — or spliced from other `Sql` — and never from a `string`
// a program computed, so the text of a query is always written in the
// source and the values travel apart from it, as parameters. Nothing here
// can put a value into the text.
use crypto, time

/// One value bound to a `$n` of a query, in the form it travels: the type the
/// server is told (`oid`, 0 to let the server infer it from the place the
/// value is used — what makes a `string` fit a `uuid`, `timestamptz` or enum
/// column), and the value as text or, for bytes, in binary. `null` is a text
/// that is absent. `fragment` is a spliced piece of query rather than a value.
public struct Arg {
  public oid:      i64 = 0
  public text:     string? = null
  public bytes:    List<u8>? = null
  public fragment: Sql? = null
  /// Whether the value is secret: sent as any other, never shown or logged.
  public secret: bool = false
}

/// What can be a `${…}` in a `sql"…"`: the numbers, `bool`, `string`,
/// `List<u8>`, `Timestamp`, `Duration`, `Uuid`, a `T?` of any of those,
/// a `Secret`, and `Sql` itself (spliced as query text). Implement it for
/// your own type — an enum, an id — by saying which of the above it is sent as:
///
/// ```veles
/// implement db.Param for UserId {
///   fun toArg(): db.Arg = this.value.toArg()
/// }
/// ```
public trait Param {
  fun toArg(): Arg
}

/// A query: text written in the program, with a value for each `$n`. Made by
/// `sql"…"`; there is no way to make one from a `string` that was computed —
/// except `Sql.dangerouslyRaw`, named for what it risks.
///
/// ```veles
/// val name = "O'Brien"
/// val q = sql"select id, name from users where name = ${name} and age > ${18}"
/// // the text the server sees: select id, name from users where name = $1 and age > $2
/// ```
public struct Sql {
  // the text between the values: one more piece than there are values
  parts: List<string>
  args:  List<Arg>

  /// A query of text only, for what is not data: a migration read from a
  /// file, DDL assembled by a tool. The name says what it risks — whatever
  /// is in `text` is the query. Never pass anything a user typed.
  public static fun dangerouslyRaw(text: string): Sql = Sql(parts: [text], args: [])

  /// The text the server receives, with `$1`, `$2`, … where the values go.
  public fun text(): string {
    val out = StringBuilder()
    loop ((i, part) in this.parts.enumerate()) {
      out.append(part)
      if (i < this.args.len()) out.append("$${i + 1}")
    }
    out.toString()
  }

  /// How many values the query carries.
  public fun valueCount(): i64 = this.args.len()

  implement Param {
    fun toArg(): Arg = Arg(fragment: this)
  }

  implement Display {
    /// The text with its placeholders; the values are not shown.
    fun toString(): string = this.text()
  }
}

/// The template behind `sql"…"`: the text between the values stays text,
/// every value becomes a parameter, and a `Sql` among the values is spliced
/// in with its own parameters.
@template
public fun sql(parts: List<string>, values: List<Param>): Sql {
  val outParts: MutableList<string> = []
  val outArgs: MutableList<Arg> = []
  // the piece being written: it continues across a spliced fragment
  var current = StringBuilder()
  loop ((i, part) in parts.enumerate()) {
    current.append(part)
    val v = values.at(i) ?: continue
    val arg = v.toArg()
    val inner = arg.fragment
    if (inner != null) {
      loop ((j, p) in inner.parts.enumerate()) {
        current.append(p)
        if (j < inner.args.len()) {
          outParts.push(current.toString())
          current = StringBuilder()
          outArgs.push(inner.args.at(j) ?: Arg())
        }
      }
    } else {
      outParts.push(current.toString())
      current = StringBuilder()
      outArgs.push(arg)
    }
  }
  outParts.push(current.toString())
  Sql(parts: outParts.toList(), args: outArgs.toList())
}

/// A table or column name written as a `Sql`, quoted: letters, digits and `_`,
/// and dots between parts (`schema.table`). Anything else — a space, a quote,
/// a semicolon — is `null`, so a name that came from a user can be checked
/// against what the program allows before it reaches a query:
///
/// ```veles
/// val order = db.ident(column) ?: throw http.badRequest("unknown column")
/// val q = sql"select * from users order by ${order} desc"
/// ```
public fun ident(name: string): Sql? {
  if (name.isEmpty()) return null
  val out = StringBuilder()
  loop ((i, part) in name.split(".").enumerate()) {
    if (part.isEmpty()) return null
    loop (b in part.bytes()) {
      val ok = (b >= 48 && b <= 57) || (b >= 65 && b <= 90) || (b >= 97 && b <= 122) || b == 95
      if (!ok) return null
    }
    if (i > 0) out.append(".")
    out.append("\"")
    out.append(part)
    out.append("\"")
  }
  Sql.dangerouslyRaw(out.toString())
}

// ---- how the standard types travel ----

// a number goes as text with the type the server should read it as
fun number(oid: i64, text: string): Arg = Arg(oid, text)

implement Param for i8 {
  fun toArg(): Arg = number(21, "$this")
}

implement Param for i16 {
  fun toArg(): Arg = number(21, "$this")
}

implement Param for i32 {
  fun toArg(): Arg = number(23, "$this")
}

implement Param for i64 {
  fun toArg(): Arg = number(20, "$this")
}

implement Param for u8 {
  fun toArg(): Arg = number(21, "$this")
}

implement Param for u16 {
  fun toArg(): Arg = number(23, "$this")
}

implement Param for u32 {
  fun toArg(): Arg = number(20, "$this")
}

implement Param for u64 {
  // past i64's range: numeric holds all of u64
  fun toArg(): Arg = number(1700, "$this")
}

implement Param for f32 {
  fun toArg(): Arg = number(700, "$this")
}

implement Param for f64 {
  fun toArg(): Arg = number(701, "$this")
}

implement Param for bool {
  fun toArg(): Arg = Arg(oid: 16, text: if (this) "t" else "f")
}

implement Param for string {
  // no type: the server reads the text as the type of the place it is used
  fun toArg(): Arg = Arg(text: this)
}

implement Param for List<u8> {
  fun toArg(): Arg = Arg(oid: 17, bytes: this)
}

implement Param for time.Timestamp {
  fun toArg(): Arg = Arg(oid: 1184, text: time.formatRfc3339(this))
}

implement Param for Duration {
  fun toArg(): Arg = Arg(oid: 1186, text: "${this.toMicros()} microseconds")
}

implement Param for crypto.Uuid {
  fun toArg(): Arg = Arg(oid: 2950, text: "$this")
}

implement Param for Secret<string> {
  fun toArg(): Arg = Arg(text: this.expose(), secret: true)
}

implement Param for Secret<List<u8>> {
  fun toArg(): Arg = Arg(oid: 17, bytes: this.expose(), secret: true)
}

implement<T: Param> Param for T? {
  // a NULL has no type of its own: the server infers it from where it is used
  fun toArg(): Arg = if (this == null) Arg() else this.toArg()
}
