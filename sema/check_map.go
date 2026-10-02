package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Maps and sets (D25) are runtime-backed reference types with an
// immutable/mutable split. Their operations lower to Builtin nodes.

func (f *fnCtx) mapLit(e *ast.MapLit, want types.Type) Expr {
	var kt, vt types.Type
	mutable := false
	if mt, ok := numericHint(want).(*types.Map); ok {
		mutable = mt.Mutable
		if !types.ContainsTypeParam(mt.Key) && !types.ContainsTypeParam(mt.Value) {
			kt, vt = mt.Key, mt.Value
		}
		if e.Mut && mutable {
			f.warnFix(e.Pos, fixDropMut(e.Pos), "redundant 'mut': the expected type '%s' already makes the literal mutable", mt)
		}
	}
	if e.Mut {
		mutable = true
	}
	lit := &MapLit{}
	for _, en := range e.Entries {
		var k, v Expr
		if kt != nil {
			k = f.checkExprTo(en.Key, kt)
		} else {
			k = f.checkExpr(en.Key, nil)
			kt = k.Type()
			f.c.checkHashable(kt, en.Key.Span())
		}
		if vt != nil {
			v = f.checkExprTo(en.Value, vt)
		} else {
			v = f.checkExpr(en.Value, nil)
			vt = v.Type()
		}
		lit.Entries = append(lit.Entries, [2]Expr{k, v})
	}
	if kt == nil {
		if e.Mut {
			f.errorf(e.Pos, "cannot infer the type of an empty map; annotate it, e.g. 'var m: MutableMap<string, i64> = [:]' (D25)")
		} else {
			f.errorf(e.Pos, "cannot infer the type of an empty map; annotate it, e.g. 'val m: Map<string, i64> = [:]' (D25)")
		}
		return bad()
	}
	lit.T = &types.Map{Key: kt, Value: vt, Mutable: mutable}
	return lit
}

