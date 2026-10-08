// D28: an `init` block may call a method only once every field the method
// reaches is assigned; handing the whole value on reaches all of them.
use io

struct Named {
  name: string
  size: i64
  init {
    this.size = this.measure() // error: 'init' calls 'measure' before assigning 'size', and the method uses the whole value; assign every field without a default first (D28)
  }
  fun measure(): i64 => weigh(this)
}

fun weigh(n: Named): i64 => 1

fun main() {
  val n = Named(name: "a")
  io.println("${n.size}")
}
