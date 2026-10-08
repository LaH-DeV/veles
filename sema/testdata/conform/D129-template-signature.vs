// D129: a @template function takes the literal's pieces and values.
use io

struct Query {
  text: string
}

@template
fun sql(parts: List<string>, values: List<i64>): Query => Query(text: parts.join("?"))

@template
fun oneParam(text: string): Query => Query(text: text) // error: is a @template with 1 parameters

@template
fun badFirst(parts: string, values: List<i64>): Query => Query(text: parts) // error: the first parameter of a @template is the pieces

@template
fun badSecond(parts: List<string>, values: i64): Query => Query(text: "") // error: the second parameter of a @template is the values

struct Holder {
  @template
  fun inside(parts: List<string>, values: List<i64>): Query => Query(text: "") // error: applies to a module-level function

  n: i64
}

fun main() {
  io.println(sql"a ${1} b".text)
}