// collectionCtor handles `MutableMap<K, V>()`, `Set<T>()`, `MutableList<T>()`.
func (f *fnCtx) collectionCtor(name string, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	if len(e.Args) != 0 {
		f.errorf(e.Pos, "'%s()' takes no arguments; use a literal to construct with contents", name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	var t types.Type
	hint := numericHint(want)
	switch name {
	case "Map", "MutableMap":
		if len(typeArgs) == 2 {
			t = &types.Map{Key: typeArgs[0], Value: typeArgs[1], Mutable: name == "MutableMap"}
		} else if mt, ok := hint.(*types.Map); ok && len(typeArgs) == 0 {
			t = &types.Map{Key: mt.Key, Value: mt.Value, Mutable: name == "MutableMap"}
		}
	case "Set", "MutableSet":
		if len(typeArgs) == 1 {
			t = &types.Set{Elem: typeArgs[0], Mutable: name == "MutableSet"}
		} else if st, ok := hint.(*types.Set); ok && len(typeArgs) == 0 {
			t = &types.Set{Elem: st.Elem, Mutable: name == "MutableSet"}
		}
	case "List", "MutableList":
		if len(typeArgs) == 1 {
			t = &types.List{Elem: typeArgs[0], Mutable: name == "MutableList"}
		} else if lt, ok := hint.(*types.List); ok && len(typeArgs) == 0 {
			t = &types.List{Elem: lt.Elem, Mutable: name == "MutableList"}
		}
	}
	if t == nil {
		f.errorf(e.Pos, "cannot infer the type arguments of '%s'; write '%s<...>()' or annotate the binding", name, name)
		return bad()
	}
	switch tt := t.(type) {
	case *types.Map:
		f.c.checkHashable(tt.Key, e.Pos)
		return &MapLit{exprBase{tt}, nil}
	case *types.Set:
		f.c.checkHashable(tt.Elem, e.Pos)
		return &Builtin{exprBase{tt}, "set.new", nil, e.Pos}
	}
	return &ListLit{exprBase{t}, nil}
}

func isCollectionCtor(name string) bool {
	switch name {
	case "Map", "MutableMap", "Set", "MutableSet", "List", "MutableList":
		return true
	}
	return false
}

func (f *fnCtx) mapMethod(recv Expr, mt *types.Map, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	need := func(n int) bool {
		if len(e.Args) != n {
			f.arityError(span, mt, name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	mutating := func() bool {
		if !mt.Mutable {
			f.errorf(span, "cannot call '%s' on an immutable Map; use MutableMap (D25)", name)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	switch name {
	case "len":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TI64}, "map.len", []Expr{recv}, span}
	case "isEmpty":
		if !need(0) {
			return bad()
		}
		return &Binary{exprBase{types.TBool}, OpEq, &Builtin{exprBase{types.TI64}, "map.len", []Expr{recv}, span}, i64c(0), span}
	case "get":
		if !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		return &Builtin{exprBase{&types.Nullable{Elem: mt.Value}}, "map.get", []Expr{recv, k}, span}
	case "getOrPanic":
		// removed (D62 C); lowered as before so nothing else cascades
		f.removedOrPanic(e, name)
		if len(e.Args) != 1 { // the removal is the one thing to report
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		ptr := &Builtin{exprBase{&types.Pointer{Elem: mt.Value}}, "map.refOrPanic", []Expr{recv, k}, span}
		return &Builtin{exprBase{mt.Value}, "deref", []Expr{ptr}, span}
	case "ref", "refOrPanic":
		// `m.ref(k)`: `(*V)?`, null when absent; `m.refOrPanic(k)`: `*V`, a
		// panic when absent. The pointer reaches the stored value, so a
		// value-struct entry is changed where it lives (D25, v0.27).
		if !mt.Mutable {
			f.errorf(span, "'%s' needs a MutableMap: a pointer into an immutable Map could change it (D25)", name)
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		if !need(1) {
			return bad()
		}
		if name == "refOrPanic" {
			f.removedOrPanic(e, name) // D62 C
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		if name == "ref" {
			return &Builtin{exprBase{&types.Nullable{Elem: &types.Pointer{Elem: mt.Value}}}, "map.ref", []Expr{recv, k}, span}
		}
		return &Builtin{exprBase{&types.Pointer{Elem: mt.Value}}, "map.refOrPanic", []Expr{recv, k}, span}
	case "getOrDefault":
		// `m.getOrDefault(k, d)` is `m.get(k) ?: d`.
		if !need(2) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		d := f.checkExprTo(e.Args[1].Value, mt.Value)
		get := &Builtin{exprBase{&types.Nullable{Elem: mt.Value}}, "map.get", []Expr{recv, k}, span}
		return &Elvis{exprBase{mt.Value}, get, d}
	case "containsKey":
		if !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		return &Builtin{exprBase{types.TBool}, "map.contains", []Expr{recv, k}, span}
	case "keys":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.List{Elem: mt.Key}}, "map.keys", []Expr{recv}, span}
	case "values":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.List{Elem: mt.Value}}, "map.values", []Expr{recv}, span}
	case "entries":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.List{Elem: &types.Tuple{Elems: []types.Type{mt.Key, mt.Value}}}}, "map.entries", []Expr{recv}, span}
	case "toMap", "toMutable":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.Map{Key: mt.Key, Value: mt.Value, Mutable: name == "toMutable"}}, "map.copy", []Expr{recv}, span}
	case "set":
		if !mutating() || !need(2) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		v := f.checkExprTo(e.Args[1].Value, mt.Value)
		return &Builtin{exprBase{types.TUnit}, "map.set", []Expr{recv, k, v}, span}
	case "remove":
		if !mutating() || !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, mt.Key)
		return &Builtin{exprBase{types.TBool}, "map.remove", []Expr{recv, k}, span}
	case "clear":
		if !mutating() || !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TUnit}, "map.clear", []Expr{recv}, span}
	case "reserve":
		// D105: room for n entries in all, so inserting up to n never rehashes
		if !mutating() || !need(1) {
			return bad()
		}
		return &Builtin{exprBase{types.TUnit}, "map.reserve", []Expr{recv, f.checkExprTo(e.Args[0].Value, types.TI64)}, span}
	}
	return nil // not a built-in: a trait impl may provide it
}

func (f *fnCtx) setMethod(recv Expr, st *types.Set, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	need := func(n int) bool {
		if len(e.Args) != n {
			f.arityError(span, st, name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	mutating := func() bool {
		if !st.Mutable {
			f.errorf(span, "cannot call '%s' on an immutable Set; use MutableSet (D25)", name)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	switch name {
	case "len":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TI64}, "map.len", []Expr{recv}, span}
	case "isEmpty":
		if !need(0) {
			return bad()
		}
		return &Binary{exprBase{types.TBool}, OpEq, &Builtin{exprBase{types.TI64}, "map.len", []Expr{recv}, span}, i64c(0), span}
	case "contains":
		if !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, st.Elem)
		return &Builtin{exprBase{types.TBool}, "map.contains", []Expr{recv, k}, span}
	case "toList":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.List{Elem: st.Elem}}, "map.keys", []Expr{recv}, span}
	case "toSet", "toMutable":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.Set{Elem: st.Elem, Mutable: name == "toMutable"}}, "map.copy", []Expr{recv}, span}
	case "add":
		if !mutating() || !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, st.Elem)
		return &Builtin{exprBase{types.TBool}, "set.add", []Expr{recv, k}, span}
	case "remove":
		if !mutating() || !need(1) {
			return bad()
		}
		k := f.checkExprTo(e.Args[0].Value, st.Elem)
		return &Builtin{exprBase{types.TBool}, "map.remove", []Expr{recv, k}, span}
	case "clear":
		if !mutating() || !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TUnit}, "map.clear", []Expr{recv}, span}
	case "reserve":
		// D105: room for n entries in all, so inserting up to n never rehashes
		if !mutating() || !need(1) {
			return bad()
		}
		return &Builtin{exprBase{types.TUnit}, "map.reserve", []Expr{recv, f.checkExprTo(e.Args[0].Value, types.TI64)}, span}
	case "union", "intersect", "difference", "isSubsetOf":
		return f.setAdapter(recv, st, name, e)
	}
	return nil // not a built-in: a trait impl may provide it
}

