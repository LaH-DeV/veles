// The scanner. Newlines are significant (spec
// §4b): a Semi is inserted at the end of a line whose last token could end a
// statement — never inside `(…)` or `[…]`, and not when the next line starts
// with `.`, `?.`, `?:`, `?!` or `??`, so chains may be written one call per
// line.
use utf8
use source { Diagnostics, File, Span }

// a documentation comment: `/// …` lines (consecutive ones merged) or a
// `/** … */` block, markers stripped
struct DocComment {
  start:    i64
  var end:  i64
  var text: string
}

/// What scanning a file gives: the tokens, the module's documentation (a doc
/// comment at the very top, set off from the first token by a blank line),
/// and every comment for tools that reproduce the source.
public struct Scan {
  public tokens:    List<Token>
  public moduleDoc: string
  public comments:  List<Comment>
}

/// Scans a whole file. Problems go into `diags`, and a character that
/// belongs to no token becomes an Illegal one, so parsing can go on.
public fun tokenize(file: File, diags: *Diagnostics): Scan {
  var lexer = Lexer(file, src: file.content, diags)
  lexer.run()
  lexer.attachDocs()
  Scan(tokens: lexer.tokens.toList(), moduleDoc: lexer.moduleDoc(), comments: lexer.comments.toList())
}

/// Scans `file.content[start, end)` with the file's own offsets: the
/// expression inside an interpolated string, `${…}`.
public fun tokenizeRange(file: File, start: i64, end: i64, diags: *Diagnostics): List<Token> {
  // the cut is where a `${` or a `}` is, so not inside a character
  val src = file.content.substring(0, end) ?: panic("tokenizeRange: $end is not at a character's edge")
  var lexer = Lexer(file, src, diags, pos: start)
  lexer.run()
  lexer.tokens.toList()
}

// the operators, longest first where one is a prefix of another
const OPERATORS: List<(string, Kind)> = [
  ("...", Kind.Ellipsis), ("..<", Kind.RangeLt), ("<<", Kind.Shl), (">>", Kind.Shr), ("::", Kind.DblColon),
  ("?.", Kind.SafeDot), ("?:", Kind.Elvis), ("?!", Kind.OrFail), ("??", Kind.Coalesce), ("=>", Kind.FatArrow),
  ("->", Kind.Arrow), ("..", Kind.Range), ("+=", Kind.PlusEq), ("-=", Kind.MinusEq), ("*=", Kind.StarEq),
  ("/=", Kind.SlashEq), ("%=", Kind.PercentEq), ("+%", Kind.WrapPlus), ("-%", Kind.WrapMinus),
  ("*%", Kind.WrapStar), ("==", Kind.Eq), ("!=", Kind.NotEq), ("<=", Kind.LtEq), (">=", Kind.GtEq),
  ("&&", Kind.AndAnd), ("||", Kind.OrOr), ("(", Kind.LParen), (")", Kind.RParen), ("{", Kind.LBrace),
  ("}", Kind.RBrace), ("[", Kind.LBracket), ("]", Kind.RBracket), (",", Kind.Comma), (";", Kind.Semi),
  (":", Kind.Colon), (".", Kind.Dot), ("?", Kind.Question), ("@", Kind.At), ("&", Kind.Amp), ("|", Kind.Pipe),
  ("=", Kind.Assign), ("+", Kind.Plus), ("-", Kind.Minus), ("*", Kind.Star), ("/", Kind.Slash),
  ("%", Kind.Percent), ("<", Kind.Lt), (">", Kind.Gt), ("!", Kind.Bang), ("^", Kind.Caret), ("~", Kind.Tilde),
]

fun isDigit(b: u8): bool => b >= '0' && b <= '9'

fun isHexDigit(b: u8): bool => isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')

fun isBinDigit(b: u8): bool => b == '0' || b == '1'

fun isOctDigit(b: u8): bool => b >= '0' && b <= '7'

// only the whitespace a gap between tokens can hold: whatever else stands
// there is a token or a comment
fun onlyWhitespace(text: string): bool {
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i)
    if (b != ' ' && b != '\t' && b != '\r' && b != '\n' && b != 0x0B && b != 0x0C) return false
  }
  true
}

