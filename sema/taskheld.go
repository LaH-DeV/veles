package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// A value that holds a running task (D111): a struct or sealed variant
// whose declared fields include a Task, or a value that holds one. It is
// received with `with` where it is made — `with srv = try serving()` — or
// returned straight to the caller, where the rule applies again. Its tasks
// are fail-fast background children of the `with`'s block: they launch
// straight into that block's scope, which the running task carries as its
// receiving scope while the `with` value is computed (a task-local under a
// reserved key, so a suspending callee inherits it), and at the end of the
// block they are cancelled and joined before the value's own `close()`.
//
// `async f()` may be a field argument of such a constructor whose value is
// received or returned. Its own arguments are evaluated where they are
// written, but the task starts only once every field has been evaluated,
// so a later argument that fails leaves no task behind (spec D111).

// heldBy says how a task-holding value made at a site is received.
type heldBy int

const (
	heldNot    heldBy = iota
	heldWith          // the value of a `with` binding
	heldReturn        // returned straight to the caller
	// heldInside: part of a value that is itself refused, which is the one
	// error reported
	heldInside
)

// taskHolding reports whether values of t hold a running task: from the
// declared fields, so a generic instantiated with Task is not one.
func (c *Checker) taskHolding(t types.Type) bool {
	return c.holdsTask(t, map[any]bool{})
}

func (c *Checker) holdsTask(t types.Type, seen map[any]bool) bool {
	switch t := t.(type) {
	case *types.Task:
		return true
	case *types.Nullable:
		return c.holdsTask(t.Elem, seen)
	case *types.Struct:
		tmpl := templateOf(t)
		if seen[tmpl] {
			return false
		}
		seen[tmpl] = true
		c.resolveStruct(tmpl)
		for _, fld := range tmpl.Fields {
			if c.holdsTask(fld.Type, seen) {
				return true
			}
		}
	case *types.Sealed:
		tmpl := sealedTemplate(t)
		if seen[tmpl] {
			return false
		}
		seen[tmpl] = true
		for _, v := range tmpl.Variants {
			if c.holdsTask(v, seen) {
				return true
			}
		}
	}
	return false
}

// heldValue is the task-holding type a call's value carries — itself, or
// the success type of the Result a throwing call returns — or nil.
func (c *Checker) heldValue(t types.Type) types.Type {
	if isResultType(t) {
		t = t.(*types.Sealed).TypeArgs[0]
	}
	if c.taskHolding(t) {
		return t
	}
	return nil
}

// heldCall is the call a received expression is: `f()` or `try f()`.
func heldCall(e ast.Expr) *ast.CallExpr {
	if t, ok := e.(*ast.TryExpr); ok {
		e = t.X
	}
	c, _ := e.(*ast.CallExpr)
	return c
}

// allowHeld lets the call e produce a task-holding value, received as how.
func (f *fnCtx) allowHeld(e ast.Expr, how heldBy) {
	call := heldCall(e)
	if call == nil || call.Async {
		return
	}
	if f.heldOK == nil {
		f.heldOK = map[*ast.CallExpr]heldBy{}
	}
	f.heldOK[call] = how
}

// heldConstructor is the task-holding struct a call constructs, or nil.
func (f *fnCtx) heldConstructor(e *ast.CallExpr) *types.Struct {
	var t types.Type
	switch callee := e.Fun.(type) {
	case *ast.NameExpr:
		if sym := f.lookup(callee.Name); sym != nil && sym.Kind == SymType {
			t = f.c.symType(sym)
			if t == nil {
				t = sym.Type
			}
		}
	case *ast.MemberExpr:
		t = f.moduleTypeNamed(callee)
	}
	st, ok := t.(*types.Struct)
	if !ok || !f.c.taskHolding(st) {
		return nil
	}
	return st
}

