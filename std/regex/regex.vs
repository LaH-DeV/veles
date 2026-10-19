use utf8

/// Regular expressions (D182): RE2's syntax, matched in time proportional to
/// the pattern times the text — there is no input that makes a search blow up,
/// so a pattern may be matched against untrusted text.
///
/// ```veles
/// use regex
///
/// val logLine = try Regex.compile("(?<date>\\d{4}-\\d\\d-\\d\\d) (?<level>[A-Z]+) (?<msg>.*)")
/// loop (line in lines) {
///   val m = logLine.find(line) ?: continue
///   io.println("${m.group("level") ?: ""}: ${m.group("msg") ?: ""}")
/// }
/// val clean = try Regex.compile("\\s+").replaceAll(text, " ")
/// ```
///
/// A pattern is compiled once and used many times; a `Regex` is immutable and
/// can be shared by tasks. `Regex.compile` throws `RegexError` for a pattern it
/// cannot read; text built from user input goes through `Regex.escape` first.
///
/// ## Syntax
///
/// Characters match themselves. `.` any character but `\n`; `[abc]` `[a-z]`
/// `[^0-9]` classes, with `\d \w \s`, `[[:alpha:]]` and `\p{Greek}` inside; `\d
/// \D \w \W \s \S` are ASCII (`\w` is `[0-9A-Za-z_]`) unless the `u` flag is on;
/// `\p{L}`, `\pL`, `\P{Lu}`, `\p{^Nd}` a Unicode category or script by name;
/// `^ $` the ends of the text (of
/// each line with the `m` flag); `\b \B` word boundaries; `\A \z` the ends of
/// the text; `a|b`; `(…)` a group, `(?<name>…)` a named one, `(?:…)` one that
/// does not capture; `* + ? {n} {n,} {n,m}`, each made lazy by a `?` after it;
/// flags `(?i)`, `(?i:…)`, `(?-i)` with `i` (ignore case), `m`, `s` (`.`
/// matches `\n`), `x` (white space and `#` comments are ignored) and `u`
/// (Unicode: `\d \w \s \b` and the case folding of `i` follow Unicode, not
/// ASCII). Escapes: `\n \t \r \f \v \a \xHH \x{HHHH}` and a backslash before
/// punctuation.
///
/// Backreferences (`\1`, `\k<name>`) and lookaround (`(?=…)`, `(?!…)`,
/// `(?<=…)`, `(?<!…)`) cannot be matched in bounded time. A pattern using
/// them is an error that says so, unless it is compiled with
/// `Regex.compile(pattern, backtracking: true)`: then it runs on an exhaustive
/// engine that can take exponential time on a hostile pattern or text, so it is
/// for patterns you wrote and text you trust. A `/…/` literal never has it.

/// A compiled pattern.
public struct Regex {
  pattern: string
  program: Program
  names:   Map<string, i64>

  /// Compiles `pattern`. The flags are the ones a `(?…)` group would set for
  /// the whole pattern.
  public static fun compile(pattern: string, ignoreCase: bool = false, multiLine: bool = false, dotAll: bool = false, extended: bool = false, unicode: bool = false, backtracking: bool = false): Regex throws RegexError {
    val flags = Flags(ignoreCase, multiLine, dotAll, extended, unicode, backtracking)
    val parsed = parse(pattern, flags)
    if (parsed.error != "") throw RegexError(message: parsed.error, position: parsed.errorAt)
    val program = build(parsed)
    if (program.error != "") throw RegexError(message: program.error, position: 0)
    Regex(pattern, program, names: parsed.names)
  }

  /// `text` with every character that means something in a pattern escaped, so
  /// it matches itself: `Regex.escape("a.b")` is `a\.b`.
  public static fun escape(text: string): string {
    val out = StringBuilder()
    loop (i in 0..<text.len()) {
      val b = text.byteAt(i)
      if (b < 128 && !isWordByte(b)) out.appendByte('\\')
      out.appendByte(b)
    }
    out.toString()
  }

  /// The pattern as it was written.
  public fun source(): string => this.pattern

  /// How many capturing groups the pattern has, not counting the whole match.
  public fun groupCount(): i64 => this.program.groups

  /// True when the pattern matches somewhere in `text`.
  public fun test(text: string): bool => Machine.of(this.program).search(text, 0, false, false) != null

