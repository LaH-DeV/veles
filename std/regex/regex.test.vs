use utf8

test fun re(pattern: string): Regex => Regex.compile(pattern) catch (e) {
  panic("bad test pattern $pattern: ${e.message}")
}

/// The text a pattern finds first, or "-" when it finds nothing.
test fun first(pattern: string, text: string): string => re(pattern).find(text)?.value() ?: "-"

test fun why(pattern: string): string {
  val r = Regex.compile(pattern) catch (e) {
    return e.message
  }
  "compiled ${r.source()}"
}

test literals "characters match themselves" {
  expect(first("abc", "xxabcxx") == "abc")
  expect(first("abc", "ab") == "-")
  expect(first("", "abc") == "")
  expect(first("é", "caffé") == "é")
  expect(first("a.c", "abc") == "abc")
  expect(first("a\\.c", "abc") == "-")
  expect(first("a\\.c", "a.c") == "a.c")
}

test classes "sets, ranges and escapes" {
  expect(first("[abc]+", "xxcabxx") == "cab")
  expect(first("[^abc]+", "abxyzc") == "xyz")
  expect(first("[a-c0-2]+", "zz1b2z") == "1b2")
  expect(first("\\d+", "year 2026!") == "2026")
  expect(first("\\D+", "12ab34") == "ab")
  expect(first("\\w+", "  snake_case9 ") == "snake_case9")
  expect(first("\\s+", "a \t b") == " \t ")
  expect(first("[[:alpha:]]+", "12abC34") == "abC")
  expect(first("[\\d.]+", "v=3.14;") == "3.14")
  expect(first("[]a]+", "x]a]") == "]a]")
  expect(first("[a-]+", "x-a-") == "-a-")
}

test negatedPosix "negated POSIX classes compose with case folding and outer negation" {
  expect(first("[[:^alpha:]]+", "abc123") == "123")
  expect(first("[[:^digit:]]+", "123abc") == "abc")
  expect(first("[^[:^alpha:]]+", "123abc") == "abc")
  expect(first("(?i)[[:^lower:]]+", "aA123") == "123")
  expect(first("(?i)[[:^upper:]]+", "aA123") == "123")
  expect(first("(?i)[^[:^lower:]]+", "123aA") == "aA")
  expect(re("[[:^ascii:]]+").matches("\u{E9}"))
  expect(!re("(?iu)[[:^lower:]]").test("\u{212A}"))
  expect(literal("[[:^digit:]]+", "").matches("abc"))
  expect(why("[[:^unknown:]]").contains("unknown class"))
}

test dot "dot stops at a newline unless dotAll" {
  expect(first("a.c", "a\nc") == "-")
  expect(Regex.compile("a.c", dotAll: true) catch (e) {
    panic("")
  }.test("a\nc"))
  expect(first(".", "é") == "é")
}

test anchors "line and text anchors, word boundaries" {
  expect(first("^abc", "abcabc") == "abc")
  expect(first("^bc", "abc") == "-")
  expect(first("bc$", "abc") == "bc")
  expect(first("ab$", "abc") == "-")
  expect(first("\\bcat\\b", "concat cat") == "cat")
  expect(re("\\bcat\\b").find("concat cat")?.start() == 7)
  expect(first("\\Bcat", "concat") == "cat")
  expect(first("\\Aab", "xab") == "-")
  expect(first("ab\\z", "xab") == "ab")
  val lines = Regex.compile("^\\w+$", multiLine: true) catch (e) {
    panic("")
  }
  expect(lines.findAll("one\ntwo three\nfour").len() == 2)
}

test alternation "the first alternative that matches wins" {
  expect(first("cat|dog", "hotdog") == "dog")
  expect(first("a|ab", "ab") == "a")
  expect(first("ab|a", "ab") == "ab")
  expect(first("(?:a|b)+", "xabbax") == "abba")
}

