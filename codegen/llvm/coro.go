package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// Suspending functions (D2) are LLVM switched-resume coroutines: the
// ramp takes the task it runs as, allocates its frame on the GC heap and
// returns the coroutine handle at the first suspension. Every suspension
// point re-checks cancellation on resume (D20: cancellation is delivered at
// a suspension point) and runs `with` cleanups before finishing.

const coroDecls = `declare token @llvm.coro.id(i32, ptr, ptr, ptr)
declare i64 @llvm.coro.size.i64()
declare ptr @llvm.coro.begin(token, ptr)
declare i8 @llvm.coro.suspend(token, i1)
declare ptr @llvm.coro.free(token, ptr)
declare i1 @llvm.coro.end(ptr, i1, token)
declare ptr @veles_task_new()
declare void @veles_task_started(ptr, ptr)
declare void @veles_task_finish(ptr, ptr, i64, i64)
declare i64 @veles_task_cancelled(ptr)
declare void @veles_task_finish_cancelled(ptr)
declare i64 @veles_task_await(ptr, ptr)
declare ptr @veles_task_result(ptr)
declare i64 @veles_task_failed(ptr)
declare ptr @veles_scope_begin(ptr, i64)
declare ptr @veles_task_launch(ptr)
declare i64 @veles_scope_wait(ptr, ptr)
declare ptr @veles_scope_failed(ptr)
declare i64 @veles_scope_failed_index(ptr)
declare ptr @veles_chan_new(ptr, i64)
declare i64 @veles_chan_send(ptr, ptr, ptr)
declare i64 @veles_chan_recv(ptr, ptr, ptr)
declare void @veles_task_leave_waits(ptr)
declare void @veles_chan_close(ptr)
declare void @veles_chan_close_after(ptr, i64)
declare i64 @veles_chan_len(ptr)
declare i64 @veles_task_sleep(ptr, i64)
declare ptr @veles_race_new(ptr)
declare void @veles_race_recv(ptr, ptr, ptr)
declare void @veles_race_sleep(ptr, i64)
declare void @veles_race_await(ptr, ptr)
declare i64 @veles_race_wait(ptr, ptr)
declare i64 @veles_race_closed(ptr)
declare void @veles_run(ptr)
declare i64 @veles_task_panicked(ptr)
declare ptr @veles_task_panic_msg(ptr, ptr)
declare void @veles_task_repanic(ptr)
declare void @veles_task_start(ptr, ptr, ptr)
`

type coroState struct {
	id, hdl  string
	task     string
	finalL   string
	cleanupL string
	suspendL string
}

// coroPrologue emits the coroutine setup at the start of a ramp.
func (g *gen) coroPrologue() {
	c := &coroState{task: "%task", finalL: "coro.final", cleanupL: "coro.cleanup", suspendL: "coro.suspend"}
	g.coro = c
	g.emit("%%coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)")
	g.emit("%%coro.size = call i64 @llvm.coro.size.i64()")
	g.emit("%%coro.mem = call ptr @veles_alloc_words(i64 %%coro.size)")
	g.emit("%%coro.hdl = call ptr @llvm.coro.begin(token %%coro.id, ptr %%coro.mem)")
	c.id, c.hdl = "%coro.id", "%coro.hdl"
}

// coroEpilogue emits the shared final-suspend, cleanup and exit blocks.
func (g *gen) coroEpilogue() {
	c := g.coro
	g.placeLabel(c.finalL)
	g.emit("%%coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)")
	g.emitTerm("switch i8 %%coro.fs, label %%%s [ i8 0, label %%coro.trap i8 1, label %%%s ]", c.suspendL, c.cleanupL)
	g.placeLabel("coro.trap")
	g.emitTerm("unreachable")
	g.placeLabel(c.cleanupL)
	g.emit("%%coro.freed = call ptr @llvm.coro.free(token %s, ptr %s)", c.id, c.hdl)
	g.emitTerm("br label %%%s", c.suspendL)
	g.placeLabel(c.suspendL)
	g.emit("%%coro.ended = call i1 @llvm.coro.end(ptr %s, i1 false, token none)", c.hdl)
	g.emitTerm("ret ptr %s", c.hdl)
}

// bodyScope is an enclosing fail-fast scope as seen from a suspension
// point in its body: where its scope pointer lives, the label of its join
// loop, and the cleanup depth at its start.
type bodyScope struct {
	slot, wait string
	cleanups   int
}

