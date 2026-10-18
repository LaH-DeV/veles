// D174: `annotation Name(params) on targets for Readers` declares typed data a
// declaration can carry; `@Name(args)` is a construction with constant
// arguments. The compiler's own names are lowercase and reserved; an
// annotation is named like a type. Nothing here changes what code does.
use io

trait Validate { }

annotation Min(value: i64) on field for Validate
annotation Label(text: string, short: string = "") on field, struct
annotation Hidden on field
annotation Route(path: string, method: string = "GET") on fun
annotation lower(n: i64) on field // error: an annotation is named like a type, 'Lower'
annotation Odd(n: i64) on nowhere // error: 'nowhere' is not a place an annotation can stand
annotation Twice(n: i64) on field, field // error: 'field' is listed twice after 'on'
annotation Bad(n: i64) on field for i64 // error: 'i64' is not a trait

struct Signup {
  @Min(18)
  age: i64
  @Label(text: "n", short: "x")
  @Hidden
  name: string
  implement Validate
}

struct Plain {
  @Min(1) // warning: nothing reads '@Min' here
  n: i64
}

@Label(text: "tagged")
struct Tagged {
  @Min("bad") // error: type mismatch
  n: i64
  @Hidden
  @Hidden // error: duplicate annotation '@Hidden'
  m: i64
  implement Validate
}

@Route(path: "/users", method: "POST")
fun users() { }

@Min(3) // error: '@Min' stands on field, and cannot stand on a function
fun f() { }

@Nope // error: unknown annotation '@Nope'
fun h() { }

@Hidden // error: '@Hidden' stands on field, and cannot stand on a struct
struct S {
  @Hidden(1) // error: construct 'Hidden' by field name
  n: i64
}

struct Sample { }

@Sample // error: '@Sample' is not an annotation
fun k() { }

fun main() {
  io.println("ok")
}
