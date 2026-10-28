package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Concurrency (build plan Stage 4). Suspension is inferred after checking
// (see suspend.go); here the surface constructs are lowered.

// sendable implements the auto-derived Sendable marker (D35): a type is
// Sendable when everything reachable through it is immutable or a
// synchronized primitive. Mutable collections, pointers, closures and trait
// objects are not.
func sendable(t types.Type) bool {
	return sendableIn(t, map[types.Type]bool{})
}

// hasVarFields reports whether a value of t can change in place: a struct
// with a `var` field (D22 v0.30), directly or inside a field, tuple
// element, nullable or sealed variant held by value. Elements of an
// immutable collection are read as copies, and Mutex/Atomic synchronize
// their contents, so they do not count. A closure shares its captures by
// reference, which is why a sendable one may not capture such a value
// (check_lambda.go).
func hasVarFields(t types.Type) bool {
	return hasVarFieldsIn(t, map[types.Type]bool{})
}

func hasVarFieldsIn(t types.Type, seen map[types.Type]bool) bool {
	switch t := t.(type) {
	case *types.Nullable:
		return hasVarFieldsIn(t.Elem, seen)
	case *types.Array:
		return hasVarFieldsIn(t.Elem, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if hasVarFieldsIn(e, seen) {
				return true
			}
		}
	case *types.Struct:
		if seen[t] {
			return false
		}
		seen[t] = true
		if ok, _ := syncWrapper(t); ok {
			return false
		}
		for _, f := range t.Fields {
			if f.Var || hasVarFieldsIn(f.Type, seen) {
				return true
			}
		}
	case *types.Sealed:
		if seen[t] {
			return false
		}
		seen[t] = true
		for _, v := range t.Variants {
			if hasVarFieldsIn(v, seen) {
				return true
			}
		}
	}
	return false
}

// syncWrapper reports whether t is a prelude type whose operations
// synchronize what it holds (D35, D66, D146), and what must be Sendable for
// it to be: nothing for a Mutex or an Atomic, Sendable whatever they hold;
// the type arguments for RwLock, Event, Lazy, Broadcast, Subscription and
// Watch, whose values reach every task that holds a copy.
func syncWrapper(t *types.Struct) (bool, []types.Type) {
	if t.Module != "std.prelude" {
		return false, nil
	}
	switch t.Name {
	case "Mutex", "Atomic", "Executor":
		return true, nil
	case "RwLock", "Event", "Lazy", "Broadcast", "Subscription", "Watch":
		return true, t.TypeArgs
	}
	return false, nil
}

// sendableIn is sendable with the structs and sealed types already on the
// path in seen, so a type that contains itself by value (a D31 error) does
// not recurse forever.
func sendableIn(t types.Type, seen map[types.Type]bool) bool {
	switch t := t.(type) {
	case *types.Basic, *types.Enum:
		return true
	case *types.Nullable:
		return sendableIn(t.Elem, seen)
	case *types.Array:
		return sendableIn(t.Elem, seen) // an array is a value: its copy crosses (D121)
	case *types.Tuple:
		for _, e := range t.Elems {
			if !sendableIn(e, seen) {
				return false
			}
		}
		return true
	case *types.Range:
		return true
	case *types.List:
		return !t.Mutable && sendableIn(t.Elem, seen)
	case *types.Map:
		return !t.Mutable && sendableIn(t.Key, seen) && sendableIn(t.Value, seen)
	case *types.Set:
		return !t.Mutable && sendableIn(t.Elem, seen)
	case *types.Struct:
		if seen[t] {
			return true
		}
		seen[t] = true
		if ok, held := syncWrapper(t); ok {
			// explicitly synchronized wrappers (D35, D146)
			for _, h := range held {
				if !sendableIn(h, seen) {
					return false
				}
			}
			return true
		}
		for _, f := range t.Fields {
			if !sendableIn(f.Type, seen) {
				return false
			}
		}
		return true
	case *types.Sealed:
		if seen[t] {
			return true
		}
		seen[t] = true
		for _, v := range t.Variants {
			if !sendableIn(v, seen) {
				return false
			}
		}
		return true
	case *types.ErrorUnion:
		for _, m := range t.Members {
			if !sendableIn(m, seen) {
				return false
			}
		}
		return true
	case *types.Channel:
		return sendableIn(t.Elem, seen)
	case *types.Task:
		return true
	case *types.TypeParam, *types.Assoc:
		return true
	case *types.Trait:
		// an object crosses when its trait says every implementor does
		// (`trait Stream : Closeable + Sendable`); boxValue enforces it
		return containsTrait(allSupers(t), sendableTraitOf(t))
	case *types.Func:
		return t.Sendable // a named function, or a closure over vals of Sendable types
	}
	return false
}

