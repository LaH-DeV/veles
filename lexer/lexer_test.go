package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// between puts one code point between two ordinary tokens, so the lexer
// reaches it the way a stray character in a real file is reached. The code
// points are spelled as numbers because several of them are invisible, and
// a test whose input cannot be seen is a test no one can review.
func between(r rune) string { return "val x = 1 " + string(r) + " 2" }

// TestUnexpectedCharacter pins how an unexpected character is spelled in
// the diagnostic, which the Veles front end has to reproduce exactly (see
// veles-selfhost-frontend-plan.md §4.2). Everything that reaches the
// message is ASCII, so the rule is showAsItself and no Unicode table is
// involved on either side.
func TestUnexpectedCharacter(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"punctuation", between('$'), "unexpected character '$'"},
		{"backslash", between('\\'), `unexpected character '\'`},
		{"hash", between('#'), "unexpected character '#'"},
		{"control", between(0x0001), "unexpected character U+0001"},
		{"delete", between(0x007F), "unexpected character U+007F"},
		{"vertical tab is not whitespace here", between(0x000B), "unexpected character U+000B"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var diags source.Diagnostics
			Tokenize(source.NewFile("t.vs", c.src), &diags)
			got := diags.Render()
			if !strings.Contains(got, c.want) {
				t.Fatalf("wanted %q in:\n%s", c.want, got)
			}
		})
	}
}

// A kind with two spellings names the current one, every time: String once
// ranged over the keyword map, so a message about `this` said 'this' or
// 'self' at random.
func TestKindStringIsDeterministic(t *testing.T) {
	for i := 0; i < 200; i++ {
		if got := KwSelf.String(); got != "'this'" {
			t.Fatalf("KwSelf.String() = %s", got)
		}
	}
}

func strParts(t *testing.T, src string) ([]StringPart, string) {
	t.Helper()
	var diags source.Diagnostics
	toks := Tokenize(source.NewFile("t.vs", src), &diags)
	for _, tk := range toks {
		if tk.Kind == String {
			return tk.Parts, diags.Render()
		}
	}
	return nil, diags.Render()
}

// D172: a text block's margin is the closing delimiter's indentation; the
// breaks next to the delimiters are not text; escapes and interpolation work.
func TestTextBlock(t *testing.T) {
	parts, diags := strParts(t, "val s = \"\"\"\n    a ${x}\n      b\\t\n\n    c\n    \"\"\"\n")
	if diags != "" {
		t.Fatalf("diagnostics: %s", diags)
	}
	if len(parts) != 3 || parts[0].Text != "a " || !parts[1].IsExpr || parts[1].Expr != "x" || parts[2].Text != "\n  b\t\n\nc" {
		t.Fatalf("parts: %#v", parts)
	}
	if p, _ := strParts(t, "val s = \"\"\"\n    \"\"\"\n"); len(p) != 1 || p[0].Text != "" {
		t.Errorf("empty block: %#v", p)
	}
	for src, want := range map[string]string{
		"val s = \"\"\"abc\"\"\"\n":           "starts on the line after the opening",
		"val s = \"\"\"\n  a\n  b\"\"\"\n":    "goes on a line of its own",
		"val s = \"\"\"\n  a\n b\n  \"\"\"\n": "indented less than the closing",
		"val s = \"\"\"\n  a\n":               "unterminated text block",
	} {
		if _, d := strParts(t, src); !strings.Contains(d, want) {
			t.Errorf("%q: wanted %q in:\n%s", src, want, d)
		}
	}
}

// A line break inside a string is "\n" in a CRLF file too, so a program
// prints the same text whatever the checkout's line ends (Windows with
// core.autocrlf printed "\r\n" from a plain string spanning lines).
func TestStringLineBreaksIgnoreCRLF(t *testing.T) {
	for _, src := range []string{
		"val s = \"a\r\nb ${x}\r\nc\"\r\n",
		"val s = \"\"\"\r\n  a\r\n  b ${x}\r\n  c\r\n  \"\"\"\r\n",
	} {
		parts, diags := strParts(t, src)
		if diags != "" || len(parts) != 3 || parts[0].Text != "a\nb " || parts[2].Text != "\nc" {
			t.Errorf("%q: parts %#v, diagnostics %s", src, parts, diags)
		}
	}
}

