package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `apply` and `edit` (D207): a block run on a receiver.
//
//	val slug = name.edit({ trim(); toLower(); replace(" ", "-") })
//	server.apply({ port = 8080 })
//
// Both are lowered here, with the block called directly on a variable of
// the caller's, so nothing is allocated but the one copy `edit` makes:
//
//   - edit: a working copy of the value — a string's bytes in a
//     StringBuilder, a List's elements in a MutableList (likewise Map, Set),
//     a struct or any other value as it is — the block run on it, and the
//     copy given back: the builder's bytes shared as the string, the
//     MutableList frozen into the List (no second copy).
//   - apply: the block run on the caller's own place — a `var`, a `var`
//     field, what a pointer points at, a temporary — and the place's new
//     value given back; on a `val` it is refused with a fix to `edit`. A
//     mutable collection (a handle) is changed in place wherever it is held.
//     A string, List, Map or Set is replaced by its edited copy.
//
// The block's `this` points at that variable. Whether the block can keep
// the pointer is known once every body is checked (onCalls, the receiver
// pass): only then does the variable move to the heap.

// onCall is a variable a function on a receiver was given a pointer to:
// the function (block), or the block's body inlined (body, with self its
// `this`). The variable is heap-allocated unless the function is a lambda,
// or the body one, that provably does not keep its `this`.
type onCall struct {
	v     *Var
	block Expr
	body  *Block
	self  *Var
}

