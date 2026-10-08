// D129: `tag"…"` is a call of a @template function; the tag must be one.
use io

trait Param {
  fun show(): string
}

implement Param for i64 {
  fun show(): string => "int"
}

struct Query {
  text: string
}

@template
fun sql(parts: List<string>, values: List<Param>): Query => Query(text: parts.join("?"))

fun plain(parts: List<string>, values: List<Param>): Query => Query(text: "")

fun main() {
  val flag = true
  val count: i64 = 3
  io.println(sql"x = ${count}".text)
  io.println(sql"x = ${flag}".text) // error: type 'bool' does not implement trait 'Param'
  io.println(plain"x".text) // error: 'plain' is not a template
  io.println(nothing"x".text) // error: unknown name 'nothing'
  io.println(count"x".text) // error: this is not a function
  val s: string = sql"y" // error: type mismatch
  io.println(s)
}