// channelCtor handles `Channel<T>()` / `Channel<T>(capacity: n)`.
func (f *fnCtx) channelCtor(typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	var elem types.Type
	if len(typeArgs) == 1 {
		elem = typeArgs[0]
	} else if ct, ok := numericHint(want).(*types.Channel); ok && len(typeArgs) == 0 {
		elem = ct.Elem
	}
	if elem == nil {
		f.errorf(e.Pos, "cannot infer the element type of the channel; write 'Channel<T>(...)'")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if !sendable(elem) {
		f.errorf(e.Pos, "'%s' is not Sendable and cannot cross a task boundary through a channel; send immutable data (a List, not a MutableList: '.toList()') or share state behind a 'Mutex' (D35/D54)", elem)
	}
	var cap Expr = &IntConst{exprBase{types.TI64}, 0, false}
	if len(e.Args) > 1 {
		f.errorf(e.Pos, "Channel takes at most one argument, 'capacity'")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if len(e.Args) == 1 {
		if e.Args[0].Name != nil && e.Args[0].Name.Name != "capacity" {
			f.errorf(e.Args[0].Name.Pos, "Channel has no parameter '%s'; use 'capacity'", e.Args[0].Name.Name)
		}
		cap = f.checkExprTo(e.Args[0].Value, types.TI64)
	}
	return &Builtin{exprBase{&types.Channel{Elem: elem}}, "chan.new", []Expr{cap}, e.Pos}
}

func (f *fnCtx) channelMethod(recv Expr, ct *types.Channel, name string, e *ast.CallExpr) Expr {
	span := e.Pos
	need := func(n int) bool {
		if len(e.Args) != n {
			f.arityError(span, ct, name, n)
			f.checkArgsLoosely(e.Args)
			return false
		}
		return true
	}
	switch name {
	case "send":
		if !need(1) {
			return bad()
		}
		f.awaitNext = false
		x := f.checkExprTo(e.Args[0].Value, ct.Elem)
		return f.suspending(&Builtin{exprBase{types.TUnit}, "chan.send", []Expr{recv, x}, span}, span, "send")
	case "recv":
		if !need(0) {
			return bad()
		}
		awaited := f.awaitNext
		f.awaitNext = false
		b := f.suspending(&Builtin{exprBase{&types.Nullable{Elem: ct.Elem}}, "chan.recv", []Expr{recv}, span}, span, "recv")
		if !awaited && !f.inRaceArm {
			f.errorf(span, "'recv()' always suspends and must be awaited: 'await ch.recv()' (D16)")
		}
		return b
	case "trySend":
		// the non-suspending forms: a caller that must not wait (a
		// metrics sample, a best-effort notification) asks, and moves on
		if !need(1) {
			return bad()
		}
		x := f.checkExprTo(e.Args[0].Value, ct.Elem)
		return &Builtin{exprBase{types.TBool}, "chan.trySend", []Expr{recv, x}, span}
	case "tryRecv":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{&types.Nullable{Elem: ct.Elem}}, "chan.tryRecv", []Expr{recv}, span}
	case "close":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TUnit}, "chan.close", []Expr{recv}, span}
	case "closeAfter":
		if !need(1) {
			return bad()
		}
		n := f.checkExprTo(e.Args[0].Value, types.TI64)
		return &Builtin{exprBase{types.TUnit}, "chan.closeAfter", []Expr{recv, n}, span}
	case "len":
		if !need(0) {
			return bad()
		}
		return &Builtin{exprBase{types.TI64}, "chan.len", []Expr{recv}, span}
	}
	return nil
}

// suspending records that the current function suspends at this point and
// rejects suspension where the executor cannot reach.
func (f *fnCtx) suspending(x Expr, span source.Span, what string) Expr {
	if f.isGlobal {
		f.errorf(span, "'%s' suspends; a global initializer cannot suspend — compute the value in 'main' and pass it down", what)
		return x
	}
	f.fn.suspends = true
	f.refuseHeldSuspension(span, "'"+what+"'")
	return x
}

// cede is the statement `cede` (D197): the task offers its executor a turn
// and goes on at the next statement — at once, when the executor picks it
// again. It is a suspension point like any other: the function suspends,
// a held lock or a global initializer refuses it, and a cancellation or a
// failed sibling is seen there. Its own operation, not a sleep of zero, so
// what was written survives into the tree and its diagnostics.
func (f *fnCtx) cede(span source.Span) Stmt {
	x := f.suspending(&Builtin{exprBase{types.TUnit}, "task.cede", nil, span}, span, "cede")
	return &ExprStmt{X: x, Pos: span}
}

