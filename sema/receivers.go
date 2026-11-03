package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The receiver pass (D22, v0.30). Every method takes a pointer to the place
// it was called on, so that it may assign the receiver's `var` fields. Two
// facts about each method are needed only once every body is checked:
//
//   - WritesSelf: the method may change its receiver — it assigns a field
//     of `this`, takes a pointer into it, or calls a method that does. A
//     call on a temporary copy then loses the change, which is reported
//     (the rule that used to guard `mut fun` on a copy).
//   - SelfEscapes: the method may keep the receiver pointer beyond the
//     call — a closure captures `this`, `&this` (or a pointer into self)
//     is taken, or a callee does either. A receiver in such a call cannot
//     stay on the caller's stack: its variable is heap-allocated
//     (AddrTaken), the same promotion `&x` performs (D10).
//
// Both are computed to a fixpoint over the call graph, then applied to
// every call site. Trait objects are heap-boxed already (D9) and a task
// receives a copy of its receiver (check_task.go), so neither needs this.

func (c *Checker) receiverPass(prog *Program) {
	changed := true
	for changed {
		changed = false
		for _, fn := range prog.Funcs {
			if fn.Body == nil {
				continue
			}
			for _, p := range fn.Params {
				if !p.PtrKept && isManagedPointer(p.Type) && pointerKept(fn.Body, p) {
					p.PtrKept = true
					changed = true
				}
				if !p.FnKept && isFuncType(p.Type) && funcKept(fn.Body, p) {
					p.FnKept = true
					changed = true
				}
			}
			if fn.SelfEscapes && fn.WritesSelf {
				continue
			}
			self := selfVarOf(fn)
			if self == nil {
				continue
			}
			esc, wr := selfUse(fn.Body, self)
			if esc && !fn.SelfEscapes {
				fn.SelfEscapes = true
				changed = true
			}
			if wr && !fn.WritesSelf {
				fn.WritesSelf = true
				changed = true
			}
		}
	}
	c.settleOnCalls()
	c.checkLentParams(prog)
	c.checkBoxedPlaces()
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		for _, p := range fn.Params {
			esc, wr := selfUse(fn.Body, p)
			p.ReadOnly = !esc && !wr && !p.AddrTaken
		}
	}
	c.fieldUsePass(prog)
	c.lintParamCopies(prog)
	c.lintNeverClosed(prog)
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		walkBlock(fn.Body, func(n any) {
			if call, ok := n.(*Call); ok && call.Fn.Receiver != nil && len(call.Args) > 0 {
				c.applyReceiver(call)
				if len(call.InitMissing) > 0 {
					c.checkInitCall(call)
				}
			}
		})
	}
	for _, g := range prog.Globals {
		if g.Init != nil {
			walkExpr(g.Init, func(n any) {
				if call, ok := n.(*Call); ok && call.Fn.Receiver != nil && len(call.Args) > 0 {
					c.applyReceiver(call)
				}
			})
		}
	}
	for _, fn := range prog.Funcs {
		if fn.Body != nil {
			lentAddrPass(fn)
		}
	}
}

// lentAddrPass keeps in the frame a variable whose address is taken only by
// `&place` pointers that are read at once (`(*&x).f`, a bit cast's
// `*(&v).cast<*raw u64>()`), lent to a pointer
// parameter that is not kept, or a receiver the method keeps no pointer to
// (`fill(&xs)`): no pointer outlives the call, so no heap cell is needed.
// AddrTaken stays set, since the variable may still change through the
// pointer during the call; only its storage changes (Var.HeapCell).
func lentAddrPass(fn *Func) {
	fine := map[*AddrOf]bool{}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Deref:
			x := n.X
			if c, ok := x.(*Cast); ok {
				x = c.X // `*(&v).cast<*raw u64>()`, a bit cast read at once
			}
			if a, ok := x.(*AddrOf); ok {
				fine[a] = true
			}
		case *Call:
			if n.Fn.Body == nil {
				return // C, or a body the passes never saw: it may keep anything
			}
			args := n.Args
			if n.Fn.Receiver != nil && len(args) > 0 {
				if a, ok := args[0].(*AddrOf); ok && !n.Fn.SelfEscapes {
					fine[a] = true
				}
				args = args[1:]
			}
			for i, a := range args {
				if ad, ok := a.(*AddrOf); ok && i < len(n.Fn.Params) && isManagedPointer(n.Fn.Params[i].Type) && !n.Fn.Params[i].PtrKept {
					fine[ad] = true
				}
			}
		case *CallIndirect:
			if ft, ok := types.Underlying(n.Fn.Type()).(*types.Func); !ok || ft.C {
				return // a C function pointer may keep anything
			}
			for _, a := range n.Args {
				if ad, ok := a.(*AddrOf); ok && isManagedPointer(ad.Type()) {
					fine[ad] = true // lent to a function value (D233)
				}
			}
		}
	})
	heap := map[*Var]bool{}
	var cands []*Var
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *AddrOf:
			if v := rootVar(n.X); v != nil && !fine[n] {
				heap[v] = true
			}
		case *Closure:
			for _, v := range n.Captures {
				heap[v] = true
			}
		case *VarDecl:
			cands = append(cands, n.Var)
		}
	})
	cands = append(cands, fn.Params...)
	for _, v := range cands {
		if v.AddrTaken && !v.addrForced && !heap[v] && !v.IsGlobal && !v.Captured && !v.IsSelf {
			v.AddrLent = true
		}
	}
}

