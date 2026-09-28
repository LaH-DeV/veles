// D16: a global is initialized before any task runs, so its initializer
// cannot wait. (Found by the suspension analysis, which runs once the
// program is otherwise free of errors — hence a file of its own.)
use io

fun fetch(): i64 {
  await sleep(Duration.millis(1))
  1
}

val fetched: i64 = fetch() // error: a global initializer cannot suspend; compute the value in 'main' and pass it down

fun main() {
  io.println("$fetched")
}