// suspendPoint yields to the executor and, on resume, honours cancellation.
func (g *gen) suspendPoint() {
	c := g.coro
	if c == nil {
		panic("codegen: suspension point in a non-suspending function")
	}
	s := g.newTmp()
	g.emit("%s = call i8 @llvm.coro.suspend(token none, i1 false)", s)
	resume := g.newLabel("resume")
	g.emitTerm("switch i8 %s, label %%%s [ i8 0, label %%%s i8 1, label %%%s ]", s, c.suspendL, resume, c.cleanupL)
	g.placeLabel(resume)
	cc := g.newTmp()
	g.emit("%s = call i64 @veles_task_cancelled(ptr %s)", cc, c.task)
	cb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", cb, cc)
	cancel, cont := g.newLabel("cancelled"), g.newLabel("cont")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", cb, cancel, cont)
	g.placeLabel(cancel)
	g.runCleanups(0)
	g.emit("call void @veles_task_finish_cancelled(ptr %s)", c.task)
	g.emitTerm("br label %%%s", c.finalL)
	g.placeLabel(cont)
	// fail-fast (D34): a child of an enclosing scope has failed, so the body
	// stops here — whatever it was waiting for will not come — and the
	// innermost scope joins its children and re-raises
	if n := len(g.bodyScopes); n > 0 {
		abort := g.newLabel("scope.abort")
		for _, bs := range g.bodyScopes {
			scv := g.newTmp()
			g.emit("%s = load ptr, ptr %s", scv, bs.slot)
			ft := g.newTmp()
			g.emit("%s = call ptr @veles_scope_failed(ptr %s)", ft, scv)
			fb := g.newTmp()
			g.emit("%s = icmp ne ptr %s, null", fb, ft)
			next := g.newLabel("scope.ok")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", fb, abort, next)
			g.placeLabel(next)
		}
		go_on := g.newLabel("cont")
		g.emitTerm("br label %%%s", go_on)
		g.placeLabel(abort)
		inner := g.bodyScopes[n-1]
		g.emit("call void @veles_task_leave_waits(ptr %s)", c.task)
		g.runCleanups(inner.cleanups)
		g.emitTerm("br label %%%s", inner.wait)
		g.placeLabel(go_on)
	}
}

// coroReturn finishes the task with the (already Result-wrapped) value.
func (g *gen) coroReturn(llt, v string, isResult bool) {
	c := g.coro
	if llt == "void" || llt == "" {
		g.emit("call void @veles_task_finish(ptr %s, ptr null, i64 0, i64 0)", c.task)
		g.emitTerm("br label %%%s", c.finalL)
		return
	}
	slot := g.alloca(llt)
	g.emit("store %s %s, ptr %s", llt, v, slot)
	failed := "0"
	if isResult {
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, llt, v)
		f := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", f, tag)
		failed = g.newTmp()
		g.emit("%s = zext i1 %s to i64", failed, f)
	}
	size := g.llSize(llt)
	g.emit("call void @veles_task_finish(ptr %s, ptr %s, i64 %d, i64 %s)", c.task, slot, size, failed)
	g.emitTerm("br label %%%s", c.finalL)
}

// llSize returns the byte size of the function's logical return value.
func (g *gen) llSize(llt string) int {
	if g.fnResult != nil {
		s, _ := g.layout(g.fnResult)
		return s
	}
	s, _ := g.layout(g.fn.Sig.Ret)
	return s
}

// awaitTask blocks until a task finishes and returns its result value.
func (g *gen) awaitTask(task string, rt types.Type) string {
	wait, got, susp := g.newLabel("await"), g.newLabel("await.got"), g.newLabel("await.susp")
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(wait)
	r := g.newTmp()
	g.emit("%s = call i64 @veles_task_await(ptr %s, ptr %s)", r, g.coro.task, task)
	rb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", rb, r)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, got, susp)
	g.placeLabel(susp)
	g.suspendPoint()
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(got)
	if types.IsUnit(rt) || types.IsNever(rt) {
		return "zeroinitializer"
	}
	rp := g.newTmp()
	g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, task)
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, g.llType(rt), rp)
	return v
}

// callSuspending runs a suspending function as a child task and awaits it.
func (g *gen) callSuspending(fn *sema.Func, argTypes []types.Type, argVals []string, rt types.Type) string {
	ct := g.newTmp()
	g.emit("%s = call ptr @veles_task_new()", ct)
	g.startTask(ct, fn, argTypes, argVals)
	return g.awaitTask(ct, rt)
}

// resultTypeOf is the logical result type of a function (Result when it throws).
func (g *gen) resultTypeOf(sig *types.Func) types.Type {
	if sig.Effects.Throws {
		return g.prog.ResultType(sig.Ret, sig.Effects.Error)
	}
	return sig.Ret
}

