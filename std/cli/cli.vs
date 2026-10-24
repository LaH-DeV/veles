/// Command-line arguments read into a struct (D167): each field is a flag,
/// named in kebab case (`minSize` is `--min-size`), its doc comment is its
/// help, and its default is the value when the flag is not given.
///
/// ```veles
/// use cli { Flag, Positional }
///
/// /// Find duplicate files under a directory.
/// struct Options {
///   /// The directory to scan.
///   @Positional
///   dir: string
///   /// How many files to hash at once.
///   @Flag(short: "w", env: "WORKERS")
///   workers: i64 = 4
///   /// Say what is compared.
///   @Flag(short: "v")
///   verbose: bool = false
///   implement cli.Args
/// }
///
/// fun main() {
///   val options = cli.args<Options>()   // --help prints and exits 0; a usage error prints and exits 2
/// }
/// ```
///
/// A `bool` is a switch (`--verbose`, and `--no-verbose` to turn it off);
/// any other field takes a value (`--workers 8`, `--workers=8`, `-w 8`,
/// `-w8`), read with its type's `Parsable` — numbers, `string`, `Duration`,
/// enums by member name. A `T?` is `null` when absent, a `List<T>` takes
/// every occurrence, and a field with neither a default nor `?` is required.
/// Short switches group (`-vq`); `--` ends the flags. Every problem is
/// reported at once, with a suggestion for a mistyped flag. A sealed trait
/// whose variants implement `cli.Args` is a set of subcommands: its first
/// argument names the variant in kebab case.
use build
use io
use os

/// How a field is spelled on the command line, every part optional:
/// `short` a one-letter `-v`, `name` the long flag in place of the field's
/// kebab-case name, `env` a variable to fall back on when the flag is not
/// given, `hidden` left out of the help.
public annotation Flag(short: string = "", name: string = "", env: string = "", hidden: bool = false) on field for Args

/// The field is an argument read by position, in field order, not a flag;
/// a `List<T>` takes the rest.
public annotation Positional on field for Args

/// Why parsing stopped: `--help` or `--version` was asked for, or the
/// arguments were wrong.
public enum ErrorKind {
  Help
  Version
  Usage
}

/// What `cli.parse` throws instead of a value: the help, the version, or
/// every problem with the arguments followed by the usage — `text` is what
/// to print.
public error Error {
  public kind: ErrorKind
  public text: string
}

/// A struct whose fields are read from the command line, or a variant of a
/// set of subcommands: `implement cli.Args` with no body.
public trait Args {
  // the struct is built in the trait's body, where the type has opted in,
  // so private fields are filled like public ones
  internal static fun readArgs(args: List<string>, command: string): This throws Error {
    val specs = specsOf<This>()
    val help = helpOf<This>(command)
    val scanned = try scan(args, specs, help)
    val values = scanned.values
    val problems = scanned.problems.toMutable()
    problems.addAll(invalidValues<This>(values, specs))
    if (!problems.isEmpty()) throw usage(problems, command, help)
    This.build(f => {
      val texts = values.get(f.name) ?: []
      when {
        f.isOptional => if (val text = texts.last()) decode<f.Inner>(text) ?: checked() else null
        f.isList     => texts.map(text => decode<f.Inner>(text) ?: checked())
        f.hasDefault => if (val text = texts.last()) decode<f.Type>(text) ?: checked() else f.default()
        else         => decode<f.Type>(texts.last() ?: checked()) ?: checked()
      }
    })
  }
}

/// Reads `args` (the arguments without the program's name, as
/// `os.args()` gives them) into a `T`: a struct that implements `Args`, or
/// a sealed trait whose variants do. Throws `Error` for `--help`,
/// `--version` and every usage problem; the program decides what to do.
public fun parse<T>(args: List<string>): T throws Error =>
  if (T implements Args) try T.readArgs(args, build.program.name) else try readCommand<T>(args, build.program.name)

/// The program's own arguments as a `T`, for `main`: on `--help` or
/// `--version` it prints them and exits 0; on a usage problem it prints
/// every problem and the usage to the error stream and exits 2.
public fun args<T>(): T {
  parse<T>(os.args()) catch (e) {
    if (e.kind == ErrorKind.Usage) {
      io.eprintln(e.text)
      os.exit(2)
    }
    io.println(e.text)
    os.exit(0)
  }
}

// ---------------------------------------------------------------------------
// subcommands: a sealed family whose variants implement Args

