/// A condition on versions.
///
/// A constraint is one or more *sets* joined by `||`; a version satisfies it
/// when it satisfies any set. A set is comparators separated by spaces or
/// commas; a version satisfies the set when it satisfies every one:
///
/// | Written | Means |
/// |---|---|
/// | `1.4.2`, `=1.4.2` | exactly that version |
/// | `>1.4.2`, `>=1.4.2`, `<2.0.0`, `<=2.0.0` | a bound |
/// | `^1.2.3` | compatible: `>=1.2.3 <2.0.0`; `^0.2.3` is `>=0.2.3 <0.3.0`; `^0.0.3` is `>=0.0.3 <0.0.4` |
/// | `~1.2.3` | patch updates: `>=1.2.3 <1.3.0`; `~1` is `>=1.0.0 <2.0.0` |
/// | `1.2`, `1.2.x`, `1.*`, `*` | any version in that range |
/// | `>=1.2 <1.5`, `>=1.2, <1.5` | both |
/// | `1.x \|\| >=3.1` | either |
///
/// A partial version (`1`, `1.2`) is a range: `=1.2` is all of `1.2.x`, `>1.2` is
/// above every `1.2.x` (`>=1.3.0`), `<=1.2` is up to every `1.2.x` (`<1.3.0`),
/// `>=1.2` and `<1.2` start from `1.2.0`.
///
/// A version with a pre-release (`2.0.0-rc.1`) satisfies a set only when one of
/// its comparators names a pre-release of the same `major.minor.patch`, so a
/// range like `^1.2` never offers a release candidate nobody asked for.
public struct Constraint {
  /// The text it was read from.
  public text: string
  sets:        List<List<Comparator>>

  /// Reads a constraint: the table above.
  public static fun parse(text: string): Constraint throws VersionError {
    val sets = try readConstraint(text)
    Constraint(text, sets)
  }

  /// The constraint that only `v` satisfies.
  public static fun exactly(v: Version): Constraint =>
    Constraint(text: "=$v", sets: [[Comparator(op: Op.Eq, version: v)]])

  /// True when `v` satisfies the constraint.
  public fun matches(v: Version): bool {
    loop (set in this.sets) {
      if (satisfiesSet(set, v)) return true
    }
    false
  }

  /// The highest of `candidates` that satisfies the constraint, or null.
  public fun newest(candidates: List<Version>): Version? {
    var best: Version? = null
    loop (candidate in candidates) {
      if (!this.matches(candidate)) continue
      val current = best
      if (current == null || candidate > current) best = candidate
    }
    best
  }

  public fun toString(): string => this.text
}

/// A bound a version is compared with.
enum Op {
  Eq
  Gt
  Ge
  Lt
  Le
}

struct Comparator {
  op:      Op
  version: Version
}

/// True when `v` is inside the comparator's bound, pre-releases aside.
fun within(c: Comparator, v: Version): bool {
  val order = v.compareTo(c.version)
  when (c.op) {
    Op.Eq => order == Ordering.Equal
    Op.Gt => order == Ordering.Greater
    Op.Ge => order != Ordering.Less
    Op.Lt => order == Ordering.Less
    Op.Le => order != Ordering.Greater
  }
}

/// Every comparator holds — and a pre-release version is let in only by a
/// comparator that names a pre-release of the same `major.minor.patch`.
fun satisfiesSet(set: List<Comparator>, v: Version): bool {
  loop (c in set) {
    if (!within(c, v)) return false
  }
  if (!v.isPreRelease()) return true
  loop (c in set) {
    val w = c.version
    if (w.isPreRelease() && w.major == v.major && w.minor == v.minor && w.patch == v.patch) return true
  }
  false
}
