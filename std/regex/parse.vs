// The pattern parser (D182): text to a tree of `Node`s. The syntax is RE2's —
// classes, `\d \w \s \b`, named groups, lazy quantifiers, flags — and what the
// linear-time engine cannot do (backreferences, lookaround) is an error that
// says so. Everything is code points, decoded once, so a multi-byte character
// in a pattern is one atom.
//
// Every function here is a `const fun`, because the compiler runs this parser
// on each `/…/` literal it reads (D182), and a `const fun` cannot throw. So a
// problem is recorded — the first one wins, in `error` and `errorAt` — and the
// parser reads on as if the pattern ended there (`peek` answers -1 once it has
// failed); `Regex.compile` turns the record into a `RegexError`.
use utf8

/// A pattern that cannot be compiled. `position` is the code point index in
/// the pattern where the problem was found.
public error RegexError {
  public message:  string
  public position: i64
}

sealed trait Node {
  /// Matches the empty string.
  struct Nothing { }
  /// One code point.
  struct Lit {
    code: i64
  }
  /// Any code point; `\n` only when `dotAll`.
  struct Dot {
    dotAll: bool
  }
  /// A set of code points: sorted, merged `lo, hi, lo, hi, …` pairs, both ends in.
  struct Ranges {
    ranges: List<i64>
  }
  struct Cat {
    items: List<Node>
  }
  struct Choice {
    items: List<Node>
  }
  /// `body{min,max}`; `max` is -1 for no upper bound.
  struct Repeat {
    body: *Node
    min:  i64
    max:  i64
    lazy: bool
  }
  /// A capturing group; `index` counts from 1 (0 is the whole match).
  struct Group {
    index: i64
    body:  *Node
  }
  /// A zero-width test: one of the `assert…` constants.
  struct Anchor {
    kind: i64
  }
  /// `\1` or `\k<name>`: the text group `index` matched. Only with
  /// `backtracking: true`; `fold` is whether `(?i)` was on where it was written.
  struct BackRef {
    index: i64
    fold:  bool
  }
  /// `(?=…)`, `(?!…)`, `(?<=…)` and `(?<!…)`: `body` must (or must not) match
  /// ahead of here, or end here for `behind`. Only with `backtracking: true`.
  struct Look {
    behind:  bool
    negated: bool
    body:    *Node
  }
}

const assertBeginLine: i64 = 0
const assertEndLine: i64 = 1
const assertBeginText: i64 = 2
const assertEndText: i64 = 3
const assertWordBoundary: i64 = 4
const assertNotWordBoundary: i64 = 5
/// The same two tests with Unicode word characters (the `u` flag).
const assertWordBoundaryUnicode: i64 = 6
const assertNotWordBoundaryUnicode: i64 = 7

const maxCode: i64 = 0x10FFFF
const maxRepeat: i64 = 1000
const maxNesting: i64 = 200

const lowerA: i64 = 97
const lowerZ: i64 = 122
const upperA: i64 = 65
const upperZ: i64 = 90
const caseGap: i64 = 32

/// What the flags of a pattern say; `(?i)` and friends change them inside it.
struct Flags {
  var ignoreCase: bool = false
  var multiLine:  bool = false
  var dotAll:     bool = false
  var extended:   bool = false
  /// `u`: `\d \w \s \b` and case folding follow Unicode, not ASCII.
  var unicode: bool = false
  /// Allows backreferences and lookaround (the exhaustive engine); it is not
  /// a `(?…)` flag, and a literal never has it.
  var backtracking: bool = false
}

/// A parsed pattern, or the first problem found in it (`error` is empty when
/// there is none). `exact` is whether the tree needs the exhaustive engine
/// (it holds a backreference or lookaround).
struct Parsed {
  root:    Node
  groups:  i64
  names:   Map<string, i64>
  exact:   bool
  error:   string
  errorAt: i64
}

