/// Settings for a program, read into a struct (D125): the environment first,
/// then dotenv or JSON files, then each field's default. Every problem is
/// reported at once, by variable name, and a `Secret` is never echoed.
///
/// ```veles
/// use config, log
///
/// struct Settings {
///   port: i64 = 8080
///   databaseUrl: Secret<string>           // DATABASE_URL
///   @key("LOG_LEVEL")
///   level: log.Level = log.Level.Info
///   timeout: Duration = Duration.seconds(30)
///   implement Decodable
/// }
///
/// fun main() throws config.Error {
///   val settings = try config.load<Settings>(files: [".env"])
///   println("listening on ${settings.port}")
/// }
/// ```
///
/// A field `databaseUrl` is the variable `DATABASE_URL`; a field of a struct
/// type reads the variables that start with its name (`db: DbSettings` with
/// `poolSize` is `DB_POOL_SIZE`); `@key("NAME")` says the name outright.
/// Numbers, `bool` (`true` or `false`), `string`, `Secret`, `Duration`
/// (`30s`, `5m`), `Timestamp` (RFC 3339), enums by member name, and `List<T>`
/// (comma separated) come from variables; a `T?` is `null` when the variable is
/// not set. A `Map` has no variable form and is refused with a panic at the
/// first call, naming the field.
use codec
use fs
use json
use os

/// The settings could not be read: one `Problem` per variable (or file) that
/// is wrong, named as the person who sets it knows it.
public error Error {
  public problems: List<codec.Problem>
  fun message(): string => this.problems.map(p => p.toString()).join("\n")
}

/// One variable a settings struct reads.
public struct Variable {
  /// The variable's name, with any prefix: `DB_POOL_SIZE`.
  public name: string
  /// What a value looks like: `an integer`, `one of "Debug", "Info"`.
  public what: string
  /// Whether `load` fails when it is not set.
  public required: bool
  /// Whether the value is a `Secret`.
  public secret: bool
  /// The default as text, when there is one that has a text form.
  public default: string?
}

/// `T` read from the process's environment and `files`. Files are read
/// from the last to the first, each beneath the ones after it: the environment
/// wins over every file, and a later file over an earlier one. A file that is
/// not there is skipped (`.env` is optional by convention); one that cannot
/// be read or parsed is a problem. `*.json` files are JSON — keys are the
/// fields' names, a struct field an object — and every other file is dotenv:
/// `NAME=value` lines, `#` comments, `export` ignored, `'…'` literal, `"…"`
/// with `\n` `\t` `\"` `\\` escapes, no interpolation. `prefix` goes in front of
/// every variable name (`"APP_"`).
///
/// An empty variable is a value: `PORT=` is `""`, fine for a `string` and a
/// problem for a number.
public fun load<T: Decodable>(files: List<string> = [], prefix: string = ""): T throws Error =>
  try loadWith<T>(name => os.env(name), files, prefix)

/// The variables `T` reads, in field order — for `--help` and documentation.
public fun describe<T: Decodable>(prefix: string = ""): List<Variable> {
  val schema = T.schema("env", codec.KeyStyle.UpperSnake)
  val out: MutableList<Variable> = []
  loop (leaf in leavesOf("config.describe", schema, prefix)) {
    out.push(Variable(name: leaf.name, what: leaf.schema.what, required: leaf.required, secret: leaf.schema.secret, default: leaf.field.fallback))
  }
  out.toList()
}

// where a value is found: a dotenv file's variables or a JSON file's tree
struct Layer {
  label: string
  vars:  Map<string, string>
  tree:  codec.Value?
}

// one variable of the struct: its name, the declared names that lead to it,
// and whether failing to set it fails the load
struct Leaf {
  name:     string
  path:     List<string>
  schema:   codec.Schema
  field:    codec.SchemaField
  required: bool
}

// The variables of a schema, nested structs flattened. What no variable can
// hold is a mistake in the type, found on the first run: it panics.
fun leavesOf(who: string, schema: codec.Schema, prefix: string): List<Leaf> {
  if (schema.kind != codec.SchemaKind.Object) {
    panic("$who: the type must be a struct whose fields are read from variables, not ${schema.what}; derive it with 'implement Decodable'")
  }
  val out: MutableList<Leaf> = []
  collect(who, schema, prefix, [], true, out)
  out.toList()
}

