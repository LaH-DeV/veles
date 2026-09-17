// Package lexer turns Veles source text into tokens.
//
// Newlines are significant: following Go's rule (spec §4b), a `;` token is
// inserted at the end of a line whose final token could end a statement.
// Unlike Go, no insertion happens while inside `(...)` or `[...]`, so
// argument lists and conditions may span lines freely; and a line that
// begins with `.`, `?.` or `?:` continues the previous one so method chains
// can be written one call per line.
package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/source"
)

type Lexer struct {
	file  *source.File
	src   string
	pos   int
	diags *source.Diagnostics

	tokens  []Token
	nesting []byte // stack of '(' '[' '{'
	docs    []docComment // documentation comments, attached to tokens after the scan
	comments []Comment   // every comment, in source order (for the formatter)
}

// Comment is one `// ...` line or `/* ... */` block, with its raw text.
type Comment struct {
	Span source.Span
	Text string
}

// Tokenize scans an entire file. Errors are reported into diags and the
// offending characters become Illegal tokens so parsing can continue.
func Tokenize(file *source.File, diags *source.Diagnostics) []Token {
	toks, _ := TokenizeFile(file, diags)
	return toks
}

// TokenizeFile scans a file and also returns its module documentation: a
// doc comment at the very top of the file, set off from the first token
// by a blank line (so it documents the module, not that declaration).
func TokenizeFile(file *source.File, diags *source.Diagnostics) ([]Token, string) {
	lx := &Lexer{file: file, src: file.Content, diags: diags}
	lx.run()
	lx.attachDocs()
	return lx.tokens, lx.moduleDoc()
}

// TokenizeAll is TokenizeFile plus every comment in the file, in source
// order, for tools that reproduce the source (the formatter).
func TokenizeAll(file *source.File, diags *source.Diagnostics) ([]Token, string, []Comment) {
	lx := &Lexer{file: file, src: file.Content, diags: diags}
	lx.run()
	lx.attachDocs()
	return lx.tokens, lx.moduleDoc(), lx.comments
}

func (lx *Lexer) span(start, end int) source.Span {
	return source.Span{File: lx.file, Start: start, End: end}
}

func (lx *Lexer) errorf(start, end int, format string, args ...any) {
	lx.diags.Errorf(lx.span(start, end), format, args...)
}

func (lx *Lexer) peekByte(off int) byte {
	if lx.pos+off < len(lx.src) {
		return lx.src[lx.pos+off]
	}
	return 0
}

func (lx *Lexer) push(kind TokenKind, start int) {
	lx.tokens = append(lx.tokens, Token{Kind: kind, Text: lx.src[start:lx.pos], Span: lx.span(start, lx.pos)})
	switch kind {
	case LParen, LBracket, LBrace:
		lx.nesting = append(lx.nesting, lx.src[start])
	case RParen, RBracket, RBrace:
		if len(lx.nesting) > 0 {
			lx.nesting = lx.nesting[:len(lx.nesting)-1]
		}
	}
}

func (lx *Lexer) lastKind() TokenKind {
	if len(lx.tokens) == 0 {
		return EOF
	}
	return lx.tokens[len(lx.tokens)-1].Kind
}

// semiAllowed reports whether a newline after the last token terminates a
// statement.
func (lx *Lexer) semiAllowed() bool {
	if len(lx.nesting) > 0 && lx.nesting[len(lx.nesting)-1] != '{' {
		return false
	}
	switch lx.lastKind() {
	case Ident, Int, Float, String, Char,
		KwTrue, KwFalse, KwNull, KwSelf, KwSelfTy, KwBreak, KwContinue, KwReturn,
		KwThrows, KwSuspends,
		RParen, RBracket, RBrace, Question, Gt, Under:
		return true
	}
	return false
}

