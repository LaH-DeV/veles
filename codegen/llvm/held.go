package llvm

import (
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// heldScope emits `with x = e` where e's value holds tasks (D111):
//
//  1. the scope begins, and is bound as the task's receiving scope while e
//     is computed, so the tasks e's constructors start join it (a failing
//     e cancels and joins them, and puts the receiving scope back);
//  2. x is bound; a task that already failed fails the block at once;
//  3. the body runs as a fail-fast scope body;
//  4. the tasks are cancelled and joined — on an early exit by the scope's
//     cleanup, which is inside x's close — and then x is closed.
//
// A failed task's panic continues here; its error is rethrown, found by
// comparing the failed task with each task field of x.
func (g *gen) heldScope(e *sema.ScopeBlock) string {
	h := e.Held
	sc := g.newTmp()
	g.emit("%s = call ptr @veles_scope_begin(ptr %s, i64 1)", sc, g.coro.task)
	slot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", sc, slot)
	g.scopeSlots[e] = slot
	wait, done, susp := g.newLabel("held.wait"), g.newLabel("held.done"), g.newLabel("held.susp")

	// 1. compute the value with the scope receiving its tasks
	computing := &sema.Builtin{Op: "scope.abandon"}
	g.abandonSlots[computing] = slot
	g.pushCleanup(computing, "@scope.cancel.thunk", slot)
	head := g.newTmp()
	g.emit("%s = call ptr @veles_receiving_bind(ptr %s)", head, sc)
	headSlot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", head, headSlot)
	restore := &sema.Builtin{Op: "receiving.restore"}
	g.receivingSlots[restore] = headSlot
	g.pushCleanup(restore, "@receiving.restore.thunk", headSlot)
	v := g.expr(h.Init)
	if g.term {
		// the value never arrives (it always throws or panics)
		g.cleanups = g.cleanups[:len(g.cleanups)-2]
		return "zeroinitializer"
	}
	st := g.declareVar(h.Var)
	g.emit("store %s %s, ptr %s", g.llType(h.Var.Type), v, st)
	g.cleanups = g.cleanups[:len(g.cleanups)-1]
	g.popCleanup(restore)
	g.expr(restore)
	g.cleanups = g.cleanups[:len(g.cleanups)-1]
	g.popCleanup(computing)

	// 2. x is bound: its close is outside the scope's own cleanup, so every
	// way out joins the tasks first
	if h.Close != nil {
		g.pushCleanup(h.Close, "@"+g.closeThunk(&sema.With{Var: h.Var, Close: h.Close}), st)
	}
	abandon := &sema.Builtin{Op: "scope.abandon"}
	g.abandonSlots[abandon] = slot
	g.pushCleanup(abandon, "@scope.cancel.thunk", slot)
	g.bodyScopes = append(g.bodyScopes, bodyScope{slot: slot, wait: wait, cleanups: len(g.cleanups)})
	g.failFastCheck()

	// 3. the body
	g.block(e.Body)
	g.bodyScopes = g.bodyScopes[:len(g.bodyScopes)-1]
	if !g.term {
		scv := g.newTmp()
		g.emit("%s = load ptr, ptr %s", scv, slot)
		g.emit("call void @veles_scope_cancel(ptr %s)", scv)
	}

	// 4. join (also the target of the fail-fast abort), then rethrow
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
	g.heldRethrow(h, slot, st)
	if h.Close != nil {
		g.cleanups = g.cleanups[:len(g.cleanups)-1]
		g.popCleanup(h.Close)
		g.expr(h.Close)
	}
	return "zeroinitializer"
}

// heldRethrow continues a held task's failure in the receiving block: a
// panic re-raises; an error is rethrown from the task field it came from.
func (g *gen) heldRethrow(h *sema.HeldValue, slot, value string) {
	scv := g.newTmp()
	g.emit("%s = load ptr, ptr %s", scv, slot)
	failed := g.newTmp()
	g.emit("%s = call ptr @veles_scope_failed(ptr %s)", failed, scv)
	has := g.newTmp()
	g.emit("%s = icmp ne ptr %s, null", has, failed)
	check, after := g.newLabel("held.check"), g.newLabel("held.after")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", has, check, after)
	g.placeLabel(check)
	pan := g.newTmp()
	g.emit("%s = call i64 @veles_task_panicked(ptr %s)", pan, failed)
	pb := g.newTmp()
	g.emit("%s = icmp ne i64 %s, 0", pb, pan)
	repanic, errs := g.newLabel("held.repanic"), g.newLabel("held.errors")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", pb, repanic, errs)
	g.placeLabel(repanic)
	g.emit("call void @veles_task_repanic(ptr %s)", failed)
	g.emitTerm("unreachable")
	g.placeLabel(errs)
	for _, ht := range h.Tasks {
		rs, ok := ht.Result.(*types.Sealed)
		if !ok || !isResultType(rs) || types.IsNever(rs.TypeArgs[1]) {
			continue
		}
		next := g.newLabel("held.next")
		task := g.heldTaskAt(h.Var.Type, value, ht.Path, next)
		same := g.newTmp()
		g.emit("%s = icmp eq ptr %s, %s", same, task, failed)
		throw := g.newLabel("held.throw")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", same, throw, next)
		g.placeLabel(throw)
		rp := g.newTmp()
		g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, failed)
		rv := g.newTmp()
		g.emit("%s = load %s, ptr %s", rv, g.llType(rs), rp)
		errVariant := rs.Variants[1]
		payload := g.extractTagged(g.llType(rs), rv, g.llType(errVariant))
		ev := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", ev, g.llType(errVariant), payload)
		g.throwValue(ev, errVariant.Fields[0].Type)
		g.placeLabel(next)
	}
	// every failing task is one of the value's fields; a task that ends
	// with an error but is not is a compiler bug, not a silent success
	msg := "internal error: a held task failed but is not one of its value's fields (D111)"
	mp, ml := g.strPtrLen(g.stringConst(msg))
	g.emit("call void @veles_panic(ptr %s, i64 %s)", mp, ml)
	g.emitTerm("unreachable")
	g.placeLabel(after)
}