fun collect(who: string, schema: codec.Schema, varPrefix: string, declared: List<string>, required: bool, out: MutableList<Leaf>) {
  loop (field in schema.fields) {
    val path = declared.concat([field.name])
    val name = varPrefix + field.key
    val inner = field.schema
    val here = required && field.required
    if (inner.kind == codec.SchemaKind.Object) {
      collect(who, inner, name + "_", path, here, out)
      continue
    }
    val elem = inner.element()
    val bad = if (inner.kind == codec.SchemaKind.Unsupported) {
      inner.what
    } else if (elem != null && (elem.kind == codec.SchemaKind.Object || elem.kind == codec.SchemaKind.List || elem.kind == codec.SchemaKind.Unsupported)) {
      "a list of ${elem.what}"
    } else {
      ""
    }
    if (!bad.isEmpty()) {
      panic("$who: field '${path.join(".")}' is $bad, which a variable cannot hold; mark it @skip (with a default), or read it yourself")
    }
    out.push(Leaf(name, path, schema: inner, field, required: here))
  }
}

// The value `path` leads to in a JSON file's tree, `null` when it is not
// there or is `null`.
fun jsonAt(tree: codec.Value, path: List<string>): codec.Value? {
  var cur = tree
  loop (name in path) {
    cur = cur.get(name) ?: return null
  }
  if (cur.isNull()) null else cur
}

// The state of one `load`: where values come from, what went wrong, and
// where each value that was found came from.
struct Load {
  lookup:   fun(string): string?
  layers:   List<Layer>
  prefix:   string
  problems: MutableList<codec.Problem> = []
  sources:  MutableMap<string, string> = [:]
  reported: MutableMap<string, bool> = [:]

  // A problem with a variable, saying where its value came from.
  fun fail(variable: string, message: string) {
    val source = this.sources.get(variable)
    val origin = if (source == null) "" else " (from $source)"
    this.reported.set(variable, true)
    this.problems.push(codec.Problem(path: variable, message: message + origin))
  }

  fun bad(variable: string, schema: codec.Schema, text: string) {
    val found = if (schema.secret) "" else ", found \"$text\""
    this.fail(variable, "expected ${schema.what}$found")
  }

  // The tree for a struct: each variable that is set, a nested struct as an
  // object. A nested struct none of whose variables is set is left out when it
  // has a default or can be null, and is there (so each missing variable is
  // reported) when it must be given.
  fun object(schema: codec.Schema, varPrefix: string, declared: List<string>, required: bool): codec.Value? {
    val fields: MutableMap<string, codec.Value> = [:]
    var found = false
    loop (field in schema.fields) {
      val name = varPrefix + field.key
      val path = declared.concat([field.name])
      val inner = field.schema
      if (inner.kind == codec.SchemaKind.Object) {
        val child = this.object(inner, name + "_", path, required && field.required)
        if (child != null) {
          fields.set(field.key, child)
          found = true
        }
        continue
      }
      val v = this.leaf(name, path, inner)
      if (v != null) {
        fields.set(field.key, v)
        found = true
      }
    }
    if (!found && !required) return null
    codec.VObject(fields: fields.toMap())
  }

  // The value of a variable: the environment, then each file, strongest first.
  fun leaf(variable: string, path: List<string>, schema: codec.Schema): codec.Value? {
    val fromEnvironment = this.lookup(variable)
    if (fromEnvironment != null) return this.typed(variable, fromEnvironment, "the environment", schema)
    loop (layer in this.layers) {
      val text = layer.vars.get(variable)
      if (text != null) return this.typed(variable, text, layer.label, schema)
      val tree = layer.tree ?: continue
      val v = jsonAt(tree, path) ?: continue
      this.sources.set(variable, layer.label)
      return v
    }
    null
  }