fun countNewlines(text: string): i64 {
  var n: i64 = 0
  loop (i in 0..<text.len()) {
    if (text.byteAt(i) == '\n') n += 1
  }
  n
}

// whether the byte is one of `chars`, which are ASCII
fun isOneOf(b: u8, chars: string): bool {
  loop (i in 0..<chars.len()) {
    if (chars.byteAt(i) == b) return true
  }
  false
}

// the text with any of the ASCII `chars` taken off its end;
// a byte of a multibyte character is never one of them, so the cut is clean
fun trimEndOf(text: string, chars: string): string {
  var end = text.len()
  loop (end > 0 && isOneOf(text.byteAt(end - 1), chars)) end -= 1
  text.substring(0, end) ?: text
}

// the text with any of the ASCII `chars` taken off its start
fun trimStartOf(text: string, chars: string): string {
  var start: i64 = 0
  loop (start < text.len() && isOneOf(text.byteAt(start), chars)) start += 1
  text.substring(start, text.len()) ?: text
}

fun dropPrefix(text: string, prefix: string): string =>
  if (text.startsWith(prefix)) text.substring(prefix.len(), text.len()) ?: text else text

fun dropSuffix(text: string, suffix: string): string =>
  if (text.endsWith(suffix)) text.substring(0, text.len() - suffix.len()) ?: text else text

struct Lexer {
  file:     File
  src:      string
  diags:    *Diagnostics
  var pos:  i64 = 0
  tokens:   MutableList<Token> = []
  nesting:  MutableList<u8> = []  // the open `(`, `[` and `{`
  docs:     MutableList<DocComment> = []
  comments: MutableList<Comment> = []

  fun span(start: i64, end: i64): Span => Span(file: this.file, start, end)

  fun errorAt(start: i64, end: i64, message: string) {
    this.diags.errorAt(this.span(start, end), message)
  }

  // src[start, end). A cut that does not fall on a character's edges
  // (a token of invalid UTF-8) reads each stray byte as U+FFFD, so a token's
  // text is always valid UTF-8
  fun cut(start: i64, end: i64): string {
    if (val text = this.src.substring(start, end)) return text
    val out = StringBuilder()
    var i = start
    loop (i < end) {
      val (code, size) = this.runeAt(i)
      if (i + size > end || (code == 0xFFFD && size == 1)) {
        out.append("\u{FFFD}")
        i += 1
      } else {
        out.append(this.src.substring(i, i + size) ?: "\u{FFFD}")
        i += size
      }
    }
    out.toString()
  }

  // the character at i and its size; in the middle of a character or in
  // invalid UTF-8, U+FFFD and one byte
  fun runeAt(i: i64): (i64, i64) {
    val rune = utf8.decode(this.src, i) ?: return (0xFFFD, 1)
    (rune.code, rune.size)
  }

  fun peekByte(offset: i64): u8 => if (this.pos + offset < this.src.len()) this.src.byteAt(this.pos + offset) else 0

  // whether src holds `text` at i
  fun holdsAt(i: i64, text: string): bool {
    if (i + text.len() > this.src.len()) return false
    loop (k in 0..<text.len()) {
      if (this.src.byteAt(i + k) != text.byteAt(k)) return false
    }
    true
  }

  fun push(kind: Kind, start: i64) {
    this.tokens.push(Token(kind, text: this.cut(start, this.pos), span: this.span(start, this.pos)))
    when (kind) {
      Kind.LParen, Kind.LBracket, Kind.LBrace => this.nesting.push(this.src.byteAt(start))
      Kind.RParen, Kind.RBracket, Kind.RBrace => {
        if (!this.nesting.isEmpty()) this.nesting.pop()
      }
      else => { }
    }
  }

  fun lastKind(): Kind {
    val [.., last] = this.tokens else return Kind.EOF
    last.kind
  }

  // whether a newline after the last token ends a statement
  fun semiAllowed(): bool {
    val [.., open] = this.nesting else return this.endsStatement()
    open == '{' && this.endsStatement()
  }

