// D205, D206: an enum reads and writes its members as text in any `Case`,
// implements `Enumerated` by itself, may override its `parse` and
// `toString(case:)`, and gives each value a predicate per member.
use io

enum Os { Windows, Linux, MacOS }

enum Status {
  Ok
  OK // error: members 'Ok' and 'OK' of enum 'Status' are the same name in another spelling: 'parse(text, case: Case.Any)' could not tell them apart; rename one (D205)
}

enum Arch { X64, Arm64 }

implement Enumerated for Arch {
  static fun parse(text: string, case: Case): Arch? => if (text == "x86_64") Arch.X64 else null
}

enum Level { Low, High }

implement Enumerated for Level {
  static fun values(): List<Level> => [] // error: 'values' of an enum is always its members in order; an implement of 'Enumerated' may override 'parse' and 'toString(case:)' (D205)
}

trait Label { fun label(): string }

implement Label for Os { // error: cannot implement 'Label' for enum 'Os': an enum is a set of values with no methods of its own
  fun label(): string => "os"
}

fun choices<E: Enumerated>(): List<string> => E.values().map(v => v.toString(case: Case.Kebab))

fun main() {
  val os = Os.parse("mac-os", case: Case.Any) ?: Os.Linux
  val text = os.toString(case: Case.Snake)
  io.println("${os.isMacOS()} $text ${Arch.parse("x86_64")} ${choices<Os>()}")
}