// constructsTaskFree: e constructs a struct that holds no task — a
// variant of a sealed type that another variant makes task-holding, typed
// as the sealed type only by where it goes (`val s: Slot = Free()`).
func (f *fnCtx) constructsTaskFree(e *ast.CallExpr) bool {
	n, ok := e.Fun.(*ast.NameExpr)
	if !ok {
		return false
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymType {
		return false
	}
	t := f.c.symType(sym)
	if t == nil {
		t = sym.Type
	}
	st, isStruct := t.(*types.Struct)
	return isStruct && !f.c.taskHolding(st)
}

// preHeld runs before a call is checked: the arguments of a task-holding
// constructor may produce held values (and `async` launches) only when the
// constructed value itself is received.
func (f *fnCtx) preHeld(e *ast.CallExpr) {
	if f.heldConstructor(e) == nil {
		return
	}
	how := f.heldOK[e]
	for _, a := range e.Args {
		call := heldCall(a.Value)
		if call == nil {
			continue
		}
		if call.Async {
			if f.heldAsync == nil {
				f.heldAsync = map[*ast.CallExpr]heldBy{}
			}
			if how == heldNot {
				f.heldAsync[call] = heldInside
			} else {
				f.heldAsync[call] = how
			}
			continue
		}
		if how == heldNot {
			how = heldInside
		}
		f.allowHeld(call, how)
	}
}

// postHeld runs after a call is checked: a task-holding value made where
// it is not received is refused, with the fix that receives it.
func (f *fnCtx) postHeld(e *ast.CallExpr, x Expr) Expr {
	if e.Async || types.IsInvalid(x.Type()) {
		return x
	}
	held := f.c.heldValue(x.Type())
	if held == nil || f.constructsTaskFree(e) {
		return x
	}
	how := f.heldOK[e]
	if how == heldNot {
		var fix *source.Fix
		if at, ok := valBefore(e.Pos); ok {
			fix = fixReplace("Receive it with 'with'", at, "with")
		}
		f.c.errorFix(e.Pos, fix, "'%s' holds a running task, so it must be received with 'with' where it is made: 'with x = …' (its tasks belong to that block), or returned to the caller (D111)", held)
		return x
	}
	return f.startHeldLast(x)
}

// heldLaunchAllowed reports whether `async` at e is a field argument of a
// received task-holding constructor; ok is false when e is not a field
// argument of one at all.
func (f *fnCtx) heldLaunchAllowed(e *ast.CallExpr) (how heldBy, ok bool) {
	how, ok = f.heldAsync[e]
	return
}

// startHeldLast makes the received tasks start only once every field of
// the value has been evaluated: the fields — and the arguments of each
// `async` among them — are evaluated in order into temporaries, then the
// tasks are launched, then the value is built. A later argument that
// fails therefore leaves no task behind. A value built any other way (an
// `init`, a helper) is left as it is.
func (f *fnCtx) startHeldLast(x Expr) Expr {
	lit, ok := x.(*StructLit)
	if !ok || !hasHeldLaunch(lit) {
		return x
	}
	var pre, launches []Stmt
	built := f.splitHeld(lit, &pre, &launches)
	stmts := append(pre, launches...)
	return &BlockExpr{exprBase{x.Type()}, &Block{Stmts: stmts, Value: built, Type: x.Type()}}
}

func hasHeldLaunch(x Expr) bool {
	switch x := x.(type) {
	case *Launch:
		return x.Scope == nil
	case *StructLit:
		for _, fx := range x.Fields {
			if hasHeldLaunch(fx) {
				return true
			}
		}
	}
	return false
}

func (f *fnCtx) splitHeld(x Expr, pre, launches *[]Stmt) Expr {
	switch x := x.(type) {
	case *StructLit:
		if hasHeldLaunch(x) {
			fields := make([]Expr, len(x.Fields))
			for i, fx := range x.Fields {
				fields[i] = f.splitHeld(fx, pre, launches)
			}
			return &StructLit{x.exprBase, x.Struct, fields}
		}
	case *Launch:
		if x.Scope == nil {
			call := *x.Call
			call.Args = make([]Expr, len(x.Call.Args))
			for i, a := range x.Call.Args {
				tmp := f.newTemp(a.Type())
				*pre = append(*pre, &VarDecl{Var: tmp, Init: a})
				call.Args[i] = ref(tmp)
			}
			l := *x
			l.Call = &call
			t := f.newTemp(x.Type())
			*launches = append(*launches, &VarDecl{Var: t, Init: &l})
			return ref(t)
		}
	}
	if types.IsUnit(x.Type()) || types.IsNever(x.Type()) {
		return x
	}
	tmp := f.newTemp(x.Type())
	*pre = append(*pre, &VarDecl{Var: tmp, Init: x})
	return ref(tmp)
}

// heldTasks lists where the tasks of a task-holding value of type t are,
// in field order, each with the Result its task ends with — what the
// receiving block rethrows when that task fails.
func (c *Checker) heldTasks(t types.Type) []HeldTask {
	var out []HeldTask
	var walk func(t types.Type, path []HeldStep, seen map[any]bool)
	walk = func(t types.Type, path []HeldStep, seen map[any]bool) {
		switch t := t.(type) {
		case *types.Task:
			out = append(out, HeldTask{Path: append([]HeldStep(nil), path...), Result: t.Result})
		case *types.Nullable:
			if _, isTask := t.Elem.(*types.Task); isTask {
				p := append(append([]HeldStep(nil), path...), HeldStep{Variant: -1, Field: -1, Nullable: true})
				out = append(out, HeldTask{Path: p, Result: t.Elem.(*types.Task).Result})
			}
		case *types.Struct:
			if seen[t] {
				return
			}
			seen[t] = true
			for i, fld := range t.Fields {
				if c.holdsTask(fld.Type, map[any]bool{}) {
					walk(fld.Type, append(path, HeldStep{Variant: -1, Field: i}), seen)
				}
			}
			delete(seen, t)
		case *types.Sealed:
			for vi, v := range t.Variants {
				for i, fld := range v.Fields {
					if c.holdsTask(fld.Type, map[any]bool{}) {
						walk(fld.Type, append(path, HeldStep{Variant: vi, Field: i}), seen)
					}
				}
			}
		}
	}
	walk(t, nil, map[any]bool{})
	return out
}

// withHeld is item i of s when its value holds tasks (D111): a held scope
// around the rest of the block. The value is computed with that scope as
// the receiving scope, so its tasks join it; when the block ends — or is
// left early — they are cancelled and joined, and then the value is
// closed if it is Closeable.
func (f *fnCtx) withHeld(s *ast.WithExpr, i int, init Expr, closeable *types.Trait, want types.Type, asValue bool, result **Var) ([]Stmt, types.Type) {
	b := s.Bindings[i]
	v := f.withVar(b, init.Type())
	var closeCall Expr
	if closeable != nil && f.findImpl(init.Type(), closeable) != nil {
		at := v.Span
		closer := &ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(v, at), Name: ast.Ident{Name: "close", Pos: at}, Pos: at}, Pos: at}
		closeCall = f.checkExpr(closer, types.TUnit)
	}
	f.markResource(v, v)
	what := "closed after its tasks are cancelled and joined"
	if closeCall == nil {
		what = "released — its tasks cancelled and joined —"
	}
	f.refWith(s, i, v, what)
	if f.c.index != nil {
		if f.c.index.Held == nil {
			f.c.index.Held = map[source.Span]bool{}
		}
		f.c.index.Held[b.Value.Span()] = true
	}
	sb := &ScopeBlock{Span: b.Value.Span(), Cancel: true}
	sb.T = types.TUnit
	sb.Held = &HeldValue{Var: v, Init: init, Close: closeCall, Tasks: f.c.heldTasks(init.Type())}
	for _, ht := range sb.Held.Tasks {
		if !isResultType(ht.Result) {
			continue
		}
		et := ht.Result.(*types.Sealed).TypeArgs[1]
		if types.IsNever(et) {
			continue
		}
		if f.catching != nil {
			// as for a scope (D98): the block hands the error to the
			// function, past the `catch`
			f.errorf(b.Value.Span(), "a value holding a task that can fail cannot be received inside a 'do' block: its block passes the task's error to the function, not to the 'catch'; receive it in a function of its own and 'try' that (D98/D111)")
			f.catching.poisoned = true
			break
		}
		for _, m := range types.UnionMembers(et) {
			f.recordError(m, b.Value.Span())
		}
	}
	inner, bodyT := f.withBindings(s, i+1, closeable, want, asValue, result)
	sb.Body = &Block{Stmts: inner, Type: types.TUnit}
	if types.IsNever(bodyT) {
		sb.Body.Type = types.TNever
	}
	sb.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		sb.ErrTo = nil // inferred: filled in at codegen time
	}
	f.suspending(sb, b.Value.Span(), "with")
	return []Stmt{sb}, bodyT
}
