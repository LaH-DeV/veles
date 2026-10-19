/// Versions of packages and the constraints that select them.
///
/// A `Version` is `major.minor.patch`, with an optional pre-release
/// (`1.4.2-rc.1`). Versions are ordered the way semver.org orders them, and
/// a `Constraint` — `^1.2`, `~1.4.2`, `>=1.0 <2.0`, `1.x || 3.1` — says which
/// versions are acceptable.
///
/// ```veles
/// use version { Constraint, Version }
///
/// val installed = try Version.parse("1.4.2")
/// val wanted = try Constraint.parse("^1.2")
/// if (wanted.matches(installed)) io.println("$installed satisfies $wanted")
///
/// val newest = wanted.newest(available)       // the highest version that qualifies
/// installed.compareTo(try Version.parse("1.10.0"))   // Less: numbers, not text
/// ```
///
/// This is the flavour of semver Veles packages use (D132): a leading `v` is
/// accepted and dropped (`v1.4.2` is how a git tag reads), build metadata
/// (`1.4.2+abc`) is refused because two versions differing only there would be
/// one tag, and each number is a whole number below 2^32 without leading zeros.

/// Why a text is not a version or a constraint: the reason, with no location
/// of its own, so a caller can put its file and line in front.
public error VersionError {
  public message: string
}

/// A package version.
public struct Version {
  public major: i64
  public minor: i64
  public patch: i64
  /// The pre-release identifiers: `rc.1` is `["rc", "1"]`; empty for a release.
  public pre: List<string> = []

  /// Reads `1.4.2`, `v1.4.2` or `1.4.2-rc.1`.
  public static fun parse(text: string): Version throws VersionError => try readVersion(text)

  /// True for a version with a pre-release part: it sorts before its release.
  public fun isPreRelease(): bool => !this.pre.isEmpty()

  /// What makes two versions of one package the same package (D132, Cargo's
  /// rule): the first non-zero component, so `1.4.2` is "1", `0.3.1` is "0.3"
  /// and `0.0.2` is "0.0.2". Versions with equal ids are compatible.
  public fun majorId(): string => when {
    this.major > 0 => "${this.major}"
    this.minor > 0 => "0.${this.minor}"
    else           => "0.0.${this.patch}"
  }

  /// The next major version: `1.4.2` and `1.4.2-rc.1` both give `2.0.0`.
  public fun bumpMajor(): Version => Version(major: this.major + 1, minor: 0, patch: 0)

  /// The next minor version, `1.5.0`.
  public fun bumpMinor(): Version => Version(major: this.major, minor: this.minor + 1, patch: 0)

  /// The next patch version, `1.4.3`; a pre-release gives its own release, `1.4.2`.
  public fun bumpPatch(): Version =>
    if (this.isPreRelease()) Version(major: this.major, minor: this.minor, patch: this.patch) else Version(major: this.major, minor: this.minor, patch: this.patch + 1)

  /// `1.4.2` or `1.4.2-rc.1`; what `"$version"` writes.
  implement Display {
    fun toString(): string {
      val release = "${this.major}.${this.minor}.${this.patch}"
      if (this.pre.isEmpty()) release else "$release-${this.pre.join(".")}"
    }
  }

  /// Precedence as semver.org defines it: the numbers in turn; a pre-release
  /// before its release; pre-release identifiers one by one, a number before a
  /// word, numbers by value, words by their bytes, a shorter list first.
  implement Comparable {
    fun compareTo(other: Version): Ordering {
      val core = [this.major.compareTo(other.major), this.minor.compareTo(other.minor), this.patch.compareTo(other.patch)]
      loop (order in core) {
        if (order != Ordering.Equal) return order
      }
      comparePre(this.pre, other.pre)
    }
  }
}

/// Pre-release lists compared: none is greater than some; then identifier by
/// identifier; then the shorter list is smaller.
fun comparePre(a: List<string>, b: List<string>): Ordering {
  if (a.isEmpty() || b.isEmpty()) return b.len().compareTo(a.len())
  loop (i in 0..<a.len().min(b.len())) {
    val order = compareIdentifier(a.at(i) ?: "", b.at(i) ?: "")
    if (order != Ordering.Equal) return order
  }
  a.len().compareTo(b.len())
}

/// Two pre-release identifiers: numbers before words; numbers by value (their
/// length first, as none has a leading zero), words by their bytes.
fun compareIdentifier(a: string, b: string): Ordering {
  val aNumber = isNumeric(a)
  val bNumber = isNumeric(b)
  if (aNumber && bNumber) {
    val byLength = a.len().compareTo(b.len())
    return if (byLength != Ordering.Equal) byLength else a.compareTo(b)
  }
  if (aNumber) return Ordering.Less
  if (bNumber) return Ordering.Greater
  a.compareTo(b)
}