// sleepCall handles the prelude `sleep(d)`. The argument is a `Duration`
// (D60): a bare number would put the unit back in the reader's head, which
// is what the type exists to prevent. The executor's timers are in
// milliseconds, so the duration is rounded **up** to one here — a sleep is
// never shorter than it was asked for, and a sub-millisecond one still
// yields.
func (f *fnCtx) sleepCall(e *ast.CallExpr) Expr {
	if len(e.Args) != 1 {
		f.errorf(e.Pos, "'sleep' takes one argument: a 'Duration'")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	awaited := f.awaitNext
	f.awaitNext = false
	dur, _ := f.c.preludeType("Duration").(*types.Struct)
	var ns Expr
	if dur == nil {
		// no prelude (a bare-file test): a number of milliseconds
		ms := f.checkExprTo(e.Args[0].Value, types.TI64)
		ns = &Binary{exprBase{types.TI64}, OpMul, ms, &IntConst{exprBase{types.TI64}, 1000000, false}, e.Pos}
	} else {
		arg := f.checkExpr(e.Args[0].Value, dur)
		switch {
		case types.IsInvalid(arg.Type()):
			return bad() // the argument reported its own problem
		case types.IsInteger(arg.Type()):
			f.errorf(e.Pos, "'sleep' takes a 'Duration', not a number of milliseconds: write 'sleep(Duration.millis(n))' or 'sleep(Duration.seconds(n))' (D60)")
			return bad()
		case !types.Identical(arg.Type(), dur):
			f.errorf(e.Pos, "'sleep' takes a 'Duration', found '%s'", arg.Type())
			return bad()
		}
		// the runtime's timers count nanoseconds (F9): the deadline is the
		// Duration itself; zero or a negative one yields
		ns = &FieldGet{exprBase{types.TI64}, arg, fieldIndex(dur, "ns"), "ns"}
	}
	b := f.suspending(&Builtin{exprBase{types.TUnit}, "task.sleep", []Expr{ns}, e.Pos}, e.Pos, "sleep")
	if !awaited && !f.inRaceArm {
		f.errorf(e.Pos, "'sleep()' always suspends and must be awaited: 'await sleep(d)' (D16)")
	}
	return b
}

// fieldIndex is the position of a field by name, so lowering that reaches
// into a prelude struct does not depend on the order its fields are written
// in. -1 is impossible for a field the prelude declares, and would be caught
// by the first program that sleeps.
func fieldIndex(s *types.Struct, name string) int {
	for i, f := range s.Fields {
		if f.Name == name {
			return i
		}
	}
	return -1
}

// ioWaitCall handles `await ioWait(fd, write)`, the standard library's
// socket wait (std/net): the task parks until the executor's poll reports
// the descriptor readable (or writable) and the non-blocking call is
// retried. Not a user-facing name; it exists only inside std.
func (f *fnCtx) ioWaitCall(e *ast.CallExpr) Expr {
	if len(e.Args) != 2 {
		f.errorf(e.Pos, "'ioWait' takes two arguments: the descriptor and whether to wait for writing")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	awaited := f.awaitNext
	f.awaitNext = false
	fd := f.checkExprTo(e.Args[0].Value, types.TI64)
	write := f.checkExprTo(e.Args[1].Value, types.TBool)
	b := f.suspending(&Builtin{exprBase{types.TUnit}, "task.ioWait", []Expr{fd, write}, e.Pos}, e.Pos, "ioWait")
	if !awaited {
		f.errorf(e.Pos, "'ioWait()' always suspends and must be awaited: 'await ioWait(fd, write)' (D16)")
	}
	return b
}

func (f *fnCtx) awaitExpr(e *ast.AwaitExpr) Expr {
	if call, ok := e.X.(*ast.CallExpr); ok && !call.Async {
		if n, isName := call.Fun.(*ast.NameExpr); isName && (n.Name == "awaitAll" || n.Name == "awaitFirst") && f.lookup(n.Name) == nil {
			drop := source.Span{File: e.Pos.File, Start: e.Pos.Start, End: e.X.Span().Start}
			f.c.errorFix(e.Pos, fixReplace("Remove 'await'", drop, ""), "'%s' waits; remove 'await' — '%s(…)' is itself the point where this task waits (D156, D168)", n.Name, n.Name)
			return f.checkExpr(e.X, nil)
		}
	}
	f.awaitNext = true
	saved := f.awaitOperand
	f.awaitOperand = e.X
	x := f.checkExpr(e.X, nil)
	f.awaitOperand = saved
	consumed := !f.awaitNext
	f.awaitNext = false
	if consumed {
		return x
	}
	if tt, ok := x.Type().(*types.Task); ok {
		f.checkNotCancelled(e)
		return f.suspending(&AwaitTask{exprBase{tt.Result}, x}, e.Pos, "await")
	}
	if types.IsInvalid(x.Type()) {
		return bad()
	}
	if e == f.raceAwait {
		f.errorf(e.Pos, "a select arm waits on 'ch.recv()', 'ch.send(v)', 'sleep(d)', a task ('async f()' or a handle), or an Event's 'wait()', a Watch's 'changed()' or a Subscription's 'recv()' (D146, D168), not '%s'", x.Type())
		return bad()
	}
	if call, ok := e.X.(*ast.CallExpr); ok && !call.Async {
		// the Kotlin/JS habit: a suspending call is written as a plain
		// call (D2), so `await f()` only ever means a stray `await`
		drop := source.Span{File: e.Pos.File, Start: e.Pos.Start, End: e.X.Span().Start}
		f.c.errorFix(e.Pos, fixReplace("Remove 'await'", drop, ""), "'await' is not written on a call: a function that suspends is called like any other and suspends by itself (D2/D16); 'await' applies to channels, timers and task handles")
		return x
	}
	f.errorf(e.Pos, "'await' applies to channels, timers and task handles, not '%s' (D16)", x.Type())
	return bad()
}

// cancelledFact is the flow fact "this handle's task was cancelled": kept
// in the narrowing state, so it holds after a `t.cancel()` only where the
// call was made on every path (joins and loops drop it, D154).
func cancelledFact(v *Var) place { return place{v: v, path: "$cancelled"} }

// noteCancelled records `t.cancel()` on a local handle.
func (f *fnCtx) noteCancelled(recv ast.Expr, at source.Span) {
	p, ok := f.placeOf(recv)
	if !ok || p.path != "" {
		return
	}
	f.narrow[cancelledFact(p.v)] = types.TUnit
	if f.c.cancelAt == nil {
		f.c.cancelAt = map[*Var]source.Span{}
	}
	f.c.cancelAt[p.v] = at
}

// checkNotCancelled refuses `await t` where `t.cancel()` was called on
// every path before it (D154): the await panics unless the task happened
// to finish first.
func (f *fnCtx) checkNotCancelled(e *ast.AwaitExpr) {
	p, ok := f.placeOf(e.X)
	if !ok || p.path != "" {
		return
	}
	if _, cancelled := f.narrow[cancelledFact(p.v)]; !cancelled {
		return
	}
	line, _ := f.c.cancelAt[p.v].File.Position(f.c.cancelAt[p.v].Start)
	f.errorf(e.Pos, "'%s' was cancelled at line %d: awaiting it panics unless the task had already finished; drop the 'await' (its scope still waits for the task to stop), or do not cancel a task whose value you need (D154)", p.v.Name, line)
}

// launch checks `async call(...)`.
func (f *fnCtx) launch(e *ast.CallExpr, want types.Type) Expr {
	savedPre := f.launchPre
	f.launchPre = nil
	x := f.launchCall(e, want)
	pre := f.launchPre
	f.launchPre = savedPre
	if len(pre) == 0 || types.IsInvalid(x.Type()) {
		return f.raceContestant(x, e.Pos)
	}
	// `async gl.f(call())`: the arguments, evaluated here first (D150)
	return f.raceContestant(&BlockExpr{exprBase{x.Type()}, &Block{Stmts: pre, Value: x, Type: x.Type()}}, e.Pos)
}

func (f *fnCtx) launchCall(e *ast.CallExpr, want types.Type) Expr {
	// D111: a field argument of a task-holding constructor starts a task
	// the value holds, which the block receiving the value owns
	// (a value that is not received is the one error: heldInside)
	how, held := f.heldLaunchAllowed(e)
	if len(f.scopes) == 0 && !held {
		f.errorf(e.Pos, "'async' must be lexically inside a 'scope' or 'gather' block, or be the whole value of a 'val' or 'with' — 'val t = async f()' runs f alongside the rest of the block, which awaits it; 'with t = async f()' runs it until the block ends: tasks cannot outlive their block (D3/D34/D100/D156)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	inner := *e
	inner.Async = false
	f.allowHeld(&inner, heldInside) // a held result is refused below, once
	saved, savedBlock := f.launching, f.asyncBlock
	f.launching = &inner
	if e.Block {
		f.asyncBlock = e.Fun
	}
	x := f.checkExpr(&inner, nil)
	f.launching, f.asyncBlock = saved, savedBlock
	if ci, isValue := x.(*CallIndirect); isValue {
		// D103: a function value runs in a task like a named function
		ft := ci.Fn.Type().(*types.Func)
		if !ft.Sendable {
			f.errorf(e.Fun.Span(), "'%s' is not a sendable function, so it cannot cross into a task; declare its type 'sendable fun(...)' (D35/D103)", srcText(e.Fun))
			return bad()
		}
		x = &Call{exprBase: exprBase{ci.Type()}, Fn: f.c.launchTrampoline(ft), Args: append([]Expr{ci.Fn}, ci.Args...), Span: e.Pos}
	}
	call, ok := x.(*Call)
	if !ok {
		if !types.IsInvalid(x.Type()) {
			f.errorf(e.Pos, "'async' starts a call: 'async f(args)' with f a named function, a method or a 'sendable fun' value, its arguments evaluated here (D3/D103)")
		}
		return bad()
	}
	for i, a := range call.Args {
		t := a.Type()
		// (a generic method's instance gets its Receiver when its body is
		// checked, which may come later: the call knows it is a method's)
		if i == 0 && (call.Fn.Receiver != nil || call.Recv != RecvNone) {
			// the receiver crosses as a copy — a snapshot, like the captures
			// of a sendable closure — never as a pointer into this task
			p := t.(*types.Pointer)
			t = p.Elem
			tmp := f.newTemp(p.Elem)
			copy := &Let{exprBase{p.Elem}, tmp, &Deref{exprBase{p.Elem}, a}, ref(tmp)}
			call.Args[0] = &AddrOf{exprBase{p}, copy}
			call.Recv = RecvNone
		}
		if !sendable(t) && !f.childStaysOnThread() {
			span := e.Pos
			if j := i - len(call.Args) + len(e.Args); j >= 0 && j < len(e.Args) {
				span = e.Args[j].Value.Span()
			}
			f.errorf(span, "argument of type '%s' is not Sendable and cannot cross a task boundary; pass immutable data (a List, not a MutableList: '.toList()') or share state behind a 'Mutex' (D35)", t)
		}
	}
	// the function throws, so its task can fail; one returning a Result
	// without `throws` hands an Err over as a value
	fails := call.Fn.Sig.Effects.Throws && isResultType(call.Type())
	if hv := f.c.heldValue(call.Type()); hv != nil {
		f.errorf(e.Pos, "'async' would start a task whose result, a '%s', holds a task of its own that no block could receive; call it and receive the value with 'with' (D111)", hv)
		return bad()
	}
	if held {
		if how == heldReturn {
			// the task outlives this function: it may not be handed what
			// this function's `with` blocks close
			for _, a := range e.Args {
				if r, _ := f.heldResource(a.Value); r != nil && f.owns(r) {
					f.refuseEscape(a.Value, "be handed to a task that outlives this function")
				}
			}
		}
		// a field's declared type carries the error to the receiving block (D111)
		return f.c.noteAlloc(&Launch{exprBase: exprBase{&types.Task{Result: call.Type()}}, Call: call, Index: -1, Fails: fails}, e.Pos)
	}
	sc := f.scopes[len(f.scopes)-1]
	handle := call.Type()
	unwrap := fails && !sc.Gather
	if unwrap {
		// D141: in a fail-fast block the child's error is the block's, so
		// awaiting the handle gives the value
		handle = call.Type().(*types.Sealed).TypeArgs[0]
		if wt, ok := want.(*types.Task); ok && types.Identical(wt.Result, call.Type()) {
			f.errorf(e.Pos, "a task of this 'scope' is 'Task<%s>', not 'Task<%s>': its error fails the scope, so awaiting it gives the value — write 'Task<%s>' and drop the 'try' on 'await' (D141)", handle, call.Type(), handle)
			return bad()
		}
	}
	l := &Launch{exprBase: exprBase{&types.Task{Result: handle}}, Call: call, Scope: sc, Index: len(sc.Launches), Fails: fails, Unwrap: unwrap}
	sc.Launches = append(sc.Launches, l)
	f.c.noteAlloc(l, e.Pos)
	if sc.Gather {
		// D52: a panic surfaces as a Result error at the gather boundary
		// a Result the child returns without `throws` is its value (D155)
		okT, errT := call.Type(), types.Type(nil)
		if rs, isR := call.Type().(*types.Sealed); isR && fails {
			okT, errT = rs.TypeArgs[0], rs.TypeArgs[1]
			if types.IsNever(errT) {
				errT = nil
			}
		}
		var members []types.Type
		members = append(members, types.UnionMembers(errT)...)
		if p := f.c.panicType(); p != nil {
			members = append(members, p)
		}
		sc.Elems = append(sc.Elems, f.c.ResultType(okT, types.MakeErrorUnion(members...)))
	} else if fails {
		// fail-fast: the scope rethrows the child's error
		et := call.Type().(*types.Sealed).TypeArgs[1]
		if !types.IsNever(et) && f.catching != nil {
			// the scope would hand the child's error to the function, past
			// the handler of the `do` block around it
			f.errorf(e.Pos, "a child that can fail cannot be launched in a 'scope' inside a 'do' block: the scope passes its error to the function, not to the 'catch'; put the scope in its own function and 'try' that (D98)")
			f.catching.poisoned = true
		} else if !types.IsNever(et) && !f.throws && !f.isGlobal {
			f.errorf(e.Pos, "this child can fail with '%s', which fails the 'scope' and leaves it as the function's error, but the function is not declared 'throws'; add 'throws', or collect the outcome as a value with 'gather' (D4/D141)", et)
		} else if !types.IsNever(et) {
			for _, m := range types.UnionMembers(et) {
				f.recordError(m, e.Pos)
			}
		}
	}
	// a test helper launched from test code: the call site is bound around
	// the launch, so the task inherits it and a failure recorded inside the
	// helper says where the test launched it from (D78); the launching task
	// has its own sites back at once
	if call.Fn.tmpl != nil && call.Fn.tmpl.TestCode && f.inTest() {
		prev := f.newTemp(types.TI64)
		enter := &Builtin{exprBase{types.TI64}, "test.enter", nil, e.Pos}
		leave := &ExprStmt{X: &Builtin{exprBase{types.TUnit}, "test.leave", []Expr{ref(prev)}, e.Pos}}
		h := f.newTemp(l.Type())
		return &Let{exprBase{l.Type()}, prev, enter, &Let{exprBase{l.Type()}, h, l,
			&BlockExpr{exprBase{l.Type()}, &Block{Stmts: []Stmt{leave}, Value: ref(h), Type: l.Type()}}}}
	}
	return l
}

func (f *fnCtx) scopeStmt(s *ast.ScopeStmt) []Stmt {
	x := f.scopeExpr(s.On, s.Workers, s.Body, s.Pos, nil, false)
	if x == nil {
		return nil
	}
	return []Stmt{x.(Stmt)}
}

// scopeExpr checks a `scope`, as a statement or (asValue) as a value
// (D156): the body's last expression, held in a temporary declared before
// the scope opens and read once every child has finished.
func (f *fnCtx) scopeExpr(on, workers ast.Expr, body *ast.Block, pos source.Span, want types.Type, asValue bool) Expr {
	if f.isGlobal {
		f.errorf(pos, "'scope' cannot appear in a global initializer; compute the value in 'main' and pass it down")
		if asValue {
			return bad()
		}
		return nil
	}
	sb := &ScopeBlock{Span: pos}
	sb.T = types.TUnit
	sb.On = f.executorOf(on, "scope")
	sb.Workers = f.workersOf(workers)
	f.scopes = append(f.scopes, sb)
	f.openTaskBlock("scope")
	var result *Var
	var valueT types.Type
	if asValue {
		b := f.checkBlock(body, want, true)
		switch {
		case types.IsNever(b.Type):
			valueT = types.TNever
		case b.Value == nil:
			f.errorf(pos, "a 'scope' used as a value ends in an expression: its value is the last one in its body, once every task it started has finished (D156)")
			valueT = types.TInvalid
		default:
			if es, ok := body.Stmts[len(body.Stmts)-1].(*ast.ExprStmt); ok {
				f.refuseEscape(es.X, "be the value of its block")
			}
			valueT = b.Type
			result = f.newTemp(b.Type)
			b.Stmts = append(b.Stmts, &Assign{Target: &VarRef{exprBase{b.Type}, result}, Value: b.Value})
			b.Value = nil
			b.Type = types.TUnit
		}
		sb.Body = b
	} else {
		sb.Body = f.checkBlock(body, nil, false)
	}
	f.closeTaskBlock()
	f.scopes = f.scopes[:len(f.scopes)-1]
	f.placedStartsTasks(sb, on, workers, pos, "scope")
	sb.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		// inferred: filled in at codegen time from the function signature
		sb.ErrTo = nil
	}
	f.suspending(sb, pos, "scope")
	if !asValue {
		return sb
	}
	switch {
	case types.IsInvalid(valueT):
		return bad()
	case result == nil:
		// a body that always returns or throws
		sb.T = types.TNever
		return sb
	}
	return &BlockExpr{exprBase{valueT}, &Block{Stmts: []Stmt{&VarDecl{Var: result}, sb}, Value: ref(result), Type: valueT}}
}

// workersOf checks the `workers:` of a scope or gather (D156): an i64,
// evaluated before the scope opens; a constant below 1 is refused here, any
// other panics at run time.
func (f *fnCtx) workersOf(workers ast.Expr) Expr {
	if workers == nil {
		return nil
	}
	n := f.checkExprTo(workers, types.TI64)
	if types.IsInvalid(n.Type()) {
		return nil
	}
	if c, ok := f.c.tryConst(n).(*CInt); ok {
		if c.V.Sign() <= 0 {
			f.errorf(workers.Span(), "'workers' must be at least 1: it is how many of the tasks run at once, here %s (D156)", c.V)
		}
		return n
	}
	// `n < 1` panics where the scope is written, as mapConcurrent does
	at := workers.Span()
	tmp := f.newTemp(types.TI64)
	low := &Binary{exprBase{types.TBool}, OpLt, ref(tmp), &IntConst{exprBase{types.TI64}, 1, false}, at}
	stop := &Block{Stmts: []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TNever}, "panic", []Expr{&StringConst{exprBase{types.TString}, "workers must be at least 1"}}, at}}}, Type: types.TNever}
	return &Let{exprBase{types.TI64}, tmp, n, &BlockExpr{exprBase{types.TI64}, &Block{Stmts: []Stmt{&ExprStmt{X: &If{exprBase{types.TUnit}, low, stop, nil}}}, Value: ref(tmp), Type: types.TI64}}}
}

