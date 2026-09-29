package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D95: `if (val x = e && ...)` binds the non-null value of a nullable e for
// the rest of the condition and the then-branch. The checker lowers it to
// what already exists: a Let that evaluates e into a variable and tests it
// against null, and a fact that narrows the variable to its non-null type
// (condFacts), so code generation sees nothing new.

// letConds are the `val x = e` operands of the top-level `&&` chain of a
// condition, in order.
func letConds(cond ast.Expr) []*ast.LetCond {
	switch c := cond.(type) {
	case *ast.LetCond:
		return []*ast.LetCond{c}
	case *ast.BinaryExpr:
		if c.Op == lexer.AndAnd {
			return append(letConds(c.L), letConds(c.R)...)
		}
	}
	return nil
}

// letCond checks one binding of an `if` condition.
func (f *fnCtx) letCond(e *ast.LetCond) Expr {
	if !f.letOK[e] {
		f.errorf(e.Pos, "'val %s = ...' is only allowed in the condition of an 'if', joined to the rest by '&&' (D95)", e.Name.Name)
		f.checkExpr(e.Value, nil)
		return bad()
	}
	init := f.checkExpr(e.Value, nil)
	t := init.Type()
	if types.IsInvalid(t) {
		// declare the name anyway, so its uses do not each report it unknown
		f.declareLocal(e.Name.Name, f.newVar(e.Name.Name, types.TInvalid, false, e.Name.Pos), e.Name.Pos)
		return bad()
	}
	if _, ok := t.(*types.Nullable); !ok {
		f.errorf(e.Value.Span(), "'val %s = ...' in a condition needs a value that can be null, and this is a '%s'; a plain 'val' binds it (D95)", e.Name.Name, t)
		f.declareLocal(e.Name.Name, f.newVar(e.Name.Name, types.TInvalid, false, e.Name.Pos), e.Name.Pos)
		return bad()
	}
	v := f.newVar(e.Name.Name, t, false, e.Name.Pos)
	f.declareChecked(e.Name.Name, v, e.Name.Pos)
	f.letVars[e] = v
	// `x != null`, spelled on invalid spans: the binding is not a reference
	// to itself, and the test is not a read that would count as a use
	none := source.Span{}
	test := f.checkExprTo(&ast.BinaryExpr{
		Op:  lexer.NotEq,
		L:   &ast.NameExpr{Name: e.Name.Name, Pos: none},
		R:   &ast.NullLit{Pos: none},
		Pos: e.Pos,
	}, types.TBool)
	v.used = false
	return &Let{exprBase{types.TBool}, v, init, test}
}
