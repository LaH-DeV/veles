// D85: `use m { f, T as U }` brings names in bare; the module stays
// reachable as `m`, and a name is checked like the qualified one.
use io { println, eprintln as warn }, os { Output }
use fs { readFile } // warning: 'readFile' is imported but never used

fun show(o: Output): string = o.stdout

fun id<T>(x: T): T {
  warn("generic")
  return x
}

fun main() {
  println("hello")
  io.println("still qualified")
  println(show(Output(stdout: "x", stderr: "", code: 0)))
  println("${id(3)}")
}
