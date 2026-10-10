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
declare i64 @veles_task_cancelled(ptr, i64)
declare void @veles_frame_park(ptr, ptr, i64)
declare void @veles_frame_return(ptr, ptr, ptr, i64, i64, ptr)
declare void @veles_frame_unwound(ptr, ptr)
declare void @veles_frame_back(ptr, ptr)
declare ptr @veles_frame_alloc(ptr, ptr, i64)
declare void @veles_frame_free(ptr, ptr)
; the link of a task's own frame: no caller (review F3)
@veles.root.link = internal constant { ptr, i64, i64 } zeroinitializer
declare void @veles_task_finish_cancelled(ptr)
declare i64 @veles_task_await(ptr, ptr)
declare ptr @veles_task_result(ptr)
declare ptr @veles_task_value(ptr)
declare void @veles_task_set_unwrap(ptr, i64)
declare i64 @veles_task_failed(ptr)
declare ptr @veles_scope_begin(ptr, i64, i64)
declare void @veles_scope_on(ptr, ptr)
declare ptr @veles_task_launch(ptr, i64)
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
declare i64 @veles_chan_try_send(ptr, ptr)
declare i64 @veles_chan_try_recv(ptr, ptr)
declare i64 @veles_task_sleep(ptr, i64)
declare i64 @veles_task_wait_io(ptr, i64, i64)
declare void @veles_task_cancel(ptr)
declare void @veles_scope_cancel(ptr)
declare void @veles_scope_abandon(ptr, ptr)
declare ptr @veles_receiving_bind(ptr)
declare void @veles_receiving_restore(ptr)
declare ptr @veles_receiving()

; a panic while a held value is computed (D111) puts the receiving scope back
define internal void @receiving.restore.thunk(ptr %env) {
entry:
  %head = load ptr, ptr %env
  call void @veles_receiving_restore(ptr %head)
  ret void
}

; a panic inside a scope body cancels the children (the join is left to
; them: their owner is gone); env is the slot holding the scope pointer
define internal void @scope.cancel.thunk(ptr %env) {
entry:
  %sc = load ptr, ptr %env
  call void @veles_scope_cancel(ptr %sc)
  ret void
}

; a cleanup's env copied to the heap, for a close() that suspends to run it
; later (D147): a slot of one word, or a heap cell already
define internal ptr @cleanup.move.word(ptr %env) {
entry:
  %v = load ptr, ptr %env
  %m = call ptr @veles_alloc_words(i64 16)
  store ptr %v, ptr %m
  ret ptr %m
}
define internal ptr @cleanup.move.same(ptr %env) {
entry:
  ret ptr %env
}
declare ptr @veles_race_new(ptr, i64)
declare void @veles_race_recv(ptr, ptr, ptr)
declare void @veles_race_send(ptr, ptr, ptr)
declare void @veles_race_sent(ptr)
declare void @veles_race_sleep(ptr, i64)
declare void @veles_race_await(ptr, ptr)
declare i64 @veles_race_wait(ptr, ptr)
declare i64 @veles_race_closed(ptr)
declare void @veles_run(ptr)
declare void @veles_runtime_threads(i64)
declare i64 @veles_task_panicked(ptr)
declare ptr @veles_task_panic_msg(ptr, ptr)
declare ptr @veles_task_panic_loc(ptr, ptr)
declare ptr @veles_task_panic_report(ptr, i64, ptr)
declare void @veles_call_push(ptr)
declare void @veles_call_pop()
declare void @veles_call_base(ptr, ptr)
declare void @veles_task_repanic(ptr)
declare void @veles_task_start(ptr, ptr, ptr)
declare void @veles_task_spawn(ptr, ptr, ptr)

; The common paths of veles_frame_alloc, veles_frame_free and
; veles_frame_back, taken inline (review F3). The offsets are the
; runtime's (veles_task: hdl 0, arena 24, depth 224, popped 246;
; frame_arena: top 8, cap 16, data 24), which veles_task.c asserts.
define internal ptr @veles.frame.alloc(ptr %t, ptr %l, i64 %size) alwaysinline {
entry:
  %par = load ptr, ptr %l
  %isroot = icmp eq ptr %par, null
  br i1 %isroot, label %slow, label %call
call:
  %ap = getelementptr inbounds i8, ptr %t, i64 24
  %a = load ptr, ptr %ap
  %noa = icmp eq ptr %a, null
  br i1 %noa, label %slow, label %have
have:
  %sz15 = add i64 %size, 15
  %sz = and i64 %sz15, -16
  %topp = getelementptr inbounds i8, ptr %a, i64 8
  %top = load i64, ptr %topp
  %capp = getelementptr inbounds i8, ptr %a, i64 16
  %cap = load i64, ptr %capp
  %end = add i64 %top, %sz
  %fits = icmp ule i64 %end, %cap
  br i1 %fits, label %fast, label %slow
fast:
  store i64 %end, ptr %topp
  %data = getelementptr inbounds i8, ptr %a, i64 24
  %p = getelementptr inbounds i8, ptr %data, i64 %top
  ret ptr %p
slow:
  %r = call ptr @veles_frame_alloc(ptr %t, ptr %l, i64 %size)
  ret ptr %r
}