  fun endsStatement(): bool => when (this.lastKind()) {
    Kind.Ident, Kind.Int, Kind.Float, Kind.String, Kind.Char, Kind.KwTrue, Kind.KwFalse, Kind.KwNull,
      Kind.KwThis, Kind.KwSelfType, Kind.KwBreak, Kind.KwContinue, Kind.KwReturn, Kind.KwThrows,
      Kind.KwSuspends, Kind.RParen, Kind.RBracket, Kind.RBrace, Kind.Question, Kind.Gt, Kind.Under,
      Kind.Shr => true  // `List<List<T>>` at a line's end
    else => false
  }

  // whether the next text that is not space or a comment starts with a
  // chain operator, so that the newline does not end the statement
  fun continuesLine(): bool {
    var i = this.pos
    loop (i < this.src.len()) {
      val b = this.src.byteAt(i)
      if (b == ' ' || b == '\t' || b == '\r' || b == '\n') {
        i += 1
      } else if (b == '/' && i + 1 < this.src.len() && this.src.byteAt(i + 1) == '/') {
        loop (i < this.src.len() && this.src.byteAt(i) != '\n') i += 1
      } else if (b == '/' && i + 1 < this.src.len() && this.src.byteAt(i + 1) == '*') {
        val end = this.findFrom(i + 2, "*/")
        if (end < 0) return false
        i = end + 2
      } else {
        if (this.holdsAt(i, "?.") || this.holdsAt(i, "?:") || this.holdsAt(i, "?!") || this.holdsAt(i, "??")) return true
        return b == '.' && i + 1 < this.src.len() && this.src.byteAt(i + 1) != '.'
      }
    }
    false
  }

  // the first index at or after `from` where `text` is, or -1
  fun findFrom(from: i64, text: string): i64 {
    loop (i in from..(this.src.len() - text.len())) {
      if (this.holdsAt(i, text)) return i
    }
    -1
  }

  fun newline(start: i64) {
    if (this.semiAllowed() && !this.continuesLine()) {
      this.tokens.push(Token(kind: Kind.Semi, text: "\n", span: this.span(start, start + 1), autoSemi: true))
    }
  }

  fun run() {
    // a byte-order mark may begin a file, and means nothing there
    if (this.pos == 0 && this.holdsAt(0, "\u{FEFF}")) this.pos = 3
    loop (this.pos < this.src.len()) {
      val b = this.src.byteAt(this.pos)
      val start = this.pos
      if (b == '\n') {
        this.pos += 1
        this.newline(start)
      } else if (b == ' ' || b == '\t' || b == '\r') {
        this.pos += 1
      } else if (b == '/' && this.peekByte(1) == '/') {
        loop (this.pos < this.src.len() && this.src.byteAt(this.pos) != '\n') this.pos += 1
        this.checkComment(start, this.pos)
        this.comments.push(Comment(span: this.span(start, this.pos), text: trimEndOf(this.cut(start, this.pos), "\r")))
        this.lineDoc(start)
      } else if (b == '/' && this.peekByte(1) == '*') {
        this.blockComment()
        this.checkComment(start, this.pos)
        this.comments.push(Comment(span: this.span(start, this.pos), text: this.cut(start, this.pos)))
        this.blockDoc(start)
      } else if (identStartLen(this.src, this.pos) > 0) {
        this.identifier()
      } else if (isDigit(b)) {
        this.number()
      } else if (b == '"') {
        this.stringLit()
      } else if (b == '\'') {
        this.charLit()
      } else {
        this.operator()
      }
    }
    // a file always ends in a statement terminator
    if (this.semiAllowed()) {
      this.tokens.push(Token(kind: Kind.Semi, text: "", span: this.span(this.pos, this.pos), autoSemi: true))
    }
    this.tokens.push(Token(kind: Kind.EOF, text: "", span: this.span(this.pos, this.pos)))
  }

  fun blockComment() {
    val start = this.pos
    var depth: i64 = 0
    loop (this.pos < this.src.len()) {
      if (this.holdsAt(this.pos, "/*")) {
        depth += 1
        this.pos += 2
      } else if (this.holdsAt(this.pos, "*/")) {
        depth -= 1
        this.pos += 2
        if (depth == 0) return
      } else {
        this.pos += 1
      }
    }
    this.errorAt(start, start + 2, "unterminated block comment")
  }

