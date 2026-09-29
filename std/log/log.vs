/// Leveled, structured logging to standard error (D91).
///
/// ```veles
/// use log { field }
///
/// log.info("served", field("path", req.path), field("ms", 3))
/// log.debug("cache miss for $key")          // built only when Debug is on
/// log.withFields([field("id", id)], () => handle(req))
/// ```
///
/// The message of `debug`, `info`, `warn` and `error` is a `lazy` parameter
/// (D90): it is evaluated only when the level is on, so a message full of
/// interpolation costs nothing while it is off. The `field(...)` arguments
/// are evaluated at the call either way; guard a hot path that builds
/// expensive fields with `log.enabled(Level.Debug)`.
///
/// One line is written per call, whole, so lines from different threads
/// never interleave. On a terminal the line is text,
///
///     2026-09-29T10:15:03.123Z INFO  served path=/x ms=3 id=abc
///
/// and anywhere else — a file, a pipe, a log collector — one JSON object:
///
///     {"time":"2026-09-29T10:15:03.123Z","level":"info","msg":"served","path":"/x","ms":3,"id":"abc"}
///
/// The level starts as `info`, or as `VELES_LOG` says (`debug`, `info`,
/// `warn`, `error`, or `off`), and `setLevel` changes it.
use io { eprintln }, json, os, time

extern "C" {
  fun veles_stderr_is_terminal(): bool
}

/// How serious a message is, and the threshold that lets messages through:
/// a message is logged when its level is at least the threshold. `Off` is a
/// threshold only — it lets nothing through.
public enum Level {
  Debug = 0
  Info  = 1
  Warn  = 2
  Error = 3
  Off   = 4
}

/// One named value of a log line. Make one with `field`.
public struct Field {
  /// The name.
  public key: string
  // the value as JSON text, and as it reads in a text line
  json: string
  text: string
}

/// A field of a log line: `field("ms", 3)`. The value keeps its type in
/// JSON output (a number stays a number); a value that cannot be encoded is
/// logged as `null`.
public fun field<T: Encodable>(key: string, value: T): Field {
  val encoded = json.encode(value) ?? "null"
  Field(key, json: encoded, text: readable(encoded))
}

// A value as a text line writes it: strings bare, quoted when they would
// run into the next field; everything else as its JSON.
fun readable(encoded: string): string {
  if (!encoded.startsWith("\"")) return encoded
  val plain = json.decode<string>(encoded) ?? encoded
  if (plain.isEmpty() || plain.contains(" ") || plain.contains("=") || plain.contains("\"") || plain.contains("\n")) {
    return encoded
  }
  plain
}

fun levelFromName(name: string): Level? = when (name.toLower()) {
  "debug" => Level.Debug
  "info"  => Level.Info
  "warn"  => Level.Warn
  "error" => Level.Error
  "off"   => Level.Off
  else    => null
}

fun startingLevel(): Level {
  val named = os.env("VELES_LOG") ?: return Level.Info
  val level = levelFromName(named)
  if (level == null) {
    eprintln("veles: VELES_LOG=$named is not debug, info, warn, error or off; logging at info")
    return Level.Info
  }
  level
}

fun onTerminal(): bool {
  // SAFETY: asks whether standard error is a terminal; no arguments, keeps nothing
  return unsafe {
    veles_stderr_is_terminal()
  }
}

val threshold = Atomic(value: startingLevel().value)
val terminal = onTerminal()
val ambient = TaskLocal<List<Field>>(fallback: [])

/// Lets messages of `level` and above through.
public fun setLevel(level: Level) {
  threshold.store(level.value)
}

/// Whether a message of `level` would be logged now.
public fun enabled(level: Level): bool = level != Level.Off && level.value >= threshold.load()

/// Logs a message that is only for finding out what a program is doing.
public fun debug(lazy msg: fun(): string, fields: Field...) {
  emit(Level.Debug, msg, fields)
}

/// Logs something that happened and is normal.
public fun info(lazy msg: fun(): string, fields: Field...) {
  emit(Level.Info, msg, fields)
}

/// Logs something odd that the program coped with.
public fun warn(lazy msg: fun(): string, fields: Field...) {
  emit(Level.Warn, msg, fields)
}

/// Logs something that went wrong: an operation failed, a request was refused.
public fun error(lazy msg: fun(): string, fields: Field...) {
  emit(Level.Error, msg, fields)
}

/// Runs `f` with `fields` added to every line logged inside it — by it, by
/// what it calls and by the tasks it starts — and returns what `f` returns.
/// A request handler binds its request id this way, once, and every line of
/// that request carries it.
public fun withFields<R, E>(fields: List<Field>, f: fun(): R suspends throws E): R throws E {
  var all: MutableList<Field> = []
  loop (x in ambient.get()) all.push(x)
  loop (x in fields) all.push(x)
  return try ambient.withValue(all.toList(), f)
}

fun emit(level: Level, msg: fun(): string, fields: List<Field>) {
  if (!enabled(level)) return
  var all: MutableList<Field> = []
  loop (x in ambient.get()) all.push(x)
  loop (x in fields) all.push(x)
  val text = msg()
  val now = time.now().toString()
  val line = all.toList()
  eprintln(if (terminal) textLine(now, level, text, line) else jsonLine(now, level, text, line))
}

fun textLine(now: string, level: Level, text: string, fields: List<Field>): string {
  val out = StringBuilder()
  out.append(now)
  out.append(" ")
  out.append(level.toString().toUpper().padEnd(5))
  out.append(" ")
  out.append(text)
  loop (f in fields) {
    out.append(" ")
    out.append(f.key)
    out.append("=")
    out.append(f.text)
  }
  out.toString()
}

fun jsonLine(now: string, level: Level, text: string, fields: List<Field>): string {
  val out = StringBuilder()
  out.append("{\"time\":\"")
  out.append(now)
  out.append("\",\"level\":\"")
  out.append(level.toString().toLower())
  out.append("\",\"msg\":")
  out.append(quote(text))
  loop (f in fields) {
    out.append(",")
    out.append(quote(f.key))
    out.append(":")
    out.append(f.json)
  }
  out.append("}")
  out.toString()
}

fun quote(s: string): string = json.encode(s) ?? "\"\""