// applyEdit checks `x.apply { … }` or `x.edit { … }` where x's type has
// no method of that name.
func (f *fnCtx) applyEdit(recv Expr, callee *ast.MemberExpr, e *ast.CallExpr) Expr {
	name := callee.Name.Name
	rt := recv.Type()
	if len(e.Args) != 1 || e.Args[0].Name != nil || e.Args[0].Spread {
		f.errorf(e.Pos, "'%s' takes one block: 'x.%s { … }' (D207)", name, name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if f.c.taskHolding(rt) {
		f.errorf(e.Pos, "'%s' holds a running task, which a copy would duplicate; '%s' is refused for task-holding types (D111, D207)", rt, name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	work := f.workForm(rt)
	if work == nil {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	block := &blockRunner{work: work, name: name}
	if l, ok := e.Args[0].Value.(*ast.LambdaExpr); ok && l.Bare {
		block.inline = l // checked where it runs, in this function
	} else if block.value = f.checkOnBlock(e.Args[0].Value, work, name); block.value == nil {
		return bad()
	}
	inPlace := types.Identical(work, rt)
	if name == "edit" {
		if r, _ := f.heldResource(callee.X); r != nil {
			f.errorf(e.Pos, "'%s' is bound by 'with', which closes it when the block ends; a copy would be a second handle to the same resource (D100, D207)", srcText(callee.X))
			return bad()
		}
		return f.editCopy(recv, rt, work, block, e.Pos)
	}
	// apply: on a mutable collection, the handle reaches the same elements
	// wherever it is held — a copy of it is the place
	if isReferenceType(rt) || !isPlaceExpr(recv) || inPacked(recv) {
		if inPlace {
			return f.runOn(recv, rt, block, false, nil, e.Pos)
		}
		return f.editCopy(recv, rt, work, block, e.Pos) // a temporary's own copy
	}
	if ok, why := placeAssignable(recv); !ok {
		fix := &source.Fix{Title: "Use 'edit', which gives a changed copy", Edits: []source.TextEdit{{Span: callee.Name.Pos, NewText: "edit"}}}
		f.c.errorFix(callee.Name.Pos, fix, "'apply' changes '%s' in place, and %s; 'edit' gives a changed copy: 'val changed = %s.edit { … }' (D207)", srcText(callee.X), why, srcText(callee.X))
		return bad()
	}
	root := rootVar(recv)
	if root != nil {
		f.invalidateVarPaths(root)
	}
	if inPlace {
		return f.runOn(recv, rt, block, true, root, e.Pos)
	}
	// a string, List, Map or Set: the place gets its edited copy
	if _, isVar := recv.(*VarRef); isVar {
		edited := f.editCopy(recv, rt, work, block, e.Pos)
		return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{&Assign{Target: recv, Value: edited, Pos: e.Pos}}, Value: recv, Type: rt}}
	}
	at := f.newTemp(&types.Pointer{Elem: rt})
	place := &Deref{exprBase{rt}, ref(at)}
	edited := f.editCopy(place, rt, work, block, e.Pos)
	return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{
		&VarDecl{Var: at, Init: &AddrOf{exprBase{at.Type}, recv}},
		&Assign{Target: place, Value: edited, Pos: e.Pos},
	}, Value: &Deref{exprBase{rt}, ref(at)}, Type: rt}}
}

// blockRunner is the block of an apply or edit: a literal block, checked
// where it runs — in the caller, `this` the place, with no closure made —
// or a receiver function value, called.
type blockRunner struct {
	inline *ast.LambdaExpr
	value  Expr
	work   types.Type
	name   string
}

// run runs the block on what ptr points at, which is in variable v (nil:
// behind a pointer); nil when the block does not check.
func (f *fnCtx) run(b *blockRunner, ptr Expr, v *Var, span source.Span) Stmt {
	if b.value != nil {
		f.c.onCalls = append(f.c.onCalls, onCall{v: v, block: b.value})
		return &ExprStmt{X: &CallIndirect{exprBase{types.TUnit}, b.value, []Expr{ptr}}, Pos: span}
	}
	l := b.inline
	self := f.newVar("this", ptr.Type(), true, l.Pos)
	self.IsSelf = true
	f.resolveBlockMembers(l, ptr.Type())
	f.recvStack = append(f.recvStack, self)
	f.blockLoops = append(f.blockLoops, len(f.loops))
	body := f.checkBlock(l.Body.(*ast.BlockExpr).Block, types.TUnit, false)
	f.recvStack = f.recvStack[:len(f.recvStack)-1]
	f.blockLoops = f.blockLoops[:len(f.blockLoops)-1]
	stmts := append([]Stmt{&VarDecl{Var: self, Init: ptr, Pos: span}}, body.Stmts...)
	if body.Value != nil {
		stmts = append(stmts, &ExprStmt{X: body.Value, Pos: body.ValuePos})
	}
	inlined := &Block{Stmts: stmts, Type: types.TUnit}
	f.c.onCalls = append(f.c.onCalls, onCall{v: v, body: inlined, self: self})
	return &ExprStmt{X: &BlockExpr{exprBase{types.TUnit}, inlined}, Pos: span}
}

// editCopy is `x.edit { … }`: the block run on a working copy of x, and
// the copy given back as an rt.
func (f *fnCtx) editCopy(x Expr, rt, work types.Type, block *blockRunner, span source.Span) Expr {
	tmp := f.newTemp(work)
	var init Expr
	freeze := ""
	switch t := rt.(type) {
	case *types.List:
		init = &Builtin{exprBase{work}, "list.copy", []Expr{x}, span}
		if !t.Mutable {
			freeze = "list.freeze"
		}
	case *types.Map:
		init = &Builtin{exprBase{work}, "map.copy", []Expr{x}, span}
		if !t.Mutable {
			freeze = "map.freeze"
		}
	case *types.Set:
		init = &Builtin{exprBase{work}, "map.copy", []Expr{x}, span}
		if !t.Mutable {
			freeze = "map.freeze"
		}
	default:
		init = x
		if rt == types.TString {
			b := &synth{sp: span}
			init = f.checkExpr(b.call(b.prelude("textBuilder"), &ast.CheckedExpr{X: x, Pos: span}), work)
		}
	}
	decl := &VarDecl{Var: tmp, Init: init, Pos: span}
	runs := f.run(block, &AddrOf{exprBase{&types.Pointer{Elem: work}}, ref(tmp)}, tmp, span)
	if runs == nil {
		return bad()
	}
	var back Expr = ref(tmp)
	switch {
	case freeze != "":
		back = &Builtin{exprBase{rt}, freeze, []Expr{ref(tmp)}, span}
	case rt == types.TString:
		b := &synth{sp: span}
		back = f.checkExpr(b.call(b.prelude("builderText"), &ast.CheckedExpr{X: ref(tmp), Pos: span}), rt)
	}
	return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{decl, runs}, Value: back, Type: rt}}
}

