// String algorithms. Strings are UTF-8 (D24): `chars()` gives one-code-point
// strings for character work, `bytes()` / `byteAt(i)` a u8 for ASCII-level
// work.

/// Reverses by code points, so multi-byte characters stay intact.
public fun reverse(s: string): string => s.chars().reversed().join("")

/// Two pointers closing in from both ends, ignoring case and spaces.
public fun isPalindrome(s: string): bool {
  val chars = s.toLower().chars().filter(c => c != " ")
  var i = 0
  var j = chars.len() - 1
  loop (i < j) {
    val front = chars.at(i) ?: panic("isPalindrome: 0 <= i < j < len")
    val back = chars.at(j) ?: panic("isPalindrome: 0 <= i < j < len")
    if (front != back) return false
    i += 1
    j -= 1
  }
  true
}

/// Two words are anagrams when their sorted characters agree.
public fun isAnagram(a: string, b: string): bool =>
  a.toLower().chars().sorted() == b.toLower().chars().sorted()

/// Word frequencies, insertion-ordered (D25: maps keep insertion order).
public fun wordCounts(text: string): Map<string, i64> {
  val counts: MutableMap<string, i64> = [:]
  loop (word in text.toLower().split(" ")) {
    if (word == "") continue
    counts.set(word, counts.getOrDefault(word, 0) + 1)
  }
  counts.toMap()
}

/// Run-length encoding: "aaabcc" -> "3a1b2c".
public fun runLengthEncode(s: string): string {
  val chars = s.chars()
  val out = StringBuilder()
  var current = chars.first() ?: return ""
  var run = 1
  loop (c in chars.drop(1)) {
    if (c == current) {
      run += 1
    } else {
      out.append("$run$current")
      current = c
      run = 1
    }
  }
  out.append("$run$current")
  out.toString()
}

/// Caesar cipher over ASCII letters; other bytes pass through.
public fun caesar(s: string, shift: i64): string {
  val out = StringBuilder()
  val by = (((shift % 26) + 26) % 26).wrapU8()
  loop (byte in s.bytes()) {
    val shifted = when {
      byte >= 'a' && byte <= 'z' => 'a' + (byte - 'a' + by) % 26
      byte >= 'A' && byte <= 'Z' => 'A' + (byte - 'A' + by) % 26
      else                       => byte
    }
    out.appendByte(shifted)
  }
  out.toString()
}

/// Naive substring search: every start position, O(n·m). Returns the byte
/// index or null.
public fun findNaive(haystack: string, needle: string): i64? {
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
public fun findKmp(haystack: string, needle: string): i64? {
  val m = needle.len()
  if (m == 0) return 0
  // fail[i] = length of the longest proper prefix of needle[0..i] that is
  // also a suffix of it
  val fail = MutableList<i64>.repeat(0, m)
  var matched = 0
  loop (i in 1..<m) {
    loop (matched > 0 && needle.byteAt(i) != needle.byteAt(matched)) matched = fail.at(matched - 1) ?: panic("findKmp: 0 < matched < m")
    if (needle.byteAt(i) == needle.byteAt(matched)) matched += 1
    fail.set(i, matched)
  }
  // the same walk over the haystack, falling back through the table on a
  // mismatch instead of restarting
  matched = 0
  loop (i in 0..<haystack.len()) {
    loop (matched > 0 && haystack.byteAt(i) != needle.byteAt(matched)) matched = fail.at(matched - 1) ?: panic("findKmp: 0 < matched < m")
    if (haystack.byteAt(i) == needle.byteAt(matched)) matched += 1
    if (matched == m) return i - m + 1
  }
  null
}

/// Longest common prefix of a list of words.
public fun commonPrefix(words: List<string>): string {
  val first = words.first() ?: return ""
  var end = first.len()
  loop (word in words.drop(1)) {
    var i = 0
    loop (i < end && i < word.len() && word.byteAt(i) == first.byteAt(i)) i += 1
    end = i
  }
  first.substring(0, end) ?: ""
}
