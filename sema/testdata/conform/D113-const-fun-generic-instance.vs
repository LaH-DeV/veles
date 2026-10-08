// D113: a generic `const fun` (`List<T>.max`) whose instance calls an ordinary
// function of its type argument (`Version.compareTo`) is fine at run time; only
// a constant that runs that instance is refused, at the constant.
use io

struct Version {
  major: i64

  implement Comparable {
    fun compareTo(other: Version): Ordering => this.major.compareTo(other.major)
  }
}

const fun newest(): i64 {
  val vs: List<Version> = [Version(major: 2), Version(major: 1)]
  vs.max()?.major ?: 0
}

const NEWEST: i64 = newest() // error: cannot run in the compiler with these type arguments: it calls

fun main() {
  val vs: List<Version> = [Version(major: 3), Version(major: 4)]
  io.println("${vs.max()?.major ?: 0} ${vs.min()?.major ?: 0} ${[3, 1].max() ?: 0}")
}