test repetition "greedy, lazy and counted" {
  expect(first("a*", "aaa") == "aaa")
  expect(first("a+?", "aaa") == "a")
  expect(first("a*?", "aaa") == "")
  expect(first("a?b", "ab") == "ab")
  expect(first("a{2}", "aaaa") == "aa")
  expect(first("a{2,}", "aaaa") == "aaaa")
  expect(first("a{1,3}", "aaaa") == "aaa")
  expect(first("a{1,3}?", "aaaa") == "a")
  expect(first("a{,3}", "a{,3}") == "a{,3}")
  expect(first("<.+>", "<a><b>") == "<a><b>")
  expect(first("<.+?>", "<a><b>") == "<a>")
  expect(first("(ab)*c", "ababc") == "ababc")
  expect(first("(a*)*b", "aaab") == "aaab")
}

test groups "numbered, named and non-capturing groups" {
  val m = re("(\\d+)-(\\d+)").find("tel 12-345 ok") ?: panic("no match")
  expect(m.value() == "12-345")
  expect(m.groupAt(0) == "12-345")
  expect(m.groupAt(1) == "12")
  expect(m.groupAt(2) == "345")
  expect(m.groupAt(3) == null)
  expect(m.groupCount() == 2)
  expect(m.span(2) == (7, 10))
  val named = re("(?<y>\\d{4})-(?<m>\\d\\d)").find("on 2026-10-19") ?: panic("no match")
  expect(named.group("y") == "2026")
  expect(named.group("m") == "10")
  expect(named.group("d") == null)
  val optional = re("(a)|(b)").find("b") ?: panic("no match")
  expect(optional.groupAt(1) == null)
  expect(optional.groupAt(2) == "b")
  expect(re("(?:ab)+(c)").find("ababc")?.groupAt(1) == "c")
  expect(re("(?P<x>a)").find("a")?.group("x") == "a")
}

test flags "case folding, extended mode, scoped flags" {
  expect(Regex.compile("hello", ignoreCase: true) catch (e) {
    panic("")
  }.test("HeLLo"))
  expect(first("(?i)hello", "HELLO") == "HELLO")
  expect(first("(?i:a)b", "Ab") == "Ab")
  expect(first("(?i:a)b", "AB") == "-")
  expect(first("a(?i)b", "aB") == "aB")
  expect(first("(?i)[a-c]+", "xABCx") == "ABC")
  expect(first("(?i)a(?-i)b", "AB") == "-")
  expect(first("(?x) a b # note\n c", "abc") == "abc")
}

test unicode "matching runs over code points" {
  expect(first("é+", "caféé!") == "éé")
  expect(first("[à-ÿ]+", "xéèy") == "éè")
  expect(first("日本", "これは日本語") == "日本")
  expect(first("\\x{65E5}", "日") == "日")
  expect(re(".").findAll("日本").len() == 2)
  expect(re("\\xe9").find("é") != null)
}

test whole "matches needs the whole text, test finds anywhere" {
  expect(re("a|ab").matches("ab"))
  expect(!re("a").matches("ab"))
  expect(re("\\d+").test("abc 12"))
  expect(!re("\\d+").matches("abc 12"))
  expect(re("").matches(""))
  expect("2026".matches(re("\\d+")))
}

test findAll "matches do not overlap and empty ones advance" {
  val digits = re("\\d+").findAll("a1b22c333")
  expect(digits.map(m => m.value()) == ["1", "22", "333"])
  expect(re("a*").findAll("baaac").map(m => m.value()) == ["", "aaa", ""])
  expect(re("").findAll("é").len() == 2)
  expect(re("x").findAll("abc").len() == 0)
}

