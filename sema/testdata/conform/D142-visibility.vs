// D142: four levels — `private` (the type's own declarations), `internal`
// (the module), nothing (the package), `public` (other packages). A program
// has no other package, so `public` there is noise; and a `public`
// declaration names only types other packages can see.
struct Token {
  text: string
}

public struct Holder { // warning: 'public' on 'Holder' does nothing in a program
  public t: Token // warning: 'public' on 't' does nothing in a program // error: field 't' is public but its type 'Token' is not visible outside the package
}

public fun parse(s: string): Token => Token(text: s) // warning: 'public' on 'parse' does nothing in a program // error: 'parse' is public but its result type 'Token' is not visible outside the package

public fun show(t: Token): string => t.text // warning: 'public' on 'show' does nothing in a program // error: 'show' is public but its parameter 't''s type 'Token' is not visible outside the package

internal fun helper(): i64 => 1

fun shared(): i64 => helper()

fun main() {
  shared()
}