// funcValueUse is a named function used as a value, where it was (D233).
type funcValueUse struct {
	fn   *Func
	span source.Span
}

// checkLentParams holds D233: a pointer parameter of a function value — a
// lambda, or a named function used as one — is lent for the call, so the
// body may not keep it (Var.PtrKept, settled by the receiver pass).
func (c *Checker) checkLentParams(prog *Program) {
	reported := map[source.Span]bool{} // a lambda in a generic body is checked once per instance
	for _, fn := range prog.Funcs {
		if !fn.IsClosure || fn.Body == nil || !fn.Span.IsValid() {
			continue
		}
		for _, p := range fn.Params {
			if !p.PtrKept || !isManagedPointer(p.Type) || !p.Span.IsValid() || reported[p.Span] {
				continue
			}
			reported[p.Span] = true
			c.errorf(p.Span, "'%s' is lent to this function only for the call (D233): the body %s, which would keep it beyond the call; read the value with '*%s'", p.Name, keptHow(fn.Body, p), p.Name)
		}
	}
	for _, use := range c.funcValues {
		for _, p := range use.fn.Params {
			if p.PtrKept && isManagedPointer(p.Type) && !reported[use.span] {
				reported[use.span] = true
				c.errorf(use.span, "'%s' keeps its pointer parameter '%s', so it cannot be used as a function value, whose pointer parameters are lent only for the call (D233); wrap a call in a lambda that passes what it needs", use.fn.Display, p.Name)
			}
		}
	}
	c.funcValues = nil
}

// boxedPlace is a copy of a named place boxed as a trait object (D232).
type boxedPlace struct {
	box  *Box
	span source.Span
}

// checkBoxedPlaces holds D232: a trait object made from a variable, a
// parameter or a field holds a copy, so a method that changes `this`
// changes the copy and the place never sees it. That is refused, with the
// fix `&x` (the object then holds the pointer) or `x.copy()` for a copy
// on purpose. A temporary is fine: nothing else could see the change.
func (c *Checker) checkBoxedPlaces() {
	reported := map[source.Span]bool{}
	for _, bp := range c.boxedPlaces {
		if reported[bp.span] {
			continue
		}
		var changer *Func
		for _, m := range bp.box.Methods {
			if m != nil && m.WritesSelf {
				changer = m
				break
			}
		}
		if changer == nil {
			continue
		}
		reported[bp.span] = true
		text := srcTextSpan(bp.span, "the value")
		ptr := "&" + text
		if strings.HasPrefix(text, "*") {
			ptr = text[1:]
		}
		c.errorFix(bp.span, fixReplace("Replace with '"+ptr+"'", bp.span, ptr),
			"the '%s' object gets a copy of '%s', and '%s' changes it, so '%s' would never see the change; pass '%s' to box the place itself, or '%s.copy()' for a copy on purpose (D232)",
			bp.box.Trait.Name, text, changer.Display, text, ptr, text)
	}
	c.boxedPlaces = nil
}

