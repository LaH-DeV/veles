package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Bracket indexing was removed from the language (2026-09-18): `[...]` is
// collection-literal syntax only, and reads and writes are methods —
// `xs.at(i)` / `xs.atOrPanic(i)` / `xs.set(i, v)` on lists, `m.get(k)` /
// `m.getOrPanic(k)` / `m.set(k, v)` on maps. The parser still accepts the
// old forms so that they can be reported with a mechanical fix.

// srcText is the source of an expression, for splicing into a fix.
func srcText(e ast.Expr) string {
	sp := e.Span()
	if sp.File == nil || sp.Start < 0 || sp.End > len(sp.File.Content) {
		return ""
	}
	return strings.TrimSpace(sp.File.Content[sp.Start:sp.End])
}

// indexRead reports `x[i]` in value position, with the replacement as a fix.
func (f *fnCtx) indexRead(e *ast.IndexExpr, isMap bool) {
	x, i := srcText(e.X), srcText(e.Index)
	var repl, alt string
	if isMap {
		repl = x + ".get(" + i + ")"
		alt = "'" + x + ".getOrPanic(" + i + ")' for a value that must be present"
	} else {
		repl = x + ".atOrPanic(" + i + ")"
		alt = "'" + x + ".at(" + i + ")' for a nullable read"
	}
	if x == "" || i == "" {
		f.errorf(e.Pos, "'[...]' after a value is not indexing; brackets are for collection literals only (D25)")
		return
	}
	f.c.errorFix(e.Pos, fixReplace("Replace with '"+repl+"'", e.Pos, repl),
		"'%s[%s]' is not indexing; brackets are for collection literals only (D25) — use '%s', or %s", x, i, repl, alt)
}

// indexWrite reports `x[i] = v` (or a compound form), with the replacement as a fix.
func (f *fnCtx) indexWrite(s *ast.AssignStmt, ix *ast.IndexExpr, isMap bool) {
	x, i, v := srcText(ix.X), srcText(ix.Index), srcText(s.Value)
	if x == "" || i == "" || v == "" {
		f.errorf(s.Pos, "'[...] =' is not index assignment; brackets are for collection literals only (D25)")
		return
	}
	var repl string
	if s.Op == lexer.Assign {
		repl = x + ".set(" + i + ", " + v + ")"
	} else {
		op := strings.TrimSuffix(s.Op.String(), "=")
		read := x + ".atOrPanic(" + i + ")"
		if isMap {
			read = x + ".getOrPanic(" + i + ")"
		}
		repl = x + ".set(" + i + ", " + read + " " + op + " " + v + ")"
	}
	f.c.errorFix(s.Pos, fixReplace("Replace with '"+repl+"'", s.Pos, repl),
		"'%s[%s] %s ...' is not index assignment; brackets are for collection literals only (D25) — use '%s'", x, i, s.Op, repl)
}

// listElemPlace is the element of `x` at `index` as an assignable place
// (`list.ref`), the lvalue behind `xs.atOrPanic(i)`. A negative index counts
// from the end, as in the value form; the list is read a second time for
// its length only on that path.
func (f *fnCtx) listElemPlace(x Expr, lt *types.List, index ast.Expr, span source.Span, mutate bool) Expr {
	if mutate && !lt.Mutable {
		f.errorf(span, "cannot assign into an immutable List; use MutableList (D25)")
	}
	idx := f.newTemp(types.TI64)
	n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{x}, span}
	fromEnd := &Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), n, span}}
	adjust := &If{exprBase{types.TUnit},
		&Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(0), span},
		&Block{Stmts: []Stmt{fromEnd}, Type: types.TUnit}, nil}
	at := &BlockExpr{exprBase{types.TI64}, &Block{
		Stmts: []Stmt{&VarDecl{Var: idx, Init: f.indexValue(index)}, &ExprStmt{X: adjust}},
		Value: ref(idx), Type: types.TI64}}
	return &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{x, at}, span}
}

// isPlaceExpr reports whether a checked expression denotes storage that
// can be written through: a dereference, a field of a place, or a list
// element (`list.ref`, the lvalue behind `xs.atOrPanic(i)`).
func isPlaceExpr(x Expr) bool {
	switch x := x.(type) {
	case *Deref, *VarRef:
		return true
	case *FieldGet:
		return isPlaceExpr(x.X)
	case *TupleGet:
		return isPlaceExpr(x.X)
	case *Unwrap:
		return isPlaceExpr(x.X)
	case *VariantCast:
		return isPlaceExpr(x.X)
	case *Builtin:
		return x.Op == "list.ref"
	}
	return false
}

// safePlace is a collection element as the receiver of a `?.` call or
// write: pre runs first (binds the collection and the index or slot), cond
// says whether the element is present, place is the element itself (valid
// only under cond) and value reads it. mutable is the collection's kind: a
// place inside an immutable List or Map is read, never written or handed
// to a `mut fun`.
type safePlace struct {
	pre     []Stmt
	cond    Expr
	place   Expr
	value   Expr
	mutable bool
}