func (g *gen) launch(e *sema.Launch) string {
	scope := g.newTmp()
	g.emit("%s = load ptr, ptr %s", scope, g.scopeSlots[e.Scope])
	ct := g.newTmp()
	g.emit("%s = call ptr @veles_task_launch(ptr %s)", ct, scope)
	var argTypes []types.Type
	var argVals []string
	for _, a := range e.Call.Args {
		argTypes = append(argTypes, a.Type())
		argVals = append(argVals, g.expr(a))
	}
	g.startTask(ct, e.Call.Fn, argTypes, argVals)
	slot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", ct, slot)
	g.launchSlots[e] = slot
	return ct
}

// rampFor wraps a non-suspending function as a coroutine so it can be
// launched as a task.
func (g *gen) rampFor(fn *sema.Func) *sema.Func {
	if r, ok := g.ramps[fn]; ok {
		return r
	}
	ramp := &sema.Func{Name: "ramp." + fn.Name, Display: fn.Display, Sig: fn.Sig, Suspends: true}
	g.ramps[fn] = ramp
	g.pending = append(g.pending, func() {
		var params, args []string
		params = append(params, "ptr %task")
		i := 0
		if fn.Receiver != nil {
			llt := g.llType(fn.Receiver.Type)
			params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
			args = append(args, fmt.Sprintf("%s %%p%d", llt, i))
			i++
		}
		for _, p := range fn.Params {
			llt := g.llType(p.Type)
			params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
			args = append(args, fmt.Sprintf("%s %%p%d", llt, i))
			i++
		}
		saved := g.fnResult
		g.defineCoroHelper(ramp, params, func() {
			ret := g.retLL(fn)
			if ret == "void" {
				g.emit("call void @%s(%s)", fn.Name, joinArgs(args))
				g.coroReturn("void", "", false)
				return
			}
			v := g.newTmp()
			g.emit("%s = call %s @%s(%s)", v, ret, fn.Name, joinArgs(args))
			g.coroReturn(ret, v, fn.Sig.Effects.Throws)
		})
		g.fnResult = saved
	})
	return ramp
}

// defineCoroHelper emits a helper that is itself a coroutine.
func (g *gen) defineCoroHelper(fn *sema.Func, params []string, body func()) {
	g.defineHelperEx(fn.Name, "ptr", params, "presplitcoroutine", func() {
		g.fn = fn
		if fn.Sig.Effects.Throws {
			g.fnResult = g.prog.ResultType(fn.Sig.Ret, fn.Sig.Effects.Error)
		}
		g.coroPrologue()
		body()
		g.coroEpilogue()
		g.coro = nil
	})
}