// keptHow says how a body keeps the pointer parameter p, for the message.
func keptHow(body *Block, p *Var) string {
	is := func(e Expr) bool {
		r, ok := e.(*VarRef)
		return ok && r.Var == p
	}
	how := ""
	note := func(s string) {
		if how == "" {
			how = s
		}
	}
	if is(body.Value) {
		note("returns it")
	}
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *Return:
			if is(n.Value) {
				note("returns it")
			}
		case *Assign:
			if is(n.Value) {
				note("stores it")
			}
		case *StructLit:
			for _, a := range n.Fields {
				if is(a) {
					note("stores it")
				}
			}
		case *Closure:
			for _, v := range n.Captures {
				if v == p {
					note("captures it")
				}
			}
		}
	})
	if how == "" {
		how = "passes it where it is kept"
	}
	return how
}

// selfVarOf is the variable standing for the receiver in fn: the receiver
// itself, or its captured stand-in inside a lambda.
func selfVarOf(fn *Func) *Var {
	if fn.Receiver != nil {
		return fn.Receiver
	}
	for _, v := range fn.CapVars {
		if v.IsSelf {
			return v
		}
	}
	return nil
}

// isManagedPointer: t is a `*T` the collector manages (not `*raw T`).
func isManagedPointer(t types.Type) bool {
	p, ok := types.Underlying(t).(*types.Pointer)
	return ok && !p.Raw
}

// pointerKept reports whether the pointer parameter p may outlive the call:
// any use of it other than reading or writing through it, calling a method
// that keeps no pointer to its receiver, or passing it to a parameter that
// is not kept either. Computed to a fixpoint with the callees' (receiverPass).
func pointerKept(body *Block, p *Var) bool {
	// the plain locals the pointer is copied into (`var cur = p`, swapped
	// with another later) are the same pointer: their uses count too; so
	// is a field of a local `val` struct built around it (`val s =
	// Saved(items: p)`), read as `s.items`
	aliases := map[*Var]bool{p: true}
	holders := map[*Var]map[int]bool{}
	fine := map[Expr]bool{}
	local := func(v *Var) bool { return !v.IsGlobal && !v.Captured && !v.AddrTaken && !v.IsParam }
	// use: e is the pointer — an alias, or a holder's field that has it
	use := func(e Expr) bool {
		switch e := e.(type) {
		case *VarRef:
			return aliases[e.Var]
		case *FieldGet:
			if r, ok := e.X.(*VarRef); ok {
				return holders[r.Var][e.Index]
			}
		}
		return false
	}
	for grew := true; grew; {
		grew = false
		walkBlock(body, func(n any) {
			var to *Var
			var from Expr
			switch n := n.(type) {
			case *VarDecl:
				to, from = n.Var, n.Init
				if lit, ok := from.(*StructLit); ok && local(to) && !to.Mutable {
					for i, f := range lit.Fields {
						if use(f) && !holders[to][i] {
							if holders[to] == nil {
								holders[to] = map[int]bool{}
							}
							holders[to][i] = true
							fine[f] = true
							grew = true
						}
					}
					return
				}
			case *Assign:
				if r, ok := n.Target.(*VarRef); ok {
					to, from = r.Var, n.Value
					if aliases[to] {
						fine[r] = true // assigned, not read
					}
				}
			}
			if to == nil || !use(from) || !local(to) {
				return
			}
			fine[from] = true
			if !aliases[to] {
				aliases[to] = true
				grew = true
			}
		})
	}
	kept := false
	lentFns := lentClosures(body)
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *Deref:
			if use(n.X) {
				fine[n.X] = true
			}
		case *Call:
			params, args := n.Fn.Params, n.Args
			if n.Fn.Receiver != nil && len(args) > 0 {
				if use(args[0]) && !n.Fn.SelfEscapes {
					fine[args[0]] = true // a method through the pointer, which keeps no pointer to its receiver
				}
				args = args[1:]
			}
			for i, a := range args {
				if use(a) && i < len(params) && isManagedPointer(params[i].Type) && !params[i].PtrKept {
					fine[a] = true
				}
			}
		case *CallIndirect:
			// a function value's pointer parameters are lent (D233)
			if ft, ok := types.Underlying(n.Fn.Type()).(*types.Func); !ok || ft.C {
				return
			}
			for _, a := range n.Args {
				if use(a) {
					fine[a] = true
				}
			}
		case *Closure:
			for _, v := range n.Captures {
				if aliases[v] || holders[v] != nil {
					// a lambda that lives only for a call keeps the pointer only
					// if its own body does (D233)
					kept = kept || holders[v] != nil || !lentFns[n] || pointerKept(n.Fn.Body, capVarOf(n.Fn, v))
				}
			}
		}
	})
	// a holder read any way but through one of its fields hands the pointer on
	inField := map[*VarRef]bool{}
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *FieldGet:
			if r, ok := n.X.(*VarRef); ok && holders[r.Var] != nil {
				inField[r] = true
				kept = kept || (use(n) && !fine[n])
			}
		case *VarRef:
			if aliases[n.Var] && !fine[n] || holders[n.Var] != nil && !inField[n] {
				kept = true
			}
		}
	})
	return kept
}