// placedStartsTasks refuses a placed or limited scope that starts no task
// of its own (D156 part 6): its body runs in the calling task, so `on:`
// would place nothing — the program meant `e.run(…)` — and `workers:`
// would limit nothing.
func (f *fnCtx) placedStartsTasks(sb *ScopeBlock, on, workers ast.Expr, pos source.Span, what string) {
	if len(sb.Launches) > 0 || (on == nil && workers == nil) || (on != nil && sb.On == nil) {
		return // (an invalid `on:` was reported)
	}
	if on != nil {
		f.errorf(pos, "this body runs in the calling task: '%s(on: %s)' places only the tasks it starts, and it starts none; run the code there with '%s.run(() => …)' (D143, D156)", what, srcText(on), srcText(on))
		return
	}
	f.errorf(pos, "'%s(workers: …)' limits how many of its tasks run at once, and this body starts none; remove 'workers:' (D156)", what)
}

func (f *fnCtx) gatherExpr(e *ast.GatherExpr) Expr {
	if f.isGlobal {
		f.errorf(e.Pos, "'gather' cannot appear in a global initializer; compute the value in 'main' and pass it down")
		return bad()
	}
	sb := &ScopeBlock{Gather: true, Span: e.Pos}
	sb.On = f.executorOf(e.On, "gather")
	sb.Workers = f.workersOf(e.Workers)
	f.scopes = append(f.scopes, sb)
	f.openTaskBlock("gather")
	sb.Body = f.checkBlock(e.Body, nil, false)
	f.closeTaskBlock()
	f.scopes = f.scopes[:len(f.scopes)-1]
	f.placedStartsTasks(sb, e.On, e.Workers, e.Pos, "gather")
	switch len(sb.Elems) {
	case 0:
		sb.T = types.TUnit
	case 1:
		// D103: one task's outcome is its Result itself — a 1-tuple exists
		// nowhere else
		sb.T = sb.Elems[0]
	default:
		sb.T = &types.Tuple{Elems: sb.Elems}
	}
	f.suspending(sb, e.Pos, "gather")
	return sb
}

