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
	case *Deref:
		return true
	case *FieldGet:
		return true
	case *Builtin:
		return x.Op == "list.ref"
	}
	return false
}

// listElemSafePlace recognises `xs.at(i)`, `xs.first()` and `xs.last()` on
// a list as the receiver of a `?.` call and returns the element as a place
// guarded by its bounds check: pre binds the list and the index, inRange is
// the condition, place is `list.ref` (valid only under inRange). A `?.`
// call then mutates the element itself rather than the copy `at` returns.
//
// When the shape matches but the call is not on a list, the receiver
// expression has still been checked; it comes back as recv so the caller
// need not check it twice (a lambda in it would otherwise be emitted twice).
func (f *fnCtx) listElemSafePlace(x ast.Expr) (pre []Stmt, inRange Expr, place Expr, recv Expr, ok bool) {
	call, isCall := x.(*ast.CallExpr)
	if !isCall {
		return nil, nil, nil, nil, false
	}
	m, isMember := call.Fun.(*ast.MemberExpr)
	if !isMember || m.Safe {
		return nil, nil, nil, nil, false
	}
	switch m.Name.Name {
	case "at":
		if len(call.Args) != 1 || call.Args[0].Name != nil {
			return nil, nil, nil, nil, false
		}
	case "first", "last":
		if len(call.Args) != 0 {
			return nil, nil, nil, nil, false
		}
	default:
		return nil, nil, nil, nil, false
	}
	// `module.f()` and `Type.f()` are not method calls (callExpr)
	if n, isName := m.X.(*ast.NameExpr); isName {
		if sym := f.lookup(n.Name); sym != nil && (sym.Kind == SymModule || sym.Kind == SymType) {
			return nil, nil, nil, nil, false
		}
		if f.typeNamed(n) != nil {
			return nil, nil, nil, nil, false
		}
	}
	if f.moduleTypeNamed(m.X) != nil {
		return nil, nil, nil, nil, false
	}
	base := f.checkExpr(m.X, nil)
	lt, isList := base.Type().(*types.List)
	if !isList {
		var typeArgs []types.Type
		for _, ta := range call.TypeArgs {
			typeArgs = append(typeArgs, f.resolve(ta))
		}
		if types.IsInvalid(base.Type()) {
			f.checkArgsLoosely(call.Args)
			return nil, nil, nil, bad(), false
		}
		return nil, nil, nil, f.dispatchMethod(base, m, typeArgs, call, nil), false
	}
	span := call.Pos
	list := f.newTemp(lt)
	idx := f.newTemp(types.TI64)
	n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
	pre = []Stmt{&VarDecl{Var: list, Init: base}}
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
	inRange = &Binary{exprBase{types.TBool}, OpAnd,
		&Binary{exprBase{types.TBool}, OpGe, ref(idx), i64c(0), span},
		&Binary{exprBase{types.TBool}, OpLt, ref(idx), n, span}, span}
	place = &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{ref(list), ref(idx)}, span}
	return pre, inRange, place, nil, true
}
