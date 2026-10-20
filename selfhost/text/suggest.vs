// "unknown name 'countr'; did you mean 'counter'?" — a name to suggest for one
// that is not known. A suggestion is a guess: callers say so.
use utf8
use lexer { isIDStart }

/// Names other languages use for what Veles calls something else (lowercased,
/// `_` dropped), most likely first. Only a name the candidates actually
/// contain is ever suggested.
const FOREIGN_NAMES: Map<string, List<string>> = [
  "size": ["len"], "length": ["len"], "count": ["len"], "put": ["set"], "insert": ["set", "add", "push"],
  "append": ["push", "add"], "add": ["push", "set"], "pushback": ["push"], "get": ["at"],
  "has": ["contains", "containsKey"], "includes": ["contains"], "containskey": ["contains"],
  "haskey": ["containsKey"], "delete": ["remove"], "erase": ["remove"], "removeat": ["remove"],
  "reduce": ["fold"], "each": ["forEach"], "sort": ["sorted"], "reverse": ["reversed"], "empty": ["isEmpty"],
  "strip": ["trim"], "trimleft": ["trimStart"], "trimright": ["trimEnd"], "lstrip": ["trimStart"],
  "rstrip": ["trimEnd"], "touppercase": ["toUpper"], "uppercase": ["toUpper"], "upper": ["toUpper"],
  "tolowercase": ["toLower"], "lowercase": ["toLower"], "lower": ["toLower"], "substr": ["substring"],
  "parseint": ["toInt"], "keyset": ["keys"], "items": ["entries"], "findindex": ["indexOf"],
  "every": ["all"], "some": ["any"],
]

/// The candidate a misspelt name most likely meant, or "".
///
/// First, a name another language uses for the same thing, when its Veles name
/// is among the candidates. Then the nearest candidate by edits — an insertion,
/// a deletion, a substitution or two neighbours swapped — ignoring case; one
/// that differs only in case always wins. A candidate further than a third of
/// the name's length is a different word, not a typo, unless one name is a long
/// prefix of the other. Names of one or two characters get no suggestion.
/// Ties go to the candidate sharing the longest subsequence with the name, then
/// to the alphabetically first, so the message is the same on every run.
fun didYouMean(name: string, candidates: List<string>): string {
  if (codePoints(name).len() < 3) return ""
  val lower = name.toLower()
  val known = candidates.toSet()
  val foreign = foreignAlternative(lower.replace("_", ""), name, known)
  if (!foreign.isEmpty()) return foreign
  val best = nearest(name, lower, known)
  if (!best.isEmpty()) return best
  // a typo of another language's name: `lenght` is `length`, so `len`
  val misspeltForeign = nearestPlain(lower.replace("_", ""), FOREIGN_NAMES.keys())
  foreignAlternative(misspeltForeign, name, known)
}

/// The Veles name for what another language calls `foreign`, when there is one
/// among `known`.
fun foreignAlternative(foreign: string, name: string, known: Set<string>): string {
  loop (alternative in FOREIGN_NAMES.get(foreign) ?: []) {
    if (known.contains(alternative) && alternative != name) return alternative
  }
  ""
}

/// The closest of `known` to `name` within a third of its length, or "".
fun nearest(name: string, lower: string, known: Set<string>): string {
  val limit = codePoints(name).len() / 3
  var best = ""
  var bestDistance = limit + 1
  var bestCommon = -1
  loop (candidate in known.toList().sorted()) {
    if (candidate == name || candidate.isEmpty() || !isPlainName(candidate)) continue
    val lowerCandidate = candidate.toLower()
    var distance = editDistance(lower, lowerCandidate, bestDistance + 1)
    if (lowerCandidate == lower) {
      distance = 0
    } else if (distance > limit && longPrefix(lower, lowerCandidate)) {
      distance = limit
    }
    if (distance > bestDistance || distance > limit) continue
    val common = commonSubsequence(lower, lowerCandidate)
    if (distance < bestDistance || common > bestCommon) {
      best = candidate
      bestDistance = distance
      bestCommon = common
    }
  }
  best
}

