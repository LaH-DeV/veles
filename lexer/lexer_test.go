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
		{"backtick", between('`'), "unexpected character '`'"},
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

// TestAboveAsciiIsAnIdentifier is the reason TestUnexpectedCharacter has no
// non-ASCII rows: every byte at or above 0x80 starts an identifier (D18),
// so nothing above ASCII ever reaches the "unexpected character" path — not
// a letter, and not an invisible one either. The Veles lexer has to agree,
// and this is where that expectation is written down.
func TestAboveAsciiIsAnIdentifier(t *testing.T) {
	for _, r := range []rune{
		0x00E9, // é, a letter
		0x00A0, // no-break space
		0x200B, // zero width space
		0x202E, // right-to-left override
		0xFEFF, // byte order mark, in the middle of a file
	} {
		var diags source.Diagnostics
		toks := Tokenize(source.NewFile("t.vs", between(r)), &diags)
		if out := diags.Render(); strings.Contains(out, "unexpected character") {
			t.Fatalf("U+%04X: wanted no complaint, got:\n%s", r, out)
		}
		found := false
		for _, tok := range toks {
			if tok.Kind == Ident && tok.Text == string(r) {
				found = true
			}
		}
		if !found {
			t.Fatalf("U+%04X: wanted it lexed as an identifier, got %v", r, toks)
		}
	}
}
