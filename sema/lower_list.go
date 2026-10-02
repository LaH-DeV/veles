package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Eager collection operations (D46) are lowered to loops in the HIR, so
// they need no runtime support and work for every element type.

func i64c(v uint64) Expr { return &IntConst{exprBase{types.TI64}, v, false} }

func ref(v *Var) Expr { return &VarRef{exprBase{v.Type}, v} }

// listLoop builds `var i = 0; loop (i < len(list)) { val x = list[i]; body; i += 1 }`
// and returns the statements; body receives the element variable.
func (f *fnCtx) listLoop(list *Var, elem types.Type, span source.Span, body func(x *Var, lp *Loop) []Stmt) []Stmt {
	idx := f.newTemp(types.TI64)
	x := f.newTemp(elem)
	f.c.nextLoop++
	lp := &Loop{ID: f.c.nextLoop}
	lenExpr := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
	lp.Cond = &Binary{exprBase{types.TBool}, OpLt, ref(idx), lenExpr, span}
	lp.Post = []Stmt{&Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), i64c(1), span}}}
	get := &Builtin{exprBase{elem}, "list.get", []Expr{ref(list), ref(idx)}, span}
	inner := []Stmt{&VarDecl{Var: x, Init: get}}
	inner = append(inner, body(x, lp)...)
	lp.Body = &Block{Stmts: inner, Type: types.TUnit}
	return []Stmt{&VarDecl{Var: idx, Init: i64c(0)}, lp}
}

// fnArg checks a function-valued argument against the expected
// type and binds it to a temporary. Throwing function values are
// rejected: what is lowered here (sortedBy's key) runs the function inline.
func (f *fnCtx) fnArg(arg ast.Expr, expected *types.Func) (*Var, bool) {
	x := f.checkExpr(arg, expected)
	ft, ok := x.Type().(*types.Func)
	if !ok {
		if !types.IsInvalid(x.Type()) {
			f.errorf(arg.Span(), "expected a function of type '%s', found '%s'", expected, x.Type())
		}
		return nil, false
	}
	if len(ft.Params) != len(expected.Params) {
		f.errorf(arg.Span(), "expected a function taking %d %s, found '%s'", len(expected.Params), plural(len(expected.Params), "argument"), ft)
		return nil, false
	}
	for i := range ft.Params {
		if !types.Identical(ft.Params[i].Type, expected.Params[i].Type) {
			f.errorf(arg.Span(), "expected a function of type '%s', found '%s'", expected, ft)
			return nil, false
		}
	}
	if expected.Ret != nil && !types.Identical(ft.Ret, expected.Ret) {
		f.errorf(arg.Span(), "expected a function returning '%s', found '%s'", expected.Ret, ft.Ret)
		return nil, false
	}
	if ft.Effects.Throws {
		f.errorf(arg.Span(), "a throwing function cannot be passed here; handle the Result inside it")
		return nil, false
	}
	v := f.newTemp(ft)
	f.pending = append(f.pending, &VarDecl{Var: v, Init: x})
	return v, true
}

