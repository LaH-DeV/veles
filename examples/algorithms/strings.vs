// String algorithms. Strings are UTF-8 (D24): `chars()` gives one-code-point
// strings for character work, `byteAt(i)` a u8 for ASCII-level work.

/// Reverses by code points, so multi-byte characters stay intact.
pub fun reverse(s: string): string = s.chars().reversed().join("")

pub fun isPalindrome(s: string): bool {
  val cs = s.toLower().chars().filter(c => c != " ")
  var i = 0
  var j = cs.len() - 1
  loop (i < j) {
    if (cs.atOrPanic(i) != cs.atOrPanic(j)) return false
    i += 1
    j -= 1
  }
  true
}

/// Two words are anagrams when their sorted characters agree.
pub fun isAnagram(a: string, b: string): bool =
  a.toLower().chars().sorted().join("") == b.toLower().chars().sorted().join("")

/// Word frequencies, insertion-ordered (D25: maps keep insertion order).
pub fun wordCounts(text: string): Map<string, i64> {
  var counts: MutableMap<string, i64> = [:]
  loop (w in text.toLower().split(" ")) {
    if (w == "") continue
    counts.set(w, (counts.get(w) ?: 0) + 1)
  }
  counts.toMap()
}

/// Run-length encoding: "aaabcc" -> "3a1b2c".
pub fun runLengthEncode(s: string): string {
  val cs = s.chars()
  if (cs.len() == 0) return ""
  val sb = stringBuilder()
  var current = cs.atOrPanic(0)
  var run = 1
  loop (c in cs.drop(1)) {
    if (c == current) {
      run += 1
    } else {
      sb.append("$run$current")
      current = c
      run = 1
    }
  }
  sb.append("$run$current")
  sb.toString()
}

/// Caesar cipher over ASCII letters; other bytes pass through.
pub fun caesar(s: string, shift: i64): string {
  val sb = stringBuilder()
  val k = ((shift % 26) + 26) % 26
  loop (i in 0..<s.len()) {
    val b = s.byteAt(i)
    val shifted = when {
      b >= 'a' && b <= 'z' => 'a' + ((b - 'a' + k as u8) % 26)
      b >= 'A' && b <= 'Z' => 'A' + ((b - 'A' + k as u8) % 26)
      else                 => b
    }
    sb.appendByte(shifted)
  }
  sb.toString()
}

/// Naive substring search: every start position, O(n·m). Returns the byte
/// index or null.
pub fun findNaive(haystack: string, needle: string): i64? {
  val n = haystack.len()
  val m = needle.len()
  if (m == 0) return 0
  loop (i in 0..n - m) {
    var j = 0
    loop (j < m && haystack.byteAt(i + j) == needle.byteAt(j)) j += 1
    if (j == m) return i
  }
  null
}

/// Knuth–Morris–Pratt: O(n + m) with a failure table over the needle.
pub fun findKmp(haystack: string, needle: string): i64? {
  val m = needle.len()
  if (m == 0) return 0
  // fail[i] = length of the longest proper prefix of needle[0..i] that is
  // also a suffix of it
  var fail: MutableList<i64> = []
  loop (_ in 0..<m) fail.push(0)
  var k = 0
  loop (i in 1..<m) {
    loop (k > 0 && needle.byteAt(i) != needle.byteAt(k)) k = fail.atOrPanic(k - 1)
    if (needle.byteAt(i) == needle.byteAt(k)) k += 1
    fail.set(i, k)
  }
  var q = 0
  loop (i in 0..<haystack.len()) {
    loop (q > 0 && haystack.byteAt(i) != needle.byteAt(q)) q = fail.atOrPanic(q - 1)
    if (haystack.byteAt(i) == needle.byteAt(q)) q += 1
    if (q == m) return i - m + 1
  }
  null
}

/// Longest common prefix of a list of words.
pub fun commonPrefix(words: List<string>): string {
  val first = words.first() ?: return ""
  var end = first.len()
  loop (w in words.drop(1)) {
    var i = 0
    loop (i < end && i < w.len() && w.byteAt(i) == first.byteAt(i)) i += 1
    end = i
  }
  first.substring(0, end) ?: ""
}
