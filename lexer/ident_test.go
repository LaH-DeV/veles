package lexer

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// TestUnicodeIdentifiers: identifiers follow UAX #31 (D18, 2026-09-25).
// Letters from any script start one; combining marks and digits continue
// one; nothing invisible or bidirectional belongs to one.
func TestUnicodeIdentifiers(t *testing.T) {
	for _, name := range []string{"café", "日本語", "µs", "naïve", "x\u0304", "λ2", "_x", "Ωmega", "ℕ"} {
		var diags source.Diagnostics
		toks := Tokenize(source.NewFile("t.vs", "val "+name+" = 1"), &diags)
		if diags.HasErrors() || len(toks) < 2 || toks[1].Kind != Ident || toks[1].Text != name {
			t.Errorf("%q: wanted one identifier, got %v\n%s", name, toks, diags.Render())
		}
	}
}

// TestInvisibleCharacters: what UAX #31 keeps out of identifiers is
// reported by name, since the reader cannot see it.
func TestInvisibleCharacters(t *testing.T) {
	for _, c := range []struct {
		r    rune
		want string
	}{
		{0x00A0, "U+00A0, a no-break space; use an ordinary space"},
		{0x200B, "U+200B, an invisible character"},
		{0x202E, "U+202E, a bidirectional control character"},
		{0x2066, "U+2066, a bidirectional control character"},
		{0xFEFF, "U+FEFF, a byte-order mark, which may only begin a file"},
		{0x20AC, "'€' (U+20AC)"},
	} {
		var diags source.Diagnostics
		Tokenize(source.NewFile("t.vs", between(c.r)), &diags)
		if got := diags.Render(); !strings.Contains(got, "unexpected character "+c.want) {
			t.Errorf("U+%04X: wanted %q in:\n%s", c.r, c.want, got)
		}
	}
}

// TestTrojanSource: the CVE-2021-42574 shapes. A bidirectional control in
// a comment is an error; in a string it is a warning that suggests the
// escape; a byte-order mark at the very start of a file is allowed.
func TestTrojanSource(t *testing.T) {
	var diags source.Diagnostics
	Tokenize(source.NewFile("t.vs", "val ok = 1 // is admin \u202E \u2066 end\n/* \u202D */ val y = 2"), &diags)
	if n := diags.ErrorCount(); n != 3 || !strings.Contains(diags.Render(), "a comment contains U+202E") {
		t.Errorf("wanted three comment errors, got %d:\n%s", n, diags.Render())
	}

	diags = source.Diagnostics{}
	Tokenize(source.NewFile("t.vs", "val s = \"abc\u202Edef\""), &diags)
	if diags.HasErrors() || !strings.Contains(diags.Render(), "write it as the escape \\u{202E}") {
		t.Errorf("wanted a warning with the escape:\n%s", diags.Render())
	}

	diags = source.Diagnostics{}
	toks := Tokenize(source.NewFile("t.vs", "\uFEFFval x = 1"), &diags)
	if diags.HasErrors() || toks[0].Kind != KwVal {
		t.Errorf("a leading byte-order mark should be skipped:\n%v\n%s", toks, diags.Render())
	}
}
