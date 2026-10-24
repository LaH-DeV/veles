// Prelude — how a name is spelled in text (D205): an enum member read from
// text or written to it, and a field's key in a document (D58).

/// The spelling of a name in text. A name is split into words at a lower
/// case letter followed by a capital, before the last capital of a run of
/// capitals (`HTTPServer` is `http`, `server`), and at `-`, `_` and spaces.
///
/// ```veles
/// Level.High.toString(case: Case.Kebab)        // "high"
/// Os.parse("mac-os", case: Case.Any)            // Os.MacOS
/// ```
public enum Case {
  /// As declared: `MacOS`, `passwordHash`.
  AsWritten
  /// Any of the spellings, when reading; as declared, when writing.
  Any
  /// `macos`: lower case, nothing between the words.
  Lower
  /// `MACOS`: upper case, nothing between the words.
  Upper
  /// `macOs`: the first word lower case, the others capitalised.
  Camel
  /// `MacOs`: every word capitalised.
  Pascal
  /// `mac_os`.
  Snake
  /// `mac-os`.
  Kebab
  /// `MAC_OS`.
  ScreamingSnake
}

/// What every enum is, with nothing written (D205): its members as a list,
/// read from text and written to it in a `Case`. An enum may write
/// `implement Enumerated for E { … }` to give `parse` or `toString(case:)`
/// a meaning of its own; `values` is always the members in order.
public trait Enumerated {
  /// Every member, in declaration order.
  static fun values(): List<This>
  /// The member `text` names in `case` — exactly as declared by default,
  /// in any spelling with `Case.Any` — or `null` when none does.
  static fun parse(text: string, case: Case = Case.AsWritten): This?
  /// The member's name spelled in `case`.
  fun toString(case: Case): string
}

// the words of a name, lower case
const fun caseWords(name: string): List<string> {
  val words: MutableList<string> = []
  var start: i64 = -1  // where the word being read began; -1 between words
  loop (i in 0..<name.len()) {
    val b = name.byteAt(i)
    if (b == '-' || b == '_' || b == ' ') {
      if (start >= 0) words.push(name.slice(start, i).toLower())
      start = -1
      continue
    }
    if (start >= 0 && b.isAsciiUpper()) {
      val before = name.byteAt(i - 1)
      val after: u8 = if (i + 1 < name.len()) name.byteAt(i + 1) else 0
      // `aB`: a new word; `ABc`: the last capital of a run starts one
      if (before.isAsciiLower() || before.isAsciiDigit() || before.isAsciiUpper() && after.isAsciiLower()) {
        words.push(name.slice(start, i).toLower())
        start = i
      }
    }
    if (start < 0) start = i
  }
  if (start >= 0) words.push(name.slice(start, name.len()).toLower())
  words
}

// `word` with its first letter a capital
const fun capitalised(word: string): string {
  if (word.isEmpty()) return word
  val out = StringBuilder()
  out.appendByte(word.byteAt(0).toAsciiUpper())
  out.append(word.slice(1, word.len()))
  out.toString()
}

// `name` spelled in `case`
const fun caseSpelled(name: string, case: Case): string {
  val words = caseWords(name)
  when (case) {
    Case.AsWritten, Case.Any => name
    Case.Lower               => words.join("")
    Case.Upper               => words.join("").toUpper()
    Case.Snake               => words.join("_")
    Case.Kebab               => words.join("-")
    Case.ScreamingSnake      => words.join("_").toUpper()
    Case.Pascal              => words.map(w => capitalised(w)).join("")
    Case.Camel               => {
      val out = StringBuilder()
      loop ((i, w) in words.enumerate()) out.append(if (i == 0) w else capitalised(w))
      out.toString()
    }
  }
}

// the form two spellings share when they name the same thing under Case.Any
const fun caseFolded(name: string): string => caseWords(name).join("")

// the index of the name `text` spells in `case`, or null
const fun caseIndex(names: List<string>, text: string, case: Case): i64? {
  loop ((i, name) in names.enumerate()) {
    val hit = when (case) {
      Case.AsWritten => text == name
      Case.Any       => caseFolded(text) == caseFolded(name)
      else           => text == caseSpelled(name, case)
    }
    if (hit) return i
  }
  null
}

// the member of an enum `text` names in `case` (E.parse with a case, D205)
const fun enumPick<E>(values: List<E>, names: List<string>, text: string, case: Case): E? {
  val i = caseIndex(names, text, case) ?: return null
  values.at(i)
}
