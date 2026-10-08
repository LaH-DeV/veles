// Named imports (D85): `use m { f, T as U }` brings names in bare, on top of
// the module's own name.
use io { eprintln as warn, println }
use os { Output }

fun describe(out: Output): string => "exit ${out.code}, ${out.stdout.trim().len()} bytes"

fun main() {
  println("bare println")
  warn("renamed eprintln goes to standard error")
  io.println("the module name still works")
  val d = Duration.millis(1500)
  println("${d.toMillis()} ms")
  println(describe(Output(stdout: "hi\n", stderr: "", code: 0)))
}
