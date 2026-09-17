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

func callFn(fv *Var, args ...Expr) Expr {
	ft := fv.Type.(*types.Func)
	return &CallIndirect{exprBase{ft.Ret}, ref(fv), args}
}

// checkLambdaArg checks a function-valued argument against the expected
// type and binds it to a temporary. Throwing function values are
// rejected: eager adapters run the function inline.
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
		f.errorf(arg.Span(), "expected a function taking %d argument(s), found '%s'", len(expected.Params), ft)
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
		f.errorf(arg.Span(), "a throwing function cannot be passed to an eager collection operation; handle the Result inside it")
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
			f.errorf(span, "'%s' takes %d argument(s)", name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	list := f.newTemp(lt)
	f.pending = nil
	pre := []Stmt{&VarDecl{Var: list, Init: recv}}
	finish := func(stmts []Stmt, value Expr) Expr {
		all := append(pre, f.pending...)
		f.pending = nil
		all = append(all, stmts...)
		return &BlockExpr{exprBase{value.Type()}, &Block{Stmts: all, Value: value, Type: value.Type()}}
	}
	switch name {
	case "map":
		if !need(1) {
			return bad()
		}
		fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}})
		if !ok {
			return bad()
		}
		ut := fv.Type.(*types.Func).Ret
		if types.IsUnit(ut) {
			f.errorf(span, "'map' needs a function that returns a value; use 'forEach' for side effects")
			return bad()
		}
		outT := &types.List{Elem: ut, Mutable: true}
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &ListLit{exprBase{outT}, nil}}}
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			return []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(out), callFn(fv, ref(x))}, span}}}
		})...)
		return finish(stmts, &Cast{exprBase{&types.List{Elem: ut}}, ref(out)})
	case "filter":
		if !need(1) {
			return bad()
		}
		fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}, Ret: types.TBool})
		if !ok {
			return bad()
		}
		outT := &types.List{Elem: lt.Elem, Mutable: true}
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &ListLit{exprBase{outT}, nil}}}
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			push := &Block{Stmts: []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(out), ref(x)}, span}}}, Type: types.TUnit}
			return []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, callFn(fv, ref(x)), push, nil}}}
		})...)
		return finish(stmts, &Cast{exprBase{&types.List{Elem: lt.Elem}}, ref(out)})
	case "forEach":
		if !need(1) {
			return bad()
		}
		fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}, Ret: types.TUnit})
		if !ok {
			return bad()
		}
		stmts := f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			return []Stmt{&ExprStmt{X: callFn(fv, ref(x))}}
		})
		return finish(stmts, &UnitConst{exprBase{types.TUnit}})
	case "fold":
		if !need(2) {
			return bad()
		}
		init := f.checkExpr(e.Args[0].Value, nil)
		if types.IsInvalid(init.Type()) {
			return bad()
		}
		acc := f.newTemp(init.Type())
		fv, ok := f.fnArg(e.Args[1].Value, &types.Func{Params: []types.Param{{Type: init.Type()}, {Type: lt.Elem}}, Ret: init.Type()})
		if !ok {
			return bad()
		}
		stmts := []Stmt{&VarDecl{Var: acc, Init: init}}
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			return []Stmt{&Assign{Target: ref(acc), Value: callFn(fv, ref(acc), ref(x))}}
		})...)
		return finish(stmts, ref(acc))
	case "any", "all":
		if !need(1) {
			return bad()
		}
		fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}, Ret: types.TBool})
		if !ok {
			return bad()
		}
		res := f.newTemp(types.TBool)
		isAny := name == "any"
		stmts := []Stmt{&VarDecl{Var: res, Init: &BoolConst{exprBase{types.TBool}, !isAny}}}
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			cond := callFn(fv, ref(x))
			if !isAny {
				cond = &Unary{exprBase{types.TBool}, OpNot, cond, span}
			}
			hit := &Block{Stmts: []Stmt{&Assign{Target: ref(res), Value: &BoolConst{exprBase{types.TBool}, isAny}}, &Break{Loop: lp}}, Type: types.TNever}
			return []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, cond, hit, nil}}}
		})...)
		return finish(stmts, ref(res))
	case "find", "indexOf":
		if !need(1) {
			return bad()
		}
		var test func(x *Var) Expr
		if name == "find" {
			fv, ok := f.fnArg(e.Args[0].Value, &types.Func{Params: []types.Param{{Type: lt.Elem}}, Ret: types.TBool})
			if !ok {
				return bad()
			}
			test = func(x *Var) Expr { return callFn(fv, ref(x)) }
		} else {
			needle := f.checkExprTo(e.Args[0].Value, lt.Elem)
			if !f.comparable(lt.Elem) {
				f.errorf(span, "elements of type '%s' cannot be compared", lt.Elem)
				return bad()
			}
			nv := f.newTemp(lt.Elem)
			pre = append(pre, &VarDecl{Var: nv, Init: needle})
			test = func(x *Var) Expr { return &Binary{exprBase{types.TBool}, OpEq, ref(x), ref(nv), span} }
		}
		var res *Var
		var found func(x *Var, idx Expr) Expr
		if name == "find" {
			rt := &types.Nullable{Elem: lt.Elem}
			res = f.newTemp(rt)
			found = func(x *Var, _ Expr) Expr { return &SomeWrap{exprBase{rt}, ref(x)} }
		} else {
			res = f.newTemp(types.TI64)
			found = func(_ *Var, idx Expr) Expr { return idx }
		}
		var init Expr = &NullConst{exprBase{res.Type}}
		if name == "indexOf" {
			init = &IntConst{exprBase{types.TI64}, 1, true}
		}
		stmts := []Stmt{&VarDecl{Var: res, Init: init}}
		idxVar := f.newTemp(types.TI64)
		stmts = append(stmts, &VarDecl{Var: idxVar, Init: i64c(0)})
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			hit := &Block{Stmts: []Stmt{&Assign{Target: ref(res), Value: found(x, ref(idxVar))}, &Break{Loop: lp}}, Type: types.TNever}
			return []Stmt{
				&ExprStmt{X: &If{exprBase{types.TUnit}, test(x), hit, nil}},
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
			f.errorf(span, "cannot order by '%s'; keys must be numbers, strings or implement 'Comparable' (D48: use a comparator otherwise)", keyT)
			return bad()
		}
		key := func(x Expr) Expr {
			if keyFn == nil {
				return x
			}
			return &CallIndirect{exprBase{keyT}, ref(keyFn), []Expr{x}}
		}
		// insertion sort on a mutable copy
		outT := &types.List{Elem: lt.Elem, Mutable: true}
		out := f.newTemp(outT)
		stmts := []Stmt{&VarDecl{Var: out, Init: &Builtin{exprBase{outT}, "list.copy", []Expr{ref(list)}, span}}}
		i := f.newTemp(types.TI64)
		j := f.newTemp(types.TI64)
		cur := f.newTemp(lt.Elem)
		curKey := f.newTemp(keyT)
		f.c.nextLoop++
		outer := &Loop{ID: f.c.nextLoop}
		f.c.nextLoop++
		inner := &Loop{ID: f.c.nextLoop}
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(out)}, span}
		getOut := func(idx Expr) Expr {
			return &Builtin{exprBase{lt.Elem}, "list.get", []Expr{ref(out), idx}, span}
		}
		setOut := func(idx Expr, v Expr) Stmt {
			return &Assign{Target: &Builtin{exprBase{lt.Elem}, "list.ref", []Expr{ref(out), idx}, span}, Value: v}
		}
		prev := &Binary{exprBase{types.TI64}, OpWrapSub, ref(j), i64c(1), span}
		// inner: loop (j > 0 && key(out[j-1]) > curKey) { out[j] = out[j-1]; j -= 1 }
		inner.Cond = &Binary{exprBase{types.TBool}, OpAnd,
			&Binary{exprBase{types.TBool}, OpGt, ref(j), i64c(0), span},
			f.greater(key(getOut(prev)), ref(curKey), span), span}
		inner.Body = &Block{Stmts: []Stmt{
			setOut(ref(j), getOut(prev)),
			&Assign{Target: ref(j), Value: prev},
		}, Type: types.TUnit}
		outer.Cond = &Binary{exprBase{types.TBool}, OpLt, ref(i), n, span}
		outer.Post = []Stmt{&Assign{Target: ref(i), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(i), i64c(1), span}}}
		outer.Body = &Block{Stmts: []Stmt{
			&VarDecl{Var: cur, Init: getOut(ref(i))},
			&VarDecl{Var: curKey, Init: key(ref(cur))},
			&VarDecl{Var: j, Init: ref(i)},
			inner,
			setOut(ref(j), ref(cur)),
		}, Type: types.TUnit}
		stmts = append(stmts, &VarDecl{Var: i, Init: i64c(1)}, outer)
		return finish(stmts, &Cast{exprBase{&types.List{Elem: lt.Elem, Mutable: false}}, ref(out)})
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
		acc := f.newTemp(types.TString)
		stmts := []Stmt{&VarDecl{Var: acc, Init: &StringConst{exprBase{types.TString}, ""}}}
		idxVar := f.newTemp(types.TI64)
		stmts = append(stmts, &VarDecl{Var: idxVar, Init: i64c(0)})
		stmts = append(stmts, f.listLoop(list, lt.Elem, span, func(x *Var, lp *Loop) []Stmt {
			first := &Binary{exprBase{types.TBool}, OpEq, ref(idxVar), i64c(0), span}
			withSep := &Block{Stmts: []Stmt{&Assign{Target: ref(acc), Value: &StringConcat{exprBase{types.TString}, []Expr{ref(acc), ref(sepV)}}}}, Type: types.TUnit}
			return []Stmt{
				&ExprStmt{X: &If{exprBase{types.TUnit}, &Unary{exprBase{types.TBool}, OpNot, first, span}, withSep, nil}},
				&Assign{Target: ref(acc), Value: &StringConcat{exprBase{types.TString}, []Expr{ref(acc), f.toString(ref(x), span)}}},
				&Assign{Target: ref(idxVar), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idxVar), i64c(1), span}},
			}
		})...)
		return finish(stmts, ref(acc))
	}
	return nil
}

// greater builds `l > r` for an orderable type: a native comparison for
// numbers and strings, `compareTo` for a Comparable type.
func (f *fnCtx) greater(l, r Expr, span source.Span) Expr {
	if cmp := f.compareOp(OpGt, l, r, span); cmp != nil {
		return cmp
	}
	return &Binary{exprBase{types.TBool}, OpGt, l, r, span}
}