// D172: a backtick string is raw — no escapes, no interpolation — and may
// span lines.
func TestRawString(t *testing.T) {
	parts, diags := strParts(t, "val s = `C:\\Users\\${x} \"q\"\nnext`\n")
	if diags != "" || len(parts) != 1 || parts[0].Text != "C:\\Users\\${x} \"q\"\nnext" {
		t.Fatalf("parts %#v, diagnostics %s", parts, diags)
	}
	if _, d := strParts(t, "val s = `abc\n"); !strings.Contains(d, "unterminated raw string") {
		t.Errorf("unterminated: %s", d)
	}
}

// D171: the spec starts at the first colon outside brackets, strings and
// character literals, and not the Elvis operator's.
func TestInterpolationSpec(t *testing.T) {
	cases := []struct {
		src, expr, spec string
		has             bool
	}{
		{`"${x:>10.2}"`, "x", ">10.2", true},
		{`"${x:}"`, "x", "", true},
		{`"${f(a: 1):x}"`, "f(a: 1)", "x", true},
		{`"${m.get("a:b"):<5}"`, `m.get("a:b")`, "<5", true},
		{`"${c == ':':^3}"`, "c == ':'", "^3", true},
		{`"${a ?: b}"`, "a ?: b", "", false},
		{`"${[1: 2].len()}"`, "[1: 2].len()", "", false},
		{`"$x:y"`, "x", "", false},
	}
	for _, c := range cases {
		parts, diags := strParts(t, "val s = "+c.src+"\n")
		if diags != "" {
			t.Errorf("%s: %s", c.src, diags)
			continue
		}
		var got *StringPart
		for i := range parts {
			if parts[i].IsExpr {
				got = &parts[i]
			}
		}
		if got == nil || got.Expr != c.expr || got.Spec != c.spec || got.HasSpec != c.has {
			t.Errorf("%s: %#v", c.src, parts)
		}
	}
}

// kindsOf lexes src and lists the token kinds, without the final newline and
// end of file.
func kindsOf(t *testing.T, src string) ([]Token, string) {
	t.Helper()
	var diags source.Diagnostics
	toks := Tokenize(source.NewFile("t.vs", src), &diags)
	var out []Token
	for _, tok := range toks {
		if tok.Kind == EOF || tok.Kind == Semi {
			continue
		}
		out = append(out, tok)
	}
	return out, diags.Render()
}

// TestRegexLiteral pins when a slash starts a regular expression (D182) and
// when it divides: after a token that ends a value it divides, as in JavaScript.
// The Veles front end reproduces the rule (selfhost/lexer).
func TestRegexLiteral(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		texts []string // the tokens that are regular expressions, in order
	}{
		{"after =", `val r = /a+b/`, []string{`/a+b/`}},
		{"flags", `val r = /a/imx`, []string{`/a/imx`}},
		{"after (", `f(/\d+/, 1)`, []string{`/\d+/`}},
		{"after comma", `f(1, /x/)`, []string{`/x/`}},
		{"after return", `return /x/`, []string{`/x/`}},
		{"after =>", `val f = s => /x/`, []string{`/x/`}},
		{"after an operator", `val b = !/x/.test(s)`, []string{`/x/`}},
		{"at the start of a statement", "val a = 1\n/x/.test(s)", []string{`/x/`}},
		{"divides after a name", `val q = a / b / c`, nil},
		{"divides after a number", `val q = 8 / 2`, nil},
		{"divides after )", `val q = (a + b) / 2`, nil},
		{"divides after ]", `val q = xs[0] / 2`, nil},
		{"divides after a string", `val q = "a" / 2`, nil},
		{"divides after this", `val q = this / 2`, nil},
		{"/= after a name", `a /= 2`, nil},
		{"escaped slash", `val r = /a\/b/`, []string{`/a\/b/`}},
		{"slash in a class", `val r = /[/]x/`, []string{`/[/]x/`}},
		{"posix class", `val r = /[[:alpha:]/]+/`, []string{`/[[:alpha:]/]+/`}},
		{"two on a line", `f(/a/, /b/i)`, []string{`/a/`, `/b/i`}},
		{"comment, not a regex", "val a = 1 // x\n", nil},
		{"block comment, not a regex", "val a = /* x */ 1", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			toks, rendered := kindsOf(t, c.src)
			if rendered != "" {
				t.Fatalf("unexpected diagnostics:\n%s", rendered)
			}
			var got []string
			for _, tok := range toks {
				if tok.Kind == Regex {
					got = append(got, tok.Text)
				}
			}
			if strings.Join(got, "|") != strings.Join(c.texts, "|") {
				t.Fatalf("regular expressions = %q, want %q", got, c.texts)
			}
		})
	}
}

