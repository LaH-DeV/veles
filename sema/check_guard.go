package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The failure forms (D30 amended 2026-09-25, spec entry D61): the value a
// nullable or a Result carries, or what to do instead.
//
//	x ?: fallback            T?  — the value, or the fallback (D30)
//	r ?? fallback            Result — the value, or the fallback
//	r ?? { e => ... }        Result — the value, or what the handler yields
//	val v = x else { ... }   T? or Result — bind the value, or leave
//	val P(f) = x else { ... }  a pattern — bind its fields, or leave
//
// `?:` and `??` are one idea split by the left side's type, each with an
// error and a fix naming the other, so the operator says which kind of
// "maybe" is being unwrapped.

// coalesceExpr checks `r ?? fallback` and `r ?? { e => ... }`.
func (f *fnCtx) coalesceExpr(e *ast.CoalesceExpr, want types.Type) Expr {
	left := e.L
	if t, isTry := e.L.(*ast.TryExpr); isTry {
		// `try f() ?? 0`: the `try` would propagate the very error `??` is
		// there to handle; one error with the fix, and the rest checked as
		// if it were not written
		cut := source.Span{File: t.Pos.File, Start: t.Pos.Start, End: t.X.Span().Start}
		f.c.errorFix(cut, fixReplace("Remove 'try'", cut, ""), "'??' handles the error itself; drop the 'try': '%s ?? …'", srcText(t.X))
		left = t.X
	}
	l := f.checkExpr(left, nil)
	lt := l.Type()
	if types.IsInvalid(lt) {
		f.checkFallbackLoosely(e)
		return bad()
	}
	if _, isNullable := lt.(*types.Nullable); isNullable {
		op := operatorSpan(e.L, "??")
		f.c.errorFix(op, fixReplace("Use '?:'", op, "?:"), "'??' is for a Result; a nullable's value-or-fallback is '?:': '%s ?: fallback'", srcText(e.L))
		f.checkFallbackLoosely(e)
		return bad()
	}
	rs, ok := lt.(*types.Sealed)
	if !ok || !isResultType(rs) {
		f.errorf(e.Pos, "'??' needs a Result on its left (a call to a 'throws' function), found '%s'", lt)
		f.checkFallbackLoosely(e)
		return bad()
	}
	okT := rs.TypeArgs[0]
	tmp := f.newTemp(lt)
	okV, errV := rs.Variants[0], rs.Variants[1]
	ref := &VarRef{exprBase{lt}, tmp}
	failed := &VariantTest{exprBase{types.TBool}, ref, errV}
	payload := &FieldGet{exprBase{okT}, &VariantCast{exprBase{okV}, ref, okV}, 0, okV.Fields[0].Name}

	var fallback *Block
	if e.Handler != nil {
		errValue := &FieldGet{exprBase{errV.Fields[0].Type}, &VariantCast{exprBase{errV}, ref, errV}, 0, errV.Fields[0].Name}
		fallback = f.handlerBlock(e.Handler, errValue, okT, false)
	} else {
		x := f.checkExprTo(e.R, okT)
		fallback = f.valueBlock(x)
	}
	pick := &If{exprBase{okT}, failed, fallback, &Block{Value: payload, Type: okT}}
	return &Let{exprBase{okT}, tmp, l, pick}
}

// checkFallbackLoosely checks the right side of a `??` whose left side was
// wrong, so its own mistakes are still reported.
func (f *fnCtx) checkFallbackLoosely(e *ast.CoalesceExpr) {
	if e.R != nil {
		f.checkExpr(e.R, nil)
	}
}

// handlerBlock checks a `{ e => ... }` handler with the error bound to
// its name. With leave set it must diverge (a let-else); otherwise it
// yields a T or diverges (a `??`).
func (f *fnCtx) handlerBlock(h *ast.Handler, errValue Expr, t types.Type, leave bool) *Block {
	f.pushScope()
	defer f.popScope()
	var decl Stmt
	if h.Err != nil {
		if errValue == nil {
			f.errorf(h.Err.Pos, "only a Result has an error to bind; write 'else { ... }' without '%s =>'", h.Err.Name)
		} else if h.Err.Name != "_" {
			v := f.newVar(h.Err.Name, errValue.Type(), false, h.Err.Pos)
			f.declareLocal(h.Err.Name, v, h.Err.Pos)
			decl = &VarDecl{Var: v, Init: errValue}
		}
	}
	var b *Block
	if leave {
		b = f.checkBlock(h.Body, types.TUnit, false)
		if !types.IsNever(b.Type) {
			f.errorf(h.Pos, "the 'else' of a 'val ... else' must leave — end it with 'return', 'break', 'continue', 'throw' or 'panic' — since the names it would bind do not exist on this path")
		}
	} else {
		b = f.checkBlock(h.Body, t, true)
	}
	if decl != nil {
		b.Stmts = append([]Stmt{decl}, b.Stmts...)
	}
	return b
}