func (f *fnCtx) selectExpr(arms []*ast.RaceArm, pos source.Span, want types.Type) Expr {
	if f.isGlobal {
		f.errorf(pos, "'select' cannot appear in a global initializer; compute the value in 'main' and pass it down")
		return bad()
	}
	r := &Race{}
	asValue := want != nil && !types.IsUnit(want)
	var resultType types.Type
	var nullArms []*NullConst // `=> null` arms, typed from the others at the end
	// exactly one arm runs: smart casts made inside an arm (an assignment
	// to a `var`) hold afterwards only when every arm agrees (D5), as for
	// the branches of an `if`
	saved := f.saveNarrow()
	var joined facts
	first := true
	// the race's own tasks (D168): `async f()` arms launch into a scope of
	// the race, cancelled and joined when an arm has won
	var group *ScopeBlock
	var launches []Stmt
	prefixes := map[*RaceArm]Stmt{}
	for _, arm := range arms {
		f.restoreNarrow(saved)
		f.pushScope()
		name, named := f.raceArmName(arm)
		if arm.Body == nil {
			// a bare arm (D168), removed by D169: every arm says what happens
			// when it wins
			text := srcText(arm.Source)
			f.c.errorFix(arm.Pos, fixReplace("Write 'val v = … => v'", arm.Pos, "val v = "+text+" => v"),
				"a select arm says what happens when it wins: 'val v = %s => v'; a task whose value is wanted races in 'race { }' (D169)", text)
		}
		f.inRaceArm = true
		// `ready.wait()`, `try sub.recv()`: an awaited prelude method, lowered
		// to a wait on its signal and a take when it wins (awaited.go)
		srcAST, tryAt := arm.Source, (*ast.TryExpr)(nil)
		if te, ok := srcAST.(*ast.TryExpr); ok {
			if ce, ok := te.X.(*ast.CallExpr); ok && !ce.Async {
				srcAST, tryAt = ce, te
			}
		}
		if aw, ok := srcAST.(*ast.AwaitExpr); ok {
			// the arm is itself the wait (D38); D168 removed the exception
			drop := source.Span{File: aw.Pos.File, Start: aw.Pos.Start, End: aw.X.Span().Start}
			f.c.errorFix(drop, fixReplace("Remove 'await'", drop, ""), "a select arm does not take 'await': the select is where the task waits; write '%s' (D168)", srcText(aw.X))
			srcAST = aw.X
		}
		launched := false
		switch s := srcAST.(type) {
		case *ast.CallExpr:
			launched = s.Async
		case *ast.TryExpr:
		default:
			// a handle: waited for as `await h` is, which also counts as the
			// await D156 requires of a `val h = async …`
			aw := &ast.AwaitExpr{X: s, Pos: s.Span()}
			f.raceAwait, srcAST = aw, aw
		}
		f.raceCall, f.raceTake = nil, nil
		if ce, ok := srcAST.(*ast.CallExpr); ok {
			f.raceCall = ce
		}
		var src Expr
		if launched {
			if group == nil {
				group = &ScopeBlock{Span: pos, Cancel: true}
				group.T = types.TUnit
			}
			f.scopes = append(f.scopes, group)
			f.openTaskBlock("select")
			task := f.checkExpr(srcAST, nil)
			f.closeTaskBlock()
			f.scopes = f.scopes[:len(f.scopes)-1]
			src = task
			if tt, ok := task.Type().(*types.Task); ok {
				handle := f.newTemp(tt)
				launches = append(launches, &VarDecl{Var: handle, Init: task})
				src = &AwaitTask{exprBase{tt.Result}, ref(handle)}
				f.refRaceLaunch(srcAST.(*ast.CallExpr))
			}
		} else {
			src = f.checkExpr(srcAST, nil)
		}
		take := f.raceTake
		f.raceCall, f.raceTake = nil, nil
		if tryAt != nil {
			if take != nil {
				take = f.tryOn(take, tryAt.Pos)
			} else {
				src = f.tryOn(src, tryAt.Pos)
			}
		}
		f.inRaceArm = false
		ha := &RaceArm{}
		var prefix Stmt // a lowered arm's take, run first when the arm wins
		var value *Var  // what the arm waited for: its name, or a bare arm's value
		if take != nil && !types.IsInvalid(src.Type()) {
			ha.Kind, ha.Source = RaceRecv, src
			ha.Var = f.newTemp(&types.Nullable{Elem: types.TBool})
			switch {
			case types.IsUnit(take.Type()) && (named || arm.Body == nil):
				f.raceArmNoValue(arm, named, "a 'wait()' or 'changed()'")
				prefix = &ExprStmt{X: take}
			case name != nil:
				value = f.newVar(name.Name, take.Type(), false, name.Pos)
				f.declareChecked(name.Name, value, name.Pos)
				prefix = &VarDecl{Var: value, Init: take}
			case arm.Body == nil:
				value = f.newTemp(take.Type())
				prefix = &VarDecl{Var: value, Init: take}
			default:
				prefix = &ExprStmt{X: take}
			}
			src = nil
		}
		switch s := src.(type) {
		case *Builtin:
			switch s.Op {
			case "chan.recv":
				ha.Kind, ha.Source = RaceRecv, s.Args[0]
			case "task.sleep":
				ha.Kind, ha.Source = RaceSleep, s.Args[0]
			case "chan.send":
				ha.Kind, ha.Source, ha.Value = RaceSend, s.Args[0], s.Args[1]
			}
		case *AwaitTask:
			ha.Kind, ha.Source = RaceTask, s.X
		}
		if ha.Source == nil {
			if !types.IsInvalid(src.Type()) {
				f.errorf(arm.Source.Span(), "a select arm waits on 'ch.recv()', 'ch.send(v)', 'sleep(d)', a task ('async f()' or a handle), or an Event's 'wait()', a Watch's 'changed()' or a Subscription's 'recv()' (D146, D168), not '%s'", src.Type())
			}
			f.popScope()
			continue
		}
		switch {
		case prefix != nil:
			// bound above, to the take
		case ha.Kind == RaceSend || ha.Kind == RaceSleep:
			what := "a send arm (when it wins, the value is in the channel, D108)"
			if ha.Kind == RaceSleep {
				what = "a 'sleep' arm"
			}
			if named || arm.Body == nil {
				f.raceArmNoValue(arm, named, what)
			}
			if name != nil { // declared, so its uses say nothing more
				f.declareLocal(name.Name, f.newVar(name.Name, types.TInvalid, false, name.Pos), name.Pos)
			}
		case name != nil:
			ha.Var = f.newVar(name.Name, src.Type(), false, name.Pos)
			f.declareChecked(name.Name, ha.Var, name.Pos)
			value = ha.Var
		default:
			ha.Var = f.newTemp(src.Type())
			value = ha.Var
		}
		// the arm's body: written, or (a bare arm, D168) the value it waited for
		bodyExpr := func(want types.Type) Expr {
			if arm.Body != nil {
				return f.checkExpr(arm.Body, want)
			}
			if value == nil {
				return bad()
			}
			return ref(value)
		}
		var body *Block
		if asValue {
			x := bodyExpr(want)
			at := arm.Source.Span()
			if arm.Body != nil {
				at = arm.Body.Span()
			}
			body = f.valueBlock(f.coerce(x, want, at))
		} else if want == nil {
			if _, isNull := arm.Body.(*ast.NullLit); isNull {
				// typed from the other arms at the end, as in `when` and
				// `if`: `ch.recv() as c => c` beside `... => null`
				nc := &NullConst{exprBase{&types.Nullable{Elem: types.TNever}}}
				nullArms = append(nullArms, nc)
				body = &Block{Value: nc, Type: nc.T}
			} else {
				x := bodyExpr(nil)
				body = f.valueBlock(x)
				if body.Value != nil && !types.IsNever(body.Type) {
					switch {
					case resultType == nil:
						resultType = body.Type
					case !f.assignableTo(body.Type, resultType) && f.assignableTo(resultType, body.Type):
						resultType = body.Type // a later arm is the wider type (T after T's variant, T? after T)
					}
				}
			}
		} else {
			x := bodyExpr(types.TUnit)
			var bt types.Type = types.TUnit
			if types.IsNever(x.Type()) {
				bt = types.TNever // the arm returns or throws
			}
			body = &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: bt}
		}
		if prefix != nil {
			prefixes[ha] = prefix // with `async` arms it stays in the race (selectGroup)
			body.Stmts = append([]Stmt{prefix}, body.Stmts...)
		}
		ha.Body = body
		r.Arms = append(r.Arms, ha)
		f.popScope()
		var state facts
		if !types.IsNever(body.Type) {
			state = f.saveNarrow()
		}
		if first {
			joined, first = state, false
		} else {
			joined = mergeFacts(joined, state)
		}
	}
	if len(r.Arms) == 0 {
		if len(arms) == 0 { // otherwise every arm was refused, and said why
			f.errorf(pos, "'select' needs at least one arm")
		}
		f.restoreNarrow(saved)
		return bad()
	}
	if joined == nil {
		joined = facts{}
	}
	f.narrow = joined
	allDiverge := true
	for _, a := range r.Arms {
		if !types.IsNever(a.Body.Type) {
			allDiverge = false
		}
	}
	switch {
	case allDiverge:
		// every arm returns or throws: nothing follows the race
		r.T = types.TNever
	case asValue:
		r.T = want
	case want == nil && len(nullArms) > 0 && nullableOf(resultType) == nil:
		f.errorf(pos, "cannot infer the type of 'null' in this 'select'; annotate the binding, e.g. 'val x: T? = select { ... }'")
		r.T = types.TInvalid
	case want == nil && resultType != nil:
		if len(nullArms) > 0 {
			resultType = nullableOf(resultType)
			for _, nc := range nullArms {
				nc.T = resultType
			}
			for _, a := range r.Arms {
				if _, ok := a.Body.Value.(*NullConst); ok {
					a.Body.Type = resultType
				}
			}
		}
		for _, a := range r.Arms {
			if a.Body.Value != nil && !types.IsNever(a.Body.Type) && !types.Identical(a.Body.Type, resultType) {
				a.Body.Value = f.coerce(a.Body.Value, resultType, pos)
				a.Body.Type = resultType
			}
		}
		r.T = resultType
	default:
		r.T = types.TUnit
		for _, a := range r.Arms {
			if a.Body.Value != nil {
				a.Body.Stmts = append(a.Body.Stmts, &ExprStmt{X: a.Body.Value})
				a.Body.Value = nil
				a.Body.Type = types.TUnit
			}
		}
	}
	f.suspending(r, pos, "select")
	if group != nil {
		return f.selectGroup(r, group, launches, prefixes)
	}
	return r
}