func TestRegexLiteralErrors(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"unterminated", "val r = /abc\nval x = 1", "unterminated regular expression"},
		{"ends the file", "val r = /abc", "unterminated regular expression"},
		{"unknown flag", `val r = /a/q`, "unknown regular expression flag 'q'"},
		{"repeated flag", `val r = /a/ii`, "the regular expression flag 'i' is written twice"},
		{"slash in a class is open", `val r = /[a/`, "unterminated regular expression"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, rendered := kindsOf(t, c.src)
			if !strings.Contains(rendered, c.want) {
				t.Fatalf("wanted %q in:\n%s", c.want, rendered)
			}
		})
	}
}

// D191: the compound assignments of the bit and wrapping operators are single
// tokens, matched longest first; the plain operators they start with are
// unchanged.
func TestCompoundAssignmentTokens(t *testing.T) {
	cases := []struct {
		src  string
		want []TokenKind
	}{
		{"a |= b", []TokenKind{Ident, PipeEq, Ident}},
		{"a &= b", []TokenKind{Ident, AmpEq, Ident}},
		{"a ^= b", []TokenKind{Ident, CaretEq, Ident}},
		{"a <<= 2", []TokenKind{Ident, ShlEq, Int}},
		{"a >>= 2", []TokenKind{Ident, ShrEq, Int}},
		{"a +%= b", []TokenKind{Ident, WrapPlusEq, Ident}},
		{"a -%= b", []TokenKind{Ident, WrapMinusEq, Ident}},
		{"a *%= b", []TokenKind{Ident, WrapStarEq, Ident}},
		// the operators themselves, and neighbours that must not change
		{"a | b & c ^ d", []TokenKind{Ident, Pipe, Ident, Amp, Ident, Caret, Ident}},
		{"a << 2 >> 1", []TokenKind{Ident, Shl, Int, Shr, Int}},
		{"a && b || c", []TokenKind{Ident, AndAnd, Ident, OrOr, Ident}},
		{"a +% b", []TokenKind{Ident, WrapPlus, Ident}},
		{"a != b", []TokenKind{Ident, NotEq, Ident}},
	}
	for _, c := range cases {
		var diags source.Diagnostics
		toks := Tokenize(source.NewFile("t.vs", c.src), &diags)
		var got []TokenKind
		for _, tk := range toks {
			if tk.Kind != EOF && tk.Kind != Semi {
				got = append(got, tk.Kind)
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: tokens %v, want %v", c.src, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: token %d is %v, want %v", c.src, i, got[i], c.want[i])
			}
		}
	}
}

// A file that is not UTF-8 does not compile (D199, as Go): every byte that
// does not decode is an error at its offset — in a comment, a string and
// code alike, a surrogate, an overlong form and a truncated sequence
// included — and nothing is lexed, so no token can carry such a byte.
func TestNotUTF8(t *testing.T) {
	src := "// caf\xe9\nval s = \"\xff\"\nval a\xc0\xaf = \xed\xa0\x80 \xe2\x82"
	var diags source.Diagnostics
	toks := Tokenize(source.NewFile("t.vs", src), &diags)
	var at []int
	for _, d := range diags.Items {
		at = append(at, d.Span.Start)
		if d.Span.End != d.Span.Start+1 || !strings.HasPrefix(d.Message, "the text is not valid UTF-8 here (byte 0x") || !strings.HasSuffix(d.Message, "): save the file as UTF-8") {
			t.Errorf("diagnostic %+v", d)
		}
	}
	want := []int{6, 17, 25, 26, 30, 31, 32, 34, 35}
	if fmt.Sprint(at) != fmt.Sprint(want) {
		t.Errorf("errors at %v, want %v", at, want)
	}
	if diags.Items[0].Message != "the text is not valid UTF-8 here (byte 0xE9): save the file as UTF-8" {
		t.Errorf("first message %q", diags.Items[0].Message)
	}
	if len(toks) != 1 || toks[0].Kind != EOF {
		t.Errorf("tokens %v, want only EOF", toks)
	}
}