// runOn runs the block on x itself and gives back its value: on the place
// when place is set (root is the variable it is in, nil behind a pointer),
// else on a temporary copy of x.
func (f *fnCtx) runOn(x Expr, rt types.Type, block *blockRunner, place bool, root *Var, span source.Span) Expr {
	pt := &types.Pointer{Elem: rt}
	if !place {
		tmp := f.newTemp(rt)
		decl := &VarDecl{Var: tmp, Init: x}
		runs := f.run(block, &AddrOf{exprBase{pt}, ref(tmp)}, tmp, span)
		if runs == nil {
			return bad()
		}
		return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{decl, runs}, Value: ref(tmp), Type: rt}}
	}
	if _, isVar := x.(*VarRef); isVar {
		runs := f.run(block, &AddrOf{exprBase{pt}, x}, root, span)
		if runs == nil {
			return bad()
		}
		return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{runs}, Value: x, Type: rt}}
	}
	at := f.newTemp(pt)
	decl := &VarDecl{Var: at, Init: &AddrOf{exprBase{pt}, x}}
	runs := f.run(block, ref(at), root, span)
	if runs == nil {
		return bad()
	}
	return &BlockExpr{exprBase{rt}, &Block{Stmts: []Stmt{decl, runs}, Value: &Deref{exprBase{rt}, ref(at)}, Type: rt}}
}

// placeAssignable reports whether the checked place x may be changed by
// the caller: a `var`, a `var` field of a place, what a pointer points at
// (`this` among them); why says what it is when it may not.
func placeAssignable(x Expr) (bool, string) {
	switch x := x.(type) {
	case *VarRef:
		if x.Var.Mutable {
			return true, ""
		}
		return false, "'" + x.Var.Name + "' is a 'val'"
	case *Deref:
		return true, ""
	case *FieldGet:
		st, ok := x.X.Type().(*types.Struct)
		if !ok || x.Index >= len(st.Fields) {
			return false, "it is not a field that can be assigned"
		}
		if fld := st.Fields[x.Index]; !fld.Var {
			return false, "'" + fld.Name + "' is not a 'var' field"
		}
		return isPlaceExpr(x.X), "it is a field of a temporary"
	}
	return false, "it is not a variable"
}

// settleOnCalls decides, once every body is checked, which variables a
// function on a receiver was given a pointer to must live on the heap.
func (c *Checker) settleOnCalls() {
	for _, oc := range c.onCalls {
		v := oc.v
		if v == nil || v.IsGlobal || v.IsSelf || v.AddrTaken {
			continue
		}
		if oc.body != nil {
			if escapes, _ := selfUse(oc.body, oc.self); !escapes {
				continue
			}
		} else if cl, ok := oc.block.(*Closure); ok && len(cl.Fn.Params) > 0 && cl.Fn.Body != nil {
			if escapes, _ := selfUse(cl.Fn.Body, cl.Fn.Params[0]); !escapes {
				continue
			}
		}
		v.AddrTaken = true
	}
	c.onCalls = nil
}

// workForm is the type an `edit` of a t runs its block on: a string's
// StringBuilder, an immutable collection's mutable form, else t itself.
func (f *fnCtx) workForm(t types.Type) types.Type {
	switch t := t.(type) {
	case *types.Basic:
		if t == types.TString {
			sb := f.c.preludeType("StringBuilder")
			if sb == nil {
				return types.TInvalid
			}
			return sb
		}
	case *types.List:
		if !t.Mutable {
			return &types.List{Elem: t.Elem, Mutable: true}
		}
	case *types.Map:
		if !t.Mutable {
			return &types.Map{Key: t.Key, Value: t.Value, Mutable: true}
		}
	case *types.Set:
		if !t.Mutable {
			return &types.Set{Elem: t.Elem, Mutable: true}
		}
	}
	return t
}

// checkOnBlock checks a receiver function value given to apply or edit: a
// `fun on work()`.
func (f *fnCtx) checkOnBlock(arg ast.Expr, work types.Type, name string) Expr {
	ptr := &types.Pointer{Elem: work}
	want := &types.Func{On: true, Params: []types.Param{{Name: "this", Type: ptr}}, Ret: types.TUnit}
	block := f.checkExpr(arg, want)
	if types.IsInvalid(block.Type()) {
		return nil
	}
	bt, ok := block.Type().(*types.Func)
	if !ok || !bt.On || len(bt.Params) != 1 || !types.Identical(bt.Params[0].Type, ptr) {
		f.errorf(arg.Span(), "'%s' takes a block that runs on a '%s' — '{ … }', a 'fun on %s()' — but this is '%s' (D207)", name, work, work, block.Type())
		return nil
	}
	if bt.Effects.Throws {
		f.errorf(arg.Span(), "a receiver function given to '%s' cannot throw; handle the error inside it, or write the block in place, where 'try' reaches the caller (D207)", name)
		return nil
	}
	return block
}