// selectGroup lowers a race with `async` arms (D168). The arms' tasks are
// children of a cancelling scope whose body is the race; the race only
// records which arm won (an arm's take, when it has one, stays in it). When
// the scope ends its other children are cancelled and joined, and only
// then does the winning arm's body run:
//
//	var won = 0
//	scope(cancel) { val t1 = async f(); …; race { t1 as v1 => won = 0; … } }
//	if (won == 0) body0 else if (won == 1) body1 … else bodyN
func (f *fnCtx) selectGroup(r *Race, group *ScopeBlock, launches []Stmt, prefixes map[*RaceArm]Stmt) Expr {
	won := f.newTemp(types.TI64)
	bodies := make([]*Block, len(r.Arms))
	for i, a := range r.Arms {
		bodies[i] = a.Body
		if p := prefixes[a]; p != nil {
			bodies[i] = &Block{Stmts: a.Body.Stmts[1:], Value: a.Body.Value, Type: a.Body.Type}
		}
		mark := &Block{Stmts: []Stmt{&Assign{Target: ref(won), Value: &IntConst{exprBase{types.TI64}, uint64(i), false}}}, Type: types.TUnit}
		if p := prefixes[a]; p != nil {
			mark.Stmts = append([]Stmt{p}, mark.Stmts...)
		}
		a.Body = mark
	}
	t := r.T
	r.T = types.TUnit
	group.Body = &Block{Stmts: append(launches, &ExprStmt{X: r}), Type: types.TUnit}
	group.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		group.ErrTo = nil // inferred: filled in at codegen time, as for `scope`
	}
	f.suspending(group, group.Span, "select")
	// the dispatch, from the last arm back
	var dispatch Expr = &BlockExpr{exprBase{bodies[len(bodies)-1].Type}, bodies[len(bodies)-1]}
	for i := len(bodies) - 2; i >= 0; i-- {
		cond := &Binary{exprBase{types.TBool}, OpEq, ref(won), &IntConst{exprBase{types.TI64}, uint64(i), false}, group.Span}
		dispatch = &If{exprBase{t}, cond, bodies[i], &Block{Value: dispatch, Type: dispatch.Type()}}
	}
	out := &Block{Stmts: []Stmt{&VarDecl{Var: won, Init: &IntConst{exprBase{types.TI64}, 0, false}}, group}, Type: t}
	if types.IsUnit(t) {
		out.Stmts = append(out.Stmts, &ExprStmt{X: dispatch})
	} else {
		out.Value = dispatch
	}
	return &BlockExpr{exprBase{t}, out}
}