  fun identifier() {
    val start = this.pos
    this.pos = identEnd(this.src, this.pos)
    val text = this.cut(start, this.pos)
    if (text == "_") return this.push(Kind.Under, start)
    this.push(KEYWORDS.get(text) ?: Kind.Ident, start)
  }

  fun digits(valid: fun(u8): bool) {
    loop (this.pos < this.src.len() && (valid(this.src.byteAt(this.pos)) || this.src.byteAt(this.pos) == '_')) this.pos += 1
  }

  fun number() {
    val start = this.pos
    if (this.src.byteAt(this.pos) == '0' && this.pos + 1 < this.src.len()) {
      val radix = this.src.byteAt(this.pos + 1)
      if (radix == 'x' || radix == 'X' || radix == 'b' || radix == 'B' || radix == 'o' || radix == 'O') {
        this.pos += 2
        when (radix) {
          'x', 'X' => this.digits(isHexDigit)
          'b', 'B' => this.digits(isBinDigit)
          else     => this.digits(isOctDigit)
        }
        return this.push(Kind.Int, start)
      }
    }
    this.digits(isDigit)
    // after a member dot a number is a tuple index: `pair.0.1` is two
    // indexes, not `pair` and the float `0.1`
    val last = this.lastKind()
    if (last == Kind.Dot || last == Kind.SafeDot) return this.push(Kind.Int, start)
    var isFloat = false
    // a '.' then a digit continues a float; `1..5` is a range
    if (this.peekByte(0) == '.' && isDigit(this.peekByte(1))) {
      isFloat = true
      this.pos += 1
      this.digits(isDigit)
    }
    val e = this.peekByte(0)
    if (e == 'e' || e == 'E') {
      var n: i64 = 1
      val sign = this.peekByte(1)
      if (sign == '+' || sign == '-') n += 1
      if (isDigit(this.peekByte(n))) {
        isFloat = true
        this.pos += n
        this.digits(isDigit)
      }
    }
    this.push(if (isFloat) Kind.Float else Kind.Int, start)
    if (identStartLen(this.src, this.pos) > 0) {
      val suffix = this.pos
      this.pos = identEnd(this.src, this.pos)
      this.errorAt(suffix, this.pos, "unexpected suffix '${this.cut(suffix, this.pos)}' on numeric literal")
    }
  }

  // decodes the escape at the backslash and moves past it
  fun escape(): string {
    val start = this.pos
    this.pos += 1  // backslash
    if (this.pos >= this.src.len()) {
      this.errorAt(start, this.pos, "unterminated escape sequence")
      return ""
    }
    val b = this.src.byteAt(this.pos)
    this.pos += 1
    when (b) {
      'n'  => return "\n"
      't'  => return "\t"
      'r'  => return "\r"
      '0'  => return "\0"
      '\\' => return "\\"
      '"'  => return "\""
      '\'' => return "'"
      '$'  => return "\$"
      'u'  => return this.unicodeEscape(start)
      else => { }
    }
    // the whole character after the backslash
    if (b >= 0x80) {
      val (code, size) = this.runeAt(this.pos - 1)
      this.pos += size - 1
      this.errorAt(start, this.pos, "unknown escape sequence '\\${utf8.char(code)}'")
      return ""
    }
    this.errorAt(start, this.pos, "unknown escape sequence '\\${utf8.char(b.toI64())}'")
    ""
  }

  // `\u{…}`, after the `u`
  fun unicodeEscape(start: i64): string {
    if (this.peekByte(0) != '{') {
      this.errorAt(start, this.pos, "expected '{' after \\u")
      return ""
    }
    this.pos += 1
    val hexStart = this.pos
    loop (this.pos < this.src.len() && this.src.byteAt(this.pos) != '}') this.pos += 1
    val hex = this.cut(hexStart, this.pos)
    if (this.pos < this.src.len()) this.pos += 1
    var code: i64 = 0
    loop (i in 0..<hex.len()) {
      val h = hex.byteAt(i)
      val digit: i64 = if (isDigit(h)) (h - '0').toI64()
      else if (h >= 'a' && h <= 'f') (h - 'a').toI64() + 10
      else if (h >= 'A' && h <= 'F') (h - 'A').toI64() + 10
      else -1
      if (digit < 0) {
        this.errorAt(hexStart, this.pos, "invalid unicode escape '\\u{$hex}'")
        return ""
      }
      code = (code * 16 + digit).wrapI32().toI64()  // kept to 32 bits: a longer escape wraps, then fails the range check
    }
    if (!utf8.isScalar(code) || hex.isEmpty()) {
      this.errorAt(start, this.pos, "invalid unicode escape '\\u{$hex}'")
      return ""
    }
    utf8.char(code)
  }

