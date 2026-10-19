// Reading a constraint: sets joined by `||`, comparators in a set, each
// comparator a partial version with an optional operator, expanded to bounds.

/// A version as a constraint writes it: the parts after the first missing one
/// are "any" (`1.2` and `1.2.x` are alike).
struct Partial {
  major: i64?
  minor: i64?
  patch: i64?
  pre:   List<string>

  /// `major.minor.patch` with the missing parts as zero.
  fun filled(): Version =>
    Version(major: this.major ?: 0, minor: this.minor ?: 0, patch: this.patch ?: 0, pre: this.pre)

  /// The first version past everything this partial covers: `1.2` gives `1.3.0`,
  /// `1` gives `2.0.0`, a full version its next patch release.
  fun end(): Version {
    val major = this.major ?: 0
    val minor = this.minor ?: return Version(major: major + 1, minor: 0, patch: 0)
    val patch = this.patch ?: return Version(major, minor: minor + 1, patch: 0)
    Version(major, minor, patch: patch + 1)
  }
}

/// Every set of a constraint.
fun readConstraint(text: string): List<List<Comparator>> throws VersionError {
  val sets: MutableList<List<Comparator>> = []
  loop (written in text.split("||")) {
    sets.push(try readSet(written.trim(), text))
  }
  sets.toList()
}

/// The comparators of one set.
fun readSet(written: string, whole: string): List<Comparator> throws VersionError {
  if (written.isEmpty()) {
    throw VersionError(message: "${quoted(whole)}: a constraint is a version, a bound like '>=1.2', or '*' for any version")
  }
  val comparators: MutableList<Comparator> = []
  loop (token in tokens(written)) {
    loop (comparator in try expand(token, whole)) comparators.push(comparator)
  }
  comparators.toList()
}

/// The words of a set, an operator on its own joined to the version after it
/// (`>= 1.2` is one comparator).
fun tokens(written: string): List<string> {
  val out: MutableList<string> = []
  var pending = ""
  loop (word in written.replace(",", " ").split(" ")) {
    if (word.isEmpty()) continue
    if (isOperator(word)) {
      pending += word
    } else {
      out.push(pending + word)
      pending = ""
    }
  }
  if (!pending.isEmpty()) out.push(pending)
  out.toList()
}

const OPERATORS: List<string> = [">=", "<=", ">", "<", "=", "^", "~"]

fun isOperator(word: string): bool => OPERATORS.contains(word)

/// The operator a token starts with, longest first, and the rest.
fun splitOperator(token: string): (string, string) {
  loop (op in OPERATORS) {
    if (token.startsWith(op)) return (op, token.substring(op.len(), token.len()) ?: "")
  }
  ("", token)
}

/// A comparator token as bounds.
fun expand(token: string, whole: string): List<Comparator> throws VersionError {
  val (op, rest) = splitOperator(token)
  if (rest.isEmpty()) throw VersionError(message: "${quoted(whole)}: '$op' needs a version after it")
  val p = try readPartial(rest, whole)
  val isFull = p.patch != null
  val anyVersion = p.major == null
  val low = p.filled()
  val atLeast = Comparator(op: Op.Ge, version: low)
  val below = Comparator(op: Op.Lt, version: p.end())
  when (op) {
    ""   => if (anyVersion) [atLeast] else if (isFull) [Comparator(op: Op.Eq, version: low)] else [atLeast, below]
    "="  => if (anyVersion) [atLeast] else if (isFull) [Comparator(op: Op.Eq, version: low)] else [atLeast, below]
    ">=" => [atLeast]
    "<"  => if (anyVersion) throw VersionError(message: "${quoted(whole)}: '<*' matches no version") else [Comparator(op: Op.Lt, version: low)]
    ">"  => if (anyVersion) throw VersionError(message: "${quoted(whole)}: '>*' matches no version") else if (isFull) [Comparator(op: Op.Gt, version: low)] else [Comparator(op: Op.Ge, version: p.end())]
    "<=" => if (anyVersion) [atLeast] else if (isFull) [Comparator(op: Op.Le, version: low)] else [below]
    "~"  => if (anyVersion) [atLeast] else [atLeast, Comparator(op: Op.Lt, version: tildeEnd(p))]
    else => if (anyVersion) [atLeast] else [atLeast, Comparator(op: Op.Lt, version: caretEnd(p))]
  }
}

/// `~1.2.3` allows patch updates: below `1.3.0`; `~1` allows below `2.0.0`.
fun tildeEnd(p: Partial): Version {
  val major = p.major ?: 0
  val minor = p.minor ?: return Version(major: major + 1, minor: 0, patch: 0)
  Version(major, minor: minor + 1, patch: 0)
}

/// `^1.2.3` allows what keeps the first non-zero number: below `2.0.0`;
/// `^0.2.3` below `0.3.0`; `^0.0.3` below `0.0.4`.
fun caretEnd(p: Partial): Version {
  val major = p.major ?: 0
  if (major > 0) return Version(major: major + 1, minor: 0, patch: 0)
  val minor = p.minor ?: return Version(major: 1, minor: 0, patch: 0)
  val patch = p.patch ?: return Version(major: 0, minor: minor + 1, patch: 0)
  if (minor > 0) return Version(major: 0, minor: minor + 1, patch: 0)
  Version(major: 0, minor: 0, patch: patch + 1)
}

fun isWildcard(part: string): bool => part == "x" || part == "X" || part == "*"

/// `1`, `1.2`, `1.2.3`, `1.2.3-rc.1`, `1.x`, `*`, with an optional `v`.
fun readPartial(written: string, whole: string): Partial throws VersionError {
  val text = if (written.startsWith("v")) written.substring(1, written.len()) ?: "" else written
  val (numbers, preText) = text.splitOnce("-") ?: (text, "")
  val pre: List<string> = if (text.indexOf("-") >= 0) try readPre(preText, whole) else []
  val parts = numbers.split(".")
  if (parts.len() > 3) throw VersionError(message: "${quoted(whole)}: ${quoted(written)} has more than three numbers")
  val values: MutableList<i64?> = []
  var wild = false
  loop (part in parts) {
    if (isWildcard(part)) {
      wild = true
      values.push(null)
      continue
    }
    if (wild) throw VersionError(message: "${quoted(whole)}: ${quoted(written)} has a number after an 'x'")
    values.push(readNumber(part) ?: throw VersionError(message: "${quoted(whole)}: ${quoted(part)} is not a number without leading zeros, or 'x'"))
  }
  if (wild && !pre.isEmpty()) throw VersionError(message: "${quoted(whole)}: ${quoted(written)} has a pre-release after an 'x'")
  if (!pre.isEmpty() && values.len() < 3) throw VersionError(message: "${quoted(whole)}: a pre-release goes with all three numbers, not ${quoted(written)}")
  Partial(major: values.at(0) ?: null, minor: values.at(1) ?: null, patch: values.at(2) ?: null, pre)
}