func (g *gen) scopeBlock(e *sema.ScopeBlock) string {
	failFast := "1"
	if e.Gather {
		failFast = "0"
	}
	sc := g.newTmp()
	g.emit("%s = call ptr @veles_scope_begin(ptr %s, i64 %s)", sc, g.coro.task, failFast)
	slot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", sc, slot)
	g.scopeSlots[e] = slot
	wait, done, susp := g.newLabel("scope.wait"), g.newLabel("scope.done"), g.newLabel("scope.susp")
	if !e.Gather {
		g.bodyScopes = append(g.bodyScopes, bodyScope{slot: slot, wait: wait, cleanups: len(g.cleanups)})
	}
	g.block(e.Body)
	if !e.Gather {
		g.bodyScopes = g.bodyScopes[:len(g.bodyScopes)-1]
	}
	if g.term {
		return "zeroinitializer"
	}
	// wait for every child
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(wait)
	scv := g.newTmp()
	g.emit("%s = load ptr, ptr %s", scv, slot)
	r := g.newTmp()
	g.emit("%s = call i64 @veles_scope_wait(ptr %s, ptr %s)", r, g.coro.task, scv)
	rb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", rb, r)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
	g.placeLabel(susp)
	g.suspendPoint()
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(done)
	if e.Gather {
		return g.gatherResults(e)
	}
	// fail-fast: a panicking child re-raises here (D52); a failed child's
	// error is rethrown
	scv2 := g.newTmp()
	g.emit("%s = load ptr, ptr %s", scv2, slot)
	failedTask := g.newTmp()
	g.emit("%s = call ptr @veles_scope_failed(ptr %s)", failedTask, scv2)
	hasFailed := g.newTmp()
	g.emit("%s = icmp ne ptr %s, null", hasFailed, failedTask)
	checkL, afterL := g.newLabel("scope.check"), g.newLabel("scope.after")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", hasFailed, checkL, afterL)
	g.placeLabel(checkL)
	pan := g.newTmp()
	g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, failedTask)
	pb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", pb, pan)
	repanicL, errorsL := g.newLabel("scope.repanic"), g.newLabel("scope.errors")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, repanicL, errorsL)
	g.placeLabel(repanicL)
	g.emit("call void @veles_task_repanic(ptr %s)", failedTask)
	g.emitTerm("unreachable")
	g.placeLabel(errorsL)
	var throwing []*sema.Launch
	for _, l := range e.Launches {
		if isResultType(l.Call.Type()) {
			throwing = append(throwing, l)
		}
	}
	if len(throwing) == 0 {
		g.placeLabel(afterL)
		return "zeroinitializer"
	}
	fi := g.newTmp()
	g.emit("%s = call i64 @veles_scope_failed_index(ptr %s)", fi, scv2)
	end := g.newLabel("scope.end")
	var cases []string
	labels := map[*sema.Launch]string{}
	for _, l := range throwing {
		labels[l] = g.newLabel("scope.fail")
		cases = append(cases, fmt.Sprintf("i64 %d, label %%%s", l.Index, labels[l]))
	}
	g.emitTerm("switch i64 %s, label %%%s [ %s ]", fi, end, strings.Join(cases, " "))
	for _, l := range throwing {
		g.placeLabel(labels[l])
		ft := g.newTmp()
		g.emit("%s = call ptr @veles_scope_failed(ptr %s)", ft, scv2)
		rp := g.newTmp()
		g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, ft)
		rs := l.Call.Type().(*types.Sealed)
		rv := g.newTmp()
		g.emit("%s = load %s, ptr %s", rv, g.llType(rs), rp)
		errVariant := rs.Variants[1]
		payload := g.extractTagged(g.llType(rs), rv, g.llType(errVariant))
		ev := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", ev, g.llType(errVariant), payload)
		g.throwValue(ev, errVariant.Fields[0].Type)
	}
	g.placeLabel(end)
	g.placeLabel(afterL)
	return "zeroinitializer"
}

func (g *gen) gatherResults(e *sema.ScopeBlock) string {
	if len(e.Launches) == 0 {
		return "zeroinitializer"
	}
	tt := e.Type().(*types.Tuple)
	llt := g.llType(tt)
	acc := "undef"
	for i, l := range e.Launches {
		slot := g.launchSlots[l]
		ct := g.newTmp()
		g.emit("%s = load ptr, ptr %s", ct, slot)
		rp := g.newTmp()
		g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, ct)
		childT := l.Call.Type()
		want := tt.Elems[i].(*types.Sealed)
		wantLL := g.llType(want)
		res := g.alloca(wantLL)
		pan := g.newTmp()
		g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, ct)
		pb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", pb, pan)
		panL, okL, joinL := g.newLabel("gather.panic"), g.newLabel("gather.value"), g.newLabel("gather.join")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, panL, okL)
		g.placeLabel(panL)
		// D52: the panic becomes Err(Panic) in the union
		pt := g.prog.PanicType.(*types.Struct)
		pv := g.convertError(g.panicValue(ct, pt), pt, want.Variants[1].Fields[0].Type)
		pe := g.makeTagged(wantLL, 1, g.llType(want.Variants[1]), g.buildStruct(want.Variants[1], []string{pv}))
		g.emit("store %s %s, ptr %s", wantLL, pe, res)
		g.emitTerm("br label %%%s", joinL)
		g.placeLabel(okL)
		var v string
		if rs, isR := childT.(*types.Sealed); isR && isResultType(rs) {
			cv := g.newTmp()
			g.emit("%s = load %s, ptr %s", cv, g.llType(rs), rp)
			v = g.convertResult(cv, rs, want)
		} else {
			val := "zeroinitializer"
			if !types.IsUnit(childT) {
				val = g.newTmp()
				g.emit("%s = load %s, ptr %s", val, g.llType(childT), rp)
			}
			v = g.makeTagged(wantLL, 0, g.llType(want.Variants[0]), g.buildStruct(want.Variants[0], []string{val}))
		}
		g.emit("store %s %s, ptr %s", wantLL, v, res)
		g.emitTerm("br label %%%s", joinL)
		g.placeLabel(joinL)
		v = g.newTmp()
		g.emit("%s = load %s, ptr %s", v, wantLL, res)
		n := g.newTmp()
		g.emit("%s = insertvalue %s %s, %s %s, %d", n, llt, acc, wantLL, v, i)
		acc = n
	}
	return acc
}