/// The nearest of `candidates` (all lowercase) within a third of the name's
/// length, without the foreign-name step.
fun nearestPlain(name: string, candidates: List<string>): string {
  var best = ""
  var bestDistance = codePoints(name).len() / 3 + 1
  loop (candidate in candidates.sorted()) {
    val distance = editDistance(name, candidate, bestDistance)
    if (distance < bestDistance) {
      best = candidate
      bestDistance = distance
    }
  }
  best
}

/// One name starts the other, and the shorter is most of it.
fun longPrefix(a: string, b: string): bool {
  val (short, long) = if (a.len() > b.len()) (b, a) else (a, b)
  short.len() >= 4 && 2 * short.len() >= long.len() && long.startsWith(short)
}

/// Excludes the checker's own names (`$testFail`, `__x`), which a program never
/// spells.
fun isPlainName(name: string): bool {
  val first = utf8.decode(name, 0) ?: return false
  isIDStart(first.code) && !name.startsWith("__") && name.indexOf("$") < 0
}

fun codePoints(text: string): List<i64> {
  val out: MutableList<i64> = []
  var at: i64 = 0
  loop (at < text.len()) {
    val rune = utf8.decode(text, at) ?: break
    out.push(rune.code)
    at += rune.size
  }
  out
}

/// The length of the longest common subsequence of two names.
fun commonSubsequence(a: string, b: string): i64 {
  val left = codePoints(a)
  val right = codePoints(b)
  var previous = Row.zeros(right.len() + 1)
  var current = Row.zeros(right.len() + 1)
  loop (i in 1..left.len()) {
    loop (j in 1..right.len()) {
      val value = if (left.at(i - 1) == right.at(j - 1)) previous.at(j - 1) + 1 else previous.at(j).max(current.at(j - 1))
      current.set(j, value)
    }
    (previous, current) = (current, previous)
  }
  previous.at(right.len())
}

/// The optimal-string-alignment distance between two names (Levenshtein plus
/// adjacent transpositions), or `stop` once it cannot come in under `stop`.
fun editDistance(a: string, b: string, stop: i64): i64 {
  val left = codePoints(a)
  val right = codePoints(b)
  val lengthGap = left.len() - right.len()
  if (lengthGap >= stop || -lengthGap >= stop) return stop
  var beforePrevious = Row.zeros(right.len() + 1)
  var previous = Row.zeros(right.len() + 1)
  var current = Row.zeros(right.len() + 1)
  loop (j in 0..right.len()) previous.set(j, j)
  loop (i in 1..left.len()) {
    current.set(0, i)
    var rowMinimum = i
    loop (j in 1..right.len()) {
      val cost = if (left.at(i - 1) == right.at(j - 1)) 0 else 1
      var best = (previous.at(j) + 1).min(current.at(j - 1) + 1).min(previous.at(j - 1) + cost)
      val swapped = i > 1 && j > 1 && left.at(i - 1) == right.at(j - 2) && left.at(i - 2) == right.at(j - 1)
      if (swapped) best = best.min(beforePrevious.at(j - 2) + 1)
      current.set(j, best)
      rowMinimum = rowMinimum.min(best)
    }
    if (rowMinimum >= stop) return stop
    (beforePrevious, previous, current) = (previous, current, beforePrevious)
  }
  previous.at(right.len())
}

/// One row of a dynamic-programming table, read with in-range indices only.
struct Row {
  cells: MutableList<i64>

  static fun zeros(width: i64): Row => Row(cells: MutableList<i64>.repeat(0, width))

  fun at(i: i64): i64 => this.cells.at(i) ?: panic("Row: the algorithms stay inside the row")

  fun set(i: i64, value: i64) {
    this.cells.set(i, value)
  }
}
