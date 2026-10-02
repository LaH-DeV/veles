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
// `xs.at(i)` / `xs.set(i, v)` on lists, `m.get(k)` / `m.set(k, v)` on
// maps. The parser still accepts the
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
		alt = "'" + x + ".get(" + i + ") ?: panic(\"…\")' for a value that must be present"
	} else {
		// `x[i]` never answered null, so the fix keeps that: a panic whose
		// reason the author writes (D62)
		repl = "(" + x + ".at(" + i + ") ?: panic(\"" + panicReasonTODO + "\"))"
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
		get := ".at("
		if isMap {
			get = ".get("
		}
		read := "(" + x + get + i + ") ?: panic(\"" + panicReasonTODO + "\"))"
		repl = x + ".set(" + i + ", " + read + " " + op + " " + v + ")"
	}
	f.c.errorFix(s.Pos, fixReplace("Replace with '"+repl+"'", s.Pos, repl),
		"'%s[%s] %s ...' is not index assignment; brackets are for collection literals only (D25) — use '%s'", x, i, s.Op, repl)
}

// listElemPlace is the element of `x` at `index` as an assignable place
// (`list.ref`): the storage behind `xs.set(i, v)`, `xs.ref(i)` and
// `loop (&x in xs)`. A negative index counts from the end, as in the value
// form; the list is read a second time for the length.
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
// element (`list.ref`, the storage behind `set` and `ref`).
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
		return x.Op == "list.ref" || x.Op == "list.refUnchecked"
	}
	return false
}

// Element reads are values (D25, v0.27): `xs.at(i)`, `m.get(k)`,
// `first()`, `last()` and `find(p)` all copy
// a value struct out of the collection, exactly like binding it to a name
// would. Writing into the collection goes through a reference — `ref` /
// (a pointer to the element, only on MutableList/MutableMap)
// or `loop (&x in xs)` — so that a read and a write never look alike.

// elemReadCall recognises a call that reads an element by value and
// returns the reference form that reaches the same element, for the hint
// on a mutation of the copy; "" when the expression is something else.
func elemReadCall(x ast.Expr) (fixed string, ok bool) {
	call, isCall := x.(*ast.CallExpr)
	if !isCall {
		return "", false
	}
	m, isMember := call.Fun.(*ast.MemberExpr)
	if !isMember {
		return "", false
	}
	var name string
	switch m.Name.Name {
	case "at", "get":
		name = "ref"
	case "atOrPanic", "getOrPanic": // removed (D62), reported on their own
		name = "ref"
	default:
		return "", false
	}
	if len(call.Args) != 1 || call.Args[0].Name != nil {
		return "", false
	}
	recv, arg := srcText(m.X), srcText(call.Args[0].Value)
	if recv == "" || arg == "" {
		return "", false
	}
	sep := "."
	if m.Safe {
		sep = "?."
	}
	return recv + sep + name + "(" + arg + ")", true
}

// copyMutationHint explains a mutation of an element copy and, when the
// receiver is `at` or `get`, attaches the rewrite to
// the reference form as a fix.
func (f *fnCtx) copyMutationHint(recv ast.Expr, span source.Span, what string) {
	if repl, ok := elemReadCall(recv); ok {
		f.c.errorFix(recv.Span(), fixReplace("Replace with '"+repl+"'", recv.Span(), repl),
			"%s: '%s' is a copy of the element, so the change would be lost; reach the element itself with '%s' (D25)", what, srcText(recv), repl)
		return
	}
	f.errorf(span, "%s; bind it with 'var' to change a copy, or reach the element with 'ref' or 'loop (&x in xs)' (D25)", what)
}

// hoistPlace binds every sub-expression that locating target evaluates —
// the collection and index of a `list.ref`, the map and key of a
// `map.refOrPanic`, the pointer under a dereference — to a temporary, so
// that a compound assignment, which reads the place and then writes it,
// evaluates `xs.set(f(), …)` with one call to `f` and one bounds
// check. pre declares the temporaries and runs before the read.
func (f *fnCtx) hoistPlace(target Expr) (Expr, []Stmt) {
	var pre []Stmt
	bind := func(x Expr) Expr {
		switch x.(type) {
		case *VarRef, *IntConst, *FloatConst, *BoolConst, *StringConst:
			return x
		}
		tmp := f.newTemp(x.Type())
		pre = append(pre, &VarDecl{Var: tmp, Init: x})
		return ref(tmp)
	}
	var walk func(x Expr) Expr
	walk = func(x Expr) Expr {
		switch x := x.(type) {
		case *FieldGet:
			n := *x
			n.X = walk(x.X)
			return &n
		case *TupleGet:
			n := *x
			n.X = walk(x.X)
			return &n
		case *Unwrap:
			n := *x
			n.X = walk(x.X)
			return &n
		case *VariantCast:
			n := *x
			n.X = walk(x.X)
			return &n
		case *Deref:
			n := *x
			if b, ok := x.X.(*Builtin); ok && b.Op == "map.refOrPanic" {
				nb := *b
				nb.Args = []Expr{bind(b.Args[0]), bind(b.Args[1])}
				n.X = &nb
			} else {
				n.X = bind(x.X)
			}
			return &n
		case *Builtin:
			if x.Op == "list.ref" || x.Op == "list.refUnchecked" {
				n := *x
				n.Args = []Expr{bind(x.Args[0]), bind(x.Args[1])}
				return &n
			}
		}
		return x
	}
	return walk(target), pre
}

// narrowLValue applies the smart casts in force to a place, so that a
// field write or a method call reaches the payload of a nullable or
// sealed variable after its test (`if (p != null) p.n = 5`), the same way
// a read does (narrowedRef).
func (f *fnCtx) narrowLValue(lv Expr, x ast.Expr) Expr {
	if p, ok := f.placeOf(x); ok {
		return f.narrowPlace(lv, p)
	}
	return lv
}
