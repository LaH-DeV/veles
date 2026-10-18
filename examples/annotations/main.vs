// Annotations are typed data a library reads while the program is compiled
// (D174). `validate` declares `Min`, `MaxLen` and `Skip` and the trait that
// reads them; this file only uses them. The rules table is a constant: the
// compiler built it, and it is part of the binary.
use io { println }
use validate { MaxLen, Min, Skip, Validate }

struct Signup {
  /// What the account is called.
  @MaxLen(8)
  name: string
  /// Must be an adult.
  @Min(18)
  age: i64
  @Skip
  note: string = ""
  implement Validate
}

struct Order {
  @Min(1)
  quantity: i64
  @MaxLen(3)
  currency: string
  implement Validate
}

const SIGNUP_RULES: List<string> = Signup.rules()
const ORDER_RULES: List<string> = Order.rules()

fun main() {
  println("signup rules:")
  loop (rule in SIGNUP_RULES) println("  $rule")

  println("${Signup(name: "ann", age: 30).problems().map(p => p.message)}")
  println("${Signup(name: "alexander", age: 12, note: "not checked").problems().map(p => p.message)}")
  println("${Order(quantity: 0, currency: "euro").problems().map(p => p.message)}")
  println("${ORDER_RULES}")
}