func (g *gen) race(e *sema.Race) string {
	r := g.newTmp()
	g.emit("%s = call ptr @veles_race_new(ptr %s)", r, g.coro.task)
	slots := make([]string, len(e.Arms))
	for i, arm := range e.Arms {
		switch arm.Kind {
		case sema.RaceRecv:
			ch := g.expr(arm.Source)
			ct := arm.Source.Type().(*types.Channel)
			slots[i] = g.alloca(g.llType(ct.Elem))
			g.emit("store %s zeroinitializer, ptr %s", g.llType(ct.Elem), slots[i])
			g.emit("call void @veles_race_recv(ptr %s, ptr %s, ptr %s)", r, ch, slots[i])
		case sema.RaceSleep:
			ms := g.expr(arm.Source)
			g.emit("call void @veles_race_sleep(ptr %s, i64 %s)", r, ms)
		case sema.RaceTask:
			t := g.expr(arm.Source)
			slots[i] = g.alloca("ptr")
			g.emit("store ptr %s, ptr %s", t, slots[i])
			g.emit("call void @veles_race_await(ptr %s, ptr %s)", r, t)
		}
	}
	wait, susp := g.newLabel("race.wait"), g.newLabel("race.susp")
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(wait)
	w := g.newTmp()
	g.emit("%s = call i64 @veles_race_wait(ptr %s, ptr %s)", w, g.coro.task, r)
	ready := g.newTmp()
	g.emit("%s = icmp sge i64 %s, 0", ready, w)
	dispatch := g.newLabel("race.dispatch")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", ready, dispatch, susp)
	g.placeLabel(susp)
	g.suspendPoint()
	g.emitTerm("br label %%%s", wait)
	g.placeLabel(dispatch)
	hasValue := !types.IsUnit(e.Type()) && !types.IsNever(e.Type())
	var res string
	llt := g.llType(e.Type())
	if hasValue {
		res = g.alloca(llt)
	}
	end := g.newLabel("race.end")
	labels := make([]string, len(e.Arms))
	var cases []string
	for i := range e.Arms {
		labels[i] = g.newLabel("race.arm")
		cases = append(cases, fmt.Sprintf("i64 %d, label %%%s", i, labels[i]))
	}
	g.emitTerm("switch i64 %s, label %%%s [ %s ]", w, end, strings.Join(cases, " "))
	for i, arm := range e.Arms {
		g.placeLabel(labels[i])
		if arm.Var != nil {
			st := g.declareVar(arm.Var)
			switch arm.Kind {
			case sema.RaceRecv:
				ct := arm.Source.Type().(*types.Channel)
				nt := arm.Var.Type.(*types.Nullable)
				closed := g.newTmp()
				g.emit("%s = call i64 @veles_race_closed(ptr %s)", closed, r)
				open := g.newTmp()
				g.emit("%s = icmp eq i64 %s, 0", open, closed)
				val := g.newTmp()
				g.emit("%s = load %s, ptr %s", val, g.llType(ct.Elem), slots[i])
				g.emit("store %s %s, ptr %s", g.llType(nt), g.makeNullable(nt, open, val), st)
			case sema.RaceTask:
				t := g.newTmp()
				g.emit("%s = load ptr, ptr %s", t, slots[i])
				rp := g.newTmp()
				g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, t)
				if !types.IsUnit(arm.Var.Type) {
					v := g.newTmp()
					g.emit("%s = load %s, ptr %s", v, g.llType(arm.Var.Type), rp)
					g.emit("store %s %s, ptr %s", g.llType(arm.Var.Type), v, st)
				}
			}
		}
		v := g.block(arm.Body)
		if hasValue && !g.term && arm.Body.Value != nil {
			g.emit("store %s %s, ptr %s", llt, v, res)
		}
		if !g.term {
			g.emitTerm("br label %%%s", end)
		}
	}
	g.placeLabel(end)
	if !hasValue {
		return "zeroinitializer"
	}
	out := g.newTmp()
	g.emit("%s = load %s, ptr %s", out, llt, res)
	return out
}