// setAdapter lowers the set algebra: loops over one set's members testing
// membership in the other.
func (f *fnCtx) setAdapter(recv Expr, st *types.Set, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	if len(e.Args) != 1 {
		f.errorf(span, "'%s' takes 1 argument", name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	a := f.newTemp(st)
	other := f.checkExpr(e.Args[0].Value, nil)
	ot, ok := other.Type().(*types.Set)
	if !ok || !types.Identical(ot.Elem, st.Elem) {
		if !types.IsInvalid(other.Type()) {
			f.errorf(e.Args[0].Value.Span(), "'%s' needs a set of '%s', found '%s'", name, st.Elem, other.Type())
		}
		return bad()
	}
	b := f.newTemp(ot)
	f.pending = nil
	pre := []Stmt{&VarDecl{Var: a, Init: recv}, &VarDecl{Var: b, Init: other}}
	finish := func(stmts []Stmt, value Expr) Expr {
		all := append(pre, f.pending...)
		f.pending = nil
		all = append(all, stmts...)
		return &BlockExpr{exprBase{value.Type()}, &Block{Stmts: all, Value: value, Type: value.Type()}}
	}
	members := func(s *Var) *Var {
		v := f.newTemp(&types.List{Elem: st.Elem})
		pre = append(pre, &VarDecl{Var: v, Init: &Builtin{exprBase{v.Type}, "map.keys", []Expr{ref(s)}, span}})
		return v
	}
	inSet := func(s *Var, x *Var) Expr {
		return &Builtin{exprBase{types.TBool}, "map.contains", []Expr{ref(s), ref(x)}, span}
	}
	outT := &types.Set{Elem: st.Elem, Mutable: true}
	resultT := &types.Set{Elem: st.Elem}
	add := func(out *Var, x *Var) Stmt {
		return &ExprStmt{X: &Builtin{exprBase{types.TBool}, "set.add", []Expr{ref(out), ref(x)}, span}}
	}
	switch name {
	case "union":
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &Builtin{exprBase{outT}, "map.copy", []Expr{ref(a)}, span}}}
		stmts = append(stmts, f.listLoop(members(b), st.Elem, span, func(x *Var, lp *Loop) []Stmt {
			return []Stmt{add(out, x)}
		})...)
		return finish(stmts, &Cast{exprBase{resultT}, ref(out)})
	case "intersect", "difference":
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &Builtin{exprBase{outT}, "set.new", nil, span}}}
		stmts = append(stmts, f.listLoop(members(a), st.Elem, span, func(x *Var, lp *Loop) []Stmt {
			var test Expr = inSet(b, x)
			if name == "difference" {
				test = &Unary{exprBase{types.TBool}, OpNot, test, span}
			}
			return []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, test, &Block{Stmts: []Stmt{add(out, x)}, Type: types.TUnit}, nil}}}
		})...)
		return finish(stmts, &Cast{exprBase{resultT}, ref(out)})
	case "isSubsetOf":
		all := f.newTemp(types.TBool)
		stmts := []Stmt{&VarDecl{Var: all, Init: &BoolConst{exprBase{types.TBool}, true}}}
		stmts = append(stmts, f.listLoop(members(a), st.Elem, span, func(x *Var, lp *Loop) []Stmt {
			miss := &Block{Stmts: []Stmt{&Assign{Target: ref(all), Value: &BoolConst{exprBase{types.TBool}, false}}, &Break{Loop: lp}}, Type: types.TNever}
			return []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, &Unary{exprBase{types.TBool}, OpNot, inSet(b, x), span}, miss, nil}}}
		})...)
		return finish(stmts, ref(all))
	}
	return bad()
}

// mapIndexAssign lowers the removed `m[k] = v` form after it is reported
// (lint_index.go), so nothing cascades.
func (f *fnCtx) mapIndexAssign(target *ast.IndexExpr, m Expr, mt *types.Map, value ast.Expr, span source.Span) []Stmt {
	if !mt.Mutable {
		f.errorf(span, "cannot assign into an immutable Map; use MutableMap (D25)")
	}
	k := f.checkExprTo(target.Index, mt.Key)
	v := f.checkExprTo(value, mt.Value)
	return []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "map.set", []Expr{m, k, v}, span}}}
}