test replace "references in the replacement" {
  expect(re("\\s+").replaceAll("a  b \t c", " ") == "a b c")
  expect(re("(\\w+)@(\\w+)").replaceAll("me@home", "$2 at $1") == "home at me")
  expect(re("(?<k>\\w+)=(?<v>\\w+)").replaceAll("a=1,b=2", `${v}:${k}`) == "1:a,2:b")
  expect(re("a").replaceAll("aaa", `$$`) == "$$$")
  expect(re("a").replaceFirst("aaa", "b") == "baa")
  expect(re("o").replaceAll("foo", "[$0]") == "f[o][o]")
  expect(re("\\d").replaceEach("a1b2", m => "<${m.value()}>") == "a<1>b<2>")
  expect(re("x*").replaceAll("abc", "-") == "-a-b-c-")
}

test split "pieces between matches" {
  expect(re(",\\s*").split("a, b,c") == ["a", "b", "c"])
  expect(re(",").split("a,b,c", limit: 2) == ["a", "b,c"])
  expect(re(",").split("a,b,c", limit: 0).isEmpty())
  expect(re("x").split("abc", limit: 0).isEmpty())
  expect(re(",").split("", limit: 0).isEmpty())
  expect(re(",").split("a,b,c", limit: 1) == ["a,b,c"])
  expect(re(",").split("a,b,c", limit: -1) == ["a", "b", "c"])
  expect(re(",").split("") == [""])
  expect(re("").split("abc") == ["a", "b", "c"])
}

test escape "escaped text matches itself" {
  val sample = "1+1=2 (really)? [yes] {no} a|b ^$ \\ ."
  expect(re(Regex.escape(sample)).matches(sample))
  expect(Regex.escape("a.b") == "a\\.b")
  expect(Regex.escape("plain_word9") == "plain_word9")
}

test errors "bad patterns say what is wrong" {
  expect(why("(").contains("missing ')'"))
  expect(why(")").contains("unmatched"))
  expect(why("[a").contains("missing ']'"))
  expect(why("*a").contains("nothing to repeat"))
  expect(why("a**").contains("cannot be repeated"))
  expect(why("a{3,2}").contains("lower bound"))
  expect(why("a{1001}").contains("over 1000"))
  expect(why("(?<n>a)(?<n>b)").contains("two groups"))
  expect(why("\\1").contains("backreferences"))
  expect(why("a(?=b)").contains("lookahead"))
  expect(why("(?<=a)b").contains("lookbehind"))
  expect(why("\\q").contains("unknown escape"))
  expect(why("[z-a]").contains("backwards"))
  expect(why("a\\").contains("backslash"))
  expect(why("^*").contains("assertion"))
}

test nullableCapture "greedy nullable repetition retains its empty capture" {
  loop (pattern in ["(a*)*b", "(a?)*b", "(a{0,2})*b"]) {
    loop (text in ["b", "a\nb"]) {
      val matched = re(pattern).find(text) ?: panic("the trailing b must match")
      expect(matched.value() == "b")
      expect(matched.groupAt(1) == "")
      var threads = Machine.of(re(pattern).program)
      threads.threadsOnly = true
      val slots = threads.search(text, 0, false, false) ?: panic("the trailing b must match")
      expect(slots.at(2) == matched.start())
      expect(slots.at(3) == matched.start())
    }
  }
  val lazy = re("(a*)*?b").find("b") ?: panic("the trailing b must match")
  expect(lazy.groupAt(1) == null)
}

test scratchGrowth "search stacks grow past their initial reservation" {
  val compiled = re("(a?)".repeat(40) + "b")
  val matches = compiled.findAll("b ab b")
  expect(matches.len() == 3)
  loop (matched in matches) {
    expect(matched.groupAt(40) == "")
    expect(matched.groupCount() == 40)
  }
  expect(matches.at(0)?.value() == "b")
  expect(matches.at(1)?.value() == "ab")
  expect(matches.at(2)?.value() == "b")
  expect(re("needle").find("x".repeat(100)) == null)
}

test malformedClasses "invalid classes and properties stop with a typed error" {
  loop (pattern in [
    "[[:notaclass:]]", "[[:alpha:]", "[a-\\d]", "[\\p{NotAProperty}]",
    "\\p{NotAProperty}", "\\p{", "\\x{", "[\\x{", "[\\", "(?i", "(?z)a",
  ]) {
    expect(Regex.compile(pattern) is Err)
  }
}

