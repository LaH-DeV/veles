// Template literals (D129): a name written straight before a string —
// `html"…"`, `sql"…"` — hands the string to a function marked `@template`
// as pieces and values, instead of pasting the values into the text. The
// function decides what a value means, so the result can be a type a plain
// string can never be: markup whose values were escaped, a query whose values
// travel apart from its text.
use io { println }

// ---- html: every value is escaped, unless it is markup already ----

trait Markup {
  fun markup(): string
}

implement Markup for string {
  fun markup(): string = this.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;").replace("\"", "&quot;")
}

implement Markup for i64 {
  fun markup(): string = "$this"
}

/// Text that is safe to put in a page. The only way to make one is `html"…"`
/// (or splicing other `Html`), so a `string` from a user can never become one
/// unescaped.
struct Html {
  private text: string

  implement Display {
    fun toString(): string = this.text
  }

  implement Markup {
    // already markup: spliced as it is, not escaped twice
    fun markup(): string = this.text
  }
}

@template
fun html(parts: List<string>, values: List<Markup>): Html {
  val out = StringBuilder()
  loop ((i, part) in parts.enumerate()) {
    out.append(part)
    val v = values.at(i)
    if (v != null) out.append(v.markup())
  }
  Html(text: out.toString())
}

// ---- sql: the text keeps $1, $2; the values are a separate list ----

trait Param {
  fun describe(): string
}

implement Param for string {
  fun describe(): string = "'$this'"
}

implement Param for i64 {
  fun describe(): string = "$this"
}

struct Query {
  private text:   string
  private params: List<string>

  fun text(): string = this.text
  fun params(): List<string> = this.params
}

@template
fun sql(parts: List<string>, values: List<Param>): Query {
  val text = StringBuilder()
  val params: MutableList<string> = []
  loop ((i, part) in parts.enumerate()) {
    text.append(part)
    val v = values.at(i)
    if (v != null) {
      params.push(v.describe())
      text.append("$${i + 1}")
    }
  }
  Query(text: text.toString(), params: params.toList())
}

fun main() {
  val name = "<script>alert(\"x\")</script> & co"
  val visits: i64 = 42
  val page = html"<h1>Hello, ${name}</h1><p>${visits} visits</p>"
  println("$page")

  // markup inside markup is not escaped again; text still is
  val card = html"<div>${page}<small>${"a < b"}</small></div>"
  println("$card")

  // the injection attempt travels as a value, never as part of the query
  val user = "x'; drop table users; --"
  val minAge: i64 = 18
  val query = sql"select id from users where name = ${user} and age >= ${minAge}"
  println(query.text())
  println("${query.params()}")
}