  /// True when the pattern matches all of `text`.
  public fun matches(text: string): bool => Machine.of(this.program).search(text, 0, true, true) != null

  /// The leftmost match at or after byte offset `from`, or null.
  public fun find(text: string, from: i64 = 0): Match? {
    if (from < 0 || from > text.len()) return null
    val slots = Machine.of(this.program).search(text, from, false, false) ?: return null
    Match(text, slots, names: this.names)
  }

  /// Every match, left to right and not overlapping; an empty match right
  /// after the previous match is skipped.
  public fun findAll(text: string): List<Match> {
    val out: MutableList<Match> = []
    val machine = Machine.of(this.program)
    var from = 0
    var lastEnd = -1
    loop (from <= text.len()) {
      val slots = machine.search(text, from, false, false) ?: break
      val m = Match(text, slots, names: this.names)
      val start = m.start()
      val end = m.end()
      if (end == start) {
        // step past one code point so the search moves on
        from = end + (utf8.decode(text, end)?.size ?: 1)
        if (start == lastEnd) continue
      } else {
        from = end
      }
      lastEnd = end
      out.push(m)
    }
    out.toList()
  }

  /// `text` with each match replaced by `replacement`. In `replacement`, `$1` or `${1}` is a
  /// group, `${name}` a named group, `$0` the whole match and `$$` a dollar;
  /// a group that did not take part is empty.
  public fun replaceAll(text: string, replacement: string): string =>
    this.replaceWith(text, m => expand(m, replacement), -1)

  /// `text` with the first match replaced by `replacement` (see `replaceAll`).
  public fun replaceFirst(text: string, replacement: string): string =>
    this.replaceWith(text, m => expand(m, replacement), 1)

  /// `text` with each match replaced by what `f` makes of it.
  public fun replaceEach(text: string, f: fun(Match): string): string => this.replaceWith(text, f, -1)

  fun replaceWith(text: string, f: fun(Match): string, limit: i64): string {
    val out = StringBuilder()
    var copied = 0
    var count = 0
    loop (m in this.findAll(text)) {
      if (limit >= 0 && count >= limit) break
      out.append(text.substring(copied, m.start()) ?: "")
      out.append(f(m))
      copied = m.end()
      count += 1
    }
    out.append(text.substring(copied, text.len()) ?: "")
    out.toString()
  }

  /// The pieces of `text` between matches. At most `limit` pieces when it is
  /// not negative, the last holding the rest.
  public fun split(text: string, limit: i64 = -1): List<string> {
    val out: MutableList<string> = []
    var copied = 0
    loop (m in this.findAll(text)) {
      if (limit >= 0 && out.len() + 1 >= limit) break
      // an empty match at the very start or end splits nothing off
      if (m.end() == 0 || m.start() == text.len()) continue
      out.push(text.substring(copied, m.start()) ?: "")
      copied = m.end()
    }
    out.push(text.substring(copied, text.len()) ?: "")
    out.toList()
  }
}

/// A match: where it is in the text and what each group took.
public struct Match {
  text:  string
  slots: List<i64>
  names: Map<string, i64>

  /// The matched text.
  public fun value(): string => this.groupAt(0) ?: ""

  /// Byte offset of the first character of the match.
  public fun start(): i64 => this.slots.at(0) ?: 0

  /// Byte offset just past the match.
  public fun end(): i64 => this.slots.at(1) ?: 0

  /// How many groups the pattern has, not counting the whole match.
  public fun groupCount(): i64 => this.slots.len() / 2 - 1

  /// Group `index` (0 is the whole match), or null when it is out of range or
  /// did not take part in the match.
  public fun groupAt(index: i64): string? {
    if (index < 0 || index >= this.slots.len() / 2) return null
    val from = this.slots.at(2 * index) ?: -1
    val to = this.slots.at(2 * index + 1) ?: -1
    if (from < 0 || to < 0) return null
    this.text.substring(from, to)
  }

  /// The group named `name`, or null when there is none or it did not take
  /// part in the match.
  public fun group(name: string): string? {
    val index = this.names.get(name) ?: return null
    this.groupAt(index)
  }