test longCounts "long repetition counts stay bounded and leading zeroes remain valid" {
  val digits = "9".repeat(2048)
  loop (pattern in ["a{$digits}", "a{1,$digits}", "a{$digits,}"]) {
    expect(why(pattern).contains("over 1000"))
  }
  val paddedOne = "0".repeat(2048) + "1"
  expect(re("a{$paddedOne}").matches("a"))
  expect(re("a{$paddedOne}").matches("aa") == false)
}

test hexBoundaries "hex escapes cover Unicode bounds and reject excessive values" {
  expect(re("\\x00").matches("\u{0}"))
  expect(re("\\x{10FFFF}").matches(utf8.char(utf8.maxCode)))
  expect(re("\\x{" + "0".repeat(2048) + "41}").matches("A"))
  loop (pattern in ["\\x{110000}", "\\x{" + "F".repeat(2048) + "}", "[\\x{110000}]", "\\x{}", "\\x0", "\\xGG"]) {
    expect(Regex.compile(pattern) is Err)
  }
}

test errorPositions "malformed escapes report code point offsets after Unicode" {
  loop ((pattern, position) in ["\\q", "\u{E9}\\q", "\u{1F600}\\q", "\u{E9}\u{1F600}\\q"].zip([1, 2, 2, 3])) {
    expect(when (Regex.compile(pattern)) {
      is Err(error) => error.position == position && error.message.contains("unknown escape")
      is Ok(_)      => false
    })
  }
}