  fun stringLit() {
    val start = this.pos
    this.pos += 1  // the opening quote
    val parts: MutableList<StringPart> = []
    val text = StringBuilder()
    loop {
      if (this.pos >= this.src.len()) {
        this.errorAt(start, start + 1, "unterminated string literal")
        break
      }
      val b = this.src.byteAt(this.pos)
      if (b == '"') {
        this.pos += 1
        break
      }
      if (b == '\\') {
        text.append(this.escape())
        continue
      }
      if (b == '$' && this.peekByte(1) == '{') {
        if (!text.isEmpty()) {
          parts.push(StringPart(text: text.toString()))
          text.clear()
        }
        val exprStart = this.pos + 2
        if (!this.interpolation(exprStart)) break
        parts.push(StringPart(isExpr: true, expr: this.cut(exprStart, this.pos), span: this.span(exprStart, this.pos)))
        this.pos += 1  // the closing brace
        continue
      }
      if (b == '$' && identStartLen(this.src, this.pos + 1) > 0) {
        if (!text.isEmpty()) {
          parts.push(StringPart(text: text.toString()))
          text.clear()
        }
        val exprStart = this.pos + 1
        this.pos = identEnd(this.src, exprStart)
        parts.push(StringPart(isExpr: true, expr: this.cut(exprStart, this.pos), span: this.span(exprStart, this.pos)))
        continue
      }
      val (code, size) = this.runeAt(this.pos)
      if (code == 0xFFFD && size == 1) this.errorAt(this.pos, this.pos + 1, "invalid UTF-8 in string literal")
      if (isBidiControl(code)) {
        // legitimate text, but invisible where it stands: an escape shows it
        this.diags.warnAt(this.span(this.pos, this.pos + size), "this string contains ${codePoint(code)}, a bidirectional control character, which changes how the code around it is displayed; write it as the escape \\u{${hexUpper(code)}}")
      }
      text.append(this.cut(this.pos, this.pos + size))
      this.pos += size
    }
    if (!text.isEmpty()) parts.push(StringPart(text: text.toString()))
    if (parts.isEmpty()) parts.push(StringPart(text: ""))
    this.tokens.push(Token(kind: Kind.String, text: this.cut(start, this.pos), span: this.span(start, this.pos), parts: parts.toList()))
  }

  // `${…}`: moves to the closing brace, over nested braces and strings;
  // false, with the error reported, when there is none
  fun interpolation(exprStart: i64): bool {
    this.pos = exprStart
    var depth: i64 = 1
    loop (this.pos < this.src.len() && depth > 0) {
      val b = this.src.byteAt(this.pos)
      if (b == '{') {
        depth += 1
      } else if (b == '}') {
        depth -= 1
      } else if (b == '"') {
        // a nested string, skipped whole
        this.pos += 1
        loop (this.pos < this.src.len() && this.src.byteAt(this.pos) != '"') {
          if (this.src.byteAt(this.pos) == '\\') this.pos += 1
          this.pos += 1
        }
      }
      if (depth > 0) this.pos += 1
    }
    if (this.pos > this.src.len()) this.pos = this.src.len()  // a nested string ran off the end
    if (depth != 0) {
      this.errorAt(exprStart - 2, exprStart, "unterminated interpolation")
      return false
    }
    true
  }

  fun charLit() {
    val start = this.pos
    this.pos += 1
    var text = ""
    if (this.peekByte(0) == '\\') {
      text = this.escape()
    } else if (this.pos < this.src.len()) {
      val (_, size) = this.runeAt(this.pos)
      text = this.cut(this.pos, this.pos + size)
      this.pos += size
    }
    if (this.peekByte(0) != '\'') {
      this.errorAt(start, this.pos, "unterminated character literal")
    } else {
      this.pos += 1
    }
    this.tokens.push(Token(kind: Kind.Char, text: this.cut(start, this.pos), span: this.span(start, this.pos), parts: [StringPart(text)]))
  }

