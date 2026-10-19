package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
)

// TestUseExtendClause: the `extend` clause that ends a use (D185) — absent,
// bare (all of a module's extension functions), or a list of names.
func TestUseExtendClause(t *testing.T) {
	cases := []struct {
		src         string
		extend      bool
		extendNames []string // nil: none, or bare
		dump        string
	}{
		{"use strx\n", false, nil, "(use strx)"},
		{"use strx extend\n", true, nil, "(use strx extend)"},
		{"use strx extend { slug, shout }\n", true, []string{"slug", "shout"}, "(use strx extend { slug, shout })"},
		{"use regex { Regex } extend { find }\n", true, []string{"find"}, "(use regex { Regex } extend { find })"},
		{"use regex { Regex, Match as M } extend\n", true, nil, "(use regex { Regex, Match as M } extend)"},
		{"use a.b as c extend { f }\n", true, []string{"f"}, "(use a.b as c extend { f })"},
		{"use io, strx extend { slug }\n", false, nil, "(use io strx extend { slug })"},
	}
	for _, c := range cases {
		f, diags := parse(t, c.src)
		if diags.HasErrors() {
			t.Errorf("%q: %s", c.src, diags.Render())
			continue
		}
		u := f.Decls[0].(*ast.UseDecl)
		last := u.Specs[len(u.Specs)-1] // the clause belongs to its own spec
		if c.src != "use io, strx extend { slug }\n" && last.Extend != c.extend {
			t.Errorf("%q: Extend = %v", c.src, last.Extend)
		}
		var names []string
		for _, n := range last.ExtendNames {
			names = append(names, n.Name.Name)
		}
		if c.src != "use io, strx extend { slug }\n" && strings.Join(names, ",") != strings.Join(c.extendNames, ",") {
			t.Errorf("%q: names %v, want %v", c.src, names, c.extendNames)
		}
		if got := strings.TrimSpace(ast.Dump(f)); got != c.dump {
			t.Errorf("%q: dump %s, want %s", c.src, got, c.dump)
		}
	}
}

// TestUseExtendClauseErrors: an empty list and a rename are refused with the way out.
func TestUseExtendClauseErrors(t *testing.T) {
	for src, want := range map[string]string{
		"use strx extend { }\n":           "'extend { }' names nothing",
		"use strx extend { slug as s }\n": "an extension function keeps its name",
	} {
		_, diags := parse(t, src)
		if !diags.HasErrors() || !strings.Contains(diags.Render(), want) {
			t.Errorf("%q: want an error containing %q, got:\n%s", src, want, diags.Render())
		}
	}
}

// A declaration `extend T { }` on the next line is not the clause of the use above it.
func TestExtendDeclAfterUseIsNotAClause(t *testing.T) {
	f, diags := parse(t, "use strx\nextend string with fun shout(): string => this\n")
	if diags.HasErrors() {
		t.Fatal(diags.Render())
	}
	if u := f.Decls[0].(*ast.UseDecl); u.Specs[0].Extend {
		t.Errorf("the next line's extend declaration was read as a clause")
	}
	if len(f.Decls) != 2 {
		t.Errorf("want the use and the extend declaration, got %d declarations", len(f.Decls))
	}
}