// lentClosures are the lambdas in body written directly as an argument for
// a function parameter the callee keeps no hold of (Var.FnKept): they live
// only for that call.
func lentClosures(body *Block) map[*Closure]bool {
	lent := map[*Closure]bool{}
	walkBlock(body, func(n any) {
		call, ok := n.(*Call)
		if !ok || call.Fn.Body == nil {
			return
		}
		params, args := call.Fn.Params, call.Args
		if call.Fn.Receiver != nil && len(args) > 0 {
			args = args[1:]
		}
		for i, a := range args {
			if c, ok := a.(*Cast); ok {
				a = c.X // a plain lambda where the parameter may suspend (D217)
			}
			if cl, ok := a.(*Closure); ok && i < len(params) && isFuncType(params[i].Type) && !params[i].FnKept && cl.Fn.Body != nil {
				lent[cl] = true
			}
		}
	})
	return lent
}

// capVarOf is the stand-in for the enclosing function's v inside a lambda.
func capVarOf(fn *Func, v *Var) *Var {
	for _, cv := range fn.CapVars {
		if cv.Outer == v {
			return cv
		}
	}
	return v
}

// isFuncType: t is a function type (a closure value).
func isFuncType(t types.Type) bool {
	_, ok := types.Underlying(t).(*types.Func)
	return ok
}

// funcKept reports whether the function-typed parameter p may outlive the
// call: any use but calling it or passing it on to a parameter that is not
// kept either. Computed to a fixpoint with the callees' (receiverPass).
func funcKept(body *Block, p *Var) bool {
	fine := map[*VarRef]bool{}
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *CallIndirect:
			if r, ok := n.Fn.(*VarRef); ok && r.Var == p {
				fine[r] = true
			}
		case *Call:
			params, args := n.Fn.Params, n.Args
			if n.Fn.Body == nil {
				return
			}
			if n.Fn.Receiver != nil && len(args) > 0 {
				args = args[1:]
			}
			for i, a := range args {
				if c, ok := a.(*Cast); ok {
					a = c.X
				}
				if r, ok := a.(*VarRef); ok && r.Var == p && i < len(params) && isFuncType(params[i].Type) && !params[i].FnKept {
					fine[r] = true
				}
			}
		}
	})
	kept := false
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *VarRef:
			kept = kept || (n.Var == p && !fine[n])
		case *Closure:
			for _, v := range n.Captures {
				kept = kept || v == p
			}
		}
	})
	return kept
}