// listAdapter lowers the higher-order List methods.
func (f *fnCtx) listAdapter(recv Expr, lt *types.List, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	need := func(n int) bool {
		if len(e.Args) != n {
			f.arityError(span, lt, name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	list := f.newTemp(lt)
	// an enclosing adapter may have pending declarations (a key function
	// bound by fnArg) and be checking a call this switch does not handle
	// (`sortedBy` rewritten to `sortedWith`): hand them back on the way out
	savedPending := f.pending
	f.pending = nil
	defer func() {
		if f.pending == nil {
			f.pending = savedPending
		}
	}()
	pre := []Stmt{&VarDecl{Var: list, Init: recv}}
	finish := func(stmts []Stmt, value Expr) Expr {
		all := append(pre, f.pending...)
		f.pending = nil
		all = append(all, stmts...)
		return &BlockExpr{exprBase{value.Type()}, &Block{Stmts: all, Value: value, Type: value.Type()}}
	}
	switch name {
	case "indexOf":
		if !need(1) {
			return bad()
		}
		needle := f.checkExprTo(e.Args[0].Value, lt.Elem)
		if !f.comparable(lt.Elem) {
			f.errorf(span, "elements of type '%s' cannot be compared with '=='; 'find' takes a test of your own", lt.Elem)
			return bad()
		}
		nv := f.newTemp(lt.Elem)
		pre = append(pre, &VarDecl{Var: nv, Init: needle})
		res := f.newTemp(types.TI64)
		stmts := []Stmt{&VarDecl{Var: res, Init: &IntConst{exprBase{types.TI64}, 1, true}}}
		idxVar := f.newTemp(types.TI64)
		stmts = append(stmts, &VarDecl{Var: idxVar, Init: i64c(0)})
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			hit := &Block{Stmts: []Stmt{&Assign{Target: ref(res), Value: ref(idxVar)}, &Break{Loop: lp}}, Type: types.TNever}
			return []Stmt{
				&ExprStmt{X: &If{exprBase{types.TUnit}, &Binary{exprBase{types.TBool}, OpEq, ref(x), ref(nv), span}, hit, nil}},
				&Assign{Target: ref(idxVar), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idxVar), i64c(1), span}},
			}
		})...)
		return finish(stmts, ref(res))
	case "contains":
		if !need(1) {
			return bad()
		}
		inner := f.listAdapter(ref(list), lt, "indexOf", e)
		if types.IsInvalid(inner.Type()) {
			return inner
		}
		return finish(nil, &Binary{exprBase{types.TBool}, OpGe, inner, i64c(0), span})
	case "at":
		// `xs.at(i)` is the checked read: `T?`, null when out of range. A
		// negative index counts from the end: `xs.at(-1)` is the last element.
		if !need(1) {
			return bad()
		}
		if f.receiverIndexProven(e, e.Args[0].Value) {
			// the facts put the index in range (D62): a `T`, not a `T?`
			f.markProven(e)
			return finish(f.uncheckedGet(list, lt, e.Args[0].Value, span))
		}
		idx := f.newTemp(types.TI64)
		pre = append(pre, &VarDecl{Var: idx, Init: f.checkExprTo(e.Args[0].Value, types.TI64)})
		rt := &types.Nullable{Elem: lt.Elem}
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
		fromEnd := &Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), n, span}}
		pre = append(pre, &ExprStmt{X: &If{exprBase{types.TUnit},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(0), span},
			&Block{Stmts: []Stmt{fromEnd}, Type: types.TUnit}, nil}})
		inRange := &Binary{exprBase{types.TBool}, OpAnd,
			&Binary{exprBase{types.TBool}, OpGe, ref(idx), i64c(0), span},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), n, span}, span}
		get := &SomeWrap{exprBase{rt}, &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), ref(idx)}, span}}
		return finish(nil, &If{exprBase{rt}, inRange, &Block{Value: get, Type: rt}, &Block{Value: &NullConst{exprBase{rt}}, Type: rt}})
	case "atOrPanic":
		// removed (D62 C); lowered as before so nothing else cascades
		f.removedOrPanic(e, name)
		if len(e.Args) != 1 { // the removal is the one thing to report
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		return finish(f.uncheckedGet(list, lt, e.Args[0].Value, span))
	case "ref", "refOrPanic":
		// `xs.ref(i)`: `(*T)?`, null when out of range; `xs.refOrPanic(i)`:
		// `*T`, a panic when out of range. A pointer to the element itself,
		// so a value struct is changed where it lives (D25, v0.27); only on
		// a MutableList, since writing through it would change the list.
		if !lt.Mutable {
			f.errorf(span, "'%s' needs a MutableList: a pointer into an immutable List could change it (D25)", name)
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		if !need(1) {
			return bad()
		}
		pt := &types.Pointer{Elem: lt.Elem}
		if name == "refOrPanic" {
			f.removedOrPanic(e, name) // D62 C
		}
		if name == "refOrPanic" || f.receiverIndexProven(e, e.Args[0].Value) {
			if name == "ref" {
				f.markProven(e)
			}
			place := f.listElemPlace(ref(list), lt, e.Args[0].Value, span, true)
			return finish(nil, &AddrOf{exprBase{pt}, place})
		}
		idx := f.newTemp(types.TI64)
		pre = append(pre, &VarDecl{Var: idx, Init: f.indexValue(e.Args[0].Value)})
		rt := &types.Nullable{Elem: pt}
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
		fromEnd := &Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), n, span}}
		pre = append(pre, &ExprStmt{X: &If{exprBase{types.TUnit},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(0), span},
			&Block{Stmts: []Stmt{fromEnd}, Type: types.TUnit}, nil}})
		inRange := &Binary{exprBase{types.TBool}, OpAnd,
			&Binary{exprBase{types.TBool}, OpGe, ref(idx), i64c(0), span},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), n, span}, span}
		elem := &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{ref(list), ref(idx)}, span}
		some := &SomeWrap{exprBase{rt}, &AddrOf{exprBase{pt}, elem}}
		return finish(nil, &If{exprBase{rt}, inRange, &Block{Value: some, Type: rt}, &Block{Value: &NullConst{exprBase{rt}}, Type: rt}})
	case "set":
		// `xs.set(i, v)` replaces the element at `i`; a panic when out of range.
		if !lt.Mutable {
			f.errorf(span, "cannot set an element of an immutable List; use MutableList (D25)")
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		if !need(2) {
			return bad()
		}
		place := f.listElemPlace(ref(list), lt, e.Args[0].Value, span, true)
		v := f.checkExprTo(e.Args[1].Value, lt.Elem)
		return finish([]Stmt{&Assign{Target: place, Value: v}}, &UnitConst{exprBase{types.TUnit}})
	case "atUnchecked", "setUnchecked":
		// D114: the local escape from the bounds check. Only in `unsafe`;
		// a debug build still checks (and panics), a release build does not.
		// No negative-from-end index: `i` is the plain offset.
		if name == "setUnchecked" && !lt.Mutable {
			f.errorf(span, "cannot set an element of an immutable List; use MutableList (D25)")
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		f.requireUnsafeIndex(span, name)
		if !need(map[string]int{"atUnchecked": 1, "setUnchecked": 2}[name]) {
			return bad()
		}
		idx := f.checkExprTo(e.Args[0].Value, types.TI64)
		elem := &Builtin{exprBase{lt.Elem}, "list.refUnchecked", []Expr{ref(list), idx}, span}
		if name == "atUnchecked" {
			return finish(nil, elem)
		}
		v := f.checkExprTo(e.Args[1].Value, lt.Elem)
		return finish([]Stmt{&Assign{Target: elem, Value: v}}, &UnitConst{exprBase{types.TUnit}})
	case "first", "last":
		if !need(0) {
			return bad()
		}
		rt := &types.Nullable{Elem: lt.Elem}
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
		var idx Expr = i64c(0)
		if name == "last" {
			idx = &Binary{exprBase{types.TI64}, OpWrapSub, n, i64c(1), span}
		}
		if m, ok := e.Fun.(*ast.MemberExpr); ok && f.indexProven(m.X, &ast.IntLit{Text: "0"}) {
			f.markProven(e)
			// the list is known not to be empty (D62)
			return finish(nil, &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), idx}, span})
		}
		empty := &Binary{exprBase{types.TBool}, OpEq, n, i64c(0), span}
		get := &SomeWrap{exprBase{rt}, &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), idx}, span}}
		return finish(nil, &If{exprBase{rt}, empty, &Block{Value: &NullConst{exprBase{rt}}, Type: rt}, &Block{Value: get, Type: rt}})
	case "reversed":
		if !need(0) {
			return bad()
		}
		outT := &types.List{Elem: lt.Elem, Mutable: true}
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &ListLit{exprBase{outT}, nil}}}
		idx := f.newTemp(types.TI64)
		f.c.nextLoop++
		lp := &Loop{ID: f.c.nextLoop}
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
		stmts = append(stmts, &VarDecl{Var: idx, Init: &Binary{exprBase{types.TI64}, OpWrapSub, n, i64c(1), span}})
		lp.Cond = &Binary{exprBase{types.TBool}, OpGe, ref(idx), i64c(0), span}
		lp.Post = []Stmt{&Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapSub, ref(idx), i64c(1), span}}}
		get := &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), ref(idx)}, span}
		lp.Body = &Block{Stmts: []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(out), get}, span}}}, Type: types.TUnit}
		stmts = append(stmts, lp)
		return finish(stmts, &Cast{exprBase{&types.List{Elem: lt.Elem, Mutable: false}}, ref(out)})
	case "sortedBy", "sorted":
		var keyFn *Var
		var keyT types.Type = lt.Elem
		if name == "sortedBy" {
			if !need(1) {
				return bad()
			}
			fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}})
			if !ok {
				return bad()
			}
			keyFn = fv
			keyT = fv.Type.(*types.Func).Ret
		} else if !need(0) {
			return bad()
		}
		if !f.ordered(keyT) {
			var fix *source.Fix
			if cmp := f.c.traitNamed("Comparable"); cmp != nil {
				fix = f.c.implementFix(keyT, cmp) // derived: field by field, in order (D58)
			}
			f.c.errorFix(span, fix, "cannot order by '%s'; keys must be numbers, strings or implement 'Comparable' (D48: use a comparator otherwise)", keyT)
			return bad()
		}
		if keyFn == nil && (types.IsInteger(lt.Elem) || types.Identical(lt.Elem, types.TString)) {
			// equal integers or strings are indistinguishable, so the order
			// among equals cannot be seen: the runtime sorts a copy with
			// direct comparisons (the comparator call was most of the cost)
			return finish(nil, &Builtin{exprBase{&types.List{Elem: lt.Elem}}, "list.sortedNative", []Expr{ref(list)}, span})
		}
		// delegate to the prelude's stable merge sort (list.vs sortedWith)
		// with the natural comparison as the comparator: every ordered type
		// answers `compareTo` (prelude impls for numbers and strings, the
		// synthesized one for tuples, D48), so the lowering is
		// `list.sortedWith(($a, $b) => key($a).compareTo(key($b)))`
		f.scope.Insert(&Symbol{Name: list.Name, Kind: SymLocal, Var: list})
		var ka, kb ast.Expr = &ast.NameExpr{Name: "$a", Pos: span}, &ast.NameExpr{Name: "$b", Pos: span}
		if keyFn != nil {
			f.scope.Insert(&Symbol{Name: keyFn.Name, Kind: SymLocal, Var: keyFn})
			ka = &ast.CallExpr{Fun: nameOf(keyFn, span), Args: []ast.Arg{{Value: ka}}, Pos: span}
			kb = &ast.CallExpr{Fun: nameOf(keyFn, span), Args: []ast.Arg{{Value: kb}}, Pos: span}
		}
		cmp := &ast.CallExpr{Fun: &ast.MemberExpr{X: ka, Name: ast.Ident{Name: "compareTo", Pos: span}, Pos: span}, Args: []ast.Arg{{Value: kb}}, Pos: span}
		lambda := &ast.LambdaExpr{Params: []ast.Param{{Name: ast.Ident{Name: "$a", Pos: span}, Pos: span}, {Name: ast.Ident{Name: "$b", Pos: span}, Pos: span}}, Body: cmp, Pos: span}
		call := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(list, span), Name: ast.Ident{Name: "sortedWith", Pos: span}, Pos: span}, Args: []ast.Arg{{Value: lambda}}, Pos: span}
		return finish(nil, f.checkExpr(call, nil))
	case "iter":
		if !need(0) {
			return bad()
		}
		call := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(list, span), Name: ast.Ident{Name: "iterator", Pos: span}, Pos: span}, Pos: span}
		f.scope.Insert(&Symbol{Name: list.Name, Kind: SymLocal, Var: list})
		return finish(nil, f.checkExpr(call, nil))
	case "join":
		if !need(1) {
			return bad()
		}
		sep := f.checkExprTo(e.Args[0].Value, types.TString)
		sepV := f.newTemp(types.TString)
		pre = append(pre, &VarDecl{Var: sepV, Init: sep})
		// the texts first, then one allocation for the result: appending to
		// an accumulator copied it whole each time, quadratic in its length
		// (80 000 numbers took 38 s)
		if types.Identical(lt.Elem, types.TString) {
			return finish(nil, &Builtin{exprBase{types.TString}, "list.joinText", []Expr{ref(list), ref(sepV)}, span})
		}
		textsT := &types.List{Elem: types.TString, Mutable: true}
		texts := f.newTemp(textsT)
		stmts := []Stmt{&VarDecl{Var: texts, Init: &ListLit{exprBase{textsT}, nil}}}
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			return []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(texts), f.toString(ref(x), span)}, span}}}
		})...)
		return finish(stmts, &Builtin{exprBase{types.TString}, "list.joinText", []Expr{ref(texts), ref(sepV)}, span})
	}
	return nil
}

