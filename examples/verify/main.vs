// verify: check the files under a directory against a SHA-256 manifest, as
// `sha256sum -c` does, and name the files the manifest does not list. The
// manifest is read while the directory is listed (`awaitAll`, D156); the
// files are hashed --workers at a time by a lazy `mapConcurrent`, whose
// results come back in the manifest's order, so each line is printed as
// soon as the files before it are done while the next ones are hashed.
//
//   verify <dir> <manifest> [--workers N]
use crypto
use fs
use io { println }
use os
use path

error UsageError {
  message: string
}

/// One line of the manifest: `<sha256 in hex>  <path under the directory>`.
struct Entry {
  hash: string
  file: string
}

/// What checking one listed file found.
struct Checked {
  file: string
  ok:   bool
  note: string
}

fun readManifest(file: string): List<Entry> throws IoError | UsageError {
  val entries: MutableList<Entry> = []
  var n = 0
  loop (line in (try fs.readFile(file)).lines()) {
    n += 1
    if (line.trim().isEmpty()) continue
    val (hash, name) = line.splitOnce("  ") ?: throw UsageError(message: "$file:$n: expected '<sha256>  <file>'")
    entries.push(Entry(hash, file: name))
  }
  entries.toList()
}

/// The files under `dir`, as paths relative to it.
fun listFiles(dir: string): List<string> throws IoError {
  val prefix = dir + "/"
  (try fs.walk(dir)).map(f => f.substring(prefix.len(), f.len()) ?: f)
}

/// Hashes one listed file and compares it with the manifest. An unreadable
/// file is a finding, not a reason to stop.
fun check(dir: string, entry: Entry): Checked {
  val full = path.join(dir, entry.file)
  if (!fs.exists(full)) return Checked(file: entry.file, ok: false, note: "missing")
  when (val bytes = fs.readBytes(full)) {
    is Err => Checked(file: entry.file, ok: false, note: "unreadable: ${bytes.message()}")
    is Ok  => {
      val ok = crypto.sha256(bytes).toHex() == entry.hash
      Checked(file: entry.file, ok, note: if (ok) "OK" else "FAILED")
    }
  }
}

struct Options {
  var dir:      string
  var manifest: string
  var workers:  i64 = 4

  static fun parse(args: List<string>): Options throws UsageError {
    val paths: MutableList<string> = []
    var opts = Options(dir: "", manifest: "")
    var i = 0
    loop (i < args.len()) {
      val arg = args.at(i)
      if (arg == "--workers") {
        val value = args.at(i + 1)?.toInt() ?: throw UsageError(message: "--workers needs a number")
        if (value < 1) throw UsageError(message: "--workers must be at least 1")
        opts.workers = value
        i += 1
      } else if (arg.startsWith("-")) {
        throw UsageError(message: "unknown option '$arg'")
      } else {
        paths.push(arg)
      }
      i += 1
    }
    if (paths.len() != 2) throw UsageError(message: "usage: verify <dir> <manifest> [--workers N]")
    opts.dir = paths.at(0)
    opts.manifest = paths.at(1)
    opts
  }
}

fun plural(n: i64, word: string): string => "$n $word${if (n == 1) "" else "s"}"

/// Checks everything; true when every listed file is present and intact.
fun run(args: List<string>): bool throws UsageError | IoError {
  val opts = try Options.parse(args)
  if (!fs.isDir(opts.dir)) throw UsageError(message: "'${opts.dir}' is not a directory")
  if (!fs.isFile(opts.manifest)) throw UsageError(message: "'${opts.manifest}' is not a file")

  // the manifest is read while the directory is listed
  val (entries, present) = awaitAll(async readManifest(opts.manifest), async listFiles(opts.dir))

  // --workers files are hashed at a time; the results arrive in order
  val dir = opts.dir
  var failed = 0
  loop (c in entries.iter().mapConcurrent(e => check(dir, e), workers: opts.workers)) {
    println("${c.file}: ${c.note}")
    if (!c.ok) failed += 1
  }

  val listed = entries.map(e => e.file).toSet()
  val extra = present.filter(f => !listed.contains(f))
  loop (f in extra) println("$f: not in the manifest")
  println("${plural(entries.len() - failed, "file")} verified, $failed failed, ${extra.len()} not listed (${plural(opts.workers, "worker")})")
  failed == 0
}

fun main() {
  when (val r = run(os.args())) {
    is Err => {
      println("verify: ${r.message()}")
      os.exit(if (r is UsageError) 2 else 1)
    }
    is Ok  => if (!r) os.exit(1)
  }
}