// raceArmName is the name an arm gives its value: `src as name` (D168), or
// the removed `val name = src`, reported with the fix and read as the
// former. named reports that a name was written; name is nil for `_`.
func (f *fnCtx) raceArmName(arm *ast.RaceArm) (name *ast.Ident, named bool) {
	if arm.As != nil {
		// `src as x` (D168), removed by D169: the name goes before the source
		whole := source.Span{File: arm.Pos.File, Start: arm.Pos.Start, End: arm.As.Pos.End}
		text := srcText(arm.Source)
		f.c.errorFix(whole, fixReplace("Write 'val "+arm.As.Name+" ='", whole, "val "+arm.As.Name+" = "+text),
			"a select arm names its value 'val %s = %s => …' (D169)", arm.As.Name, text)
		return arm.As, true
	}
	b := arm.Binding
	if b == nil {
		return nil, false
	}
	head := source.Span{File: arm.Pos.File, Start: arm.Pos.Start, End: arm.Source.Span().Start}
	switch {
	case b.Name == nil:
		f.errorf(b.Pos, "a select arm names one value: 'val name = src => …' (D169)")
		return nil, true
	case b.Name.Name == "_":
		f.c.errorFix(head, fixReplace("Remove 'val _ ='", head, ""), "write '%s => …': an arm need not name a value it does not use (D168)", srcText(arm.Source))
		return nil, false
	}
	return b.Name, true
}

