package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
)

// D198: `indirect` and `inline` before a variant in a sealed trait's body,
// and `indirect` before `sealed trait`; elsewhere the words are names.
func TestBoxedVariantWords(t *testing.T) {
	src := "indirect sealed trait Shape {\n  struct Circle {\n    r: f64\n  }\n}\n" +
		"sealed trait Expr {\n  indirect struct Add {\n    a: i64\n  }\n  inline struct Hole { }\n}\n" +
		"fun f(indirect: i64, inline: i64): i64 => indirect + inline\n"
	f, diags := parse(t, src)
	if diags.HasErrors() {
		t.Fatal(diags.Render())
	}
	var shape, expr *ast.TraitDecl
	for _, d := range f.Decls {
		if td, ok := d.(*ast.TraitDecl); ok {
			switch td.Name.Name {
			case "Shape":
				shape = td
			case "Expr":
				expr = td
			}
		}
	}
	if shape == nil || !shape.Indirect || expr == nil || expr.Indirect {
		t.Fatalf("indirect on the trait: %+v %+v", shape, expr)
	}
	if !expr.Variants[0].Indirect || expr.Variants[0].Inline || !expr.Variants[1].Inline || expr.Variants[1].Indirect {
		t.Errorf("indirect/inline on the variants: %+v %+v", expr.Variants[0], expr.Variants[1])
	}
	dump := ast.Dump(f)
	for _, want := range []string{"(trait indirect sealed Shape", "(struct indirect Add", "(struct inline Hole"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}
