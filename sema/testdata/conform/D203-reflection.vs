// D203: what a library needs to read a type while compiling — a field's
// declared default, whether it is optional or a list and what is inside,
// constant branches on those, an enum's members and a sealed family's
// variants, and every enum read back from its name (Parsable).
use io

annotation Alias(name: string) on variant, enumMember

enum Level {
  /// Few messages.
  Low
  @Alias("hi")
  High
}

sealed trait Command {
  /// Builds the program.
  struct Build {
    release: bool = false
  }
  struct Test {
    filter: string = ""
  }
}

struct Options {
  workers: i64 = 4
  limit:   i64? = null
  files:   List<string> = []
  name:    string
}

fun decode<V>(text: string): V? {
  if (V implements Parsable) return V.parse(text)
  null
}

// each field takes the branch its constants decide; the others are not
// checked for it, so `f.Inner` and `f.default()` appear only where they apply
fun filled(): Options => Options.build(f => when {
  f.isOptional => decode<f.Inner>("7")
  f.isList     => [decode<f.Inner>("a") ?: ""]
  f.hasDefault => f.default()
  else         => decode<f.Type>("x") ?: f.Type.default()
})

const fun memberNames<T>(): List<string> {
  val out: MutableList<string> = []
  loop (m in T.members) out.push("${m.index} ${m.name} ${m.doc}")
  out
}

const LEVELS: List<string> = memberNames<Level>()
const COMMANDS: List<string> = memberNames<Command>()

fun variantNames(): string {
  var out = ""
  inline loop (m in Command.members) {
    out += m.name + "=" + m.Type.name() + " "
  }
  out
}

extend Build { static fun name(): string => "build" }
extend Test { static fun name(): string => "test" }

fun aliases(): i64 {
  var n = 0
  inline loop (m in Level.members) {
    if (m.has<Alias>()) n += 1
    if (val a = m.attr<Alias>()) n += a.name.len()
  }
  n
}

struct Required { name: string }
struct Count { n: i64 = 0 }

fun mistakes(): i64 {
  var n = 0
  inline loop (f in Required.fields) {
    n += f.default().len() // error: 'name' has no default: test 'f.hasDefault' before 'f.default()' (D203)
  }
  inline loop (g in Count.fields) {
    val t: g.Inner? = null // error: 'g.Inner' is the type inside a 'T?' or 'List<T>' field, and this one is a 'i64': test 'g.isOptional' or 'g.isList' first (D203)
    n += if (t == null) 0 else 1
  }
  inline loop (h in Count.fields) {
    n += h.Inner.default() // error: 'h.Inner' is the type inside a 'T?' or 'List<T>' field, and this is not one: test 'h.isOptional' or 'h.isList' first (D203)
    n += h.default(1) // error: 'default' takes no arguments: f.default() is the field's declared default (D203)
  }
  inline loop (m in Level.members) {
    val x = m.get(Level.Low) // error: 'get' reads a field; a member handle has the methods has and attr (D203)
    val y = m.typeName // error: a member handle has name, doc, index and, for a variant, Type, and the methods has and attr; 'typeName' is none of them (D203)
    val z = m.Type.default() // error: 'm' is an enum's member, which has no type of its own; 'Type' is a field's or a variant's (D203)
  }
  inline loop (o in Options.members) { } // error: 'Options' has no members to read: 'members' lists an enum's members or a sealed family's variants; a struct has 'fields' (D203)
  n
}

fun main() {
  val parsed: Level? = decode("High")
  io.println("${filled().limit} ${filled().files} ${filled().workers} ${LEVELS} ${COMMANDS} ${variantNames()} ${aliases()} ${parsed} ${mistakes()}")
}