  // Text from the environment or a dotenv file, as the value its type reads.
  fun typed(variable: string, text: string, source: string, schema: codec.Schema): codec.Value? {
    this.sources.set(variable, source)
    var out: codec.Value? = null
    if (schema.kind == codec.SchemaKind.Int) {
      val n = text.toInt()
      if (n == null) this.bad(variable, schema, text) else out = codec.VInt(value: n)
    } else if (schema.kind == codec.SchemaKind.Float) {
      val x = text.toF64()
      if (x == null) this.bad(variable, schema, text) else out = codec.VFloat(value: x)
    } else if (schema.kind == codec.SchemaKind.Bool) {
      if (text == "true") {
        out = codec.VBool(value: true)
      } else if (text == "false") {
        out = codec.VBool(value: false)
      } else {
        this.bad(variable, schema, text)
      }
    } else if (schema.kind == codec.SchemaKind.List) {
      val elem = schema.element() ?: return null
      val items: MutableList<codec.Value> = []
      var whole = true
      if (!text.trim().isEmpty()) {
        loop (part in text.split(",")) {
          val v = this.typed(variable, part.trim(), source, elem)
          if (v == null) whole = false else items.push(v)
        }
      }
      if (whole) out = codec.VList(items: items.toList())
    } else {
      out = codec.VString(value: text)
    }
    out
  }

  // A problem the decoder found, as the person who sets the variables
  // knows it: `DB.POOL_SIZE` is `DB_POOL_SIZE`, "missing" says it is not set,
  // and a secret's value never appears.
  fun rewrite(p: codec.Problem, leaves: Map<string, Leaf>) {
    val variable = this.prefix + (p.path.split("[").at(0) ?: "").replace(".", "_")
    if (this.reported.get(variable) == true) return
    val leaf = leaves.get(variable)
    val what = if (leaf == null) "a value" else leaf.schema.what
    val secret = leaf != null && leaf.schema.secret
    if (p.message == "missing") {
      this.fail(variable, "not set (expected $what)")
    } else if (secret) {
      this.fail(variable, "is not valid (expected $what)")
    } else {
      this.fail(variable, p.message)
    }
  }
}

// The files as layers, strongest first; what could not be read is a problem.
fun readLayers(files: List<string>, problems: MutableList<codec.Problem>): List<Layer> {
  val layers: MutableList<Layer> = []
  loop (file in files.reversed()) {
    if (!fs.exists(file)) continue
    val text = fs.readFile(file) catch (e) {
      problems.push(codec.Problem(path: file, message: "cannot be read: ${e.message()}"))
      continue
    }
    if (file.toLower().endsWith(".json")) {
      val tree = json.parse(text) catch (e) {
        loop (p in e.problems) {
          problems.push(codec.Problem(path: file, message: p.toString()))
        }
        continue
      }
      if (tree is codec.VObject) {
        layers.push(Layer(label: file, vars: [:], tree))
      } else {
        problems.push(codec.Problem(path: file, message: "expected an object of settings"))
      }
      continue
    }
    val dotenv = parseDotenv(text)
    loop (p in dotenv.problems) {
      problems.push(codec.Problem(path: file, message: p))
    }
    layers.push(Layer(label: file, vars: dotenv.vars, tree: null))
  }
  layers.toList()
}

// `load` with the lookup of the environment passed in, which is how the tests
// give it variables.
fun loadWith<T: Decodable>(lookup: fun(string): string?, files: List<string>, prefix: string): T throws Error {
  val schema = T.schema("env", codec.KeyStyle.UpperSnake)
  val leaves: MutableMap<string, Leaf> = [:]
  loop (leaf in leavesOf("config.load", schema, prefix)) {
    leaves.set(leaf.name, leaf)
  }
  val problems: MutableList<codec.Problem> = []
  val state = Load(lookup, layers: readLayers(files, problems), prefix)
  state.problems.addAll(problems.toList())
  val tree = state.object(schema, prefix, [], true) ?: codec.VObject(fields: [:])
  val decoder = codec.ValueDecoder.of(tree, format: "env", enums: codec.EnumStyle.Name, keys: codec.KeyStyle.UpperSnake, durations: codec.DurationStyle.Text)
  val decoded = T.decode(decoder)
  val found = when (decoded) {
    is Err(e) => e.problems
    is Ok(_)  => decoder.problems()
  }
  val table = leaves.toMap()
  loop (p in found) {
    state.rewrite(p, table)
  }
  if (!state.problems.isEmpty()) throw Error(problems: state.problems.toList())
  when (decoded) {
    is Ok(value) => value
    is Err(_)    => throw Error(problems: [codec.Problem(path: "", message: "the settings could not be read")])
  }
}