// listFilterIs lowers `xs.filterIs<Variant>()` on a list of a sealed type: the
// elements of that variant, as a `List<Variant>`. It is `filter(x => x is V)`
// with the promise kept in the type — a predicate cannot narrow across the
// call boundary, a type argument can (D13).
func (f *fnCtx) listFilterIs(recv Expr, lt *types.List, typeArgs []types.Type, e *ast.CallExpr) Expr {
	span := e.Pos
	if len(e.Args) != 0 {
		f.errorf(span, "'filterIs' takes no arguments; the variant is its type argument: 'xs.filterIs<Circle>()'")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	sealed, isSealed := lt.Elem.(*types.Sealed)
	if !isSealed {
		f.errorf(span, "'filterIs' works on a list of a sealed type, not 'List<%s>'; a list of 'T?' has 'filterNotNull()'", lt.Elem)
		return bad()
	}
	if len(typeArgs) != 1 {
		f.errorf(span, "'filterIs' needs one type argument, a variant of '%s': 'xs.filterIs<%s>()'", sealed.Name, firstVariantName(sealed))
		return bad()
	}
	variant, ok := typeArgs[0].(*types.Struct)
	if !ok || variant.Sealed == nil || !types.Identical(variant.Sealed, sealed) {
		f.errorf(span, "'%s' is not a variant of '%s'", typeArgs[0], sealed)
		return bad()
	}
	list := f.newTemp(lt)
	outT := &types.List{Elem: variant, Mutable: true}
	out := f.newTemp(outT)
	stmts := []Stmt{&VarDecl{Var: list, Init: recv}, &VarDecl{Var: out, Init: &ListLit{exprBase{outT}, nil}}}
	stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
		test := &VariantTest{exprBase{types.TBool}, ref(x), variant}
		push := &Block{Stmts: []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(out), &VariantCast{exprBase{variant}, ref(x), variant}}, span}}}, Type: types.TUnit}
		return []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, test, push, nil}}}
	})...)
	result := &Cast{exprBase{&types.List{Elem: variant}}, ref(out)}
	return &BlockExpr{exprBase{result.Type()}, &Block{Stmts: stmts, Value: result, Type: result.Type()}}
}