// channel and timer builtins
func (g *gen) taskBuiltin(e *sema.Builtin) (string, bool) {
	switch e.Op {
	case "chan.new":
		ct := e.Type().(*types.Channel)
		cap := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call ptr @veles_chan_new(ptr %s, i64 %s)", v, g.arrayDescOf(ct.Elem), cap)
		return v, true
	case "chan.send":
		ct := e.Args[0].Type().(*types.Channel)
		ch := g.expr(e.Args[0])
		x := g.expr(e.Args[1])
		slot := g.alloca(g.llType(ct.Elem))
		g.emit("store %s %s, ptr %s", g.llType(ct.Elem), x, slot)
		loop, done, susp := g.newLabel("send"), g.newLabel("send.done"), g.newLabel("send.susp")
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(loop)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_chan_send(ptr %s, ptr %s, ptr %s)", r, g.coro.task, ch, slot)
		rb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", rb, r)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
		g.placeLabel(susp)
		g.suspendPoint()
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(done)
		return "zeroinitializer", true
	case "chan.recv":
		ct := e.Args[0].Type().(*types.Channel)
		nt := e.Type().(*types.Nullable)
		ch := g.expr(e.Args[0])
		slot := g.alloca(g.llType(ct.Elem))
		g.emit("store %s zeroinitializer, ptr %s", g.llType(ct.Elem), slot)
		loop, done, susp := g.newLabel("recv"), g.newLabel("recv.done"), g.newLabel("recv.susp")
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(loop)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_chan_recv(ptr %s, ptr %s, ptr %s)", r, g.coro.task, ch, slot)
		rb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", rb, r)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
		g.placeLabel(susp)
		g.suspendPoint()
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(done)
		got := g.newTmp()
		g.emit("%s = icmp eq i64 %s, 1", got, r)
		val := g.newTmp()
		g.emit("%s = load %s, ptr %s", val, g.llType(ct.Elem), slot)
		return g.makeNullable(nt, got, val), true
	case "chan.close":
		ch := g.expr(e.Args[0])
		g.emit("call void @veles_chan_close(ptr %s)", ch)
		return "zeroinitializer", true
	case "chan.closeAfter":
		ch := g.expr(e.Args[0])
		n := g.expr(e.Args[1])
		g.emit("call void @veles_chan_close_after(ptr %s, i64 %s)", ch, n)
		return "zeroinitializer", true
	case "chan.len":
		ch := g.expr(e.Args[0])
		v := g.newTmp()
		g.emit("%s = call i64 @veles_chan_len(ptr %s)", v, ch)
		return v, true
	case "task.sleep":
		ms := g.expr(e.Args[0])
		loop, done, susp := g.newLabel("sleep"), g.newLabel("sleep.done"), g.newLabel("sleep.susp")
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(loop)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_task_sleep(ptr %s, i64 %s)", r, g.coro.task, ms)
		rb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", rb, r)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
		g.placeLabel(susp)
		g.suspendPoint()
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(done)
		return "zeroinitializer", true
	}
	return "", false
}

func isResultType(t types.Type) bool {
	s, ok := t.(*types.Sealed)
	return ok && s.Template != nil && s.Template.Name == "Result" && s.Template.Module == "<prelude>"
}

// runRoot starts main as the root task and drives the executor until it
// completes; returns the root task.
// startRoot runs fn as the root task to completion (panics are captured
// in the task) and returns the task.
func (g *gen) startRoot(fn *sema.Func) string {
	root := g.newTmp()
	g.emit("%s = call ptr @veles_task_new()", root)
	g.startTask(root, fn, nil, nil)
	g.emit("call void @veles_run(ptr %s)", root)
	return root
}

func (g *gen) runRoot(main *sema.Func) string {
	root := g.startRoot(main)
	// a panic that reached the root task ends the program
	pan := g.newTmp()
	g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, root)
	pb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", pb, pan)
	dieL, okL := g.newLabel("root.panic"), g.newLabel("root.ok")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, dieL, okL)
	g.placeLabel(dieL)
	g.emit("call void @veles_task_repanic(ptr %s)", root)
	g.emitTerm("unreachable")
	g.placeLabel(okL)
	return root
}

// panicValue builds the prelude Panic struct for a panicked task.
func (g *gen) panicValue(task string, pt *types.Struct) string {
	lenSlot := g.alloca("i64")
	p := g.newTmp()
	g.emit("%s = call ptr @veles_task_panic_msg(ptr %s, ptr %s)", p, task, lenSlot)
	l := g.newTmp()
	g.emit("%s = load i64, ptr %s", l, lenSlot)
	a := g.newTmp()
	g.emit("%s = insertvalue %s undef, ptr %s, 0", a, strType, p)
	s := g.newTmp()
	g.emit("%s = insertvalue %s %s, i64 %s, 1", s, strType, a, l)
	return g.buildStruct(pt, []string{s})
}

