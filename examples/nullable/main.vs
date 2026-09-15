use io

struct User { name: string, age: i32 }

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
  io.println(greet(a))
  io.println(greet(b))
  val age = a?.age ?: -1
  val bage = b?.age ?: -1
  io.println("ages: $age $bage")
  when (a) {
    null => io.println("none")
    is User(name) => io.println("some $name")
  }
  val nested: i32?? = Some(null)
  when (nested) {
    null => io.println("outer null")
    Some(null) => io.println("inner null")
    Some(Some(x)) => io.println("value $x")
  }
  var maybe: string? = null
  maybe = "set"
  if (maybe != null && maybe.len() > 2) {
    io.println("len ${maybe.len()}")
  }
}
