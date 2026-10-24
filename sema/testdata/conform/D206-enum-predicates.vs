// D206: each member of an enum gives its values a predicate, `isM()` —
// true for that member alone, taking no arguments; no other `is…` exists.
use io

enum Os { Windows, Linux, MacOS }

fun separator(os: Os): string => if (os.isWindows()) "\\" else "/"

fun main() {
  val os = Os.parse("mac-os", case: Case.Any) ?: Os.Linux
  io.println("${os.isMacOS()} ${!os.isLinux()} ${separator(os)}")
  val wrong = os.isMacOS(true) // error: 'isMacOS' takes no arguments: it says whether the value is Os.MacOS (D206)
  val nope = os.isSolaris() // error: no method 'isSolaris' on enum 'Os'
  io.println("$wrong $nope")
}