// continuesLine reports whether the next significant text begins with a
// chain operator, in which case the newline does not end the statement.
func (lx *Lexer) continuesLine() bool {
	i := lx.pos
	for i < len(lx.src) {
		c := lx.src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < len(lx.src) && lx.src[i+1] == '/':
			for i < len(lx.src) && lx.src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(lx.src) && lx.src[i+1] == '*':
			end := strings.Index(lx.src[i+2:], "*/")
			if end < 0 {
				return false
			}
			i += end + 4
		default:
			rest := lx.src[i:]
			if strings.HasPrefix(rest, "?.") || strings.HasPrefix(rest, "?:") {
				return true
			}
			if c == '.' && len(rest) > 1 && rest[1] != '.' {
				return true
			}
			return false
		}
	}
	return false
}

func (lx *Lexer) newline(start int) {
	if lx.semiAllowed() && !lx.continuesLine() {
		lx.tokens = append(lx.tokens, Token{Kind: Semi, Text: "\n", Span: lx.span(start, start+1), AutoSemi: true})
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (lx *Lexer) run() {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		start := lx.pos
		switch {
		case c == '\n':
			lx.pos++
			lx.newline(start)
		case c == ' ' || c == '\t' || c == '\r':
			lx.pos++
		case c == '/' && lx.peekByte(1) == '/':
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
				lx.pos++
			}
			lx.comments = append(lx.comments, Comment{Span: lx.span(start, lx.pos), Text: strings.TrimRight(lx.src[start:lx.pos], "\r")})
			lx.lineDoc(start)
		case c == '/' && lx.peekByte(1) == '*':
			lx.blockComment()
			lx.comments = append(lx.comments, Comment{Span: lx.span(start, lx.pos), Text: lx.src[start:lx.pos]})
			lx.blockDoc(start)
		case isIdentStart(c):
			lx.identifier()
		case isDigit(c):
			lx.number()
		case c == '"':
			lx.stringLit()
		case c == '\'':
			lx.charLit()
		default:
			lx.operator()
		}
	}
	// A file always ends in a statement terminator.
	if lx.semiAllowed() {
		lx.tokens = append(lx.tokens, Token{Kind: Semi, Text: "", Span: lx.span(lx.pos, lx.pos), AutoSemi: true})
	}
	lx.tokens = append(lx.tokens, Token{Kind: EOF, Span: lx.span(lx.pos, lx.pos)})
}

func (lx *Lexer) blockComment() {
	start := lx.pos
	depth := 0
	for lx.pos < len(lx.src) {
		if strings.HasPrefix(lx.src[lx.pos:], "/*") {
			depth++
			lx.pos += 2
		} else if strings.HasPrefix(lx.src[lx.pos:], "*/") {
			depth--
			lx.pos += 2
			if depth == 0 {
				return
			}
		} else {
			lx.pos++
		}
	}
	lx.errorf(start, start+2, "unterminated block comment")
}

func (lx *Lexer) identifier() {
	start := lx.pos
	for lx.pos < len(lx.src) && isIdentChar(lx.src[lx.pos]) {
		lx.pos++
	}
	text := lx.src[start:lx.pos]
	if text == "_" {
		lx.push(Under, start)
		return
	}
	if kind, ok := keywords[text]; ok {
		lx.push(kind, start)
		return
	}
	lx.push(Ident, start)
}

func (lx *Lexer) digits(valid func(byte) bool) {
	for lx.pos < len(lx.src) && (valid(lx.src[lx.pos]) || lx.src[lx.pos] == '_') {
		lx.pos++
	}
}

