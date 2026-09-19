// dedup: find duplicate files under a directory. The files are hashed by a
// pool of tasks (`mapConcurrent`, at most --workers at a time), each result
// a `Result<Hashed, IoError>` so one unreadable file is reported and
// skipped rather than stopping the run; the main task then groups the
// results by content. Nothing is deleted; the report says what is
// duplicated and how much space it costs.
//
//   dedup <dir> [--workers N] [--min-size BYTES] [--verbose]
use fs, io, os, path

error UsageError {
  message: string
}

struct Hashed {
  file: string
  size: i64
  hash: u64
}

/// What makes two files "the same": equal size and equal hash.
type Fingerprint = (i64, u64)

// ---------------------------------------------------------------------------
// hashing: FNV-1a over the bytes, a fingerprint that is cheap to write in
// the language itself (no crypto in std yet)

const FNV_OFFSET: u64 = 14695981039346656037
const FNV_PRIME: u64 = 1099511628211

fun fnv1a(bytes: List<u8>): u64 {
  var h = FNV_OFFSET
  loop (b in bytes) {
    h = (h ^ (b as u64)) *% FNV_PRIME
  }
  h
}

// ---------------------------------------------------------------------------
// hashing one file; the pool is the prelude's mapConcurrent

/// Reads and fingerprints one file. A file that cannot be read is an
/// IoError; the caller decides that one bad file does not stop the run.
fun hashFile(file: string, verbose: bool): Hashed throws IoError {
  val bytes = try fs.readBytes(file)
  if (verbose) io.println("hashed ${bytes.len()} bytes ${path.base(file)}")
  Hashed(file, size: bytes.len(), hash: fnv1a(bytes))
}

/// Every regular file under `dir`, recursively, in a stable order.
fun walk(dir: string, out: MutableList<string>) throws IoError {
  loop (name in try fs.listDir(dir)) {
    val p = path.join(dir, name)
    if (fs.isDir(p)) try walk(p, out) else out.push(p)
  }
}

struct Options {
  dir:     string
  workers: i64 = 4
  minSize: i64 = 1
  verbose: bool = false

  static fun parse(args: List<string>): Options throws UsageError {
    var dir: string? = null
    var opts = Options(dir: "")
    var i = 0
    loop (i < args.len()) {
      val arg = args.atOrPanic(i)
      when (arg) {
        "--workers", "--min-size" => {
          val value = args.at(i + 1)?.toInt() ?: throw UsageError(message: "$arg needs a number")
          if (arg == "--workers") {
            if (value < 1) throw UsageError(message: "--workers must be at least 1")
            opts.workers = value
          } else opts.minSize = value
          i += 1
        }
        "--verbose", "-v"         => opts.verbose = true
        else                      => {
          if (arg.startsWith("-")) throw UsageError(message: "unknown option '$arg'")
          if (dir != null) throw UsageError(message: "one directory at a time")
          dir = arg
        }
      }
      i += 1
    }
    opts.dir = dir ?: throw UsageError(message: "usage: dedup <dir> [--workers N] [--min-size BYTES] [--verbose]")
    opts
  }
}

fun plural(n: i64, word: string): string = "$n $word${if (n == 1) "" else "s"}"

fun run(args: List<string>) throws UsageError | IoError {
  val opts = try Options.parse(args)
  if (!fs.isDir(opts.dir)) throw UsageError(message: "'${opts.dir}' is not a directory")

  val files: MutableList<string> = []
  try walk(opts.dir, files)

  // hash everything, --workers files at a time; the lambda returns the
  // Result itself (no `try`), so a failure is a value in the list
  val verbose = opts.verbose
  val outcomes = files.toList().mapConcurrent(file => hashFile(file, verbose), workers: opts.workers)
  val hashed: MutableList<Hashed> = []
  loop (outcome in outcomes) {
    if (outcome is Err) {
      io.println("cannot read: ${outcome.message()}")
      continue
    }
    hashed.push(outcome)
  }

  // group by (size, hash); a group of one is not a duplicate
  val groups: MutableMap<Fingerprint, MutableList<string>> = [:]
  loop (h in hashed) {
    if (h.size < opts.minSize) continue
    groups.getOrPut((h.size, h.hash), () => []).push(h.file)
  }
  val dupes = groups.entries()
    .filter(e => e.1.len() > 1)
    .sortedWith((a, b) => {
      val (sizeA, sizeB) = (a.0.0, b.0.0)
      if (sizeA != sizeB) return sizeB - sizeA  // largest first
      a.1.atOrPanic(0).compareTo(b.1.atOrPanic(0))
    })

  var wasted = 0
  loop ((key, members) in dupes) {
    val (size, _) = key
    io.println("${members.len()} identical files, ${plural(size, "byte")} each:")
    loop (file in members.sorted()) {
      io.println("  $file")
    }
    wasted += size * (members.len() - 1)
  }
  val total = hashed.fold(0, (acc, h) => acc + h.size)
  io.println("${plural(hashed.len(), "file")}, ${plural(total, "byte")}, ${plural(opts.workers, "worker")}: ${plural(dupes.len(), "duplicate group")}, ${plural(wasted, "byte")} recoverable")
}

fun main() {
  when (val r = run(os.args())) {
    is Ok  => { }
    is Err => {
      io.println("dedup: ${r.message()}")
      os.exit(if (r is UsageError) 2 else 1)
    }
  }
}
