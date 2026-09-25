package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// The receiver pass (D22, v0.30). Every method takes a pointer to the place
// it was called on, so that it may assign the receiver's `var` fields. Two
// facts about each method are needed only once every body is checked:
//
//   - WritesSelf: the method may change its receiver — it assigns a field
//     of `self`, takes a pointer into it, or calls a method that does. A
//     call on a temporary copy then loses the change, which is reported
//     (the rule that used to guard `mut fun` on a copy).
//   - SelfEscapes: the method may keep the receiver pointer beyond the
//     call — a closure captures `self`, `&self` (or a pointer into self)
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
			if fn.Body == nil || (fn.SelfEscapes && fn.WritesSelf) {
				continue
			}
			self := selfVarOf(fn)
			if self == nil {
				continue
			}
			esc, wr := selfUse(fn, self)
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
	c.fieldUsePass(prog)
	c.lintParamCopies(prog)
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

// selfUse scans fn's body for the ways self can escape or be written.
func selfUse(fn *Func, self *Var) (escapes, writes bool) {
	reads := map[*AddrOf]bool{}  // `*&place`: the pointer is consumed at once
	recvs := map[*AddrOf]*Func{} // a call's receiver pointer: the callee decides
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Assign:
			if rootVar(n.Target) == self {
				writes = true
			}
		case *Deref:
			if a, ok := n.X.(*AddrOf); ok {
				reads[a] = true
			}
		case *Call:
			if n.Fn.Receiver != nil && len(n.Args) > 0 {
				if a, ok := n.Args[0].(*AddrOf); ok {
					recvs[a] = n.Fn
				}
			}
		case *AddrOf:
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
// the receiver's, which is how `self` reads its place.
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
		default:
			return nil
		}
	}
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
			root.AddrTaken = true
		}
		if fn.WritesSelf && root != nil && root.IsGlobal && !root.Mutable {
			c.errorf(call.RecvSpan, "cannot call '%s' on '%s': the method changes its receiver and a global 'val' is a constant shared by every task; declare it 'var' (D11/D35)", fn.Display, root.Name)
		}
	case RecvTemp, RecvHandle:
		if fn.SelfEscapes {
			if v, ok := a.X.(*VarRef); ok {
				v.Var.AddrTaken = true
			}
		}
		if call.Recv == RecvTemp && fn.WritesSelf && fn.Sig != nil && types.IsUnit(fn.Sig.Ret) && !c.implementsPrelude(call.RecvType, "Iterator") {
			// a method that changes its receiver, returning nothing, called
			// on a copy (`xs.at(i)?.bump()`, `make().bump()`): the change is
			// lost. Iterators are consumed by their methods and stay exempt.
			c.lostCopy(call, fmt.Sprintf("'%s' changes a temporary copy of '%s' that is then discarded", fn.Display, call.RecvType))
		}
	}
}

// lostCopy reports a write to a copy, with the fix when the copy is an
// element read (`xs.atOrPanic(i)` → `*xs.refOrPanic(i)`, D25).
func (c *Checker) lostCopy(call *Call, what string) {
	if call.RecvExpr != nil {
		if repl, ok := elemReadCall(call.RecvExpr); ok {
			c.errorFix(call.RecvExpr.Span(), fixReplace("Replace with '"+repl+"'", call.RecvExpr.Span(), repl),
				"%s: '%s' is a copy of the element, so the change would be lost; reach the element itself with '%s' (D25)", what, srcText(call.RecvExpr), repl)
			return
		}
	}
	c.errorf(call.RecvSpan, "%s; bind it with 'var' to change a copy, or reach the element with 'ref' / 'refOrPanic' or 'loop (&x in xs)' (D25)", what)
}

// fieldUsePass computes, for every method, which of the receiver's fields
// it uses (Func.FieldsUsed / AllFields), to a fixpoint over the calls on
// self. `self.f` reads field f; any other use of `self` as a whole — a
// copy, `&self`, a capture, a task launch — counts as every field.
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