func (lx *Lexer) number() {
	start := lx.pos
	if lx.src[lx.pos] == '0' && lx.pos+1 < len(lx.src) {
		switch lx.src[lx.pos+1] {
		case 'x', 'X':
			lx.pos += 2
			lx.digits(func(c byte) bool { return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') })
			lx.push(Int, start)
			return
		case 'b', 'B':
			lx.pos += 2
			lx.digits(func(c byte) bool { return c == '0' || c == '1' })
			lx.push(Int, start)
			return
		case 'o', 'O':
			lx.pos += 2
			lx.digits(func(c byte) bool { return c >= '0' && c <= '7' })
			lx.push(Int, start)
			return
		}
	}
	lx.digits(isDigit)
	isFloat := false
	// A '.' followed by a digit continues a float; `1..5` is a range.
	if lx.peekByte(0) == '.' && isDigit(lx.peekByte(1)) {
		isFloat = true
		lx.pos++
		lx.digits(isDigit)
	}
	if c := lx.peekByte(0); c == 'e' || c == 'E' {
		n := 1
		if s := lx.peekByte(1); s == '+' || s == '-' {
			n++
		}
		if isDigit(lx.peekByte(n)) {
			isFloat = true
			lx.pos += n
			lx.digits(isDigit)
		}
	}
	if isFloat {
		lx.push(Float, start)
	} else {
		lx.push(Int, start)
	}
	if lx.pos < len(lx.src) && isIdentStart(lx.src[lx.pos]) {
		s := lx.pos
		for lx.pos < len(lx.src) && isIdentChar(lx.src[lx.pos]) {
			lx.pos++
		}
		lx.errorf(s, lx.pos, "unexpected suffix '%s' on numeric literal", lx.src[s:lx.pos])
	}
}

// escape decodes one escape sequence starting at the backslash; returns the
// decoded text and advances past it.
func (lx *Lexer) escape() string {
	start := lx.pos
	lx.pos++ // backslash
	if lx.pos >= len(lx.src) {
		lx.errorf(start, lx.pos, "unterminated escape sequence")
		return ""
	}
	c := lx.src[lx.pos]
	lx.pos++
	switch c {
	case 'n':
		return "\n"
	case 't':
		return "\t"
	case 'r':
		return "\r"
	case '0':
		return "\x00"
	case '\\':
		return "\\"
	case '"':
		return "\""
	case '\'':
		return "'"
	case '$':
		return "$"
	case 'u':
		if lx.peekByte(0) != '{' {
			lx.errorf(start, lx.pos, "expected '{' after \\u")
			return ""
		}
		lx.pos++
		hexStart := lx.pos
		for lx.pos < len(lx.src) && lx.src[lx.pos] != '}' {
			lx.pos++
		}
		hex := lx.src[hexStart:lx.pos]
		if lx.pos < len(lx.src) {
			lx.pos++
		}
		var r rune
		for _, h := range hex {
			var v rune
			switch {
			case h >= '0' && h <= '9':
				v = h - '0'
			case h >= 'a' && h <= 'f':
				v = h - 'a' + 10
			case h >= 'A' && h <= 'F':
				v = h - 'A' + 10
			default:
				lx.errorf(hexStart, lx.pos, "invalid unicode escape '\\u{%s}'", hex)
				return ""
			}
			r = r*16 + v
		}
		if !utf8.ValidRune(r) || len(hex) == 0 {
			lx.errorf(start, lx.pos, "invalid unicode escape '\\u{%s}'", hex)
			return ""
		}
		return string(r)
	}
	lx.errorf(start, lx.pos, "unknown escape sequence '\\%c'", c)
	return ""
}

func (lx *Lexer) stringLit() {
	start := lx.pos
	lx.pos++ // opening quote
	var parts []StringPart
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			parts = append(parts, StringPart{Text: buf.String()})
			buf.Reset()
		}
	}
	for {
		if lx.pos >= len(lx.src) {
			lx.errorf(start, start+1, "unterminated string literal")
			break
		}
		c := lx.src[lx.pos]
		if c == '"' {
			lx.pos++
			break
		}
		if c == '\\' {
			buf.WriteString(lx.escape())
			continue
		}
		if c == '$' {
			// `$ident` or `${expr}`
			if lx.peekByte(1) == '{' {
				flush()
				exprStart := lx.pos + 2
				lx.pos += 2
				depth := 1
				for lx.pos < len(lx.src) && depth > 0 {
					switch lx.src[lx.pos] {
					case '{':
						depth++
					case '}':
						depth--
					case '"':
						// skip nested string
						lx.pos++
						for lx.pos < len(lx.src) && lx.src[lx.pos] != '"' {
							if lx.src[lx.pos] == '\\' {
								lx.pos++
							}
							lx.pos++
						}
					}
					if depth > 0 {
						lx.pos++
					}
				}
				if depth != 0 {
					lx.errorf(exprStart-2, exprStart, "unterminated interpolation")
					break
				}
				parts = append(parts, StringPart{IsExpr: true, Expr: lx.src[exprStart:lx.pos], Span: lx.span(exprStart, lx.pos)})
				lx.pos++ // closing brace
				continue
			}
			if isIdentStart(lx.peekByte(1)) {
				flush()
				exprStart := lx.pos + 1
				lx.pos++
				for lx.pos < len(lx.src) && isIdentChar(lx.src[lx.pos]) {
					lx.pos++
				}
				parts = append(parts, StringPart{IsExpr: true, Expr: lx.src[exprStart:lx.pos], Span: lx.span(exprStart, lx.pos)})
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(lx.src[lx.pos:])
		if r == utf8.RuneError && size == 1 {
			lx.errorf(lx.pos, lx.pos+1, "invalid UTF-8 in string literal")
		}
		buf.WriteString(lx.src[lx.pos : lx.pos+size])
		lx.pos += size
	}
	flush()
	if parts == nil {
		parts = []StringPart{{Text: ""}}
	}
	lx.tokens = append(lx.tokens, Token{Kind: String, Text: lx.src[start:lx.pos], Span: lx.span(start, lx.pos), Parts: parts})
}

func (lx *Lexer) charLit() {
	start := lx.pos
	lx.pos++
	var text string
	if lx.peekByte(0) == '\\' {
		text = lx.escape()
	} else if lx.pos < len(lx.src) {
		r, size := utf8.DecodeRuneInString(lx.src[lx.pos:])
		text = string(r)
		lx.pos += size
	}
	if lx.peekByte(0) != '\'' {
		lx.errorf(start, lx.pos, "unterminated character literal")
	} else {
		lx.pos++
	}
	lx.tokens = append(lx.tokens, Token{Kind: Char, Text: lx.src[start:lx.pos], Span: lx.span(start, lx.pos), Parts: []StringPart{{Text: text}}})
}

// operators are matched longest-first.
var operators = []struct {
	text string
	kind TokenKind
}{
	{"..<", RangeLt}, {"::", DblColon}, {"?.", SafeDot}, {"?:", Elvis}, {"=>", FatArrow},
	{"->", Arrow}, {"..", Range}, {"+=", PlusEq}, {"-=", MinusEq}, {"*=", StarEq},
	{"/=", SlashEq}, {"%=", PercentEq}, {"+%", WrapPlus}, {"-%", WrapMinus}, {"*%", WrapStar},
	{"==", Eq}, {"!=", NotEq}, {"<=", LtEq}, {">=", GtEq}, {"&&", AndAnd}, {"||", OrOr},
	{"(", LParen}, {")", RParen}, {"{", LBrace}, {"}", RBrace}, {"[", LBracket}, {"]", RBracket},
	{",", Comma}, {";", Semi}, {":", Colon}, {".", Dot}, {"?", Question}, {"@", At},
	{"&", Amp}, {"|", Pipe}, {"=", Assign}, {"+", Plus}, {"-", Minus}, {"*", Star},
	{"/", Slash}, {"%", Percent}, {"<", Lt}, {">", Gt}, {"!", Bang},
}

func (lx *Lexer) operator() {
	start := lx.pos
	rest := lx.src[lx.pos:]
	for _, op := range operators {
		if strings.HasPrefix(rest, op.text) {
			lx.pos += len(op.text)
			lx.push(op.kind, start)
			return
		}
	}
	r, size := utf8.DecodeRuneInString(rest)
	lx.pos += size
	if unicode.IsPrint(r) {
		lx.errorf(start, lx.pos, "unexpected character '%c'", r)
	} else {
		lx.errorf(start, lx.pos, "unexpected character U+%04X", r)
	}
	lx.push(Illegal, start)
}

// TokenizeRange scans file.Content[start:end] with file-absolute positions.
// It is used to lex the expressions embedded in interpolated strings.
func TokenizeRange(file *source.File, start, end int, diags *source.Diagnostics) []Token {
	lx := &Lexer{file: file, src: file.Content[:end], pos: start, diags: diags}
	lx.run()
	return lx.tokens
}

// ---------------------------------------------------------------------------
// documentation comments

// docComment is a `/// ...` line or a `/** ... */` block; consecutive
// `///` lines merge into one comment.
type docComment struct {
	start, end int
	text       string
}

// lineDoc records a `///` comment just scanned from start (pos is at the
// newline). `////` and plain `//` are ordinary comments.
func (lx *Lexer) lineDoc(start int) {
	body := lx.src[start:lx.pos]
	if !strings.HasPrefix(body, "///") || strings.HasPrefix(body, "////") {
		return
	}
	line := strings.TrimPrefix(strings.TrimPrefix(body, "///"), " ")
	line = strings.TrimRight(line, "\r")
	// merge with a `///` line directly above
	if n := len(lx.docs); n > 0 && onlyWhitespace(lx.src[lx.docs[n-1].end:start]) && strings.Count(lx.src[lx.docs[n-1].end:start], "\n") == 1 {
		lx.docs[n-1].text += "\n" + line
		lx.docs[n-1].end = lx.pos
		return
	}
	lx.docs = append(lx.docs, docComment{start: start, end: lx.pos, text: line})
}

// blockDoc records a `/** ... */` comment just scanned from start.
func (lx *Lexer) blockDoc(start int) {
	body := lx.src[start:lx.pos]
	if !strings.HasPrefix(body, "/**") || body == "/**/" || !strings.HasSuffix(body, "*/") {
		return
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(body, "/**"), "*/")
	var lines []string
	for _, l := range strings.Split(inner, "\n") {
		l = strings.TrimRight(l, " \t\r")
		l = strings.TrimLeft(l, " \t")
		l = strings.TrimPrefix(strings.TrimPrefix(l, "*"), " ")
		lines = append(lines, l)
	}
	// drop blank first/last lines from `/**\n ... \n */`
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lx.docs = append(lx.docs, docComment{start: start, end: lx.pos, text: strings.Join(lines, "\n")})
}

// attachDocs gives each documentation comment to the first token after it,
// provided nothing but whitespace (at most one newline of it) separates
// them. Automatic semicolons are skipped over.
func (lx *Lexer) attachDocs() {
	ti := 0
	for _, d := range lx.docs {
		for ti < len(lx.tokens) && (lx.tokens[ti].Span.Start < d.end || (lx.tokens[ti].Kind == Semi && lx.tokens[ti].AutoSemi)) {
			ti++
		}
		if ti >= len(lx.tokens) {
			return
		}
		gap := lx.src[d.end:lx.tokens[ti].Span.Start]
		if onlyWhitespace(gap) && strings.Count(gap, "\n") <= 1 {
			lx.tokens[ti].Doc = d.text
		}
	}
}

func onlyWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}

// moduleDoc is the first doc comment when nothing but whitespace precedes
// it and a blank line separates it from the first token.
func (lx *Lexer) moduleDoc() string {
	if len(lx.docs) == 0 {
		return ""
	}
	d := lx.docs[0]
	if !onlyWhitespace(lx.src[:d.start]) {
		return ""
	}
	for _, t := range lx.tokens {
		if t.Span.Start < d.end || (t.Kind == Semi && t.AutoSemi) {
			continue
		}
		if strings.Count(lx.src[d.end:t.Span.Start], "\n") >= 2 {
			return d.text
		}
		return ""
	}
	return d.text
}