// elemSafePlace recognises `xs.at(i)`, `xs.first()`, `xs.last()` on a list
// and `m.get(k)` on a map as the receiver of a `?.` call or assignment. A
// `?.` call then mutates the element itself rather than the copy `at`/`get`
// returns.
//
// When the shape matches but the call is not on a collection, the receiver
// expression has still been checked; it comes back as recv so the caller
// need not check it twice (a lambda in it would otherwise be emitted twice).
func (f *fnCtx) elemSafePlace(x ast.Expr) (sp *safePlace, recv Expr) {
	call, isCall := x.(*ast.CallExpr)
	if !isCall {
		return nil, nil
	}
	m, isMember := call.Fun.(*ast.MemberExpr)
	if !isMember || m.Safe {
		return nil, nil
	}
	switch m.Name.Name {
	case "at", "get":
		if len(call.Args) != 1 || call.Args[0].Name != nil {
			return nil, nil
		}
	case "first", "last":
		if len(call.Args) != 0 {
			return nil, nil
		}
	default:
		return nil, nil
	}
	// `module.f()` and `Type.f()` are not method calls (callExpr)
	if n, isName := m.X.(*ast.NameExpr); isName {
		if sym := f.lookup(n.Name); sym != nil && (sym.Kind == SymModule || sym.Kind == SymType) {
			return nil, nil
		}
		if f.typeNamed(n) != nil {
			return nil, nil
		}
	}
	if f.moduleTypeNamed(m.X) != nil {
		return nil, nil
	}
	base := f.checkExpr(m.X, nil)
	span := call.Pos
	if mt, isMap := base.Type().(*types.Map); isMap && m.Name.Name == "get" {
		k := f.checkExprTo(call.Args[0].Value, mt.Key)
		slot := f.newTemp(&types.Nullable{Elem: &types.Pointer{Elem: mt.Value}})
		ptr := &Unwrap{exprBase{&types.Pointer{Elem: mt.Value}}, ref(slot)}
		return &safePlace{
			pre:     []Stmt{&VarDecl{Var: slot, Init: &Builtin{exprBase{slot.Type}, "map.ref", []Expr{base, k}, span}}},
			cond:    &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, ref(slot)}, span},
			place:   &Deref{exprBase{mt.Value}, ptr},
			value:   &Deref{exprBase{mt.Value}, ptr},
			mutable: mt.Mutable,
		}, nil
	}
	lt, isList := base.Type().(*types.List)
	if !isList || m.Name.Name == "get" {
		var typeArgs []types.Type
		for _, ta := range call.TypeArgs {
			typeArgs = append(typeArgs, f.resolve(ta))
		}
		if types.IsInvalid(base.Type()) {
			f.checkArgsLoosely(call.Args)
			return nil, bad()
		}
		return nil, f.dispatchMethod(base, m, typeArgs, call, nil)
	}
	list := f.newTemp(lt)
	idx := f.newTemp(types.TI64)
	n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
	pre := []Stmt{&VarDecl{Var: list, Init: base}}
	switch m.Name.Name {
	case "at":
		pre = append(pre, &VarDecl{Var: idx, Init: f.indexValue(call.Args[0].Value)})
		fromEnd := &Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), n, span}}
		pre = append(pre, &ExprStmt{X: &If{exprBase{types.TUnit},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(0), span},
			&Block{Stmts: []Stmt{fromEnd}, Type: types.TUnit}, nil}})
	case "first":
		pre = append(pre, &VarDecl{Var: idx, Init: i64c(0)})
	case "last":
		pre = append(pre, &VarDecl{Var: idx, Init: &Binary{exprBase{types.TI64}, OpWrapSub, n, i64c(1), span}})
	}
	return &safePlace{
		pre: pre,
		cond: &Binary{exprBase{types.TBool}, OpAnd,
			&Binary{exprBase{types.TBool}, OpGe, ref(idx), i64c(0), span},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), n, span}, span},
		place:   &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{ref(list), ref(idx)}, span},
		value:   &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), ref(idx)}, span},
		mutable: lt.Mutable,
	}, nil
}

// narrowLValue applies the smart casts in force to a place, so that a
// field write or a `mut fun` call reaches the payload of a nullable or
// sealed variable after its test (`if (p != null) p.n = 5`), the same way
// a read does (narrowedRef).
func (f *fnCtx) narrowLValue(lv Expr, x ast.Expr) Expr {
	if p, ok := f.placeOf(x); ok {
		return f.narrowPlace(lv, p)
	}
	return lv
}

// mapElemPlace is the value stored under `key` in `m` as an assignable
// place: a dereference of `map.refOrPanic` (which panics when the key is
// absent). It is both the value and the lvalue behind `m.getOrPanic(k)`.
func (f *fnCtx) mapElemPlace(m Expr, mt *types.Map, key ast.Expr, span source.Span, mutate bool) Expr {
	if mutate && !mt.Mutable {
		f.errorf(span, "cannot assign into an immutable Map; use MutableMap (D25)")
	}
	k := f.checkExprTo(key, mt.Key)
	ptr := &Builtin{exprBase{&types.Pointer{Elem: mt.Value}}, "map.refOrPanic", []Expr{m, k}, span}
	return &Deref{exprBase{mt.Value}, ptr}
}

// checkElemWritable reports a write through a place that is rooted in an
// element of an immutable collection (`xs.atOrPanic(i).n = 1` on a List,
// `m.getOrPanic(k).f.g = 1` on a Map). The lvalue chain is walked down to
// the element access; anything behind a pointer is writable (D11).
func (f *fnCtx) checkElemWritable(x Expr, span source.Span) {
	for {
		switch e := x.(type) {
		case *FieldGet:
			x = e.X
		case *TupleGet:
			x = e.X
		case *Unwrap:
			x = e.X
		case *VariantCast:
			x = e.X
		case *Deref:
			if b, ok := e.X.(*Builtin); ok && b.Op == "map.refOrPanic" {
				if mt, ok := b.Args[0].Type().(*types.Map); ok && !mt.Mutable {
					f.errorf(span, "cannot assign into an immutable Map; use MutableMap (D25)")
				}
			}
			return
		case *Builtin:
			if e.Op == "list.ref" {
				if lt, ok := e.Args[0].Type().(*types.List); ok && !lt.Mutable {
					f.errorf(span, "cannot assign into an immutable List; use MutableList (D25)")
				}
			}
			return
		default:
			return
		}
	}
}
