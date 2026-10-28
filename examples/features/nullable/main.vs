use io { println }

struct User {
  name: string
  age:  i64
}

fun find(name: string): User? {
  if (name == "ann") return User(name: "Ann", age: 41)
  null
}

fun greet(u: User?): string {
  if (u == null) return "nobody"
  "hello ${u.name}, age ${u.age}"
}

fun main() {
  val a = find("ann")
  val b = find("bob")
  println(greet(a))
  println(greet(b))
  val age = a?.age ?: -1
  val bage = b?.age ?: -1
  println("ages: $age $bage")
  when (a) {
    null          => println("none")
    is User(name) => println("some $name")
  }
  val nested: i64?? = Some(null)
  when (nested) {
    null          => println("outer null")
    Some(null)    => println("inner null")
    Some(Some(x)) => println("value $x")
  }
  var maybe: string? = null
  maybe = "set"
  if (maybe != null && maybe.len() > 2) {
    println("len ${maybe.len()}")
  }
}