  fun operator() {
    val start = this.pos
    loop ((text, kind) in OPERATORS) {
      if (this.holdsAt(this.pos, text)) {
        this.pos += text.len()
        return this.push(kind, start)
      }
    }
    val (code, size) = this.runeAt(this.pos)
    this.pos += size
    // not an identifier character and not an operator: name what it is,
    // since an invisible one cannot be seen in the source
    val what = if (code >= 0x80) describeRune(code)
    else if (showAsItself(code)) "'${utf8.char(code)}'"
    else codePoint(code)
    this.errorAt(start, this.pos, "unexpected character $what")
    this.push(Kind.Illegal, start)
  }

  // the bidirectional controls in a comment: there they can make the code
  // after it display as comment, or the comment as code — an error
  fun checkComment(start: i64, end: i64) {
    var i = start
    loop (i < end) {
      val (code, size) = this.runeAt(i)
      if (isBidiControl(code)) this.errorAt(i, i + size, "a comment contains ${describeRune(code)}; remove it")
      i += size
    }
  }

  // a `///` comment just scanned from start (pos is at its end); `////` and
  // `//` are ordinary comments
  fun lineDoc(start: i64) {
    val body = this.cut(start, this.pos)
    if (!body.startsWith("///") || body.startsWith("////")) return
    val line = trimEndOf(dropPrefix(dropPrefix(body, "///"), " "), "\r")
    // merged with a `///` line directly above
    val [.., last] = this.docs else {
      this.docs.push(DocComment(start, end: this.pos, text: line))
      return
    }
    val gap = this.cut(last.end, start)
    if (onlyWhitespace(gap) && countNewlines(gap) == 1) {
      val i = this.docs.len() - 1
      this.docs.set(i, DocComment(start: last.start, end: this.pos, text: last.text + "\n" + line))
      return
    }
    this.docs.push(DocComment(start, end: this.pos, text: line))
  }

  // a `/** … */` comment just scanned from start
  fun blockDoc(start: i64) {
    val body = this.cut(start, this.pos)
    if (!body.startsWith("/**") || body == "/**/" || !body.endsWith("*/")) return
    val inner = dropSuffix(dropPrefix(body, "/**"), "*/")
    val lines = inner.split("\n").map(line => dropPrefix(dropPrefix(trimStartOf(trimEndOf(line, " \t\r"), " \t"), "*"), " "))
    // without the blank first and last lines of `/**\n … \n */`
    var first: i64 = 0
    var end = lines.len()
    loop (first < end && (lines.at(first) ?: "x").isEmpty()) first += 1
    loop (end > first && (lines.at(end - 1) ?: "x").isEmpty()) end -= 1
    val text = StringBuilder()
    loop (i in first..<end) {
      if (i > first) text.append("\n")
      text.append(lines.at(i) ?: "")
    }
    this.docs.push(DocComment(start, end: this.pos, text: text.toString()))
  }

  // each documentation comment goes to the first token after it, when only
  // whitespace with at most one newline separates them; inserted
  // semicolons are passed over
  fun attachDocs() {
    var t: i64 = 0
    loop (doc in this.docs) {
      loop {
        val token = this.tokens.at(t) ?: return
        if (token.span.start >= doc.end && !(token.kind == Kind.Semi && token.autoSemi)) break
        t += 1
      }
      var token = this.tokens.at(t) ?: return
      val gap = this.cut(doc.end, token.span.start)
      if (onlyWhitespace(gap) && countNewlines(gap) <= 1) {
        token.doc = doc.text
        this.tokens.set(t, token)
      }
    }
  }

  // the first doc comment, when nothing but whitespace is above it and a
  // blank line is below
  fun moduleDoc(): string {
    val [doc, ..] = this.docs else return ""
    if (!onlyWhitespace(this.cut(0, doc.start))) return ""
    loop (token in this.tokens) {
      if (token.span.start < doc.end || (token.kind == Kind.Semi && token.autoSemi)) continue
      return if (countNewlines(this.cut(doc.end, token.span.start)) >= 2) doc.text else ""
    }
    doc.text
  }
}
