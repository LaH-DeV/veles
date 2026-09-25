package sema

import (
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
		if t.Module == "std.prelude" && (t.Name == "Mutex" || t.Name == "Atomic") {
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

// sendableIn is sendable with the structs and sealed types already on the
// path in seen, so a type that contains itself by value (a D31 error) does
// not recurse forever.
func sendableIn(t types.Type, seen map[types.Type]bool) bool {
	switch t := t.(type) {
	case *types.Basic, *types.Enum:
		return true
	case *types.Nullable:
		return sendableIn(t.Elem, seen)
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
		if t.Module == "std.prelude" && (t.Name == "Mutex" || t.Name == "Atomic") {
			return true // explicitly synchronized wrappers (D35)
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
		f.errorf(e.Pos, "'%s' is not Sendable and cannot cross a task boundary through a channel (D35/D54)", elem)
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
			f.errorf(span, "'%s' takes %d argument(s)", name, n)
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
	f.errorf(e.Fun.Span(), "no method '%s' on '%s'", name, ct)
	f.checkArgsLoosely(e.Args)
	return bad()
}

// suspending records that the current function suspends at this point and
// rejects suspension where the executor cannot reach.
func (f *fnCtx) suspending(x Expr, span source.Span, what string) Expr {
	if f.isGlobal {
		f.errorf(span, "'%s' suspends; a global initializer cannot suspend", what)
		return x
	}
	f.fn.suspends = true
	return x
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
	var ms Expr
	if dur == nil {
		// no prelude (a bare-file test): fall back to the old milliseconds
		ms = f.checkExprTo(e.Args[0].Value, types.TI64)
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
		ms = durationMillis(dur, arg, e.Pos)
	}
	b := f.suspending(&Builtin{exprBase{types.TUnit}, "task.sleep", []Expr{ms}, e.Pos}, e.Pos, "sleep")
	if !awaited && !f.inRaceArm {
		f.errorf(e.Pos, "'sleep()' always suspends and must be awaited: 'await sleep(d)' (D16)")
	}
	return b
}

// durationMillis is `(d.ns + 999999) / 1000000`: the whole milliseconds a
// Duration covers, rounded up, so a sleep is never shorter than it was asked
// for. `d` is read once, so an argument with side effects is evaluated once.
// Zero gives 0 and a negative duration a negative count, both of which
// `veles_task_sleep` reads as "yield". The one value this does not hold for
// is a duration within a millisecond of the largest one representable, where
// the `+ 999999` overflows — 292 years, checked in a debug build (D21).
func durationMillis(dur *types.Struct, d Expr, span source.Span) Expr {
	ns := &FieldGet{exprBase{types.TI64}, d, fieldIndex(dur, "ns"), "ns"}
	up := &Binary{exprBase{types.TI64}, OpAdd, ns, &IntConst{exprBase{types.TI64}, 999999, false}, span}
	return &Binary{exprBase{types.TI64}, OpDiv, up, &IntConst{exprBase{types.TI64}, 1000000, false}, span}
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
	f.awaitNext = true
	x := f.checkExpr(e.X, nil)
	consumed := !f.awaitNext
	f.awaitNext = false
	if consumed {
		return x
	}
	if tt, ok := x.Type().(*types.Task); ok {
		return f.suspending(&AwaitTask{exprBase{tt.Result}, x}, e.Pos, "await")
	}
	if !types.IsInvalid(x.Type()) {
		f.errorf(e.Pos, "'await' applies to channels, timers and task handles, not '%s' (D16)", x.Type())
	}
	return bad()
}

// launch checks `async call(...)`.
func (f *fnCtx) launch(e *ast.CallExpr, want types.Type) Expr {
	if len(f.scopes) == 0 {
		f.errorf(e.Pos, "'async' must be lexically inside a 'scope' or 'gather' block: tasks cannot outlive their scope (D3/D34)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	inner := *e
	inner.Async = false
	x := f.checkExpr(&inner, nil)
	call, ok := x.(*Call)
	if !ok {
		if !types.IsInvalid(x.Type()) {
			f.errorf(e.Pos, "'async' launches a direct call of a named function or method")
		}
		return bad()
	}
	for i, a := range call.Args {
		t := a.Type()
		if i == 0 && call.Fn.Receiver != nil {
			// the receiver crosses as a copy — a snapshot, like the captures
			// of a sendable closure — never as a pointer into this task
			p := t.(*types.Pointer)
			t = p.Elem
			tmp := f.newTemp(p.Elem)
			copy := &Let{exprBase{p.Elem}, tmp, &Deref{exprBase{p.Elem}, a}, ref(tmp)}
			call.Args[0] = &AddrOf{exprBase{p}, copy}
			call.Recv = RecvNone
		}
		if !sendable(t) {
			span := e.Pos
			if j := i - len(call.Args) + len(e.Args); j >= 0 && j < len(e.Args) {
				span = e.Args[j].Value.Span()
			}
			f.errorf(span, "argument of type '%s' is not Sendable and cannot cross a task boundary (D35)", t)
		}
	}
	sc := f.scopes[len(f.scopes)-1]
	l := &Launch{exprBase{&types.Task{Result: call.Type()}}, call, sc, len(sc.Launches)}
	sc.Launches = append(sc.Launches, l)
	if sc.Gather {
		// D52: a panic surfaces as a Result error at the gather boundary
		okT, errT := call.Type(), types.Type(nil)
		if rs, isR := call.Type().(*types.Sealed); isR && isResultType(rs) {
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
	} else if isResultType(call.Type()) {
		// fail-fast: the scope rethrows the child's error
		et := call.Type().(*types.Sealed).TypeArgs[1]
		if !types.IsNever(et) {
			for _, m := range types.UnionMembers(et) {
				f.recordError(m, e.Pos)
			}
		}
	}
	return l
}

func (f *fnCtx) scopeStmt(s *ast.ScopeStmt) []Stmt {
	if f.isGlobal {
		f.errorf(s.Pos, "'scope' cannot appear in a global initializer")
		return nil
	}
	sb := &ScopeBlock{Span: s.Pos}
	sb.T = types.TUnit
	f.scopes = append(f.scopes, sb)
	sb.Body = f.checkBlock(s.Body, nil, false)
	f.scopes = f.scopes[:len(f.scopes)-1]
	sb.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		// inferred: filled in at codegen time from the function signature
		sb.ErrTo = nil
	}
	f.suspending(sb, s.Pos, "scope")
	return []Stmt{sb}
}

func (f *fnCtx) gatherExpr(e *ast.GatherExpr) Expr {
	if f.isGlobal {
		f.errorf(e.Pos, "'gather' cannot appear in a global initializer")
		return bad()
	}
	sb := &ScopeBlock{Gather: true, Span: e.Pos}
	f.scopes = append(f.scopes, sb)
	sb.Body = f.checkBlock(e.Body, nil, false)
	f.scopes = f.scopes[:len(f.scopes)-1]
	sb.T = &types.Tuple{Elems: sb.Elems}
	if len(sb.Elems) == 0 {
		sb.T = types.TUnit
	}
	f.suspending(sb, e.Pos, "gather")
	return sb
}

func (f *fnCtx) raceExpr(e *ast.RaceExpr, want types.Type) Expr {
	if f.isGlobal {
		f.errorf(e.Pos, "'race' cannot appear in a global initializer")
		return bad()
	}
	r := &Race{}
	asValue := want != nil && !types.IsUnit(want)
	var resultType types.Type
	// exactly one arm runs: smart casts made inside an arm (an assignment
	// to a `var`) hold afterwards only when every arm agrees (D5), as for
	// the branches of an `if`
	saved := f.saveNarrow()
	var joined facts
	first := true
	for _, arm := range e.Arms {
		f.restoreNarrow(saved)
		f.pushScope()
		f.inRaceArm = true
		src := f.checkExpr(arm.Source, nil)
		f.inRaceArm = false
		ha := &RaceArm{}
		switch s := src.(type) {
		case *Builtin:
			switch s.Op {
			case "chan.recv":
				ha.Kind, ha.Source = RaceRecv, s.Args[0]
			case "task.sleep":
				ha.Kind, ha.Source = RaceSleep, s.Args[0]
			}
		case *AwaitTask:
			ha.Kind, ha.Source = RaceTask, s.X
		}
		if ha.Source == nil {
			if !types.IsInvalid(src.Type()) {
				f.errorf(arm.Source.Span(), "a race arm waits on 'ch.recv()', 'sleep(d)' or 'await task', not '%s'", src.Type())
			}
			f.popScope()
			continue
		}
		if arm.Binding != nil {
			if arm.Binding.Name == nil {
				f.errorf(arm.Binding.Pos, "race arms bind a single name")
			} else {
				vt := src.Type()
				ha.Var = f.newVar(arm.Binding.Name.Name, vt, false, arm.Binding.Name.Pos)
				f.declareChecked(arm.Binding.Name.Name, ha.Var, arm.Binding.Name.Pos)
			}
		} else if ha.Kind != RaceSleep {
			ha.Var = f.newTemp(src.Type())
		}
		var body *Block
		if asValue {
			body = f.valueBlock(f.checkExprTo(arm.Body, want))
		} else if want == nil {
			x := f.checkExpr(arm.Body, nil)
			body = f.valueBlock(x)
			if body.Value != nil && resultType == nil && !types.IsNever(body.Type) {
				resultType = body.Type
			}
		} else {
			x := f.checkExpr(arm.Body, types.TUnit)
			var bt types.Type = types.TUnit
			if types.IsNever(x.Type()) {
				bt = types.TNever // the arm returns or throws
			}
			body = &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: bt}
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
		f.errorf(e.Pos, "'race' needs at least one arm")
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
	case want == nil && resultType != nil:
		for _, a := range r.Arms {
			if a.Body.Value != nil && !types.IsNever(a.Body.Type) && !types.Identical(a.Body.Type, resultType) {
				a.Body.Value = f.coerce(a.Body.Value, resultType, e.Pos)
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
	f.suspending(r, e.Pos, "race")
	return r
}

// panicType is the prelude's Panic struct.
func (c *Checker) panicType() types.Type {
	if sym := c.universe.LookupLocal("Panic"); sym != nil && sym.Kind == SymType {
		return sym.Type
	}
	return nil
}
