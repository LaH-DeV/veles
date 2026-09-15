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
		kt, vt, mutable = mt.Key, mt.Value, mt.Mutable
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
			f.errorf(e.Pos, "cannot infer the type of an empty map; annotate it, e.g. 'var m: MutableMap<string, i32> = mut [:]' (D25)")
		} else {
			f.errorf(e.Pos, "cannot infer the type of an empty map; annotate it, e.g. 'val m: Map<string, i32> = [:]' (D25)")
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
			f.errorf(span, "'%s' takes %d argument(s)", name, n)
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
	case "set", "put":
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
	}
	f.errorf(e.Fun.Span(), "no method '%s' on '%s'", name, mt)
	f.checkArgsLoosely(e.Args)
	return bad()
}

func (f *fnCtx) setMethod(recv Expr, st *types.Set, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	need := func(n int) bool {
		if len(e.Args) != n {
			f.errorf(span, "'%s' takes %d argument(s)", name, n)
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
	}
	f.errorf(e.Fun.Span(), "no method '%s' on '%s'", name, st)
	f.checkArgsLoosely(e.Args)
	return bad()
}

// mapIndexAssign lowers `m[k] = v`.
func (f *fnCtx) mapIndexAssign(target *ast.IndexExpr, m Expr, mt *types.Map, value ast.Expr, span source.Span) []Stmt {
	if !mt.Mutable {
		f.errorf(span, "cannot assign into an immutable Map; use MutableMap (D25)")
	}
	k := f.checkExprTo(target.Index, mt.Key)
	v := f.checkExprTo(value, mt.Value)
	return []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "map.set", []Expr{m, k, v}, span}}}
}
