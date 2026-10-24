package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// D207: a block on the same line after a call or a name is the call's last
// argument; `{ … }` alone is a block lambda; `fun on T(A): R` is a type.
func TestReceiverBlocks(t *testing.T) {
	cases := map[string]string{
		"x.apply { port = 1 }":       "(call (. x apply) (block-lambda (block\n  (= port 1))))",
		"configure(&s) { port = 1 }": "(call configure (& s), (block-lambda (block\n  (= port 1))))",
		"measure { work() }":         "(call measure (block-lambda (block\n  (call work))))",
		"f({ a() })":                 "(call f (block-lambda (block\n  (call a))))",
		"x.edit { trim() }.len()":    "(call (. (call (. x edit) (block-lambda (block\n  (call trim)))) len))",
	}
	for src, want := range cases {
		diags := &source.Diagnostics{}
		e := ParseExprString(src, diags)
		if diags.HasErrors() {
			t.Errorf("%s: %s", src, diags.Render())
			continue
		}
		if got := ast.DumpExpr(e); got != want {
			t.Errorf("%s:\n  got  %q\n  want %q", src, got, want)
		}
		if c, ok := e.(*ast.CallExpr); ok && src != "f({ a() })" && !strings.Contains(src, ".len()") && !c.Trailing {
			t.Errorf("%s: not marked trailing", src)
		}
	}
	// a block on the next line is a statement of its own
	diags := &source.Diagnostics{}
	f := ParseFile(source.NewFile("t.vs", "fun f() {\n  g()\n  {\n    h()\n  }\n}\n"), diags)
	body := f.Decls[0].(*ast.FunDecl).Body
	if len(body.Stmts) < 2 {
		t.Fatalf("a block on the next line was taken as an argument: %d statements", len(body.Stmts))
	}
	if call, ok := body.Stmts[0].(*ast.ExprStmt).X.(*ast.CallExpr); !ok || call.Trailing {
		t.Errorf("g() took the next line's block")
	}
	// the receiver type
	diags = &source.Diagnostics{}
	f = ParseFile(source.NewFile("t.vs", "fun build(body: fun on StringBuilder(i64): bool) { }\n"), diags)
	if diags.HasErrors() {
		t.Fatal(diags.Render())
	}
	ft := f.Decls[0].(*ast.FunDecl).Params[0].Type.(*ast.FunType)
	if ft.On == nil || len(ft.Params) != 1 || ft.Ret == nil {
		t.Errorf("fun on T(A): R parsed as %+v", ft)
	}
}