// selfUse scans a body for the ways self can escape or be written.
func selfUse(body *Block, self *Var) (escapes, writes bool) {
	reads := map[*AddrOf]bool{}  // `*&place`: the pointer is consumed at once
	recvs := map[*AddrOf]*Func{} // a call's receiver pointer: the callee decides
	lent := map[*AddrOf]bool{}   // `&place` passed to a pointer parameter the callee does not keep
	walkBlock(body, func(n any) {
		switch n := n.(type) {
		case *Assign:
			if collRoot(n.Target) == self {
				writes = true // into its own storage, or an element of a list in it (D231)
			}
		case *Builtin:
			if n.Op == "array.rawData" && rootVar(n.Args[0]) == self {
				writes = true // the storage is lent to C, which may write it (withRaw)
			}
			if collectionWrites[n.Op] && collRoot(n.Args[0]) == self {
				writes = true // a change to a collection it holds (D231)
			}
		case *Deref:
			if a, ok := n.X.(*AddrOf); ok {
				reads[a] = true
			}
		case *Call:
			args := n.Args
			if n.Fn.Receiver != nil && len(args) > 0 {
				if a, ok := args[0].(*AddrOf); ok {
					recvs[a] = n.Fn
				}
				args = args[1:]
			}
			for i, a := range args {
				if ad, ok := a.(*AddrOf); ok && i < len(n.Fn.Params) && isManagedPointer(n.Fn.Params[i].Type) && !n.Fn.Params[i].PtrKept {
					lent[ad] = true
				}
			}
		case *AddrOf:
			if lent[n] && collRoot(n.X) == self {
				writes = true // lent for the call: it may be written through, not kept
				return
			}
			if rootVar(n.X) != self && !reads[n] && collRoot(n.X) == self {
				writes = true // a pointer to an element of a list it holds (D231)
				return
			}
			if rootVar(n.X) != self || reads[n] {
				return
			}
			if callee, isRecv := recvs[n]; isRecv {
				if callee.SelfEscapes {
					escapes = true
				}
				if callee.WritesSelf {
					writes = true
				}
				return
			}
			// a pointer into the receiver handed out: it may be stored and
			// written through
			escapes, writes = true, true
		case *Closure:
			for _, v := range n.Captures {
				if v == self {
					escapes = true
					if n.Fn.WritesSelf {
						writes = true
					}
				}
			}
		}
	})
	return
}

// rootVar is the variable whose own storage the place expression e
// denotes: through fields and payloads, never through a pointer — except
// the receiver's, which is how `this` reads its place.
func rootVar(e Expr) *Var {
	for {
		switch x := e.(type) {
		case *VarRef:
			return x.Var
		case *Deref:
			if v, ok := x.X.(*VarRef); ok && v.Var.IsSelf {
				return v.Var
			}
			return nil
		case *FieldGet:
			e = x.X
		case *TupleGet:
			e = x.X
		case *Unwrap:
			e = x.X
		case *VariantCast:
			e = x.X
		case *Builtin:
			// an element of an Array lies inside the value holding it (D121)
			if x.Op == "list.ref" || x.Op == "list.refUnchecked" {
				if _, isArray := x.Args[0].Type().(*types.Array); isArray {
					e = x.Args[0]
					continue
				}
			}
			return nil
		default:
			return nil
		}
	}
}

// RootVar is rootVar for the code generator: the variable whose own
// storage a place expression lies in, or nil.
func RootVar(e Expr) *Var { return rootVar(e) }

// MayWrite reports whether evaluating e may change v's own storage: an
// assignment into it, or a pointer into it handed out (a method's
// receiver included). A capture makes v a heap cell, which is not asked.
func MayWrite(e Expr, v *Var) bool {
	found := false
	walkExpr(e, func(n any) {
		switch n := n.(type) {
		case *Assign:
			found = found || collRoot(n.Target) == v
		case *AddrOf:
			found = found || collRoot(n.X) == v
		case *Builtin:
			found = found || (collectionWrites[n.Op] && collRoot(n.Args[0]) == v) // D231
		}
	})
	return found
}

// applyReceiver settles one method call once the callee's facts are known:
// heap-allocates a receiver the callee may keep a pointer to, and reports
// a change the callee makes to a copy that is then thrown away.
func (c *Checker) applyReceiver(call *Call) {
	fn := call.Fn
	a, ok := call.Args[0].(*AddrOf)
	if !ok {
		return
	}
	switch call.Recv {
	case RecvPlace:
		root := call.RecvRoot
		if fn.SelfEscapes && root != nil && !root.IsGlobal && !root.IsSelf {
			root.heapPromote()
		}
		if fn.WritesSelf && root != nil && root.IsGlobal && !root.Mutable {
			c.errorf(call.RecvSpan, "cannot call '%s' on '%s': the method changes its receiver and a global 'val' is a constant shared by every task%s", fn.Display, root.Name, c.globalWriteHint(root.Global))
		}
	case RecvTemp, RecvHandle:
		if fn.SelfEscapes {
			if v, ok := a.X.(*VarRef); ok {
				v.Var.heapPromote()
			}
		}
		if call.Recv == RecvTemp && fn.WritesSelf && fn.Sig != nil && types.IsUnit(fn.Sig.Ret) && !c.implementsPrelude(call.RecvType, "Iterator") {
			// a method that changes its receiver, returning nothing, called
			// on a copy (`xs.get(i)?.bump()`, `make().bump()`): the change is
			// lost. Iterators are consumed by their methods and stay exempt.
			c.lostCopy(call, fmt.Sprintf("'%s' changes a temporary copy of '%s' that is then discarded", fn.Display, call.RecvType))
		}
	}
}