  /// Where group `index` starts and ends, as byte offsets, or null.
  public fun span(index: i64): (i64, i64)? {
    if (index < 0 || index >= this.slots.len() / 2) return null
    val from = this.slots.at(2 * index) ?: -1
    val to = this.slots.at(2 * index + 1) ?: -1
    if (from < 0 || to < 0) return null
    (from, to)
  }
}

/// `template` with its `$` references filled in from `m`.
fun expand(m: Match, template: string): string {
  val out = StringBuilder()
  var i = 0
  loop (i < template.len()) {
    val b = template.byteAt(i)
    if (b != '$' || i + 1 >= template.len()) {
      out.appendByte(b)
      i += 1
      continue
    }
    val next = template.byteAt(i + 1)
    if (next == '$') {
      out.appendByte('$')
      i += 2
    } else if (next == '{') {
      val close = template.indexOf("}", i + 2)
      if (close < 0) {
        out.appendByte(b)
        i += 1
      } else {
        val name = template.substring(i + 2, close) ?: ""
        out.append(referenced(m, name))
        i = close + 1
      }
    } else if (next >= '0' && next <= '9') {
      var j = i + 1
      loop (j < template.len() && template.byteAt(j) >= '0' && template.byteAt(j) <= '9') j += 1
      out.append(referenced(m, template.substring(i + 1, j) ?: ""))
      i = j
    } else {
      out.appendByte(b)
      i += 1
    }
  }
  out.toString()
}

fun referenced(m: Match, name: string): string {
  val index = name.toInt()
  if (index != null) return m.groupAt(index) ?: ""
  m.group(name) ?: ""
}

// String extensions (D173, D182): the same operations read from the text's
// side. An inherent `replace`, `split` or `contains` wins over an extension of
// the same name and there is no overloading, so these have names of their own.
// They are visible where a file says `use regex`.

/// True when `re` matches all of this text: `"2026".matches(Regex.compile("\d+"))`.
public extend string with fun matches(re: Regex): bool => re.matches(this)

/// The first match of `re` in this text, or null.
public extend string with fun find(re: Regex): Match? => re.find(this)

/// Every match of `re` in this text.
public extend string with fun findAll(re: Regex): List<Match> => re.findAll(this)

/// This text with each match of `re` replaced (see `Regex.replaceAll`).
public extend string with fun replaceAll(re: Regex, replacement: string): string => re.replaceAll(this, replacement)

/// The pieces of this text between matches of `re`.
public extend string with fun splitBy(re: Regex): List<string> => re.split(this)

/// Why the compiler refuses the literal `/pattern/flags`, or "" when it is
/// fine. The checker runs this at compile time on every regex literal (D182):
/// the same parser and compiler `Regex.compile` uses, so a literal that passes
/// here cannot fail when the program runs.
public const fun validate(pattern: string, flags: string): string {
  val parsed = parse(pattern, flagsOf(flags))
  if (parsed.error != "") return parsed.error
  build(parsed).error
}

const fun flagsOf(flags: string): Flags =>
  Flags(ignoreCase: flags.indexOf("i") >= 0, multiLine: flags.indexOf("m") >= 0, dotAll: flags.indexOf("s") >= 0, extended: flags.indexOf("x") >= 0, unicode: flags.indexOf("u") >= 0)

/// Literals already compiled, by flags and pattern: `/…/` in a loop compiles
/// once, not on every pass.
struct Literals {
  entries: MutableMap<string, Regex> = [:]
}

val literals = Mutex(value: Literals())

/// The value of the literal `/pattern/flags`. The compiler calls this for a
/// regex literal, after `validate` accepted it; write `Regex.compile` for a
/// pattern that is built at run time.
public fun literal(pattern: string, flags: string): Regex {
  val key = flags + "/" + pattern
  val known = literals.withLock(l => l.entries.get(key))
  if (known != null) return known
  val made = Regex.compile(pattern, ignoreCase: flags.indexOf("i") >= 0, multiLine: flags.indexOf("m") >= 0, dotAll: flags.indexOf("s") >= 0, extended: flags.indexOf("x") >= 0, unicode: flags.indexOf("u") >= 0) catch (e) {
    panic("regex literal /$pattern/$flags: ${e.message}")
  }
  literals.withLock(l => l.entries.set(key, made))
  made
}