const fun parse(pattern: string, flags: Flags): Parsed {
  val codes: MutableList<i64> = []
  var at = 0
  loop (at < pattern.len()) {
    val r = utf8.decode(pattern, at) ?: panic("regex: a string is valid UTF-8")
    codes.push(r.code)
    at += r.size
  }
  val parser = Parser(codes, flags)
  val root = parser.alternation(0)
  if (parser.pos < parser.codes.len()) {
    // only a stray `)` can stop the top level before the end
    parser.fail("unmatched ')'")
  }
  if (parser.maxBackRef > parser.groups && !parser.failed()) {
    parser.pos = parser.backRefAt
    parser.fail("\\${parser.maxBackRef} refers to a group the pattern does not have")
  }
  Parsed(root, groups: parser.groups, names: parser.names.toMap(), exact: parser.exact, error: parser.error, errorAt: parser.errorAt)
}

/// A code point as the ASCII byte it is, or 255 for one that is not ASCII (or
/// the end of the pattern): what lets `when` and comparisons use byte literals.
const fun sym(c: i64): u8 => if (c in 0..<128) c.wrapU8() else 255

const fun isDigit(c: i64): bool {
  val b = sym(c)
  b in '0'..'9'
}

const fun isLetter(c: i64): bool {
  val b = sym(c)
  b in 'a'..'z' || b in 'A'..'Z'
}

const fun hexValue(c: i64): i64 {
  val b = sym(c)
  if (b in '0'..'9') return c - 48
  if (b in 'a'..'f') return c - 97 + 10
  if (b in 'A'..'F') return c - 65 + 10
  -1
}

struct Parser {
  codes:       List<i64>
  var flags:   Flags
  var pos:     i64 = 0
  var groups:  i64 = 0
  var error:   string = ""
  var errorAt: i64 = 0
  /// A backreference or lookaround was read (see `Parsed.exact`).
  var exact:      bool = false
  var maxBackRef: i64 = 0
  var backRefAt:  i64 = 0
  names:          MutableMap<string, i64> = [:]

  /// Records the first problem; later ones are consequences of it.
  const fun fail(message: string) {
    if (this.error == "") {
      this.error = message
      this.errorAt = this.pos
    }
  }

  const fun failed(): bool => this.error != ""

  /// The code point here, or -1 at the end — and once a problem is recorded,
  /// so every loop that reads ahead stops.
  const fun peek(): i64 => if (this.failed()) -1 else this.codes.at(this.pos) ?: -1

  const fun peekAt(ahead: i64): i64 => if (this.failed()) -1 else this.codes.at(this.pos + ahead) ?: -1

  const fun look(ch: u8): bool => sym(this.peek()) == ch

  const fun eat(ch: u8): bool {
    if (this.look(ch)) {
      this.pos += 1
      return true
    }
    false
  }

  /// In extended mode, white space and `#` comments between atoms are skipped.
  const fun skipSpace() {
    if (!this.flags.extended) return
    loop (this.pos < this.codes.len() && !this.failed()) {
      val b = sym(this.peek())
      if (b in [' ', '\t', '\n', '\r', 12]) {
        this.pos += 1
      } else if (b == '#') {
        loop (this.pos < this.codes.len() && !this.look('\n')) this.pos += 1
      } else {
        break
      }
    }
  }

  const fun alternation(depth: i64): Node {
    if (depth > maxNesting) {
      this.fail("groups nested more than $maxNesting deep")
      return Nothing()
    }
    val saved = this.flags
    val items: MutableList<Node> = [this.sequence(depth)]
    loop (this.eat('|')) items.push(this.sequence(depth))
    // `(?i)` ends with its group
    this.flags = saved
    if (items.len() == 1) items.at(0) else Choice(items)
  }

  const fun sequence(depth: i64): Node {
    val items: MutableList<Node> = []
    loop {
      this.skipSpace()
      val b = sym(this.peek())
      if (this.peek() == -1 || b in ['|', ')']) break
      val atom = this.atom(depth) ?: continue
      items.push(this.quantified(atom))
    }
    if (items.len() == 0) return Nothing()
    if (items.len() == 1) return items.at(0)
    Cat(items)
  }