fun readCommand<T>(args: List<string>, command: string): T throws Error {
  val help = commandsHelp<T>(command)
  val [word, ..rest] = args else throw usage(["a command is missing"], command, help)
  if (word == "--help" || word == "-h") throw Error(kind: ErrorKind.Help, text: help)
  if (word == "--version" && !build.program.version.isEmpty()) throw versionError()
  val names: MutableList<string> = []
  inline loop (m in T.members) {
    val name = kebab(m.name)
    if (word == name) return try m.Type.readArgs(rest, "$command $name")
    names.push(name)
  }
  val hint = if (val near = nearest(word, names)) "; did you mean '$near'?" else ""
  throw usage(["unknown command '$word'$hint"], command, help)
}

fun commandsHelp<T>(command: string): string {
  val out = StringBuilder()
  if (!T.doc.isEmpty()) out.append("${T.doc}\n\n")
  out.append("Usage: $command <command> [options]\n\nCommands:\n")
  val rows: MutableList<(string, string)> = []
  inline loop (m in T.members) {
    rows.push((kebab(m.name), firstLine(m.doc)))
  }
  table(out, rows)
  out.append("\nRun '$command <command> --help' for a command's options.")
  out.toString()
}

// ---------------------------------------------------------------------------
// a struct's flags

// one field as the command line sees it
struct Spec {
  field:        string
  long:         string
  short:        string = ""
  env:          string = ""
  hidden:       bool = false
  positional:   bool = false
  switchFlag:   bool = false
  repeated:     bool = false
  required:     bool = false
  doc:          string = ""
  shownDefault: string = ""
  choices:      string = ""
}

fun specsOf<T>(): List<Spec> {
  val out: MutableList<Spec> = []
  inline loop (f in T.fields) {
    val flag = f.attr<Flag>() ?: Flag()
    val long = if (flag.name.isEmpty()) kebab(f.name) else flag.name
    // refused while compiling: every part here is a constant
    static assert(kebab(f.name) != "help" && kebab(f.name) != "version" || f.has<Flag>(), "cli: field '${f.name}' would be --${kebab(f.name)}, which is cli's own flag; rename it or give it @Flag(name: …)")
    static assert((f.attr<Flag>()?.name ?: "") != "help" && (f.attr<Flag>()?.name ?: "") != "version", "cli: field '${f.name}': --help and --version are cli's own flags; give it another @Flag(name: …)")
    static assert((f.attr<Flag>()?.short ?: "") != "h", "cli: field '${f.name}': -h is cli's own help flag; give it another @Flag(short: …)")
    static assert((f.attr<Flag>()?.short ?: "").len() <= 1, "cli: field '${f.name}': a short flag is one letter, as in @Flag(short: \"v\")")
    out.push(Spec(
      field: f.name,
      long,
      short: flag.short,
      env: flag.env,
      hidden: flag.hidden,
      positional: f.has<Positional>(),
      switchFlag: f.typeName == "bool" || f.typeName == "bool?",
      repeated: f.isList,
      required: !f.hasDefault && !f.isOptional && !f.isList,
      doc: f.doc,
      shownDefault: if (f.hasDefault && !f.isOptional) shown(shownValue(f.default()), f.typeName) else "",
      choices: when {
        f.isOptional || f.isList => choicesOf<f.Inner>()
        else                     => choicesOf<f.Type>()
      },
    ))
  }
  out
}

// a default worth printing in the help: not an empty text, list or `false`
fun shown(text: string, typeName: string): string {
  if (text.isEmpty() || text == "[]" || typeName == "bool") return ""
  text
}

// what scan finds: each field's texts, by field name, and every problem
struct Scanned {
  values:   Map<string, List<string>>
  problems: List<string>
}

