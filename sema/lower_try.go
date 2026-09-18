package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Throwing functions in eager collection operations (2026-09-18, found by
// the calculator program): `args.map(a => try self.eval(a))` types as
// `Result<List<U>, E>`. The adapters are inlined loops, so a failing call
// records the error, leaves the loop and the whole operation yields
// `Err`; the partial result is dropped and nothing panics. Callers `try`
// it like any other fallible call.

// adapterState is what one eager operation needs to know about the
// functions it was given: whether any of them throws, the union of their
// error types, and the slots that carry a failure out of the loop.
type adapterState struct {
	errT   types.Type // union of the throwing functions' errors; nil when none throws
	failed *Var       // bool: set when a call failed
	errVal *Var       // E?: the error of the failed call
}

// throwingFn registers a throwing function argument with the current
// adapter; it reports whether one is active (getOrPut and sortedBy run
// their function outside a loop the adapter controls and refuse).
func (f *fnCtx) throwingFn(ft *types.Func, span source.Span) bool {
	a := f.adapter
	if a == nil {
		return false
	}
	e := ft.Effects.Error
	if e == nil {
		e = types.TNever
	}
	if a.errT == nil {
		a.errT = e
	} else {
		a.errT = types.MakeErrorUnion(a.errT, e)
	}
	return true
}

// call calls an adapter's function argument. A throwing one is called
// for its Result: `Err` stores the error, marks the failure and leaves the
// loop; `Ok` yields the payload.
func (f *fnCtx) call(lp *Loop, fv *Var, args ...Expr) Expr {
	ft := fv.Type.(*types.Func)
	if !ft.Effects.Throws || f.adapter == nil {
		return &CallIndirect{exprBase{ft.Ret}, ref(fv), args}
	}
	a := f.adapter
	rt := f.c.ResultType(ft.Ret, ft.Effects.Error)
	r := f.newTemp(rt)
	call := &CallIndirect{exprBase{rt}, ref(fv), args}
	okV, errV := rt.Variants[0], rt.Variants[1]
	payload := &FieldGet{exprBase{ft.Ret}, &VariantCast{exprBase{okV}, ref(r), okV}, 0, okV.Fields[0].Name}
	if a.errT == nil || types.IsNever(a.errT) {
		// declared `throws` but nothing can be thrown: only the unwrap
		return &Let{exprBase{ft.Ret}, r, call, payload}
	}
	f.adapterSlots()
	errPayload := &FieldGet{exprBase{errV.Fields[0].Type}, &VariantCast{exprBase{errV}, ref(r), errV}, 0, errV.Fields[0].Name}
	var stored Expr = errPayload
	if !types.Identical(errPayload.Type(), a.errT) {
		stored = &ErrorConvert{exprBase{a.errT}, errPayload, errPayload.Type()}
	}
	fail := &Block{Stmts: []Stmt{
		&Assign{Target: ref(a.failed), Value: &BoolConst{exprBase{types.TBool}, true}},
		&Assign{Target: ref(a.errVal), Value: &SomeWrap{exprBase{a.errVal.Type}, stored}},
		&Break{Loop: lp},
	}, Type: types.TNever}
	lp.hasBreak = true
	body := &If{exprBase{ft.Ret}, &VariantTest{exprBase{types.TBool}, ref(r), errV}, fail, &Block{Value: payload, Type: ft.Ret}}
	return &Let{exprBase{ft.Ret}, r, call, body}
}

// adapterSlots creates the failure slots on first use.
func (f *fnCtx) adapterSlots() {
	a := f.adapter
	if a.failed != nil {
		return
	}
	a.failed = f.newTemp(types.TBool)
	a.errVal = f.newTemp(&types.Nullable{Elem: a.errT})
	f.pending = append(f.pending,
		&VarDecl{Var: a.failed, Init: &BoolConst{exprBase{types.TBool}, false}},
		&VarDecl{Var: a.errVal, Init: &NullConst{exprBase{a.errVal.Type}}})
}

// adapterResult wraps an adapter's value once its loops are done: when a
// function argument threw, the operation is `Result<T, E>` — `Err(e)` on
// failure, `Ok(value)` otherwise.
func (f *fnCtx) adapterResult(value Expr) Expr {
	a := f.adapter
	if a == nil || a.errT == nil || types.IsNever(a.errT) || a.failed == nil {
		return value
	}
	rt := f.c.ResultType(value.Type(), a.errT)
	okV, errV := rt.Variants[0], rt.Variants[1]
	okLit := &StructLit{exprBase{okV}, okV, []Expr{value}}
	errLit := &StructLit{exprBase{errV}, errV, []Expr{&Unwrap{exprBase{a.errT}, ref(a.errVal)}}}
	return &If{exprBase{rt}, ref(a.failed),
		&Block{Value: &MakeVariant{exprBase{rt}, rt, errV, errLit}, Type: rt},
		&Block{Value: &MakeVariant{exprBase{rt}, rt, okV, okLit}, Type: rt}}
}

// fnArgNoThrow is fnArg for an operation whose function does not run
// inside a loop the adapter controls (a sort comparator, `getOrPut`), so
// a throwing one cannot be propagated and is refused.
func (f *fnCtx) fnArgNoThrow(arg ast.Expr, expected *types.Func) (*Var, bool) {
	saved := f.adapter
	f.adapter = nil
	defer func() { f.adapter = saved }()
	return f.fnArg(arg, expected)
}