// lostCopy reports a write to a copy, with the fix when the copy is an
// element read (`xs.get(i)` → `xs.at(i)`, D25).
func (c *Checker) lostCopy(call *Call, what string) {
	if call.RecvExpr != nil {
		if repl, ok := elemReadCall(call.RecvExpr); ok {
			c.errorFix(call.RecvExpr.Span(), fixReplace("Replace with '"+repl+"'", call.RecvExpr.Span(), repl),
				"%s: '%s' is a copy of the element, so the change would be lost; reach the element itself with '%s' (D25)", what, srcText(call.RecvExpr), repl)
			return
		}
	}
	c.errorf(call.RecvSpan, "%s; bind it with 'var' to change a copy, or reach the element with 'ref' or 'loop (&x in xs)' (D25)", what)
}

// fieldUsePass computes, for every method, which of the receiver's fields
// it uses (Func.FieldsUsed / AllFields), to a fixpoint over the calls on
// self. `this.f` reads field f; any other use of `this` as a whole — a
// copy, `&this`, a capture, a task launch — counts as every field.
func (c *Checker) fieldUsePass(prog *Program) {
	changed := true
	for changed {
		changed = false
		for _, fn := range prog.Funcs {
			if fn.Body == nil || fn.AllFields {
				continue
			}
			self := selfVarOf(fn)
			if self == nil {
				continue
			}
			used, all := fieldUse(fn, self)
			if all && !fn.AllFields {
				fn.AllFields = true
				changed = true
				continue
			}
			for i := range used {
				if !fn.FieldsUsed[i] {
					if fn.FieldsUsed == nil {
						fn.FieldsUsed = map[int]bool{}
					}
					fn.FieldsUsed[i] = true
					changed = true
				}
			}
		}
	}
}

func fieldUse(fn *Func, self *Var) (used map[int]bool, all bool) {
	used = map[int]bool{}
	consumed := map[Expr]bool{} // a Deref of self already accounted for
	isSelf := func(e Expr) bool {
		d, ok := e.(*Deref)
		if !ok {
			return false
		}
		v, ok := d.X.(*VarRef)
		return ok && v.Var == self
	}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *FieldGet:
			if isSelf(n.X) {
				used[n.Index] = true
				consumed[n.X] = true
			}
		case *Call:
			if n.Fn.Receiver != nil && len(n.Args) > 0 {
				if a, ok := n.Args[0].(*AddrOf); ok && isSelf(a.X) {
					// a method on self: what it uses, we use
					consumed[a.X] = true
					consumed[a] = true
					if n.Fn.AllFields {
						all = true
					}
					for i := range n.Fn.FieldsUsed {
						used[i] = true
					}
				}
			}
		case *AddrOf:
			if isSelf(n.X) && !consumed[n] {
				all = true // a pointer to the whole receiver handed out
			}
		case *Deref:
			if isSelf(n) && !consumed[n] {
				all = true // the receiver used as a value
			}
		case *Closure:
			for _, v := range n.Captures {
				if v == self {
					if n.Fn.AllFields {
						all = true
					}
					for i := range n.Fn.FieldsUsed {
						used[i] = true
					}
				}
			}
		}
	})
	return used, all
}

// checkInitCall reports a method called from an `init` block that reaches
// a field the block has not assigned yet.
func (c *Checker) checkInitCall(call *Call) {
	fn := call.Fn
	st, ok := call.RecvType.(*types.Struct)
	if !ok {
		return
	}
	names := func(idx []int) string {
		var out []string
		for _, i := range idx {
			out = append(out, st.Fields[i].Name)
		}
		return strings.Join(out, "', '")
	}
	if fn.AllFields {
		c.errorf(call.RecvSpan, "'init' calls '%s' before assigning '%s', and the method uses the whole value; assign every field without a default first (D28)", fn.Display, names(call.InitMissing))
		return
	}
	for _, i := range call.InitMissing {
		if fn.FieldsUsed[i] {
			c.errorf(call.RecvSpan, "'init' calls '%s' before assigning '%s', which the method reads (D28)", fn.Display, st.Fields[i].Name)
			return
		}
	}
}