// the arguments sorted into each field's texts; throws only for --help and
// --version, every other problem is listed
fun scan(args: List<string>, specs: List<Spec>, help: string): Scanned throws Error {
  val values: MutableMap<string, MutableList<string>> = [:]
  val problems: MutableList<string> = []
  val positionals: MutableList<string> = []
  val add = (field: string, text: string) => {
    if (val list = values.get(field)) list.push(text) else values.set(field, [text])
  }
  var i: i64 = 0
  var flagsEnded = false
  loop (i < args.len()) {
    val arg = args.at(i)
    i += 1
    when {
      flagsEnded || arg == "-" || !arg.startsWith("-") => positionals.push(arg)
      arg == "--" => flagsEnded = true
      arg == "--help" || arg == "-h" => throw Error(kind: ErrorKind.Help, text: help)
      arg == "--version" && !build.program.version.isEmpty() => throw versionError()
      arg.startsWith("--") => {
        var name = arg.slice(2, arg.len())
        var attached: string? = null  // `--name=value`
        if (val parts = name.splitOnce("=")) {
          name = parts.before
          attached = parts.after
        }
        val negated = name.startsWith("no-")
        var spec = findLong(specs, name)
        if (spec == null && negated) spec = findLong(specs, name.slice(3, name.len()))
        val found = spec ?: Spec(field: "", long: "", positional: true)
        if (found.positional) {
          val hint = if (val near = nearest(name, specs.filter(s => !s.positional && !s.hidden).map(s => s.long))) "; did you mean '--$near'?" else ""
          problems.push("unknown flag '--$name'$hint")
          continue
        }
        if (found.switchFlag) {
          val on = !(negated && found.long != name)
          if (val text = attached) add(found.field, text) else add(found.field, if (on) "true" else "false")
        } else if (val text = attached) {
          add(found.field, text)
        } else if (val next = args.at(i)) {
          add(found.field, next)
          i += 1
        } else {
          problems.push("'--${found.long}' needs a value")
        }
      }
      else => {
        // short flags: `-v`, grouped switches `-vq`, a value `-w 8` or `-w8`
        var k: i64 = 1
        loop (k < arg.len()) {
          val letter = arg.slice(k, k + 1)
          k += 1
          val spec = specs.find(s => s.short == letter) ?: Spec(field: "", long: "", positional: true)
          if (spec.positional) {
            problems.push("unknown flag '-$letter'")
            continue
          }
          if (spec.switchFlag) {
            add(spec.field, "true")
            continue
          }
          if (k < arg.len()) {
            add(spec.field, arg.slice(k, arg.len()))
          } else if (val next = args.at(i)) {
            add(spec.field, next)
            i += 1
          } else {
            problems.push("'-$letter' needs a value")
          }
          break
        }
      }
    }
  }
  // positionals, in field order; a list takes the rest
  var p: i64 = 0
  loop (spec in specs) {
    if (!spec.positional) continue
    if (spec.repeated) {
      loop (p < positionals.len()) {
        add(spec.field, positionals.at(p))
        p += 1
      }
    } else if (val text = positionals.at(p)) {
      add(spec.field, text)
      p += 1
    }
  }
  loop (extra in positionals.drop(p)) problems.push("unexpected argument '$extra'")
  // the environment, then what is still missing
  loop (spec in specs) {
    if (values.containsKey(spec.field)) continue
    if (!spec.env.isEmpty()) {
      if (val text = os.env(spec.env)) {
        add(spec.field, text)
        continue
      }
    }
    if (spec.required) problems.push(if (spec.positional) "the argument <${spec.long}> is missing" else "the flag '--${spec.long}' is missing")
  }
  val out: MutableMap<string, List<string>> = [:]
  loop ((field, texts) in values.entries()) out.set(field, texts.toList())
  Scanned(values: out, problems)
}

// each text that does not read as its field's type, as a sentence
fun invalidValues<T>(values: Map<string, List<string>>, specs: List<Spec>): List<string> {
  val out: MutableList<string> = []
  inline loop (f in T.fields) {
    val texts = values.get(f.name) ?: []
    val spec = specs.find(s => s.field == f.name)
    val flagName = if (spec == null) f.name else if (spec.positional) "<${spec.long}>" else "'--${spec.long}'"
    loop (text in texts) {
      val reads = when {
        f.isOptional || f.isList => decode<f.Inner>(text) != null
        else                     => decode<f.Type>(text) != null
      }
      if (!reads) out.push("invalid value '$text' for $flagName: expected ${f.typeName}")
    }
  }
  out
}

fun findLong(specs: List<Spec>, name: string): Spec? => specs.find(s => s.long == name)

// the help of a struct's flags
fun helpOf<T>(command: string): string {
  val specs = specsOf<T>()
  val out = StringBuilder()
  if (!T.doc.isEmpty()) out.append("${T.doc}\n\n")
  out.appendLine("Usage: ${usageLine(specs, command)}")
  val arguments: MutableList<(string, string)> = []
  val options: MutableList<(string, string)> = []
  loop (spec in specs) {
    if (spec.hidden) continue
    var text = firstLine(spec.doc)
    if (!spec.shownDefault.isEmpty()) text = (text + " (default: ${spec.shownDefault})").trim()
    if (!spec.choices.isEmpty()) text = (text + " [one of: ${spec.choices}]").trim()
    if (!spec.env.isEmpty()) text = (text + " [env: ${spec.env}]").trim()
    if (spec.positional) {
      arguments.push(("<${spec.long}>" + (if (spec.repeated) "..." else ""), text))
    } else {
      val short = if (spec.short.isEmpty()) "    " else "-${spec.short}, "
      val value = if (spec.switchFlag) "" else " <${spec.long.toUpper()}>"
      options.push(("$short--${spec.long}$value", text))
    }
  }
  options.push(("-h, --help", "Print this help"))
  if (!build.program.version.isEmpty()) options.push(("    --version", "Print the version"))
  if (!arguments.isEmpty()) {
    out.append("\nArguments:\n")
    table(out, arguments)
  }
  out.append("\nOptions:\n")
  table(out, options)
  out.toString().trimEnd()
}