// heldTaskAt loads the task at path inside the value at ptr p of type t,
// branching to miss when the path goes through another variant of a
// sealed value or a null `Task?`.
func (g *gen) heldTaskAt(t types.Type, p string, path []sema.HeldStep, miss string) string {
	for _, step := range path {
		switch {
		case step.Nullable:
			// a `Task<…>?` is the pointer itself, null when absent
			tv := g.newTmp()
			g.emit("%s = load ptr, ptr %s", tv, p)
			isNull := g.newTmp()
			g.emit("%s = icmp eq ptr %s, null", isNull, tv)
			present := g.newLabel("held.present")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", isNull, miss, present)
			g.placeLabel(present)
			return tv
		case step.Variant >= 0:
			st := t.(*types.Sealed)
			tagp := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 0", tagp, g.llType(st), p)
			tag := g.newTmp()
			g.emit("%s = load i32, ptr %s", tag, tagp)
			is := g.newTmp()
			g.emit("%s = icmp eq i32 %s, %d", is, tag, step.Variant)
			in := g.newLabel("held.variant")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", is, in, miss)
			g.placeLabel(in)
			payload := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", payload, g.llType(st), p)
			v := st.Variants[step.Variant]
			fp := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, g.llType(v), payload, step.Field)
			p, t = fp, v.Fields[step.Field].Type
		default:
			st := t.(*types.Struct)
			fp := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, g.llType(st), p, step.Field)
			p, t = fp, st.Fields[step.Field].Type
		}
	}
	tv := g.newTmp()
	g.emit("%s = load ptr, ptr %s", tv, p)
	return tv
}