define internal void @veles.frame.free(ptr %t, ptr %h) alwaysinline {
entry:
  %ap = getelementptr inbounds i8, ptr %t, i64 24
  %a = load ptr, ptr %ap
  %noa = icmp eq ptr %a, null
  br i1 %noa, label %done, label %have
have:
  %data = getelementptr inbounds i8, ptr %a, i64 24
  %capp = getelementptr inbounds i8, ptr %a, i64 16
  %cap = load i64, ptr %capp
  %hi = ptrtoint ptr %h to i64
  %di = ptrtoint ptr %data to i64
  %off = sub i64 %hi, %di
  %in = icmp ult i64 %off, %cap
  br i1 %in, label %fast, label %slow
fast:
  %topp = getelementptr inbounds i8, ptr %a, i64 8
  store i64 %off, ptr %topp
  ret void
slow:
  call void @veles_frame_free(ptr %t, ptr %h)
  ret void
done:
  ret void
}

define internal void @veles.frame.back(ptr %t, ptr %l) alwaysinline {
entry:
  %donep = getelementptr inbounds { ptr, i64, i64 }, ptr %l, i32 0, i32 2
  store i64 1, ptr %donep
  %par = load ptr, ptr %l
  store ptr %par, ptr %t
  %dp = getelementptr inbounds { ptr, i64, i64 }, ptr %l, i32 0, i32 1
  %d = load i64, ptr %dp
  %d1 = sub i64 %d, 1
  %d32 = trunc i64 %d1 to i32
  %tdp = getelementptr inbounds i8, ptr %t, i64 224
  store i32 %d32, ptr %tdp
  %pp = getelementptr inbounds i8, ptr %t, i64 246
  store i8 1, ptr %pp
  ret void
}
`

type coroState struct {
	id, hdl  string
	task     string
	link     string // the frame's link: where it returns to (review F3)
	depth    string // how deep in calls the frame is: 0 for a task's own
	finalL   string
	cleanupL string
	suspendL string
}

// coroPrologue emits the coroutine setup at the start of a ramp.
func (g *gen) coroPrologue() {
	c := &coroState{task: "%task", link: "%link", depth: "%coro.depth", finalL: "coro.final", cleanupL: "coro.cleanup", suspendL: "coro.suspend"}
	g.coro = c
	// the frame is a collector object, whose body is 8-aligned: said so,
	// LLVM realigns a field that needs more (an @align(n) local, D120)
	// rather than assume the 16 it takes by default
	g.emit("%%coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)")
	g.emit("%%coro.size = call i64 @llvm.coro.size.i64()")
	// a call's frame comes from the task's arena, a task's own from the
	// collector (veles_frame_alloc)
	g.emit("%%coro.mem = call ptr @veles.frame.alloc(ptr %%task, ptr %%link, i64 %%coro.size)")
	g.emit("%%coro.hdl = call ptr @llvm.coro.begin(token %%coro.id, ptr %%coro.mem)")
	g.emit("%%coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %%link, i32 0, i32 1")
	g.emit("%%coro.depth = load i64, ptr %%coro.depthp")
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
	g.suspendAt(true)
}

// suspendAt is a suspension point; park says the task resumes in this
// frame. A frame waiting for a call it made does not park: the call's
// frame, deeper, parked the task, and returns to this one (review F3).
func (g *gen) suspendAt(park bool) {
	c := g.coro
	if c == nil {
		panic("codegen: suspension point in a non-suspending function")
	}
	if park {
		g.emit("call void @veles_frame_park(ptr %s, ptr %s, i64 %s)", c.task, c.hdl, c.depth)
	}
	s := g.newTmp()
	g.emit("%s = call i8 @llvm.coro.suspend(token none, i1 false)", s)
	resume := g.newLabel("resume")
	g.emitTerm("switch i8 %s, label %%%s [ i8 0, label %%%s i8 1, label %%%s ]", s, c.suspendL, resume, c.cleanupL)
	g.placeLabel(resume)
	if g.inCleanup > 0 {
		// D47: cleanup is non-cancellable — the join of an abandoned scope
		// waits for its children whatever happens around it
		return
	}
	cc := g.newTmp()
	g.emit("%s = call i64 @veles_task_cancelled(ptr %s, i64 %s)", cc, c.task, c.depth)
	cb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", cb, cc)
	cancel, cont := g.newLabel("cancelled"), g.newLabel("cont")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", cb, cancel, cont)
	g.placeLabel(cancel)
	g.runCleanups(0)
	g.emit("call void @veles_frame_unwound(ptr %s, ptr %s)", c.task, c.link)
	g.emitTerm("br label %%%s", c.finalL)
	g.placeLabel(cont)
	g.failFastCheck()
}

// failFastCheck: a child of an enclosing scope has failed (D34), so the
// body stops here — whatever it was waiting for will not come — and the
// innermost scope joins its children and re-raises.
func (g *gen) failFastCheck() {
	c := g.coro
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
		// the body is abandoned: what the innermost scope still runs is
		// cancelled before its join (a `with` task, D100, would otherwise
		// be waited for; a scope whose own child failed has cancelled them),
		// and so is the suspending call the body is inside, which the join
		// then waits for (D3)
		isc := g.newTmp()
		g.emit("%s = load ptr, ptr %s", isc, inner.slot)
		g.emit("call void @veles_scope_abandon(ptr %s, ptr %s)", isc, c.task)
		g.runCleanups(inner.cleanups)
		g.emitTerm("br label %%%s", inner.wait)
		g.placeLabel(go_on)
	}
}

// coroReturn returns the (already Result-wrapped) value v of type t from
// the frame: it finishes the task when the frame is the task's own, and
// is written into the caller's link otherwise (review F3); nil t is no
// value.
func (g *gen) coroReturn(t types.Type, v string, isResult bool) {
	c := g.coro
	if t == nil {
		g.emit("call void @veles_frame_return(ptr %s, ptr %s, ptr null, i64 0, i64 0, ptr null)", c.task, c.link)
		g.emitTerm("br label %%%s", c.finalL)
		return
	}
	slot := v // a memory-class value is already in memory
	if !g.isMem(t) {
		slot = g.scratch(g.llType(t)) // veles_frame_return copies it before the frame ends
		g.emit("store %s %s, ptr %s", g.llType(t), v, slot)
	}
	failed := "0"
	if isResult {
		tag := g.tagOf(t, v)
		f := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", f, tag)
		failed = g.newTmp()
		g.emit("%s = zext i1 %s to i64", failed, f)
	}
	size := g.memSize(t)
	lt := g.linkType(t)
	if size == 0 || lt == "{ ptr, i64, i64 }" {
		g.emit("call void @veles_frame_return(ptr %s, ptr %s, ptr %s, i64 %d, i64 %s, ptr null)", c.task, c.link, slot, size, failed)
		g.emitTerm("br label %%%s", c.finalL)
		return
	}
	// a call's frame (its link has a parent) writes the result straight
	// into the link; a task's own hands it to the runtime to keep
	pp := g.newTmp()
	g.emit("%s = load ptr, ptr %s", pp, c.link)
	root := g.newTmp()
	g.emit("%s = icmp eq ptr %s, null", root, pp)
	rootL, callL := g.newLabel("ret.task"), g.newLabel("ret.call")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", root, rootL, callL)
	g.placeLabel(rootL)
	g.emit("call void @veles_frame_return(ptr %s, ptr %s, ptr %s, i64 %d, i64 %s, ptr null)", c.task, c.link, slot, size, failed)
	g.emitTerm("br label %%%s", c.finalL)
	g.placeLabel(callL)
	out := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 3", out, lt, c.link)
	g.storeVal(t, v, out)
	g.emit("call void @veles.frame.back(ptr %s, ptr %s)", c.task, c.link)
	g.emitTerm("br label %%%s", c.finalL)
}

// linkType is the LLVM type of the link a call of a suspending function
// returning rt passes (review F3): the caller's frame, the callee's
// depth, the done flag, then the result.
func (g *gen) linkType(rt types.Type) string {
	if rt == nil || types.IsUnit(rt) || types.IsNever(rt) {
		return "{ ptr, i64, i64 }"
	}
	return "{ ptr, i64, i64, " + g.llType(rt) + " }"
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
	// a panic in the awaited task continues in this one (D20: it unwinds
	// to the enclosing task scope, and `await handle` on a panicked child
	// rethrows)
	pan := g.newTmp()
	g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, task)
	pb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", pb, pan)
	repanic, fine := g.newLabel("await.repanic"), g.newLabel("await.fine")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, repanic, fine)
	g.placeLabel(repanic)
	g.emit("call void @veles_task_repanic(ptr %s)", task)
	g.emitTerm("unreachable")
	g.placeLabel(fine)
	if types.IsUnit(rt) || types.IsNever(rt) {
		return "zeroinitializer"
	}
	rp := g.newTmp()
	g.emit("%s = call ptr @veles_task_value(ptr %s)", rp, task)
	return g.loadVal(rt, rp)
}

// callSuspending calls a suspending function in this task (review F3).
func (g *gen) callSuspending(fn *sema.Func, argTypes []types.Type, argVals []string, rt types.Type) string {
	var args []string
	for i, t := range argTypes {
		args = append(args, g.vt(t)+" "+argVals[i])
	}
	return g.callFrame("@"+fn.Name, args, rt)
}

// callFrame calls the coroutine code (a function, a function value's code
// or a method table slot) with args after the task and a link, and gives
// its result of type rt (review F3). The callee's frame runs in this
// task: if it finishes without waiting, its result is in the link when the
// call returns, and nothing else happens — no task, no queue. If it waits,
// it has parked the task on its own frame, and this frame suspends without
// parking; the callee hands the task back when it finishes (or unwinds), so
// when this frame resumes the call is done, and the checks of any
// suspension point follow: a cancellation, a failed child of a scope.
func (g *gen) callFrame(code string, args []string, rt types.Type) string {
	c := g.coro
	lt := g.linkType(rt)
	lk := g.alloca(lt)
	p := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 0", p, lt, lk)
	g.emit("store ptr %s, ptr %s", c.hdl, p)
	d := g.newTmp()
	g.emit("%s = add i64 %s, 1", d, c.depth)
	p = g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", p, lt, lk)
	g.emit("store i64 %s, ptr %s", d, p)
	done := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 2", done, lt, lk)
	g.emit("store i64 0, ptr %s", done)
	h := g.newTmp()
	g.emit("%s = call ptr %s(%s)", h, code, joinArgs(append([]string{"ptr " + c.task, "ptr " + lk}, args...)))
	check, got, wait := g.newLabel("call.check"), g.newLabel("call.done"), g.newLabel("call.wait")
	g.emitTerm("br label %%%s", check)
	g.placeLabel(check)
	dv := g.newTmp()
	g.emit("%s = load i64, ptr %s", dv, done)
	db := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", db, dv)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", db, got, wait)
	g.placeLabel(wait)
	g.suspendAt(false)
	g.emitTerm("br label %%%s", check)
	g.placeLabel(got)
	// the callee's frame is over; what it gave back is in the link
	g.emit("call void @veles.frame.free(ptr %s, ptr %s)", c.task, h)
	if g.inCleanup == 0 {
		// it unwound instead (done is 2) — cancelled, or abandoned by a
		// failed child, at a loop's back edge with no suspension between:
		// the link holds no result, and this frame follows it as after a
		// suspension point that saw the same (D145)
		ub := g.newTmp()
		g.emit("%s = icmp eq i64 %s, 2", ub, dv)
		unwound, ok := g.newLabel("call.unwound"), g.newLabel("call.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s, !prof !{!\"branch_weights\", i32 1, i32 100000}", ub, unwound, ok)
		g.placeLabel(unwound)
		cc := g.newTmp()
		g.emit("%s = call i64 @veles_task_cancelled(ptr %s, i64 %s)", cc, c.task, c.depth)
		cb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", cb, cc)
		cancel, look := g.newLabel("call.cancelled"), g.newLabel("call.look")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", cb, cancel, look)
		g.placeLabel(look)
		g.failFastCheck()
		g.emitTerm("br label %%%s", cancel)
		g.placeLabel(cancel)
		g.runCleanups(0)
		g.emit("call void @veles_frame_unwound(ptr %s, ptr %s)", c.task, c.link)
		g.emitTerm("br label %%%s", c.finalL)
		g.placeLabel(ok)
	}
	if lt == "{ ptr, i64, i64 }" {
		return "zeroinitializer"
	}
	rp := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 3", rp, lt, lk)
	return g.loadVal(rt, rp)
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
	if e.Scope == nil {
		// held by a value (D111): the scope of the `with` receiving it
		g.emit("%s = call ptr @veles_receiving()", scope)
	} else {
		g.emit("%s = load ptr, ptr %s", scope, g.scopeSlots[e.Scope])
	}
	ct := g.newTmp()
	g.emit("%s = call ptr @veles_task_launch(ptr %s, i64 %d)", ct, scope, e.Index)
	if e.Unwrap {
		// D141: `await` gives the Ok payload, which follows the tag word
		g.emit("call void @veles_task_set_unwrap(ptr %s, i64 8)", ct)
	}
	var argTypes []types.Type
	var argVals []string
	for _, a := range e.Call.Args {
		argTypes = append(argTypes, a.Type())
		argVals = append(argVals, g.exprOwned(a))
	}
	g.startTaskAs(ct, e.Call.Fn, argTypes, argVals, true)
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
		params = append(params, "ptr %task", "ptr %link")
		i := 0
		if fn.Receiver != nil {
			llt := g.vt(fn.Receiver.Type)
			params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
			args = append(args, fmt.Sprintf("%s %%p%d", llt, i))
			i++
		}
		for _, p := range fn.Params {
			llt := g.vt(p.Type)
			params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
			args = append(args, fmt.Sprintf("%s %%p%d", llt, i))
			i++
		}
		saved := g.fnResult
		g.defineCoroHelper(ramp, params, func() {
			rt := g.fnRet(fn)
			v := g.callRet(rt, "@"+fn.Name, args)
			g.coroReturn(rt, v, fn.Sig.Effects.Throws)
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
		g.markLifetimes = true
		body()
		g.markLifetimes = false
		g.coroEpilogue()
		g.coro = nil
	})
}

func (g *gen) scopeBlock(e *sema.ScopeBlock) string {
	if e.Held != nil {
		return g.heldScope(e)
	}
	failFast := "1"
	if e.Gather {
		failFast = "0"
	}
	var on string
	if e.On != nil {
		on = g.expr(e.On) // before the scope opens (D143)
	}
	sc := g.newTmp()
	g.emit("%s = call ptr @veles_scope_begin(ptr %s, i64 %s, i64 %s)", sc, g.coro.task, failFast, g.coro.depth)
	if on != "" {
		g.emit("call void @veles_scope_on(ptr %s, ptr %s)", sc, on)
	}
	slot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", sc, slot)
	g.scopeSlots[e] = slot
	wait, done, susp := g.newLabel("scope.wait"), g.newLabel("scope.done"), g.newLabel("scope.susp")
	// leaving the body early cancels and joins the children (a cleanup,
	// like a `with` close); the fail-fast abort path below the entry runs
	// only the cleanups inside the body and joins through `wait` itself
	abandon := &sema.Builtin{Op: "scope.abandon"}
	g.abandonSlots[abandon] = slot
	g.pushCleanup(abandon, "@scope.cancel.thunk", slot, "@cleanup.move.word")
	if !e.Gather {
		g.bodyScopes = append(g.bodyScopes, bodyScope{slot: slot, wait: wait, cleanups: len(g.cleanups)})
	}
	g.block(e.Body)
	if !e.Gather {
		g.bodyScopes = g.bodyScopes[:len(g.bodyScopes)-1]
	}
	if e.Cancel && !g.term {
		// `with t = async f()` (D100): the body is over, so the task is
		// stopped and then joined, as when the body leaves early
		scv := g.newTmp()
		g.emit("%s = load ptr, ptr %s", scv, slot)
		g.emit("call void @veles_scope_cancel(ptr %s)", scv)
	}
	// wait for every child; the join is emitted even after a body that
	// always returns or throws, because the fail-fast abort branch of a
	// suspension point inside the body jumps here. The abandon cleanup
	// stays active through the join: an owner cancelled while it waits
	// here must cancel its children and wait for them to unwind before it
	// unwinds itself, or they would outlive the block (D34) with their
	// `with` blocks never closed (D43).
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
	g.cleanups = g.cleanups[:len(g.cleanups)-1]
	g.popCleanup(abandon)
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
		if l.Fails {
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
		rv := g.loadVal(rs, rp)
		errVariant := rs.Variants[1]
		payload := g.extractTaggedT(rs, rv, errVariant)
		ev := g.part(errVariant, payload, 0, errVariant.Fields[0].Type)
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
	// one launch: the gather's value is that Result itself (D103)
	tt, isTuple := e.Type().(*types.Tuple)
	if !isTuple {
		tt = &types.Tuple{Elems: []types.Type{e.Type()}}
	}
	llt := g.llType(tt)
	acc := "undef"
	var tupleMem string // the gathered tuple, when it is in the memory class
	if g.isMem(tt) {
		tupleMem = g.newMem(tt)
	}
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
		pe := g.makeTaggedT(want, 1, want.Variants[1], g.buildStruct(want.Variants[1], []string{pv}))
		g.storeVal(want, pe, res)
		g.emitTerm("br label %%%s", joinL)
		g.placeLabel(okL)
		var v string
		if rs, isR := childT.(*types.Sealed); isR && isResultType(rs) {
			cv := g.loadVal(rs, rp)
			v = g.convertResult(cv, rs, want)
		} else {
			val := "zeroinitializer"
			if !types.IsUnit(childT) {
				val = g.loadVal(childT, rp)
			}
			v = g.makeTaggedT(want, 0, want.Variants[0], g.buildStruct(want.Variants[0], []string{val}))
		}
		g.storeVal(want, v, res)
		g.emitTerm("br label %%%s", joinL)
		g.placeLabel(joinL)
		v = g.loadVal(want, res)
		if tupleMem != "" {
			g.storeVal(want, v, g.fieldPtr(llt, tupleMem, i))
			continue
		}
		n := g.newTmp()
		g.emit("%s = insertvalue %s %s, %s %s, %d", n, llt, acc, wantLL, v, i)
		acc = n
	}
	if tupleMem != "" {
		if !isTuple {
			return g.part(tt, tupleMem, 0, e.Type())
		}
		return tupleMem
	}
	if !isTuple {
		v := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", v, llt, acc)
		return v
	}
	return acc
}

func (g *gen) race(e *sema.Race) string {
	r := g.newTmp()
	g.emit("%s = call ptr @veles_race_new(ptr %s, i64 %d)", r, g.coro.task, len(e.Arms))
	slots := make([]string, len(e.Arms))
	for i, arm := range e.Arms {
		switch arm.Kind {
		case sema.RaceRecv:
			ch := g.expr(arm.Source)
			ct := arm.Source.Type().(*types.Channel)
			slots[i] = g.zeroSlot(ct.Elem)
			g.emit("call void @veles_race_recv(ptr %s, ptr %s, ptr %s)", r, ch, slots[i])
		case sema.RaceSend:
			// the channel and the value, once, as the race starts (D108)
			ch := g.expr(arm.Source)
			v := g.exprOwned(arm.Value)
			slots[i] = g.slotOf(arm.Value.Type(), v)
			g.emit("call void @veles_race_send(ptr %s, ptr %s, ptr %s)", r, ch, slots[i])
		case sema.RaceSleep:
			ns := g.expr(arm.Source)
			g.emit("call void @veles_race_sleep(ptr %s, i64 %s)", r, ns)
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
	if hasValue {
		res = g.resultSlot(e.Type())
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
		if arm.Kind == sema.RaceSend {
			g.emit("call void @veles_race_sent(ptr %s)", r)
		}
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
				val := g.loadVal(ct.Elem, slots[i])
				g.storeVal(nt, g.makeNullableT(nt, open, val), st)
			case sema.RaceTask:
				t := g.newTmp()
				g.emit("%s = load ptr, ptr %s", t, slots[i])
				rp := g.newTmp()
				g.emit("%s = call ptr @veles_task_value(ptr %s)", rp, t)
				if !types.IsUnit(arm.Var.Type) {
					g.storeVal(arm.Var.Type, g.loadVal(arm.Var.Type, rp), st)
				}
			}
		}
		v := g.block(arm.Body)
		if hasValue && !g.term && arm.Body.Value != nil {
			g.storeVal(e.Type(), v, res)
		}
		if !g.term {
			g.emitTerm("br label %%%s", end)
		}
	}
	g.placeLabel(end)
	if !hasValue {
		return "zeroinitializer"
	}
	return g.loadVal(e.Type(), res)
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
		x := g.exprOwned(e.Args[1])
		slot := g.slotOf(ct.Elem, x)
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
		slot := g.zeroSlot(ct.Elem)
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
		val := g.loadVal(ct.Elem, slot)
		return g.makeNullableT(nt, got, val), true
	case "chan.trySend":
		ct := e.Args[0].Type().(*types.Channel)
		ch := g.expr(e.Args[0])
		x := g.exprOwned(e.Args[1])
		slot := g.slotOf(ct.Elem, x)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_chan_try_send(ptr %s, ptr %s)", r, ch, slot)
		ok := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", ok, r)
		return ok, true
	case "chan.tryRecv":
		ct := e.Args[0].Type().(*types.Channel)
		nt := e.Type().(*types.Nullable)
		ch := g.expr(e.Args[0])
		slot := g.zeroSlot(ct.Elem)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_chan_try_recv(ptr %s, ptr %s)", r, ch, slot)
		got := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", got, r)
		val := g.loadVal(ct.Elem, slot)
		return g.makeNullableT(nt, got, val), true
	case "chan.close":
		ch := g.expr(e.Args[0])
		g.emit("call void @veles_chan_close(ptr %s)", ch)
		return "zeroinitializer", true
	case "task.cancel":
		t := g.expr(e.Args[0])
		g.emit("call void @veles_task_cancel(ptr %s)", t)
		return "zeroinitializer", true
	case "receiving.restore":
		head := g.newTmp()
		g.emit("%s = load ptr, ptr %s", head, g.receivingSlots[e])
		g.emit("call void @veles_receiving_restore(ptr %s)", head)
		return "zeroinitializer", true
	case "scope.abandon":
		// the body is being left early: cancel the children and join them
		slot := g.abandonSlots[e]
		sc := g.newTmp()
		g.emit("%s = load ptr, ptr %s", sc, slot)
		g.emit("call void @veles_scope_cancel(ptr %s)", sc)
		wait, done, susp := g.newLabel("abandon.wait"), g.newLabel("abandon.done"), g.newLabel("abandon.susp")
		g.emitTerm("br label %%%s", wait)
		g.placeLabel(wait)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_scope_wait(ptr %s, ptr %s)", r, g.coro.task, sc)
		rb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", rb, r)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
		g.placeLabel(susp)
		g.suspendPoint()
		g.emitTerm("br label %%%s", wait)
		g.placeLabel(done)
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
		ns := g.expr(e.Args[0]) // the deadline in nanoseconds (F9)
		loop, done, susp := g.newLabel("sleep"), g.newLabel("sleep.done"), g.newLabel("sleep.susp")
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(loop)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_task_sleep(ptr %s, i64 %s)", r, g.coro.task, ns)
		rb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", rb, r)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", rb, done, susp)
		g.placeLabel(susp)
		g.suspendPoint()
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(done)
		return "zeroinitializer", true
	case "task.ioWait":
		// std/net: park until the executor's poll reports the socket ready;
		// the same retry loop as sleep
		fd := g.expr(e.Args[0])
		write := g.expr(e.Args[1])
		wz := g.newTmp()
		g.emit("%s = zext i1 %s to i64", wz, write)
		loop, done, susp := g.newLabel("iowait"), g.newLabel("iowait.done"), g.newLabel("iowait.susp")
		g.emitTerm("br label %%%s", loop)
		g.placeLabel(loop)
		r := g.newTmp()
		g.emit("%s = call i64 @veles_task_wait_io(ptr %s, i64 %s, i64 %s)", r, g.coro.task, fd, wz)
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
	field := func(accessor string) string {
		lenSlot := g.alloca("i64")
		p := g.newTmp()
		g.emit("%s = call ptr @%s(ptr %s, ptr %s)", p, accessor, task, lenSlot)
		l := g.newTmp()
		g.emit("%s = load i64, ptr %s", l, lenSlot)
		a := g.newTmp()
		g.emit("%s = insertvalue %s undef, ptr %s, 0", a, strType, p)
		s := g.newTmp()
		g.emit("%s = insertvalue %s %s, i64 %s, 1", s, strType, a, l)
		return s
	}
	return g.buildStruct(pt, []string{field("veles_task_panic_msg"), field("veles_task_panic_loc")})
}

// convertResult widens Result<T, E1> to Result<T, E2> (E1 within E2).
func (g *gen) convertResult(v string, from, to *types.Sealed) string {
	if from == to {
		return v
	}
	toLL := g.llType(to)
	res := g.alloca(toLL)
	tag := g.tagOf(from, v)
	isErr := g.newTmp()
	g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
	errL, okL, endL := g.newLabel("conv.err"), g.newLabel("conv.ok"), g.newLabel("conv.end")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
	g.placeLabel(okL)
	okPayload := g.extractTaggedT(from, v, from.Variants[0])
	okVal := "zeroinitializer"
	if g.llType(from.Variants[0]) != "{}" {
		okVal = g.part(from.Variants[0], okPayload, 0, from.Variants[0].Fields[0].Type)
	}
	okNew := g.makeTaggedT(to, 0, to.Variants[0], g.buildStruct(to.Variants[0], []string{okVal}))
	g.storeVal(to, okNew, res)
	g.emitTerm("br label %%%s", endL)
	g.placeLabel(errL)
	errPayload := g.extractTaggedT(from, v, from.Variants[1])
	errVal := "zeroinitializer"
	if g.llType(from.Variants[1]) != "{}" {
		errVal = g.part(from.Variants[1], errPayload, 0, from.Variants[1].Fields[0].Type)
	}
	conv := g.convertError(errVal, from.Variants[1].Fields[0].Type, to.Variants[1].Fields[0].Type)
	errNew := g.makeTaggedT(to, 1, to.Variants[1], g.buildStruct(to.Variants[1], []string{conv}))
	g.storeVal(to, errNew, res)
	g.emitTerm("br label %%%s", endL)
	g.placeLabel(endL)
	return g.loadVal(to, res)
}

// testRunner emits the `veles test` entry point: every test runs
// as a task in declaration order; a returned error, a panic or a recorded
// failure (expect, require, fail — D78) fails the
// test, and the exit code reports the outcome (§4b: testing is a build
// mode).
func (g *gen) testRunner() {
	failures, passes := g.alloca("i64"), g.alloca("i64")
	g.emit("store i64 0, ptr %s", failures)
	g.emit("store i64 0, ptr %s", passes)
	// the failed tests' names, each after ", ", for the summary
	failedNames := g.alloca(strType)
	g.emit("store %s %s, ptr %s", strType, g.stringConst(""), failedNames)
	bump := func(counter string) {
		n, n1 := g.newTmp(), g.newTmp()
		g.emit("%s = load i64, ptr %s", n, counter)
		g.emit("%s = add i64 %s, 1", n1, n)
		g.emit("store i64 %s, ptr %s", n1, counter)
	}
	say := func(text string) {
		p, l := g.strPtrLen(text)
		g.emit("call void @veles_print(ptr %s, i64 %s)", p, l)
	}
	var fail func(text string)
	okMsg := g.stringConst("ok\n")
	// what the test's expect/require/fail calls recorded (D78): how many,
	// whether one ended the test, and the report lines
	depth := 0 // the running test's suite depth: its report lines are indented to it
	rec := ""  // the reported test's record (D80)
	take := func() (n, stopped, text string) {
		buf, st := g.alloca(strType), g.alloca("i64")
		n = g.newTmp()
		g.emit("%s = call i64 @veles_test_take(ptr %s, ptr %s, ptr %s, i64 %d)", n, rec, buf, st, 2*depth)
		text, s, stopped := g.newTmp(), g.newTmp(), g.newTmp()
		g.emit("%s = load %s, ptr %s", text, strType, buf)
		g.emit("%s = load i64, ptr %s", s, st)
		g.emit("%s = icmp ne i64 %s, 0", stopped, s)
		return n, stopped, text
	}
	// passOrRecorded ends a test that returned: ok, unless it recorded
	// failures on the way
	passOrRecorded := func(doneL string) {
		n, _, text := take()
		any := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", any, n)
		recL, passL := g.newLabel("test.recorded"), g.newLabel("test.pass")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", any, recL, passL)
		g.placeLabel(recL)
		fail(g.concat(g.stringConst("FAILED\n"), text))
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(passL)
		say(okMsg)
		bump(passes)
		g.emitTerm("br label %%%s", doneL)
	}
	// every test is queued first, each with its own record bound around its
	// task's creation so the task and all it starts record into it; the
	// runtime runs up to `jobs` at once (D80). The reports then follow in
	// declaration order, each printed when its test is done.
	g.emit("call void @veles_test_setup(i64 %d, i64 %d, i64 %d)", g.prog.TestJobs, g.prog.TestTimeoutMs, len(g.prog.Tests))
	recs, tasks := make([]string, len(g.prog.Tests)), make([]string, len(g.prog.Tests))
	for i, t := range g.prog.Tests {
		np, nl := g.strPtrLen(g.stringConst(t.Display))
		recs[i], tasks[i] = g.newTmp(), g.newTmp()
		g.emit("%s = call ptr @veles_test_new(ptr %s, i64 %s)", recs[i], np, nl)
		prev := g.newTmp()
		g.emit("%s = call ptr @veles_test_bind(ptr %s)", prev, recs[i])
		g.emit("%s = call ptr @veles_task_new()", tasks[i])
		g.emit("call void @veles_test_unbind(ptr %s)", prev)
		fn := t
		if !fn.Suspends {
			fn = g.rampFor(fn)
		}
		g.emit("call void @veles_test_queue(ptr %s, ptr @%s, ptr null)", tasks[i], g.entryThunk(fn, nil, nil))
	}
	var open []string // the suites whose heading is printed, outermost first
	for i, t := range g.prog.Tests {
		// suites are headings, their tests indented under them (D78):
		// print the ones this test enters
		same := 0
		for same < len(open) && same < len(t.Suite) && open[same] == t.Suite[same] {
			same++
		}
		for d := same; d < len(t.Suite); d++ {
			say(g.stringConst(strings.Repeat("  ", d) + t.Suite[d] + "\n"))
		}
		open = t.Suite
		depth = len(t.Suite)
		leaf := t.Display
		if depth > 0 {
			leaf = strings.TrimPrefix(leaf, strings.Join(t.Suite, " / ")+" / ")
		}
		// wait for this test — the ones after it run on meanwhile — then
		// report it; a test that outlives --timeout is the watchdog's to report
		root := tasks[i]
		rec = recs[i]
		g.emit("call void @veles_run(ptr %s)", root)
		say(g.stringConst(strings.Repeat("  ", depth) + "test " + leaf + " ... "))
		name := t.Display
		fail = func(text string) {
			say(text)
			bump(failures)
			names := g.newTmp()
			g.emit("%s = load %s, ptr %s", names, strType, failedNames)
			g.emit("store %s %s, ptr %s", strType, g.concat(names, g.stringConst(", "+name)), failedNames)
		}
		var rs *types.Sealed
		if t.Sig.Effects.Throws {
			rs = g.prog.ResultType(t.Sig.Ret, t.Sig.Effects.Error).(*types.Sealed)
		}
		pan := g.newTmp()
		g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, root)
		pb := g.newTmp()
		g.emit("%s = icmp ne i64 %s, 0", pb, pan)
		panL, resL, doneL := g.newLabel("test.panic"), g.newLabel("test.result"), g.newLabel("test.done")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, panL, resL)
		g.placeLabel(panL)
		pt := g.prog.PanicType.(*types.Struct)
		pv := g.panicValue(root, pt)
		pmsg := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", pmsg, g.llType(pt), pv)
		// under the message: where it panicked, the call chain of a debug
		// build (D81) and, inside a test helper, where the test called it (D78)
		tailLen := g.alloca("i64")
		tailPtr := g.newTmp()
		g.emit("%s = call ptr @veles_task_panic_report(ptr %s, i64 %d, ptr %s)", tailPtr, root, 2*depth+2, tailLen)
		tl, t0, tail := g.newTmp(), g.newTmp(), g.newTmp()
		g.emit("%s = load i64, ptr %s", tl, tailLen)
		g.emit("%s = insertvalue %s undef, ptr %s, 0", t0, strType, tailPtr)
		g.emit("%s = insertvalue %s %s, i64 %s, 1", tail, strType, t0, tl)
		report := g.concat(g.concat(g.stringConst("FAILED: panic: "), pmsg), tail)
		_, stopped, recorded := take()
		// a require or fail ended the test with a panic: its own line says why
		asPanic := g.concat(g.concat(report, g.stringConst("\n")), recorded)
		asStop := g.concat(g.stringConst("FAILED\n"), recorded)
		shown := g.newTmp()
		g.emit("%s = select i1 %s, %s %s, %s %s", shown, stopped, strType, asStop, strType, asPanic)
		fail(shown)
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(resL)
		if rs == nil {
			passOrRecorded(doneL)
			g.placeLabel(doneL)
			continue
		}
		rp := g.newTmp()
		g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, root)
		result := g.loadVal(rs, rp)
		tag := g.tagOf(rs, result)
		isErr := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
		errL, okL := g.newLabel("test.err"), g.newLabel("test.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
		g.placeLabel(errL)
		errVariant := rs.Variants[1]
		payload := g.extractTaggedT(rs, result, errVariant)
		errVal := g.part(errVariant, payload, 0, errVariant.Fields[0].Type)
		_, _, thrownRecorded := take()
		fail(g.concat(g.concat(g.concat(g.stringConst("FAILED: "), g.show(errVariant.Fields[0].Type, errVal)), g.stringConst("\n")), thrownRecorded))
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(okL)
		passOrRecorded(doneL)
		g.placeLabel(doneL)
	}
	// the summary: `3 passed, 1 failed: parsesDates` (and what a filter left out)
	np, nf := g.newTmp(), g.newTmp()
	g.emit("%s = load i64, ptr %s", np, passes)
	g.emit("%s = load i64, ptr %s", nf, failures)
	summary := g.concat(g.concat(g.stringConst("\n"), g.show(types.TI64, np)), g.stringConst(" passed, "))
	summary = g.concat(g.concat(summary, g.show(types.TI64, nf)), g.stringConst(" failed"))
	names := g.newTmp()
	g.emit("%s = load %s, ptr %s", names, strType, failedNames)
	// drop the first ", " — or, with nothing failed, nothing to drop
	ptr, n := g.strPtrLen(names)
	has := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", has, n)
	cut, rest := g.newTmp(), g.newTmp()
	g.emit("%s = select i1 %s, i64 2, i64 0", cut, has)
	g.emit("%s = getelementptr i8, ptr %s, i64 %s", rest, ptr, cut)
	restLen := g.newTmp()
	g.emit("%s = sub i64 %s, %s", restLen, n, cut)
	listed0, listed := g.newTmp(), g.newTmp()
	g.emit("%s = insertvalue %s undef, ptr %s, 0", listed0, strType, rest)
	g.emit("%s = insertvalue %s %s, i64 %s, 1", listed, strType, listed0, restLen)
	colon := g.newTmp()
	g.emit("%s = select i1 %s, %s %s, %s %s", colon, has, strType, g.stringConst(": "), strType, g.stringConst(""))
	summary = g.concat(g.concat(summary, colon), listed)
	if k := g.prog.TestsFiltered; k > 0 {
		summary = g.concat(summary, g.stringConst(fmt.Sprintf("; %d filtered out", k)))
	}
	say(g.concat(summary, g.stringConst("\n")))
	anyFail := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", anyFail, nf)
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
			args := []string{"ptr %task", "ptr @veles.root.link"}
			args = append(args, extraLead...)
			for i, ll := range lls {
				p := g.newTmp()
				g.emit("%s = getelementptr inbounds %s, ptr %%args, i32 0, i32 %d", p, blk, i)
				if g.isMem(paramTypes[i]) {
					args = append(args, "ptr "+p) // the argument block's own storage
					continue
				}
				v := g.newTmp()
				g.emit("%s = load %s, ptr %s", v, ll, p)
				args = append(args, ll+" "+v)
			}
			if !g.prog.Release {
				// the first frame of the task's call chain (D81)
				g.emit("call void @veles_call_base(ptr %%task, ptr %s)", g.chainRecord("", fn))
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
	g.startTaskAs(task, fn, argTypes, argVals, false)
}

// startTaskAs is startTask; spawn hands the task to the run queue instead
// of running its ramp here (an `async` launch, D66): any worker may start
// it, which is what lets the children of a scope run in parallel.
func (g *gen) startTaskAs(task string, fn *sema.Func, argTypes []types.Type, argVals []string, spawn bool) {
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
			s, _ := g.layout(t)
			a := g.llAlign(t)
			size = (size + a - 1) / a * a
			size += s
		}
		args = g.newTmp()
		g.emit("%s = call ptr @veles_alloc_words(i64 %d)", args, (size+7)/8*8+8)
		for i, v := range argVals {
			p := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", p, blk, args, i)
			g.storeVal(argTypes[i], v, p)
		}
	}
	thunk := g.entryThunk(fn, argTypes, nil)
	if spawn {
		g.emit("call void @veles_task_spawn(ptr %s, ptr @%s, ptr %s)", task, thunk, args)
		return
	}
	g.emit("call void @veles_task_start(ptr %s, ptr @%s, ptr %s)", task, thunk, args)
}

// runtimeConfig hands the program's `[runtime]` settings (D143) to the
// runtime before the first task runs; nothing when they are the defaults.
func (g *gen) runtimeConfig() {
	if g.prog.Threads > 0 {
		g.emit("call void @veles_runtime_threads(i64 %d)", g.prog.Threads)
	}
}