// convertResult widens Result<T, E1> to Result<T, E2> (E1 within E2).
func (g *gen) convertResult(v string, from, to *types.Sealed) string {
	if from == to {
		return v
	}
	fromLL, toLL := g.llType(from), g.llType(to)
	res := g.alloca(toLL)
	tag := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", tag, fromLL, v)
	isErr := g.newTmp()
	g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
	errL, okL, endL := g.newLabel("conv.err"), g.newLabel("conv.ok"), g.newLabel("conv.end")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
	g.placeLabel(okL)
	okPayload := g.extractTagged(fromLL, v, g.llType(from.Variants[0]))
	okVal := "zeroinitializer"
	if g.llType(from.Variants[0]) != "{}" {
		okVal = g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", okVal, g.llType(from.Variants[0]), okPayload)
	}
	okNew := g.makeTagged(toLL, 0, g.llType(to.Variants[0]), g.buildStruct(to.Variants[0], []string{okVal}))
	g.emit("store %s %s, ptr %s", toLL, okNew, res)
	g.emitTerm("br label %%%s", endL)
	g.placeLabel(errL)
	errPayload := g.extractTagged(fromLL, v, g.llType(from.Variants[1]))
	errVal := "zeroinitializer"
	if g.llType(from.Variants[1]) != "{}" {
		errVal = g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", errVal, g.llType(from.Variants[1]), errPayload)
	}
	conv := g.convertError(errVal, from.Variants[1].Fields[0].Type, to.Variants[1].Fields[0].Type)
	errNew := g.makeTagged(toLL, 1, g.llType(to.Variants[1]), g.buildStruct(to.Variants[1], []string{conv}))
	g.emit("store %s %s, ptr %s", toLL, errNew, res)
	g.emitTerm("br label %%%s", endL)
	g.placeLabel(endL)
	out := g.newTmp()
	g.emit("%s = load %s, ptr %s", out, toLL, res)
	return out
}

// testRunner emits the `veles test` entry point: every @test function runs
// as a task in declaration order; a returned error or a panic fails the
// test, and the exit code reports the outcome (§4b: testing is a build
// mode).
func (g *gen) testRunner() {
	failures := g.alloca("i32")
	g.emit("store i32 0, ptr %s", failures)
	fail := func(text string) {
		p, l := g.strPtrLen(text)
		g.emit("call void @veles_print(ptr %s, i64 %s)", p, l)
		n := g.newTmp()
		g.emit("%s = load i32, ptr %s", n, failures)
		n1 := g.newTmp()
		g.emit("%s = add i32 %s, 1", n1, n)
		g.emit("store i32 %s, ptr %s", n1, failures)
	}
	okMsg := g.stringConst("ok\n")
	for _, t := range g.prog.Tests {
		p, l := g.strPtrLen(g.stringConst("test " + t.Display + " ... "))
		g.emit("call void @veles_print(ptr %s, i64 %s)", p, l)
		var rs *types.Sealed
		if t.Sig.Effects.Throws {
			rs = g.prog.ResultType(t.Sig.Ret, t.Sig.Effects.Error).(*types.Sealed)
		}
		root := g.startRoot(t)
		pan := g.newTmp()
		g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, root)
		pb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", pb, pan)
		panL, resL, doneL := g.newLabel("test.panic"), g.newLabel("test.result"), g.newLabel("test.done")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, panL, resL)
		g.placeLabel(panL)
		pt := g.prog.PanicType.(*types.Struct)
		pmsg := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", pmsg, g.llType(pt), g.panicValue(root, pt))
		fail(g.concat(g.concat(g.stringConst("FAILED: panic: "), pmsg), g.stringConst("\n")))
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(resL)
		if rs == nil {
			p, l := g.strPtrLen(okMsg)
			g.emit("call void @veles_print(ptr %s, i64 %s)", p, l)
			g.emitTerm("br label %%%s", doneL)
			g.placeLabel(doneL)
			continue
		}
		rp := g.newTmp()
		g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, root)
		result := g.newTmp()
		g.emit("%s = load %s, ptr %s", result, g.llType(rs), rp)
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(rs), result)
		isErr := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
		errL, okL := g.newLabel("test.err"), g.newLabel("test.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
		g.placeLabel(errL)
		errVariant := rs.Variants[1]
		payload := g.extractTagged(g.llType(rs), result, g.llType(errVariant))
		errVal := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", errVal, g.llType(errVariant), payload)
		fail(g.concat(g.concat(g.stringConst("FAILED: "), g.show(errVariant.Fields[0].Type, errVal)), g.stringConst("\n")))
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(okL)
		p2, l2 := g.strPtrLen(okMsg)
		g.emit("call void @veles_print(ptr %s, i64 %s)", p2, l2)
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(doneL)
	}
	n := g.newTmp()
	g.emit("%s = load i32, ptr %s", n, failures)
	anyFail := g.newTmp()
	g.emit("%s = icmp ne i32 %s, 0", anyFail, n)
	code := g.newTmp()
	g.emit("%s = select i1 %s, i32 1, i32 0", code, anyFail)
	g.emitTerm("ret i32 %s", code)
}