// letElse checks `val binding = x else { ... }` and `val P(f) = x else { ... }`.
func (f *fnCtx) letElse(s *ast.ValStmt) []Stmt {
	mutable := s.Kind == ast.BindVar
	if s.Kind == ast.BindConst {
		f.errorf(s.Pos, "'const' is only allowed at module level; use 'val' for a local immutable binding")
	}
	x := f.checkExpr(s.Value, nil)
	xt := x.Type()
	if types.IsInvalid(xt) {
		return f.brokenBinding(s)
	}
	tmp := f.newTemp(xt)
	ref := &VarRef{exprBase{xt}, tmp}
	out := []Stmt{&VarDecl{Var: tmp, Init: x}}

	var failed Expr
	var errValue, payload Expr
	var binds []Stmt
	switch {
	case s.Pattern != nil:
		test, pbinds, irrefutable := f.compilePattern(s.Pattern, ref, xt, s.Pattern.Span())
		if irrefutable || test == nil {
			f.errorf(s.Pattern.Span(), "this pattern always matches, so the 'else' can never run; drop it")
			return f.brokenBinding(s)
		}
		failed = &Unary{exprBase{types.TBool}, OpNot, test, s.Pattern.Span()}
		binds = pbinds
	default:
		switch t := xt.(type) {
		case *types.Nullable:
			failed = &IsNull{exprBase{types.TBool}, ref}
			payload = &Unwrap{exprBase{t.Elem}, ref}
		case *types.Sealed:
			if isResultType(t) {
				okV, errV := t.Variants[0], t.Variants[1]
				failed = &VariantTest{exprBase{types.TBool}, ref, errV}
				payload = &FieldGet{exprBase{t.TypeArgs[0]}, &VariantCast{exprBase{okV}, ref, okV}, 0, okV.Fields[0].Name}
				errValue = &FieldGet{exprBase{errV.Fields[0].Type}, &VariantCast{exprBase{errV}, ref, errV}, 0, errV.Fields[0].Name}
			}
		}
		if failed == nil {
			f.errorf(s.Value.Span(), "nothing here can fail: 'val ... else' needs a nullable, a Result or a pattern, found '%s'; drop the 'else'", xt)
			return f.brokenBinding(s)
		}
	}

	// the else branch, where none of the names below exist
	saved := f.saveNarrow()
	elseBlock := f.handlerBlock(s.Else, errValue, nil, true)
	f.restoreNarrow(saved)
	out = append(out, &ExprStmt{X: &If{exprBase{types.TUnit}, failed, elseBlock, nil}})

	if s.Pattern != nil {
		for _, b := range binds {
			if vd, ok := b.(*VarDecl); ok {
				vd.Var.Mutable = mutable
				f.declareLocal(vd.Var.Name, vd.Var, vd.Var.Span)
			}
		}
		return append(out, binds...)
	}

	// the value is there: bind it as an ordinary `val` would
	b := s.Binding
	if b.Name != nil {
		init := payload
		if b.Type != nil {
			init = f.coerce(payload, f.resolve(b.Type), s.Value.Span())
		}
		v := f.newVar(b.Name.Name, init.Type(), mutable, b.Name.Pos)
		v.InitText = srcText(s.Value)
		f.declareChecked(b.Name.Name, v, b.Name.Pos)
		return append(out, &VarDecl{Var: v, Init: init})
	}
	tt, ok := payload.Type().(*types.Tuple)
	if !ok || len(tt.Elems) != len(b.Tuple) {
		f.errorf(b.Pos, "cannot destructure a value of type '%s' into %d names (D37)", payload.Type(), len(b.Tuple))
		return out
	}
	whole, parts := f.bindPattern(&b, tt, mutable)
	return append(append(out, &VarDecl{Var: whole, Init: payload}), parts...)
}

// operatorSpan is the span of op written right after the expression l —
// the operator of `l op r`, which the syntax tree does not keep.
func operatorSpan(l ast.Expr, op string) source.Span {
	sp := l.Span()
	if sp.File == nil {
		return sp
	}
	rest := sp.File.Content[sp.End:]
	i := strings.Index(rest, op)
	if i < 0 {
		return sp
	}
	return source.Span{File: sp.File, Start: sp.End + i, End: sp.End + i + len(op)}
}

// brokenBinding declares the names a let-else that failed to check would
// have bound, with the invalid type, so each use of them is not a second
// "unknown name" error on top of the one that was reported.
func (f *fnCtx) brokenBinding(s *ast.ValStmt) []Stmt {
	var declare func(b *ast.Binding)
	declare = func(b *ast.Binding) {
		if b.Name != nil {
			f.declareLocal(b.Name.Name, f.newVar(b.Name.Name, types.TInvalid, false, b.Name.Pos), b.Name.Pos)
		}
		for i := range b.Tuple {
			declare(&b.Tuple[i])
		}
	}
	if s.Pattern == nil {
		declare(&s.Binding)
	}
	return nil
}