  /// A quantifier after an atom; a second one in a row is an error.
  const fun quantified(atom: Node): Node {
    this.skipSpace()
    var min = 0
    var max = -1
    val b = sym(this.peek())
    when {
      b == '*' => {
        this.pos += 1
      }
      b == '+' => {
        this.pos += 1
        min = 1
      }
      b == '?' => {
        this.pos += 1
        max = 1
      }
      b == '{' => {
        val bounds = this.braces() ?: return atom
        min = bounds.0
        max = bounds.1
      }
      else     => {
        return atom
      }
    }
    val lazy = this.eat('?')
    this.skipSpace()
    val after = sym(this.peek())
    if (after in ['*', '+', '?']) {
      this.fail("a repetition cannot be repeated again: group it with (?:…)")
    }
    if (atom is Anchor) this.fail("an assertion cannot be repeated")
    Repeat(body: &atom, min, max, lazy)
  }

  /// `{n}`, `{n,}` or `{n,m}`; null (and nothing consumed) when the brace is
  /// just a brace.
  const fun braces(): (i64, i64)? {
    val start = this.pos
    this.pos += 1
    val first = this.number()
    if (first == null) {
      this.pos = start
      return null
    }
    val low = first
    var high = low
    if (this.eat(',')) {
      high = this.number() ?: -1
    }
    if (!this.eat('}')) {
      this.pos = start
      return null
    }
    if (low > maxRepeat || high > maxRepeat) {
      this.fail("repeat count over $maxRepeat")
    } else if (high != -1 && high < low) {
      this.fail("{$low,$high}: the lower bound is above the upper")
    }
    (low, high)
  }

  const fun number(): i64? {
    var n: i64? = null
    loop (isDigit(this.peek())) {
      val digit = this.peek() - 48
      n = ((n ?: 0) * 10 + digit).min(maxRepeat + 1)
      this.pos += 1
    }
    n
  }

  /// One atom, or null for a construct that only sets flags (`(?i)`).
  const fun atom(depth: i64): Node? {
    val c = this.peek()
    this.pos += 1
    when (sym(c)) {
      '('           => this.group(depth)
      '['           => this.bracket()
      '.'           => Dot(dotAll: this.flags.dotAll)
      '^'           => Anchor(kind: if (this.flags.multiLine) assertBeginLine else assertBeginText)
      '$'           => Anchor(kind: if (this.flags.multiLine) assertEndLine else assertEndText)
      '\\'          => this.escape()
      '*', '+', '?' => {
        this.pos -= 1
        this.fail("nothing to repeat before '${utf8String(c)}'")
        this.pos += 1
        Nothing()
      }
      else          => this.literal(c)
    }
  }

  const fun literal(c: i64): Node {
    if (this.flags.ignoreCase) {
      if (this.flags.unicode) {
        val orbit = foldUnicode([c, c])
        if (orbit != [c, c]) return Ranges(ranges: orbit)
      } else if (isLetter(c)) {
        return Ranges(ranges: foldRanges([c, c]))
      }
    }
    Lit(code: c)
  }

  /// `ranges` plus the other cases of what is in them (see `Flags.unicode`).
  const fun fold(ranges: List<i64>): List<i64> => if (this.flags.unicode) foldUnicode(ranges) else foldRanges(ranges)

  /// The sets `\d \w \s` mean under the flags in force.
  const fun digits(): List<i64> => if (this.flags.unicode) catNd else digitRanges()

  const fun words(): List<i64> => if (this.flags.unicode) unicodeWord else wordRanges()

  const fun spaces(): List<i64> => if (this.flags.unicode) unicodeSpace else spaceRanges()

  /// After `\p` or `\P` (`pos` at the character after it): `{Name}`, `{^Name}`
  /// or a one-letter name such as `\pL`. Null after a problem.
  const fun property(negated: bool): List<i64>? {
    var invert = negated
    val name = StringBuilder()
    if (this.eat('{')) {
      if (this.eat('^')) invert = !invert
      loop {
        val c = this.peek()
        if (this.eat('}')) break
        if (!(isLetter(c) || isDigit(c) || sym(c) == '_')) {
          this.fail("\\p{…} holds the name of a Unicode class or script, then '}'")
          return null
        }
        name.appendByte(sym(c))
        this.pos += 1
      }
    } else {
      val c = this.peek()
      if (!isLetter(c)) {
        this.fail("\\p needs a class name: \\pL or \\p{Greek}")
        return null
      }
      name.appendByte(sym(c))
      this.pos += 1
    }
    val text = name.toString()
    val set: List<i64>? = if (text == "Any") [0, maxCode] else unicodeProperty(text)
    if (set == null) {
      this.fail("unknown Unicode class or script '$text'")
      return null
    }
    if (invert) negate(set) else set
  }

