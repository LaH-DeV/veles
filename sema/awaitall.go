package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// awaitAll (D156) is a built-in, a suspension point like `await`, whose
// name is the mark D16 asks for. `awaitAll(t1, …, tn)` waits for every
// task and gives the tuple of their values in argument order; an argument
// is a handle (one that must be awaited counts as awaited here) or an
// inline `async f()`, which belongs to the awaitAll itself: those launch
// into a fail-fast scope of its own, so the first of them to fail cancels
// the others and the error leaves from here. `awaitAll(tasks)` over a list
// of handles gives the list of their values (a prelude function).

const awaitAllDoc = "Waits for every task and gives their values: `awaitAll(a, b)` is `(await a, await b)` as a tuple, in argument order; `awaitAll(tasks)` gives a `List` in the list's order. An argument may be an inline `async f()`: the first of those to fail cancels the others and its error leaves from here (D156)."

func (f *fnCtx) awaitAllCall(e *ast.CallExpr) Expr {
	if n, ok := e.Fun.(*ast.NameExpr); ok && f.c.index != nil {
		f.c.index.Refs = append(f.c.index.Refs, Ref{Span: n.Pos, Kind: "fun", Name: "awaitAll", Detail: "fun awaitAll(tasks…)  (built in)", Doc: awaitAllDoc})
	}
	if f.isGlobal {
		f.errorf(e.Pos, "'awaitAll' cannot appear in a global initializer; compute the value in 'main' and pass it down")
		return bad() // (its `async` arguments would only repeat it)
	}
	if len(e.Args) == 0 {
		f.errorf(e.Pos, "'awaitAll' waits for tasks: 'awaitAll(a, b)' with handles or 'async f()' calls, or 'awaitAll(tasks)' with a list of handles (D156)")
		return bad()
	}
	if c, isCall := e.Args[0].Value.(*ast.CallExpr); len(e.Args) == 1 && (!isCall || !c.Async) {
		x := f.awaitedOperand(e.Args[0].Value)
		if l, isList := x.Type().(*types.List); isList {
			if _, ofTasks := l.Elem.(*types.Task); ofTasks {
				return f.awaitList(x, l.Mutable, e.Pos)
			}
		}
		if tt, isTask := x.Type().(*types.Task); isTask {
			f.warnf(e.Pos, "'awaitAll' of one task is an 'await': use 'await %s' (D156)", srcText(e.Args[0].Value))
			return f.suspending(&AwaitTask{exprBase{tt.Result}, x}, e.Pos, "awaitAll")
		}
		f.notATask(e.Args[0].Value, x)
		return bad()
	}
	var sb *ScopeBlock
	for _, a := range e.Args {
		if c, isCall := a.Value.(*ast.CallExpr); isCall && c.Async {
			sb = &ScopeBlock{Span: e.Pos}
			sb.T = types.TUnit
			f.scopes = append(f.scopes, sb)
			f.openTaskBlock("awaitAll")
			break
		}
	}
	var launches []Stmt
	var elems []Expr
	var elemTypes []types.Type
	failed := false
	for _, a := range e.Args {
		if c, isCall := a.Value.(*ast.CallExpr); isCall && c.Async {
			l := f.launch(c, nil)
			tt, ok := l.Type().(*types.Task)
			if !ok {
				failed = true
				continue
			}
			tmp := f.newTemp(tt)
			launches = append(launches, &VarDecl{Var: tmp, Init: l})
			elems = append(elems, f.suspending(&AwaitTask{exprBase{tt.Result}, ref(tmp)}, a.Value.Span(), "awaitAll"))
			elemTypes = append(elemTypes, tt.Result)
			continue
		}
		x := f.awaitedOperand(a.Value)
		tt, ok := x.Type().(*types.Task)
		if !ok {
			f.notATask(a.Value, x)
			failed = true
			continue
		}
		elems = append(elems, f.suspending(&AwaitTask{exprBase{tt.Result}, x}, a.Value.Span(), "awaitAll"))
		elemTypes = append(elemTypes, tt.Result)
	}
	if sb != nil {
		f.closeTaskBlock()
		f.scopes = f.scopes[:len(f.scopes)-1]
	}
	if failed {
		return bad()
	}
	var value Expr = elems[0]
	if len(elems) > 1 {
		value = &TupleLit{exprBase{&types.Tuple{Elems: elemTypes}}, elems}
	}
	if sb == nil {
		return value
	}
	vt := value.Type()
	res := f.newTemp(vt)
	sb.Body = &Block{Stmts: append(launches, &Assign{Target: ref(res), Value: value}), Type: types.TUnit}
	sb.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		sb.ErrTo = nil // inferred: filled in at codegen time
	}
	f.suspending(sb, e.Pos, "awaitAll")
	return &BlockExpr{exprBase{vt}, &Block{Stmts: []Stmt{&VarDecl{Var: res}, sb}, Value: ref(res), Type: vt}}
}

// awaitedOperand checks what an awaitAll waits for: a handle named here
// counts as awaited (D156 part 1).
func (f *fnCtx) awaitedOperand(e ast.Expr) Expr {
	saved := f.awaitOperand
	f.awaitOperand = e
	x := f.checkExpr(e, nil)
	f.awaitOperand = saved
	return x
}

// awaitList is `awaitAll(tasks)`: the prelude's loop over the handles.
func (f *fnCtx) awaitList(x Expr, mutable bool, at source.Span) Expr {
	v, decl := f.hidden("tasks", x, false)
	name := "awaitEach"
	if mutable {
		name = "awaitEachMut"
	}
	call := &ast.CallExpr{Fun: &ast.PreludeName{Name: name, Pos: at}, Args: []ast.Arg{{Value: nameOf(v, at)}}, Pos: at}
	r := f.checkExpr(call, nil)
	return &BlockExpr{exprBase{r.Type()}, &Block{Stmts: []Stmt{decl}, Value: r, Type: r.Type()}}
}

// notATask: an awaitAll argument that is not a task — most often a call
// that finished before awaitAll began.
func (f *fnCtx) notATask(arg ast.Expr, x Expr) {
	if types.IsInvalid(x.Type()) {
		return
	}
	if c, isCall := arg.(*ast.CallExpr); isCall {
		at := source.Span{File: c.Pos.File, Start: c.Pos.Start, End: c.Pos.Start}
		f.c.errorFix(arg.Span(), fixReplace("Start it as a task", at, "async "), "'%s' finishes before 'awaitAll' starts, so these run one after the other; start it with 'async %s' (D156)", srcText(arg), srcText(arg))
		return
	}
	f.errorf(arg.Span(), "'awaitAll' waits for tasks, and '%s' is a '%s' (D156)", srcText(arg), x.Type())
}
