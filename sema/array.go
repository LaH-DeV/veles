package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `Array<T, N>` methods and loops (D121). An array is a value: assignment
// and passing copy it, `a.set(i, v)` changes the variable it is called on.
// Reads are a list's — `at(i)` is `T?`, and `T` where the index is a
// constant in range (so it compiles to a plain load) — and the read-only
// collection methods run over a copy of the elements.

// arrayPlace is the expression an array method reads: the receiver itself
// when it is a place (reading it twice is harmless), else a temporary
// holding it, so that it is evaluated once.
func (f *fnCtx) arrayPlace(recv Expr) ([]Stmt, Expr) {
	if _, isTable := recv.(*ConstTable); isTable || isPlaceExpr(recv) {
		return nil, recv // read where it is
	}
	tmp := f.newTemp(recv.Type())
	return []Stmt{&VarDecl{Var: tmp, Init: recv}}, ref(tmp)
}

func (f *fnCtx) arrayMethod(recv Expr, at *types.Array, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	n, known := at.N()
	if !known {
		f.errorf(span, "the length of '%s' is not known here (D121)", at)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	need := func(k int) bool {
		if len(e.Args) != k {
			f.arityError(span, at, name, k)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	wrap := func(pre []Stmt, value Expr) Expr {
		if len(pre) == 0 {
			return value
		}
		return &BlockExpr{exprBase{value.Type()}, &Block{Stmts: pre, Value: value, Type: value.Type()}}
	}
	lenOf := func(x Expr) Expr { return &Builtin{exprBase{types.TI64}, "list.len", []Expr{x}, span} }
	switch name {
	case "len":
		if !need(0) {
			return bad()
		}
		return lenOf(recv)
	case "isEmpty":
		if !need(0) {
			return bad()
		}
		return &Binary{exprBase{types.TBool}, OpEq, lenOf(recv), i64c(0), span}
	case "indices":
		if !need(0) {
			return bad()
		}
		// the valid indexes, `0..<N`: a range the bounds facts know (D62)
		return &RangeLit{exprBase{&types.Range{Elem: types.TI64}}, i64c(0), i64c(uint64(n)), false}
	case "toList", "toMutable":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.List{Elem: at.Elem, Mutable: name == "toMutable"}}, "array.toList", []Expr{recv}, span}
	case "at":
		if !need(1) {
			return bad()
		}
		return f.arrayAt(recv, at, e)
	case "first", "last":
		if !need(0) {
			return bad()
		}
		if n == 0 {
			rt := &types.Nullable{Elem: at.Elem}
			return wrap([]Stmt{&ExprStmt{X: recv}}, &NullConst{exprBase{rt}})
		}
		// the array is never empty, so the element is always there
		i := int64(0)
		if name == "last" {
			i = n - 1
		}
		return &Builtin{exprBase{at.Elem}, "list.get", []Expr{recv, i64c(uint64(i))}, span}
	case "set", "setUnchecked":
		m, isMember := e.Fun.(*ast.MemberExpr)
		if !isMember || !need(2) {
			return bad()
		}
		if name == "setUnchecked" {
			f.requireUnsafeIndex(span, name)
		}
		place, _ := f.checkLValue(m.X, true)
		if place == nil {
			f.checkArgsLoosely(e.Args)
			return bad()
		}
		var target Expr
		if name == "set" {
			target = f.listElemPlace(place, &types.List{Elem: at.Elem, Mutable: true}, e.Args[0].Value, span, true)
		} else {
			idx := f.checkExprTo(e.Args[0].Value, types.TI64)
			target = &Builtin{exprBase{at.Elem}, "list.refUnchecked", []Expr{place, idx}, span}
		}
		v := f.checkExprTo(e.Args[1].Value, at.Elem)
		return &BlockExpr{exprBase{types.TUnit}, &Block{Stmts: []Stmt{&Assign{Target: target, Value: v}}, Value: &UnitConst{exprBase{types.TUnit}}, Type: types.TUnit}}
	case "atUnchecked":
		f.requireUnsafeIndex(span, name)
		if !need(1) {
			return bad()
		}
		idx := f.checkExprTo(e.Args[0].Value, types.TI64)
		return &Builtin{exprBase{at.Elem}, "list.refUnchecked", []Expr{recv, idx}, span}
	case "push", "pop", "clear", "reserve", "addFirst", "addLast", "insert", "remove", "removeAt":
		f.errorf(span, "an Array has a fixed length, so '%s' has nothing to do; a MutableList grows and shrinks (D121)", name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	// the compiler's own list operations (`indexOf`, `contains`, …) over a
	// copy of the elements; the prelude's (`map`, `fold`, …) are found next,
	// as the List's, on a copy
	lt := &types.List{Elem: at.Elem}
	switch name {
	case "sorted", "join", "iter":
		// these call into the runtime's list: a list they are given
		recv = &Builtin{exprBase{lt}, "array.toList", []Expr{recv}, span}
	}
	return f.listAdapter(recv, lt, name, e)
}

// arrayAt is `a.at(i)`: `T?`, null when out of range, a negative index
// counting from the end; `T` when the index is a constant in range.
func (f *fnCtx) arrayAt(recv Expr, at *types.Array, e *ast.CallExpr) Expr {
	span := e.Pos
	n, _ := at.N()
	arg := e.Args[0].Value
	idx := f.checkExprTo(arg, types.TI64)
	if types.IsInvalid(idx.Type()) {
		return bad()
	}
	pre, x := f.arrayPlace(recv)
	if k, ok := f.c.tryConst(idx).(*CInt); ok {
		if tbl, isTable := recv.(*ConstTable); isTable && k.V.IsInt64() {
			// a constant array read at a constant index is the element itself (D113)
			elems := tbl.Value.(*CArray).Elems
			if i := k.V.Int64(); i >= -int64(len(elems)) && i < int64(len(elems)) {
				if i < 0 {
					i += int64(len(elems))
				}
				return constExpr(elems[i])
			}
		}
		i := int64(0)
		if k.V.IsInt64() {
			i = k.V.Int64()
		}
		if i < 0 {
			i += n
		}
		if !k.V.IsInt64() || i < 0 || i >= n {
			f.errorf(arg.Span(), "index %s is out of range for '%s', so 'at' is always null (D121)", k.V, at)
			return bad()
		}
		get := &Builtin{exprBase{at.Elem}, "list.get", []Expr{x, i64c(uint64(i))}, span}
		if len(pre) == 0 {
			return get
		}
		return &BlockExpr{exprBase{at.Elem}, &Block{Stmts: pre, Value: get, Type: at.Elem}}
	}
	tmp := f.newTemp(types.TI64)
	pre = append(pre, &VarDecl{Var: tmp, Init: idx})
	nExpr := i64c(uint64(n))
	pre = append(pre, &ExprStmt{X: &If{exprBase{types.TUnit},
		&Binary{exprBase{types.TBool}, OpLt, ref(tmp), i64c(0), span},
		&Block{Stmts: []Stmt{&Assign{Target: ref(tmp), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(tmp), nExpr, span}}}, Type: types.TUnit}, nil}})
	get := &Builtin{exprBase{at.Elem}, "list.get", []Expr{x, ref(tmp)}, span}
	if f.receiverIndexProven(e, arg) || f.indexBelow(arg, n) {
		f.markProven(e)
		return &BlockExpr{exprBase{at.Elem}, &Block{Stmts: pre, Value: get, Type: at.Elem}}
	}
	rt := &types.Nullable{Elem: at.Elem}
	inRange := &Binary{exprBase{types.TBool}, OpAnd,
		&Binary{exprBase{types.TBool}, OpGe, ref(tmp), i64c(0), span},
		&Binary{exprBase{types.TBool}, OpLt, ref(tmp), nExpr, span}, span}
	value := &If{exprBase{rt}, inRange, &Block{Value: &SomeWrap{exprBase{rt}, get}, Type: rt}, &Block{Value: &NullConst{exprBase{rt}}, Type: rt}}
	return &BlockExpr{exprBase{rt}, &Block{Stmts: pre, Value: value, Type: rt}}
}

// arrayLoop lowers `loop (x in a)` (over a copy of the array, since an
// array is a value) and `loop (&x in a)` (a pointer to each element where it
// lives, so the body changes the array itself).
func (f *fnCtx) arrayLoop(s *ast.LoopStmt, iter Expr, it *types.Array, lp *Loop, label string) []Stmt {
	n, _ := it.N()
	byRef := s.Var.Name != nil && s.Var.Ref
	if !byRef && hasRefBinding(s.Var) {
		f.errorf(s.Var.Pos, "'&' binds an array element as a whole: 'loop (&x in a)' (D42)")
	}
	var pre []Stmt
	var base func() Expr
	if byRef {
		place, root := f.checkLValue(s.Iter, true)
		if place == nil {
			return nil
		}
		if root != nil {
			root.AddrTaken = true
			f.invalidatePaths(root)
		}
		ptrT := &types.Pointer{Elem: it}
		ptr := f.newTemp(ptrT)
		pre = append(pre, &VarDecl{Var: ptr, Init: &AddrOf{exprBase{ptrT}, place}})
		base = func() Expr { return &Deref{exprBase{it}, ref(ptr)} }
	} else {
		tmp := f.newTemp(it)
		pre = append(pre, &VarDecl{Var: tmp, Init: iter})
		base = func() Expr { return ref(tmp) }
	}
	idx := f.newTemp(types.TI64)
	pre = append(pre, &VarDecl{Var: idx, Init: i64c(0)})
	lp.Cond = &Binary{exprBase{types.TBool}, OpLt, ref(idx), i64c(uint64(n)), s.Pos}
	lp.Post = []Stmt{&Assign{Target: ref(idx), Value: &Binary{exprBase{types.TI64}, OpWrapAdd, ref(idx), i64c(1), s.Pos}}}
	var elemT types.Type = it.Elem
	if byRef {
		elemT = &types.Pointer{Elem: it.Elem}
	}
	v, parts := f.bindLoopVar(s.Var, elemT)
	f.loops = append(f.loops, &loopFrame{hir: lp, label: label})
	body := f.checkBlock(s.Body, nil, false)
	f.loops = f.loops[:len(f.loops)-1]
	var get Expr = &Builtin{exprBase{it.Elem}, "list.get", []Expr{base(), ref(idx)}, s.Pos}
	if byRef {
		get = &AddrOf{exprBase{elemT}, &Builtin{exprBase{it.Elem}, "list.refUnchecked", []Expr{base(), ref(idx)}, s.Pos}}
	}
	body.Stmts = append(append([]Stmt{&VarDecl{Var: v, Init: get}}, parts...), body.Stmts...)
	lp.Body = body
	return append(pre, lp)
}

// arrayMake is `Array<T, N>.make(value)`: N copies of value.
func (f *fnCtx) arrayMake(at *types.Array, args []ast.Arg, span source.Span) Expr {
	if _, known := at.N(); !known || types.ContainsTypeParam(at.Elem) {
		f.errorf(span, "write the whole type to fill an array: 'Array<i64, 8>.make(0)' (D121)")
		f.checkArgsLoosely(args)
		return bad()
	}
	if len(args) != 1 {
		f.errorf(span, "'make' takes the value every element starts as: 'Array<i64, 8>.make(0)' (D121)")
		f.checkArgsLoosely(args)
		return bad()
	}
	v := f.checkExprTo(args[0].Value, at.Elem)
	return &Builtin{exprBase{at}, "array.make", []Expr{v}, span}
}

// listTemp is the temporary a list adapter reads its receiver through: an
// array's elements are read from a copy of the array (D121).
func (f *fnCtx) listTemp(recv Expr, lt *types.List) *Var {
	if at, ok := recv.Type().(*types.Array); ok {
		return f.newTemp(at)
	}
	return f.newTemp(lt)
}

// arrayMakeCall is `Array<T, N>.make(value)`, or `Array.make(value)` where
// the expected type names the array.
func (f *fnCtx) arrayMakeCall(n *ast.NameExpr, e *ast.CallExpr, want types.Type) Expr {
	var at *types.Array
	if len(n.TypeArgs) > 0 {
		at, _ = f.resolve(&ast.NamedType{Path: []ast.Ident{{Name: "Array", Pos: n.Pos}}, Args: n.TypeArgs, Pos: n.Pos}).(*types.Array)
	} else {
		at, _ = numericHint(want).(*types.Array)
	}
	if at == nil {
		if len(n.TypeArgs) == 0 {
			f.errorf(e.Pos, "cannot tell which array to make; write 'Array<i64, 8>.make(0)', or bind it where the type is written: 'val a: Array<i64, 8> = Array.make(0)' (D121)")
		}
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	return f.arrayMake(at, e.Args, e.Pos)
}

// listToArray is `xs.toArray<N>()`: the elements as an `Array<T, N>`, or
// null when the list does not hold exactly N (D121).
func (f *fnCtx) listToArray(recv Expr, lt *types.List, typeArgs []types.Type, e *ast.CallExpr) Expr {
	if len(e.Args) != 0 {
		f.errorf(e.Pos, "'toArray' takes no arguments; the length is its type argument: 'xs.toArray<4>()' (D121)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if len(typeArgs) != 1 {
		f.errorf(e.Pos, "'toArray' needs the length as its type argument: 'xs.toArray<4>()' (D121)")
		return bad()
	}
	n, ok := typeArgs[0].(*types.Const)
	if !ok {
		if !types.IsInvalid(typeArgs[0]) {
			f.errorf(e.Pos, "'toArray' takes a length, not the type '%s': 'xs.toArray<4>()' (D121)", typeArgs[0])
		}
		return bad()
	}
	at := &types.Array{Elem: lt.Elem, Len: n}
	return &Builtin{exprBase{&types.Nullable{Elem: at}}, "list.toArray", []Expr{recv}, e.Pos}
}

// isArrayOrNullableArray: t is an `Array<T, N>` or an `Array<T, N>?`.
func isArrayOrNullableArray(t types.Type) bool {
	_, ok := numericHint(t).(*types.Array)
	return ok
}