  const fun group(depth: i64): Node? {
    if (!this.eat('?')) {
      this.groups += 1
      val index = this.groups
      val body = this.alternation(depth + 1)
      this.close()
      return Group(index, body: &body)
    }
    val b = sym(this.peek())
    if (b == ':') {
      this.pos += 1
      val body = this.alternation(depth + 1)
      this.close()
      return body
    }
    if (b in ['<', 'P']) {
      if (b == '<' && (sym(this.peekAt(1)) == '=' || sym(this.peekAt(1)) == '!')) {
        val negated = sym(this.peekAt(1)) == '!'
        if (!this.flags.backtracking) {
          this.pos += 1
          this.fail("lookbehind is not supported by the linear-time engine; Regex.compile(pattern, backtracking: true) allows it, and can take exponential time")
          return Nothing()
        }
        this.pos += 2
        return this.lookaround(depth, true, negated)
      }
      this.pos += 1
      if (b == 'P' && !this.eat('<')) {
        this.fail("expected (?P<name>…)")
        return Nothing()
      }
      val name = this.groupName()
      if (this.names.containsKey(name)) this.fail("two groups are named '$name'")
      this.groups += 1
      val index = this.groups
      this.names.set(name, index)
      val body = this.alternation(depth + 1)
      this.close()
      return Group(index, body: &body)
    }
    if (b in ['=', '!']) {
      if (!this.flags.backtracking) {
        this.fail("lookahead is not supported by the linear-time engine; Regex.compile(pattern, backtracking: true) allows it, and can take exponential time")
        return Nothing()
      }
      this.pos += 1
      return this.lookaround(depth, false, b == '!')
    }
    // flags: (?i) (?i-s) (?i:…)
    var on = true
    var next = this.flags
    loop {
      val f = this.peek()
      this.pos += 1
      when (sym(f)) {
        'i'  => next.ignoreCase = on
        'm'  => next.multiLine = on
        's'  => next.dotAll = on
        'x'  => next.extended = on
        'u'  => next.unicode = on
        '-'  => {
          if (!on) {
            this.fail("flags: a second '-'")
            return Nothing()
          }
          on = false
        }
        ')'  => {
          this.flags = next
          return null
        }
        ':'  => {
          val saved = this.flags
          this.flags = next
          val body = this.alternation(depth + 1)
          this.flags = saved
          this.close()
          return body
        }
        else => {
          this.pos -= 1
          this.fail("unknown group syntax or flag '${utf8String(f)}'")
          return Nothing()
        }
      }
    }
  }

  const fun close() {
    if (!this.eat(')')) this.fail("missing ')'")
  }

  /// The body of `(?=…)`, `(?!…)`, `(?<=…)` or `(?<!…)`, after its opening.
  const fun lookaround(depth: i64, behind: bool, negated: bool): Node {
    this.exact = true
    val body = this.alternation(depth + 1)
    this.close()
    Look(behind, negated, body: &body)
  }

  const fun groupName(): string {
    val out = StringBuilder()
    loop {
      val c = this.peek()
      if (this.eat('>')) break
      if (!(isLetter(c) || isDigit(c) || sym(c) == '_')) {
        this.fail("a group name is letters, digits and '_', then '>'")
        break
      }
      out.appendByte(sym(c))
      this.pos += 1
    }
    val name = out.toString()
    if (name.len() == 0) {
      this.fail("a group needs a name")
    } else if (isDigit(name.byteAt(0).toI64())) {
      this.fail("a group name cannot start with a digit")
    }
    name
  }