test linear "a pattern that blows up a backtracker finishes at once" {
  val hostile = re("(a+)+$")
  val text = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
  expect(!hostile.test(text))
  expect(re("(x+x+)+y").find("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx") == null)
  expect(re("(a|aa)+$").test("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
}

test counted "large counts and nesting stay bounded" {
  expect(re("a{1000}").test("a".repeat(1000)))
  expect(why("(a{1000}){1000}").contains("instructions"))
  expect(why("(".repeat(300)).contains("nested"))
}

/// The two engines agree: the backtracker (small jobs) and the threads
/// (large ones) return the same capture slots for every pattern and text.
test engines "backtracker and threads agree" {
  val patterns = [
    `a`, `abc`, `a|ab`, `ab|a`, `(a|ab)(c|bcd)(d*)`, `a*`, `a+?`, `(a*)*b`, `(a+)+$`,
    `(\d+)-(\d+)`, `(?<y>\d{4})-(?<m>\d\d)`, `\bcat\b`, `^\w+$`, `[a-c]+x?`, `(?i)hello`,
    `é+`, `(x*)(y*)z?`, `.*`, `.+?b`, `(a)|(b)|(c)`, `(?:ab)*c`, `$`, `^`, `x*$`,
    `é*x`, `.*?é`, `[a-z]*[0-9]`, `w+?d`, `(w*)-(w*)`, `.*é.*`, `a*?b*?c`, `[^x]*x`,
  ]
  val texts = ["", "a", "abc", "ab", "abcd", "aaab", "2026-10-19", "cat concat cat", "hello HELLO", "ééa", "xyz", "xxyyz", "bab", "a\nb\nc", "ééx", "abc123é", "foo-bar-é", "axbxc", "aaaaaaaaaaaaaaaaaaaaaaaaaaaa"]
  var compared = 0
  loop (pattern in patterns) {
    val regex = re(pattern)
    loop (text in texts) {
      loop (from in 0..<(text.len() + 1)) {
        loop (anchored in [false, true]) {
          loop (full in [false, true]) {
            val small = Machine.of(regex.program)
            var large = Machine.of(regex.program)
            large.threadsOnly = true
            if (utf8.decode(text, from) == null && from < text.len()) continue
            expect(small.search(text, from, anchored, full) == large.search(text, from, anchored, full))
            compared += 1
          }
        }
      }
    }
  }
  expect(compared > 1000)
}

test big "a text too big for the backtracker still matches" {
  val text = "x".repeat(20000) + " needle 42 end"
  val m = re(`(\w+) (\d+)`).find(text) ?: panic("no match")
  expect(m.groupAt(1) == "needle")
  expect(m.groupAt(2) == "42")
  expect(m.start() == 20001)
  expect(re(`\d+`).findAll(text + " 7").len() == 2)
}

// The compiler runs the parser itself on every literal (D182): these are
// computed while compiling, so they fail the build if the evaluator cannot run
// the parser, the compiler of programs, or finds a different answer.
const okLiteral = validate(`(?<y>\d{4})-(?<m>\d\d)|[a-z]+?x*`, "i")
const badLiteral = validate("a(", "")
const lookaround = validate("a(?=b)", "")
const tooBig = validate("(a{1000}){1000}", "")
const nullableLiteral = validate("(a*)*b", "")
const negatedPosixLiteral = validate("[[:^digit:]]+", "")
static assert(okLiteral == "", "a good pattern validates")
static assert(badLiteral == "missing ')'", "a bad pattern says why")
static assert(lookaround.contains("lookahead"), "lookaround is refused")
static assert(tooBig.contains("instructions"), "a huge pattern is refused")
static assert(nullableLiteral == "", "nullable repetition compiles in the evaluator")
static assert(negatedPosixLiteral == "", "negated POSIX classes compile in the evaluator")

test validated "validate agrees at run time" {
  expect(validate("a+", "") == "")
  expect(validate("a(", "") == "missing ')'")
  expect(validate("a", "i") == "")
}

/// A pattern compiled for the exhaustive engine.
test fun rb(pattern: string): Regex => Regex.compile(pattern, backtracking: true) catch (e) {
  panic("bad test pattern $pattern: ${e.message}")
}

test properties "Unicode categories and scripts by name" {
  expect(first(`\p{L}+`, "123 héllo wörld") == "héllo")
  expect(first(`\pL+`, "—日本語—") == "日本語")
  expect(first(`\p{Greek}+`, "abc αβγ def") == "αβγ")
  expect(first(`\p{Lu}`, "abC") == "C")
  expect(first(`\P{L}+`, "ab12cd") == "12")
  expect(first(`\p{^L}+`, "ab12cd") == "12")
  expect(first(`[\p{Nd}_]+`, "x٣٤_y") == "٣٤_")
  expect(first(`\p{Han}+`, "日本語 kanji") == "日本語")
  expect(first(`\p{Old_Italic}`, "a𐌀b") == "𐌀")
  expect(first(`\p{Any}`, "é") == "é")
  expect(first(`[^\p{L}]+`, "ab12cd") == "12")
  expect(re(`\p{Lu}\p{Ll}+`).findAll("Hello World ünï Ünï").len() == 3)
  expect(why(`\p{Nope}`).contains("unknown Unicode class or script 'Nope'"))
  expect(why(`\p`).contains("needs a class name"))
  expect(why(`\p{L`).contains("then '}'"))
}

test unicodeFlag "the u flag makes classes and case folding Unicode" {
  expect(first(`\w+`, "héllo") == "h")
  expect(first(`(?u)\w+`, "héllo wörld") == "héllo")
  expect(first(`(?u)\W+`, "héllo, wörld") == ", ")
  expect(first(`(?u)\d+`, "x٣٤y") == "٣٤")
  expect(first(`\d+`, "x٣٤y") == "-")
  expect(first(`(?u)\s`, "a\u{00A0}b") == "\u{00A0}")
  expect(first(`\s`, "a\u{00A0}b") == "-")
  expect(first(`(?u)\bé`, "xé é") == "é")
  expect(re(`(?u)\bé`).find("xé é")?.start() == 4)
  expect(first(`\bé`, "xé é") == "é")
  expect(first(`(?iu)é`, "É") == "É")
  expect(first(`(?i)é`, "É") == "-")
  expect(first(`(?iu)[à-ý]+`, "ÀÉÜ") == "ÀÉÜ")
  expect(first(`(?iu)σ+`, "Σσς") == "Σσς")
  expect(first(`(?iu)ǆ`, "ǅ") == "ǅ")
  expect(Regex.compile("é+", unicode: true) catch (e) {
    panic("")
  }.test("é"))
  expect(Regex.compile("É", ignoreCase: true, unicode: true) catch (e) {
    panic("")
  }.test("é"))
  expect(first(`(?u:\w+) \w+`, "héllo wörld") == "héllo w")
}

test backtracking "backreferences and lookaround, on request" {
  expect(rb(`(\w)\1`).find("abccd")?.value() == "cc")
  expect(rb(`(?<q>['"]).*?\k<q>`).find(`say "hi" and 'yo'`)?.value() == `"hi"`)
  expect(rb(`(a+)b\1`).find("aabaa")?.value() == "aabaa")
  expect(rb(`(a+)b\1`).find("aaba")?.value() == "aba")
  expect(rb(`(?i)(ab)\1`).test("abAB"))
  expect(!rb(`(ab)\1`).test("abAB"))
  expect(rb(`(a)|b\1`).find("b") == null)
  expect(rb(`foo(?=bar)`).find("foobaz foobar")?.start() == 7)
  expect(rb(`foo(?!bar)`).find("foobar foobaz")?.start() == 7)
  expect(rb(`(?<=\$)\d+`).find("cost $42 or 17")?.value() == "42")
  expect(rb(`(?<!\$)\b\d+`).find("$42 17")?.value() == "17")
  expect(rb(`(?<=ab)c`).find("xabc")?.start() == 3)
  expect(rb(`(?<=é)x`).find("éx")?.start() == 2)
  expect(rb(`^(?=.*\d)(?=.*[a-z]).{6,}$`).test("abc123"))
  expect(!rb(`^(?=.*\d)(?=.*[a-z]).{6,}$`).test("abcdef"))
  expect(rb(`(?=(\w+))\w`).find("word")?.groupAt(1) == "word")
  expect(rb(`(?!(a))b`).find("b")?.groupAt(1) == null)
  expect(rb(`(\d+)-\1`).matches("12-12"))
  expect(!rb(`(\d+)-\1`).matches("12-13"))
  expect(rb(`(a*)*b`).find("xaaab")?.value() == "aaab")
  expect(rb(`(a*)*c`).find("aaab") == null)
  expect(rb(`(a|b)*\1`).find("abb")?.value() == "abb")
  expect(rb(`\p{Lu}(?=\p{Ll})`).findAll("ABc dEf").len() == 2)
}

test backtrackingErrors "what the exhaustive engine still refuses" {
  expect(why(`(\w)\1`).contains("backtracking: true"))
  expect(why(`a(?=b)`).contains("backtracking: true"))
  expect(why(`(?<=a)b`).contains("backtracking: true"))
  expect(why(`\k<x>`).contains("backtracking: true"))
  expect(whyB(`(a)\2`).contains("refers to a group the pattern does not have"))
  expect(whyB(`\k<x>(?<x>a)`).contains("has not been opened yet"))
  expect(whyB(`(?=a`).contains("missing ')'"))
}

/// Why a pattern is refused even for the exhaustive engine.
test fun whyB(pattern: string): string {
  val r = Regex.compile(pattern, backtracking: true) catch (e) {
    return e.message
  }
  "compiled ${r.source()}"
}

const unicodeLiteral = validate(`\p{L}+\s\p{Greek}*[\p{Nd}_]`, "u")
const foldedLiteral = validate(`straße|É`, "iu")
const badProperty = validate(`\p{Nope}`, "")
static assert(unicodeLiteral == "", "Unicode classes validate at compile time")
static assert(foldedLiteral == "", "Unicode case folding runs in the compiler")
static assert(badProperty.contains("unknown Unicode class"), "an unknown class is refused while compiling")
