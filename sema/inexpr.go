package sema

import (
	"strconv"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// D189: `x in c` and `x !in c`.
//
// A list, string or range literal on the right is a set known at compile time
// and becomes comparisons — no list is built, no call made:
//
//	b in [' ', '\t']    is   b == ' ' || b == '\t'
//	c in 'a'..'z'       is   c >= 'a' && c <= 'z'
//	b in "+-*/"         is   b == 43 || b == 45 || b == 42 || b == 47
//
// Anything else is a method call on the container, so every container, effect
// and error rule is the method's: `contains` for a list, set, range or text
// (a substring), `containsKey` for a map, and for a byte in a text the
// prelude's `containsByte`. The operand that is a literal adapts to the other
// one, as it does for `==`.

// maxByteSet is the longest string literal compiled to comparisons; a longer
// one is a call.
const maxByteSet = 16

func (f *fnCtx) inExpr(e *ast.InExpr, want types.Type) Expr {
	res := f.membership(e)
	if e.Not && !types.IsInvalid(res.Type()) {
		return &Unary{exprBase{types.TBool}, OpNot, res, e.Pos}
	}
	return res
}

func (f *fnCtx) membership(e *ast.InExpr) Expr {
	switch c := e.C.(type) {
	case *ast.ListLit:
		if !c.Mut && len(c.Elems) > 0 {
			return f.inEquals(e, c.Elems)
		}
	case *ast.RangeExpr:
		return f.inRange(e, c)
	case *ast.StringLit:
		if text, plain := plainText(c); plain && len(text) > 0 && len(text) <= maxByteSet {
			x := f.checkExpr(e.X, nil)
			if types.IsInteger(x.Type()) {
				bytes := make([]ast.Expr, len(text))
				for i := 0; i < len(text); i++ {
					bytes[i] = &ast.IntLit{Text: strconv.Itoa(int(text[i])), Pos: c.Pos}
				}
				return f.inEqualsOf(e, x, bytes)
			}
			return f.inCall(e, x, f.checkExpr(e.C, nil))
		}
	}
	return f.inGeneral(e)
}

// plainText is the text of a string literal with no interpolation.
func plainText(s *ast.StringLit) (string, bool) {
	text := ""
	for _, p := range s.Parts {
		if p.Expr != nil {
			return "", false
		}
		text += p.Text
	}
	return text, true
}

func (f *fnCtx) inEquals(e *ast.InExpr, elems []ast.Expr) Expr {
	x := f.checkExpr(e.X, nil)
	return f.inEqualsOf(e, x, elems)
}

// inEqualsOf is `x == e1 || x == e2 || …`, with x evaluated once.
func (f *fnCtx) inEqualsOf(e *ast.InExpr, x Expr, elems []ast.Expr) Expr {
	if types.IsInvalid(x.Type()) {
		for _, el := range elems {
			f.checkExpr(el, nil)
		}
		return bad()
	}
	tmp := f.newTemp(x.Type())
	var chain ast.Expr
	for _, el := range elems {
		eq := &ast.BinaryExpr{Op: lexer.Eq, L: f.tempRef(tmp, e), R: el, Pos: el.Span()}
		if chain == nil {
			chain = eq
		} else {
			chain = &ast.BinaryExpr{Op: lexer.OrOr, L: chain, R: eq, Pos: e.Pos}
		}
	}
	return &Let{exprBase{types.TBool}, tmp, x, f.checkExprTo(chain, types.TBool)}
}

// inRange is `x >= lo && x <= hi` (`< hi` for `lo..<hi`), with x evaluated once.
func (f *fnCtx) inRange(e *ast.InExpr, r *ast.RangeExpr) Expr {
	x := f.checkExpr(e.X, nil)
	if types.IsInvalid(x.Type()) {
		f.checkExpr(r.Lo, nil)
		f.checkExpr(r.Hi, nil)
		return bad()
	}
	tmp := f.newTemp(x.Type())
	hiOp := lexer.LtEq
	if !r.Inclusive {
		hiOp = lexer.Lt
	}
	chain := &ast.BinaryExpr{Op: lexer.AndAnd,
		L:   &ast.BinaryExpr{Op: lexer.GtEq, L: f.tempRef(tmp, e), R: r.Lo, Pos: r.Lo.Span()},
		R:   &ast.BinaryExpr{Op: hiOp, L: f.tempRef(tmp, e), R: r.Hi, Pos: r.Hi.Span()},
		Pos: e.Pos}
	return &Let{exprBase{types.TBool}, tmp, x, f.checkExprTo(chain, types.TBool)}
}

func (f *fnCtx) tempRef(v *Var, e *ast.InExpr) ast.Expr {
	return &ast.CheckedExpr{X: &VarRef{exprBase{v.Type}, v}, Pos: e.X.Span()}
}

// inGeneral is the method call. The operand that is a literal is checked
// after the other, against the type it must have.
func (f *fnCtx) inGeneral(e *ast.InExpr) Expr {
	var x, c Expr
	if isLiteralExpr(e.X) && !isLiteralExpr(e.C) {
		c = f.checkExpr(e.C, nil)
		x = f.checkExpr(e.X, memberType(c.Type()))
	} else {
		x = f.checkExpr(e.X, nil)
		c = f.checkExpr(e.C, nil)
	}
	if types.IsInvalid(x.Type()) || types.IsInvalid(c.Type()) {
		return bad()
	}
	return f.inCall(e, x, c)
}

// inCall is `c.contains(x)` — `containsKey` for a map, `containsByte` for a
// byte in a text — checked like any method call.
func (f *fnCtx) inCall(e *ast.InExpr, x, c Expr) Expr {
	if types.IsInvalid(x.Type()) || types.IsInvalid(c.Type()) {
		return bad()
	}
	name := "contains"
	switch t := types.Underlying(c.Type()).(type) {
	case *types.Map:
		name = "containsKey"
	case *types.Basic:
		if types.IsString(t) && types.IsInteger(x.Type()) {
			name = "containsByte"
		}
	}
	member := &ast.MemberExpr{X: &ast.CheckedExpr{X: c, Pos: e.C.Span()}, Name: ast.Ident{Name: name, Pos: e.Pos}, Pos: e.Pos}
	call := &ast.CallExpr{Fun: member, Args: []ast.Arg{{Value: &ast.CheckedExpr{X: x, Pos: e.X.Span()}}}, Pos: e.Pos}
	return f.checkExprTo(call, types.TBool)
}

// memberType is what a container's `contains` takes: the element of a list,
// set, range or array, the key of a map.
func memberType(t types.Type) types.Type {
	switch t := types.Underlying(t).(type) {
	case *types.List:
		return t.Elem
	case *types.Set:
		return t.Elem
	case *types.Range:
		return t.Elem
	case *types.Array:
		return t.Elem
	case *types.Map:
		return t.Key
	}
	return nil
}