  /// After a backslash outside a class.
  const fun escape(): Node {
    val c = this.peek()
    if (c == -1) {
      this.fail("a pattern cannot end in a backslash")
      return Nothing()
    }
    when (sym(c)) {
      'd'      => this.classEscape(this.digits(), false)
      'D'      => this.classEscape(this.digits(), true)
      'w'      => this.classEscape(this.words(), false)
      'W'      => this.classEscape(this.words(), true)
      's'      => this.classEscape(this.spaces(), false)
      'S'      => this.classEscape(this.spaces(), true)
      'b'      => this.assertion(if (this.flags.unicode) assertWordBoundaryUnicode else assertWordBoundary)
      'B'      => this.assertion(if (this.flags.unicode) assertNotWordBoundaryUnicode else assertNotWordBoundary)
      'A'      => this.assertion(assertBeginText)
      'z'      => this.assertion(assertEndText)
      'p', 'P' => {
        this.pos += 1
        val set = this.property(sym(c) == 'P') ?: return Nothing()
        Ranges(ranges: set)
      }
      'k'      => this.namedBackRef()
      else     => {
        if (isDigit(c) && c != 48) return this.numberedBackRef()
        this.literal(this.escapedCode())
      }
    }
  }

  /// `\1`, `\2`, … (`pos` at the first digit).
  const fun numberedBackRef(): Node {
    if (!this.flags.backtracking) {
      this.fail("backreferences are not supported by the linear-time engine; Regex.compile(pattern, backtracking: true) allows them, and can take exponential time")
      return Nothing()
    }
    val at = this.pos - 1
    var index = 0
    loop (isDigit(this.peek()) && index < 100) {
      index = index * 10 + (this.peek() - 48)
      this.pos += 1
    }
    this.exact = true
    if (index > this.maxBackRef) {
      this.maxBackRef = index
      this.backRefAt = at
    }
    BackRef(index, fold: this.flags.ignoreCase)
  }

  /// `\k<name>` (`pos` at the `k`): the group of that name, which must come
  /// before it.
  const fun namedBackRef(): Node {
    if (!this.flags.backtracking) {
      this.fail("backreferences are not supported by the linear-time engine; Regex.compile(pattern, backtracking: true) allows them, and can take exponential time")
      return Nothing()
    }
    this.pos += 1
    if (!this.eat('<')) {
      this.fail("expected \\k<name>")
      return Nothing()
    }
    val name = this.groupName()
    val index = this.names.get(name)
    if (index == null) {
      this.fail("\\k<$name> names a group that has not been opened yet")
      return Nothing()
    }
    this.exact = true
    BackRef(index, fold: this.flags.ignoreCase)
  }

  const fun classEscape(set: List<i64>, negated: bool): Node {
    this.pos += 1
    Ranges(ranges: if (negated) negate(set) else set)
  }

  const fun assertion(kind: i64): Node {
    this.pos += 1
    Anchor(kind)
  }

  /// The code point an escape such as `\n`, `\x41`, `\x{1F600}` or `\.` names;
  /// `pos` is at the character after the backslash.
  const fun escapedCode(): i64 {
    val c = this.peek()
    this.pos += 1
    when (sym(c)) {
      'n'  => 10
      't'  => 9
      'r'  => 13
      'f'  => 12
      'v'  => 11
      'a'  => 7
      '0'  => 0
      'x'  => this.hexEscape()
      else => {
        if (isLetter(c) || isDigit(c)) {
          this.pos -= 1
          this.fail("unknown escape '\\${utf8String(c)}'")
          return 0
        }
        c
      }
    }
  }

  const fun hexEscape(): i64 {
    if (this.eat('{')) {
      var code = 0
      var digits = 0
      loop (!this.look('}')) {
        val v = hexValue(this.peek())
        if (v < 0) {
          this.fail("\\x{…} holds hexadecimal digits")
          return 0
        }
        code = code * 16 + v
        digits += 1
        if (code > maxCode) {
          this.fail("\\x{…} is above U+10FFFF")
          return 0
        }
        this.pos += 1
      }
      this.pos += 1
      if (digits == 0) this.fail("\\x{} needs digits")
      return code
    }
    val high = hexValue(this.peek())
    val low = hexValue(this.peekAt(1))
    if (high < 0 || low < 0) {
      this.fail("\\x needs two hexadecimal digits")
      return 0
    }
    this.pos += 2
    high * 16 + low
  }

