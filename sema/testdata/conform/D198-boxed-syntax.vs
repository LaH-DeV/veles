// D198: `indirect` and `inline` say where a variant of a sealed trait is
// stored, so they stand only before a struct in a sealed trait's body.
trait Named {
  indirect struct Label { // error: only a sealed trait declares its variants: write 'sealed trait Named { struct Label … }' (D193)
    text: string
  }
}

fun main() { }