// raceArmNoValue refuses a name on an arm that waits for nothing it could
// give: a sleep, a send, an Event's wait, a Watch's changed.
func (f *fnCtx) raceArmNoValue(arm *ast.RaceArm, named bool, what string) {
	switch {
	case !named || arm.Body == nil || arm.As != nil:
		// nothing named, a bare arm or an `as` (D168), each reported as one
	case arm.Binding != nil:
		head := source.Span{File: arm.Pos.File, Start: arm.Pos.Start, End: arm.Source.Span().Start}
		f.c.errorFix(head, fixReplace("Remove the name", head, ""), "%s has no value to name; remove '%s' (D169)", what, strings.TrimSpace(srcTextSpan(head, "")))
	}
}

// refRaceLaunch records the hover of an `async` arm's keyword.
func (f *fnCtx) refRaceLaunch(call *ast.CallExpr) {
	if f.c.index == nil || !call.Pos.IsValid() {
		return
	}
	kw := source.Span{File: call.Pos.File, Start: call.Pos.Start, End: call.Pos.Start + len("async")}
	f.c.index.Refs = append(f.c.index.Refs, Ref{Span: kw, Kind: "keyword", Name: "async", Detail: "async " + srcText(call.Fun),
		Doc: "Started by this race when it starts; cancelled, and waited for, if another arm wins (D168)."})
}

// panicType is the prelude's Panic struct.
func (c *Checker) panicType() types.Type {
	if sym := c.universe.LookupLocal("Panic"); sym != nil && sym.Kind == SymType {
		return sym.Type
	}
	return nil
}

// launchTrampoline is the function `async f(args)` starts when f is a
// function value (D103): it takes the value and the arguments, evaluated in
// the parent, and calls one with the others. One per function type; its
// suspension is inferred from the call, its error is the type's.
func (c *Checker) launchTrampoline(ft *types.Func) *Func {
	key := types.Key(ft)
	if fn, ok := c.trampolines[key]; ok {
		return fn
	}
	if c.trampolines == nil {
		c.trampolines = map[string]*Func{}
	}
	newVar := func(name string, t types.Type) *Var {
		c.nextVar++
		return &Var{Name: name, Type: t, ID: c.nextVar, IsParam: true}
	}
	fv := newVar("$f", ft)
	vars := []*Var{fv}
	params := []types.Param{{Name: "f", Type: ft}}
	var args []Expr
	for i, p := range ft.Params {
		v := newVar("$a"+itoa(i), p.Type)
		vars = append(vars, v)
		params = append(params, types.Param{Name: v.Name, Type: p.Type})
		args = append(args, &VarRef{exprBase{p.Type}, v})
	}
	rt := ft.Ret
	if ft.Effects.Throws {
		rt = c.ResultType(ft.Ret, ft.Effects.Error)
	}
	var value Expr = &CallIndirect{exprBase{rt}, &VarRef{exprBase{ft}, fv}, args}
	if ft.Effects.Throws {
		value = &Try{exprBase{ft.Ret}, value, ft.Effects.Error, ft.Effects.Error}
	}
	body := &Block{Type: types.TNever}
	if types.IsUnit(ft.Ret) {
		body.Stmts = []Stmt{&ExprStmt{X: value}, &Return{}}
	} else {
		body.Stmts = []Stmt{&Return{Value: value}}
	}
	fn := &Func{Name: mangleName("veles.launch<" + key + ">"), Display: "async", Params: vars, Body: body, checked: true,
		Sig: &types.Func{Params: params, Ret: ft.Ret, Effects: types.Effects{Throws: ft.Effects.Throws, Error: ft.Effects.Error}, Sendable: true}}
	c.funcs = append(c.funcs, fn)
	c.trampolines[key] = fn
	return fn
}

// executorOf checks the executor of `scope(on: e)` / `gather(on: e)`
// (D143) and lowers it to the Executor's runtime handle; nil when there is
// none.
func (f *fnCtx) executorOf(on ast.Expr, what string) Expr {
	if on == nil {
		return nil
	}
	ex, _ := f.c.preludeType("Executor").(*types.Struct)
	var want types.Type
	if ex != nil {
		want = ex
	}
	x := f.checkExpr(on, want)
	if types.IsInvalid(x.Type()) {
		return nil
	}
	st, ok := x.Type().(*types.Struct)
	if !ok || ex == nil || !types.Identical(st, ex) {
		f.errorf(on.Span(), "'%s(on: …)' takes an Executor, not '%s': make one with 'Executor.pool(threads: n, name: …)' or 'Executor.thread(name: …)' (D143)", what, x.Type())
		return nil
	}
	for i, fld := range st.Fields {
		if fld.Name == "handle" {
			return &FieldGet{exprBase{fld.Type}, x, i, fld.Name}
		}
	}
	return nil
}