  /// `[…]`, after the opening bracket.
  const fun bracket(): Node {
    val pairs: MutableList<i64> = []
    val negated = this.eat('^')
    var first = true
    loop {
      if (this.peek() == -1) {
        this.fail("missing ']'")
        break
      }
      if (this.look(']') && !first) {
        this.pos += 1
        break
      }
      first = false
      if (this.look('[') && sym(this.peekAt(1)) == ':') {
        if (this.posixClass(pairs)) continue
      }
      val lo = this.classAtom(pairs) ?: continue
      var hi = lo
      if (this.look('-') && sym(this.peekAt(1)) != ']' && this.peekAt(1) != -1) {
        this.pos += 1
        val end = this.classAtom(pairs)
        if (end == null) {
          this.fail("a range cannot end in a class such as \\d")
          break
        }
        hi = end
        if (hi < lo) {
          this.fail("the range ${utf8String(lo)}-${utf8String(hi)} runs backwards")
          break
        }
      }
      pairs.push(lo)
      pairs.push(hi)
    }
    var ranges = normalize(pairs.toList())
    if (this.flags.ignoreCase) ranges = this.fold(ranges)
    Ranges(ranges: if (negated) negate(ranges) else ranges)
  }

  /// One member of a class: its code point, or null after adding a class
  /// escape (`\d`) straight to `pairs`.
  const fun classAtom(pairs: MutableList<i64>): i64? {
    val c = this.peek()
    this.pos += 1
    if (sym(c) != '\\') return c
    val e = this.peek()
    if (e == -1) {
      this.fail("missing ']'")
      return null
    }
    if (sym(e) == 'p' || sym(e) == 'P') {
      this.pos += 1
      val named = this.property(sym(e) == 'P')
      if (named != null) {
        loop (x in named) pairs.push(x)
      }
      return null
    }
    val set: List<i64>? = when (sym(e)) {
      'd'  => this.digits()
      'D'  => negate(this.digits())
      'w'  => this.words()
      'W'  => negate(this.words())
      's'  => this.spaces()
      'S'  => negate(this.spaces())
      else => null
    }
    if (set != null) {
      this.pos += 1
      loop (x in set) pairs.push(x)
      return null
    }
    if (this.failed()) return null
    // `\b` in a class is a backspace, as in RE2
    if (sym(e) == 'b') {
      this.pos += 1
      return 8
    }
    this.escapedCode()
  }

  /// `[:alpha:]` and friends inside a class; false when it is not one.
  const fun posixClass(pairs: MutableList<i64>): bool {
    var end = this.pos + 2
    loop (isLetter(this.codes.at(end) ?: -1)) end += 1
    if (sym(this.codes.at(end) ?: -1) != ':' || sym(this.codes.at(end + 1) ?: -1) != ']') return false
    val name = StringBuilder()
    var i = this.pos + 2
    loop (i < end) {
      name.appendByte(sym(this.codes.at(i) ?: -1))
      i += 1
    }
    val set = posix(name.toString())
    if (set == null) {
      this.fail("unknown class [:${name.toString()}:]")
      return true
    }
    loop (x in set) pairs.push(x)
    this.pos = end + 2
    true
  }
}

const fun posix(name: string): List<i64>? => when (name) {
  "alpha"  => [65, 90, 97, 122]
  "digit"  => [48, 57]
  "alnum"  => [48, 57, 65, 90, 97, 122]
  "upper"  => [65, 90]
  "lower"  => [97, 122]
  "space"  => [9, 13, 32, 32]
  "blank"  => [9, 9, 32, 32]
  "punct"  => [33, 47, 58, 64, 91, 96, 123, 126]
  "xdigit" => [48, 57, 65, 70, 97, 102]
  "word"   => wordRanges()
  "print"  => [32, 126]
  "graph"  => [33, 126]
  "cntrl"  => [0, 31, 127, 127]
  "ascii"  => [0, 127]
  else     => null
}

const fun digitRanges(): List<i64> => [48, 57]

