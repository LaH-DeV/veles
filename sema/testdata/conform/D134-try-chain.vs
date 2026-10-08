// D134: one `try` covers every failing call in the receiver chain after it.
use io

error NotFound { }
error Bad { }

struct Page {
  body: string

  fun parse(): i64 throws Bad => this.body.toInt() ?: throw Bad()
  fun words(): List<string> => this.body.split(" ")
  fun next(): Page throws Bad => if (this.body == "end") throw Bad() else Page(body: this.body + "1")
}

struct Client {
  fun fetch(url: string): Page throws NotFound => if (url == "") throw NotFound() else Page(body: url)
}

fun client(): Client throws NotFound => Client()

// two failing links: the error type is their union
fun twoLinks(url: string): i64 throws NotFound | Bad => try client().fetch(url).parse()

// three failing links, and a plain method after them
fun threeLinks(url: string): i64 throws NotFound | Bad => try client().fetch(url).next().parse() + try client().fetch(url).next().words().len()

// a field link
fun field(url: string): string throws NotFound => try client().fetch(url).body

// a Result method in the chain applies to the Result
fun resultMethod(url: string): Page throws NotFound | Bad => try client().fetch(url).mapError(e => Bad())

// an argument is not a link: its Result is passed as a value
fun takes(r: Result<i64, Bad>): i64 => r.getOrDefault(0)
fun argument(): i64 throws NotFound => try client().fetch(takes(Page(body: "1").parse()).toString()).words().len()

// with `catch`, the handler sees the union
fun caught(url: string): i64 => try client().fetch(url).parse() catch (e) { -1 }

// the old spelling compiles, with the inner `try` called redundant
fun redundant(url: string): i64 throws NotFound | Bad => try (try client().fetch(url)).parse() // warning: the inner 'try' is redundant

// the union must be declared: a throws list without Bad is too narrow
fun narrow(url: string): i64 throws NotFound => try client().fetch(url).parse() // error: Bad

fun main() {
  io.println("${caught("12")} ${caught("")} ${caught("x")}")
}