fun usageLine(specs: List<Spec>, command: string): string {
  var line = "$command [options]"
  loop (spec in specs) {
    if (!spec.positional) continue
    line += if (spec.repeated) " [${spec.long}...]" else if (spec.required) " <${spec.long}>" else " [${spec.long}]"
  }
  line
}

// two columns, the first padded to the widest
fun table(out: StringBuilder, rows: List<(string, string)>) {
  var width: i64 = 0
  loop ((left, _) in rows) width = width.max(left.len())
  loop ((left, right) in rows) {
    out.appendLine(if (right.isEmpty()) "  $left" else "  ${left.padEnd(width)}  $right")
  }
}

fun firstLine(doc: string): string => (doc.lines().first() ?: "").trim()

fun usage(problems: List<string>, command: string, help: string): Error {
  val out = StringBuilder()
  loop (problem in problems) out.appendLine("error: $problem")
  val usageText = help.lines().find(line => line.startsWith("Usage: ")) ?: "Usage: $command"
  out.append("\n$usageText\nRun '$command --help' for more.")
  Error(kind: ErrorKind.Usage, text: out.toString())
}

fun versionError(): Error => Error(kind: ErrorKind.Version, text: "${build.program.name} ${build.program.version}")

// a value of V from its text; every flag type is Parsable, and an enum is
// read in any spelling (`--format json` is Format.Json, D205)
fun decode<V>(text: string): V? {
  static assert(V implements Parsable, "cli: a flag's type is read with Parsable (numbers, string, bool, Duration, enums, …); give the field one")
  if (V implements Enumerated) return V.parse(text, case: Case.Any)
  if (V implements Parsable) return V.parse(text)
  null
}

// a default as the help shows it: an enum in kebab case, as it is typed
fun shownValue<V>(value: V): string {
  if (V implements Enumerated) return value.toString(case: Case.Kebab)
  "$value"
}

// an enum flag's choices for the help, in kebab case; "" for any other type
fun choicesOf<V>(): string {
  if (V implements Enumerated) return V.values().map(v => v.toString(case: Case.Kebab)).join(", ")
  ""
}

// a value scan has already read: decoding cannot fail here
fun checked(): Never => panic("cli: a value scan accepted did not decode")

// `minSize` is `min-size`, `RunTests` is `run-tests`
const fun kebab(name: string): string {
  val out = StringBuilder()
  loop (i in 0..<name.len()) {
    val b = name.byteAt(i)
    if (b.isAsciiUpper()) {
      if (i > 0) out.appendByte('-')
      out.appendByte(b.toAsciiLower())
    } else {
      out.appendByte(b)
    }
  }
  out.toString()
}

// the candidate a mistyped word most likely meant, within a third of its
// length (the compiler's own rule for "did you mean")
fun nearest(word: string, candidates: List<string>): string? {
  val limit = (word.len() / 3).max(1)
  var best: string? = null
  var bestDistance = limit + 1
  loop (candidate in candidates) {
    val d = distance(word, candidate)
    if (d < bestDistance) {
      best = candidate
      bestDistance = d
    }
  }
  best
}

// optimal string alignment: Levenshtein plus swapping two neighbours
fun distance(a: string, b: string): i64 {
  val n = b.len()
  var before = MutableList<i64>.repeat(0, n + 1)
  var previous = MutableList<i64>.make(n + 1, j => j)
  var current = MutableList<i64>.repeat(0, n + 1)
  loop (i in 1..a.len()) {
    current.set(0, i)
    loop (j in 1..n) {
      val cost = if (a.byteAt(i - 1) == b.byteAt(j - 1)) 0 else 1
      var d = ((previous.at(j) ?: 0) + 1).min((current.at(j - 1) ?: 0) + 1).min((previous.at(j - 1) ?: 0) + cost)
      if (i > 1 && j > 1 && a.byteAt(i - 1) == b.byteAt(j - 2) && a.byteAt(i - 2) == b.byteAt(j - 1)) {
        d = d.min((before.at(j - 2) ?: 0) + 1)
      }
      current.set(j, d)
    }
    val oldest = before
    before = previous
    previous = current
    current = oldest
  }
  previous.at(n) ?: 0
}
