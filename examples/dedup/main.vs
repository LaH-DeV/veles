// dedup: find duplicate files under a directory. The files are hashed by a
// pool of tasks (`mapConcurrent`, at most --workers at a time), each result
// a `Result<Hashed, IoError>` so one unreadable file is reported and
// skipped rather than stopping the run; the main task then groups the
// results by content. Nothing is deleted; the report says what is
// duplicated and how much space it costs.
//
// The arguments are read by std/cli from the doc comments of Options:
// `dedup --help` prints them.
use cli { Flag, Positional }
use fs
use io { println }
use os
use path

error UsageError {
  message: string
}

struct Hashed {
  file: string
  size: i64
  hash: u64
}

/// What makes two files "the same": equal size and equal hash. A struct
/// rather than a `(i64, u64)` tuple so that a group's key reads `key.size`,
/// and it is a map key for free (structural equality and hashing).
struct Fingerprint {
  size: i64
  hash: u64
}

// ---------------------------------------------------------------------------
// hashing: FNV-1a over the bytes, a fingerprint that is cheap to write in
// the language itself (no crypto in std yet)

const FNV_OFFSET: u64 = 14695981039346656037
const FNV_PRIME: u64 = 1099511628211

fun fnv1a(bytes: List<u8>): u64 {
  var h = FNV_OFFSET
  loop (b in bytes) {
    h = (h ^ (b.toU64())) *% FNV_PRIME
  }
  h
}

// ---------------------------------------------------------------------------
// hashing one file; the pool is the prelude's mapConcurrent

/// Reads and fingerprints one file. A file that cannot be read is an
/// IoError; the caller decides that one bad file does not stop the run.
fun hashFile(file: string, verbose: bool): Hashed throws IoError {
  val bytes = try fs.readBytes(file)
  if (verbose) println("hashed ${bytes.len()} bytes ${path.base(file)}")
  Hashed(file, size: bytes.len(), hash: fnv1a(bytes))
}

/// Find duplicate files under a directory.
///
/// Files are hashed a few at a time and grouped by size and hash; nothing is
/// deleted.
struct Options {
  /// The directory to scan.
  @Positional
  dir: string
  /// How many files to hash at once.
  @Flag(short: "w")
  workers: i64 = 4
  /// Smallest file to consider, in bytes.
  minSize: i64 = 1
  /// Say which file is hashed.
  @Flag(short: "v")
  verbose: bool = false
  implement cli.Args
}

fun plural(n: i64, word: string): string => "$n $word${if (n == 1) "" else "s"}"

fun run(opts: Options) throws UsageError | IoError {
  if (opts.workers < 1) throw UsageError(message: "--workers must be at least 1")
  if (!fs.isDir(opts.dir)) throw UsageError(message: "'${opts.dir}' is not a directory")

  val files = try fs.walk(opts.dir)  // every file below, in a stable order

  // hash everything, --workers files at a time; the lambda returns the
  // Result itself (no `try`), so a failure is a value in the list
  val verbose = opts.verbose
  val outcomes = files.mapConcurrent(file => hashFile(file, verbose), workers: opts.workers)
  loop (err in outcomes.errors()) println("cannot read: ${err.message()}")
  val hashed = outcomes.oks()

  // group by fingerprint; a group of one is not a duplicate
  val groups: MutableMap<Fingerprint, MutableList<string>> = [:]
  loop (h in hashed) {
    if (h.size < opts.minSize) continue
    groups.getOrPut(Fingerprint(size: h.size, hash: h.hash), () => []).push(h.file)
  }
  val dupes = groups.entries()
    .filter(e => e.value.len() > 1)
    .sortedWith((a, b) => {
      if (a.key.size != b.key.size) return b.key.size.compareTo(a.key.size)  // largest first
      val first = a.value.first() ?: panic("dedup: a group of duplicates has at least two files")
      first.compareTo(b.value.first() ?: panic("dedup: a group of duplicates has at least two files"))
    })

  var wasted = 0
  loop ((key, members) in dupes) {
    println("${members.len()} identical files, ${plural(key.size, "byte")} each:")
    loop (file in members.sorted()) {
      println("  $file")
    }
    wasted += key.size * (members.len() - 1)
  }
  val total = hashed.fold(0, (acc, h) => acc + h.size)
  println("${plural(hashed.len(), "file")}, ${plural(total, "byte")}, ${plural(opts.workers, "worker")}: ${plural(dupes.len(), "duplicate group")}, ${plural(wasted, "byte")} recoverable")
}

fun main() {
  when (val r = run(cli.args<Options>())) {
    is Err => {
      println("dedup: ${r.message()}")
      os.exit(if (r is UsageError) 2 else 1)
    }
    is Ok  => { }
  }
}
