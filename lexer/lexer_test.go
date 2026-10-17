package lexer

import (
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