// entryThunk returns a function `void (ptr task, ptr args)` that unpacks
// the arguments of fn from a heap block and runs its ramp, recording the
// handle. Every coroutine starts through veles_task_start this way.
func (g *gen) entryThunk(fn *sema.Func, paramTypes []types.Type, extraLead []string) string {
	name := "entry." + fn.Name
	if g.thunks[name] {
		return name
	}
	g.thunks[name] = true
	g.pending = append(g.pending, func() {
		g.defineHelper(name, "void", []string{"ptr %task", "ptr %args"}, func() {
			var lls []string
			for _, pt := range paramTypes {
				lls = append(lls, g.llType(pt))
			}
			blk := "{ " + strings.Join(lls, ", ") + " }"
			args := []string{"ptr %task"}
			args = append(args, extraLead...)
			for i, ll := range lls {
				p := g.newTmp()
				g.emit("%s = getelementptr inbounds %s, ptr %%args, i32 0, i32 %d", p, blk, i)
				v := g.newTmp()
				g.emit("%s = load %s, ptr %s", v, ll, p)
				args = append(args, ll+" "+v)
			}
			h := g.newTmp()
			g.emit("%s = call ptr @%s(%s)", h, fn.Name, joinArgs(args))
			g.emit("call void @veles_task_started(ptr %%task, ptr %s)", h)
			g.emitTerm("ret void")
		})
	})
	return name
}

// startTask packs args and runs fn as task through its entry thunk.
func (g *gen) startTask(task string, fn *sema.Func, argTypes []types.Type, argVals []string) {
	if !fn.Suspends {
		fn = g.rampFor(fn)
	}
	var lls []string
	for _, t := range argTypes {
		lls = append(lls, g.llType(t))
	}
	blk := "{ " + strings.Join(lls, ", ") + " }"
	args := "null"
	if len(lls) > 0 {
		size := 0
		for _, t := range argTypes {
			s, a := g.layout(t)
			size = (size + a - 1) / a * a
			size += s
		}
		args = g.newTmp()
		g.emit("%s = call ptr @veles_alloc_words(i64 %d)", args, (size+7)/8*8+8)
		for i, v := range argVals {
			p := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, blk, args, i)
			g.emit("store %s %s, ptr %s", lls[i], v, p)
		}
	}
	thunk := g.entryThunk(fn, argTypes, nil)
	g.emit("call void @veles_task_start(ptr %s, ptr @%s, ptr %s)", task, thunk, args)
}

// startIndirect starts a suspending function value as a task: the args
// block holds the code pointer, the environment and the arguments.
func (g *gen) startIndirect(task string, ft *types.Func, argTypes []types.Type, argVals []string) {
	var lls []string
	for _, t := range argTypes {
		lls = append(lls, g.llType(t))
	}
	blk := "{ " + strings.Join(lls, ", ") + " }"
	size := 0
	for _, t := range argTypes {
		s, a := g.layout(t)
		size = (size + a - 1) / a * a
		size += s
	}
	args := g.newTmp()
	g.emit("%s = call ptr @veles_alloc_words(i64 %d)", args, (size+7)/8*8+8)
	for i, v := range argVals {
		p := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, blk, args, i)
		g.emit("store %s %s, ptr %s", lls[i], v, p)
	}
	name := "entry.ft." + mangleType(ft)
	if !g.thunks[name] {
		g.thunks[name] = true
		g.pending = append(g.pending, func() {
			g.defineHelper(name, "void", []string{"ptr %task", "ptr %args"}, func() {
				var callArgs []string
				callArgs = append(callArgs, "ptr %task")
				var code string
				for i, ll := range lls {
					p := g.newTmp()
					g.emit("%s = getelementptr inbounds %s, ptr %%args, i32 0, i32 %d", p, blk, i)
					v := g.newTmp()
					g.emit("%s = load %s, ptr %s", v, ll, p)
					if i == 0 {
						code = v
						continue
					}
					callArgs = append(callArgs, ll+" "+v)
				}
				h := g.newTmp()
				g.emit("%s = call ptr %s(%s)", h, code, joinArgs(callArgs))
				g.emit("call void @veles_task_started(ptr %%task, ptr %s)", h)
				g.emitTerm("ret void")
			})
		})
	}
	g.emit("call void @veles_task_start(ptr %s, ptr @%s, ptr %s)", task, name, args)
}
