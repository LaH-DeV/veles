// std/cli (D167, D203, D204): arguments read into a struct, every problem
// at once, help from the doc comments, subcommands as a sealed family.

/// Find duplicate files under a directory.
struct Dedup {
  /// The directory to scan.
  @Positional
  dir: string
  /// How many files to hash at once.
  @Flag(short: "w", env: "VELES_CLI_TEST_WORKERS")
  workers: i64 = 4
  /// Smallest file to consider, in bytes.
  minSize: i64 = 1
  /// Say what is compared.
  @Flag(short: "v")
  verbose: bool = false
  /// Leave out files matching this, as often as needed.
  @Flag(short: "x", name: "exclude")
  excludes: List<string> = []
  /// Stop after this long.
  timeout: Duration? = null
  @Flag(hidden: true)
  debugDump: bool = false
  implement Args
}

enum Format {
  Text
  Json
}

/// Builds and tests packages.
sealed trait Tool {
  /// Compile a package.
  struct Build {
    /// Optimize.
    release: bool = false
    format:  Format = Format.Text
    implement Args
  }
  /// Run the tests.
  struct RunTests {
    @Positional
    filters: List<string> = []
    @Flag(short: "j")
    jobs: i64 = 1
    implement Args
  }
}

test fun dedup(args: List<string>): Dedup throws Error => try parse<Dedup>(args)

// what parsing threw, or null when it read a value
test fun failure<T>(args: List<string>): Error? {
  val _ = parse<T>(args) catch (e) {
    return e
  }
  null
}

test fun problem(args: List<string>): string {
  val d = parse<Dedup>(args) catch (e) {
    return e.text
  }
  "parsed ${d.dir}"
}

test "flags, values and defaults" {
  val d = try dedup(["src", "--workers", "8", "-v", "--min-size=10"])
  expect(d.dir == "src")
  expect(d.workers == 8)
  expect(d.minSize == 10)
  expect(d.verbose)
  expect(d.excludes.isEmpty())
  expect(d.timeout == null)
  val plain = try dedup(["."])
  expect(plain.workers == 4 && plain.minSize == 1 && !plain.verbose)
}

test "short flags group and take values attached or apart" {
  val d = try dedup(["-vw16", "lib"])
  expect(d.verbose && d.workers == 16 && d.dir == "lib")
  val e = try dedup(["-w", "2", "lib", "-x", "*.tmp", "--exclude=*.bak"])
  expect(e.workers == 2 && e.excludes == ["*.tmp", "*.bak"])
}

test "a switch turns off with --no-, an optional and a Duration read" {
  val d = try dedup([".", "--verbose", "--no-verbose", "--timeout", "5s"])
  expect(!d.verbose)
  expect(d.timeout == Duration.seconds(5))
}

test "-- ends the flags" {
  val d = try dedup(["--", "-odd-name"])
  expect(d.dir == "-odd-name")
}

test "every problem is reported at once, with a suggestion" {
  val text = problem(["--wokers", "--min-size", "many", "-q"])
  expect(text.contains("error: unknown flag '--wokers'; did you mean '--workers'?"))
  expect(text.contains("error: unknown flag '-q'"))
  expect(text.contains("error: invalid value 'many' for '--min-size': expected i64"))
  expect(text.contains("error: the argument <dir> is missing"))
  expect(text.contains("Usage: "))
  expect(text.contains(" [options] <dir>"))
}

test "a value that does not read is a problem, not a panic" {
  val text = problem([".", "--workers", "eight"])
  expect(text.contains("error: invalid value 'eight' for '--workers': expected i64"))
}

test "help comes from the doc comments" {
  val e = failure<Dedup>(["--help"])
  expect(e != null)
  if (e != null) {
    expect(e.kind == ErrorKind.Help)
    expect(e.text.startsWith("Find duplicate files under a directory."))
    expect(e.text.contains("  <dir>"))
    expect(e.text.contains("-w, --workers <WORKERS>"))
    expect(e.text.contains("How many files to hash at once. (default: 4) [env: VELES_CLI_TEST_WORKERS]"))
    expect(e.text.contains("    --min-size <MIN-SIZE>"))
    expect(e.text.contains("-v, --verbose"))
    expect(!e.text.contains("debug-dump"))
    expect(e.text.contains("-h, --help"))
  }
}

test "subcommands name a variant in kebab case" {
  val built = try parse<Tool>(["build", "--release", "--format", "Json"])
  expect(built is Build)
  if (built is Build) expect(built.release && built.format == Format.Json)
  val ran = try parse<Tool>(["run-tests", "-j", "4", "parse", "lex"])
  expect(ran is RunTests)
  if (ran is RunTests) expect(ran.jobs == 4 && ran.filters == ["parse", "lex"])
}

test "a mistyped subcommand is suggested; help lists them" {
  val wrong = failure<Tool>(["biuld"])
  expect(wrong != null && wrong.text.contains("unknown command 'biuld'; did you mean 'build'?"))
  val help = failure<Tool>(["--help"])
  expect(help != null)
  if (help != null) {
    expect(help.text.startsWith("Builds and tests packages."))
    expect(help.text.contains("build      Compile a package."))
    expect(help.text.contains("run-tests  Run the tests."))
  }
}

test "an enum flag reads any spelling and the help lists its choices" {
  val lower = try parse<Tool>(["build", "--format", "json"])
  expect(lower is Build)
  if (lower is Build) expect(lower.format == Format.Json)
  val upper = try parse<Tool>(["build", "--format", "JSON"])
  if (upper is Build) expect(upper.format == Format.Json)
  val bad = failure<Tool>(["build", "--format", "yaml"])
  expect(bad != null && bad.text.contains("invalid value 'yaml' for '--format'"))
  val help = failure<Tool>(["build", "--help"])
  expect(help != null && help.text.contains("--format <FORMAT>  (default: text) [one of: text, json]"))
}

test "kebab case and the nearest name" {
  expect(kebab("minSize") == "min-size")
  expect(kebab("RunTests") == "run-tests")
  expect(nearest("wokers", ["workers", "verbose"]) == "workers")
  expect(nearest("zzz", ["workers"]) == null)
  expect(distance("ab", "ba") == 1)
}