const fun wordRanges(): List<i64> => [48, 57, 65, 90, 95, 95, 97, 122]

const fun spaceRanges(): List<i64> => [9, 10, 12, 13, 32, 32]

/// Sorted, merged ranges from `lo, hi, …` pairs in any order.
const fun normalize(pairs: List<i64>): List<i64> {
  // insertion sort by the low end: pattern classes are short, and tuples do not
  // compare in the compiler's evaluator
  val lows: MutableList<i64> = []
  val highs: MutableList<i64> = []
  var i = 0
  loop (i + 1 < pairs.len()) {
    val lo = pairs.at(i) ?: 0
    var at = lows.len()
    loop (at > 0 && (lows.at(at - 1) ?: 0) > lo) at -= 1
    lows.insert(at, lo)
    highs.insert(at, pairs.at(i + 1) ?: 0)
    i += 2
  }
  val out: MutableList<i64> = []
  loop (k in 0..<lows.len()) {
    val lo = lows.at(k)
    val hi = highs.at(k) ?: 0
    val last = out.len() - 1
    if (last >= 1 && lo <= (out.at(last) ?: 0) + 1) {
      if (hi > (out.at(last) ?: 0)) out.set(last, hi)
    } else {
      out.push(lo)
      out.push(hi)
    }
  }
  out
}

/// Everything in 0..U+10FFFF that `ranges` leaves out.
const fun negate(ranges: List<i64>): List<i64> {
  val out: MutableList<i64> = []
  var next = 0
  var i = 0
  loop (i + 1 < ranges.len()) {
    val lo = ranges.at(i) ?: 0
    val hi = ranges.at(i + 1) ?: 0
    if (lo > next) {
      out.push(next)
      out.push(lo - 1)
    }
    next = hi + 1
    i += 2
  }
  if (next <= maxCode) {
    out.push(next)
    out.push(maxCode)
  }
  out
}

/// `ranges` plus the other case of every ASCII letter in them.
const fun foldRanges(ranges: List<i64>): List<i64> {
  val out: MutableList<i64> = []
  var i = 0
  loop (i + 1 < ranges.len()) {
    val lo = ranges.at(i) ?: 0
    val hi = ranges.at(i + 1) ?: 0
    out.push(lo)
    out.push(hi)
    // the lower-case letters in the range as upper case, and the reverse
    val lowLo = lo.max(lowerA)
    val lowHi = hi.min(lowerZ)
    if (lowLo <= lowHi) {
      out.push(lowLo - caseGap)
      out.push(lowHi - caseGap)
    }
    val upLo = lo.max(upperA)
    val upHi = hi.min(upperZ)
    if (upLo <= upHi) {
      out.push(upLo + caseGap)
      out.push(upHi + caseGap)
    }
    i += 2
  }
  normalize(out)
}

/// One code point as text, for messages.
const fun utf8String(code: i64): string {
  val out: MutableList<u8> = []
  utf8.encodeTo(out, code)
  out.decodeUtf8() ?: "?"
}

/// `ranges` plus every other case of the code points in them: Unicode simple
/// case folding, for `(?i)` under the `u` flag.
const fun foldUnicode(ranges: List<i64>): List<i64> {
  val out: MutableList<i64> = []
  loop (x in ranges) out.push(x)
  var i = 0
  loop (i + 1 < foldNext.len()) {
    val c = foldNext.at(i) ?: 0
    if (inSet(ranges, c)) {
      var member = foldNext.at(i + 1) ?: c
      loop (member != c) {
        out.push(member)
        out.push(member)
        member = foldOf(member)
      }
    }
    i += 2
  }
  normalize(out)
}

/// The next code point in the case orbit of `c` (`c` itself when it has no
/// other case).
const fun foldOf(c: i64): i64 {
  var lo = 0
  var hi = foldNext.len() / 2 - 1
  loop (lo <= hi) {
    val mid = (lo + hi) / 2
    val key = foldNext.at(2 * mid) ?: 0
    if (c < key) {
      hi = mid - 1
    } else if (c > key) {
      lo = mid + 1
    } else {
      return foldNext.at(2 * mid + 1) ?: c
    }
  }
  c
}