// uncheckedGet is the read behind `atOrPanic` and a proven `at` (D62): the
// element as a value, a negative index counted from the end; the runtime
// still panics out of range.
// requireUnsafeIndex refuses an unchecked access (D114) outside `unsafe`,
// naming the checked spelling to use instead.
func (f *fnCtx) requireUnsafeIndex(span source.Span, name string) {
	if f.unsafe > 0 {
		return
	}
	checked := map[string]string{"atUnchecked": "at(i)", "setUnchecked": "set(i, v)", "byteAtUnchecked": "byteAt(i)"}[name]
	f.errorf(span, "'%s' skips the bounds check in a release build, so it needs an 'unsafe' block (D114); write '%s', or wrap the call in 'unsafe { }' with a '// SAFETY:' comment saying why the index is in range", name, checked)
}

func (f *fnCtx) uncheckedGet(list *Var, lt *types.List, index ast.Expr, span source.Span) ([]Stmt, Expr) {
	idx := f.newTemp(types.TI64)
	n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(list)}, span}
	fromEnd := &Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), n, span}}
	stmts := []Stmt{
		&VarDecl{Var: idx, Init: f.indexValue(index)},
		&ExprStmt{X: &If{exprBase{types.TUnit},
			&Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(0), span},
			&Block{Stmts: []Stmt{fromEnd}, Type: types.TUnit}, nil}},
	}
	return stmts, &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(list), ref(idx)}, span}
}
