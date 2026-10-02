package sema

import (
	"github.com/LaH-DeV/veles/types"
)

// Suspension inference (D2, amended by D40 and D116): a function suspends
// when it contains a suspension point or calls something that suspends.
// The set is computed to a fixpoint over the call graph; dynamic dispatch
// (function values, trait objects) uses the effect declared on the type.
//
// D116: a parameter of a suspending function type (`f: fun(T): U
// suspends`) means *may* suspend. A function that suspends only by
// calling such parameters, or by passing them on to another function's, is
// conditional: a call of it suspends only when a function value it is
// given for one does — a lambda whose body suspends, or a value whose type
// says so. Such a function has two instances, a coroutine one and a plain
// one (codegen emits the plain one, `.plain`, when a call needs it), and a
// plain call is an ordinary call: allowed under a lock, in `init`, in a
// global initializer. A lambda passed to it follows the call: plain when
// the call is.
//
// Only first-order parameters count: one whose own parameters are not
// suspending functions, so that the values passed to it are known by the
// same rule. A trait method keeps the effects it declares (D40).

// susp walks a body for suspension. env holds the suspend-parameters known
// not to suspend in the instance being looked at — a function's own in its
// plain instance, and through a closure's captures the enclosing
// function's. The zero susp is the coroutine instance's view: every value
// suspends as its type says.
type susp struct{ env map[*Var]bool }

// plainVar reports whether v, or the variable it captures, is known not to
// suspend.
func (w susp) plainVar(v *Var) bool {
	for ; v != nil; v = v.Outer {
		if w.env[v] {
			return true
		}
	}
	return false
}

// call reports whether a call of a named function suspends here (its
// arguments' own evaluation aside).
func (w susp) call(e *Call) bool {
	fn := e.Fn
	if !fn.Suspends {
		return false
	}
	if !fn.Conditional {
		return true
	}
	off := 0
	if fn.Receiver != nil {
		off = 1 // the receiver's pointer comes first
	}
	for i, p := range fn.Params {
		if i+off < len(e.Args) && isSuspendParam(p) && w.argSuspends(e.Args[i+off]) {
			return true
		}
	}
	return false
}

// argSuspends reports whether a function value passed for a
// suspend-parameter suspends when it is called.
func (w susp) argSuspends(a Expr) bool {
	switch v := FuncValueOf(a).(type) {
	case *Closure:
		return v.Fn.suspends || w.block(v.Fn.Body)
	case *NullConst:
		return false
	case *VarRef:
		if w.plainVar(v.Var) {
			return false
		}
	}
	return funcTypeSuspends(a.Type())
}

// FuncValueOf is the function value an expression passes on unchanged:
// through a nullable's wrapping or unwrapping, and a conversion between
// function types that keeps the representation (dropping `sendable`).
func FuncValueOf(x Expr) Expr {
	for {
		switch y := x.(type) {
		case *SomeWrap:
			x = y.X
		case *Unwrap:
			x = y.X
		case *Cast:
			if funcTypeOf(y.X.Type()) == nil || funcTypeOf(y.Type()) == nil {
				return x
			}
			x = y.X
		default:
			return x
		}
	}
}

// indirect reports whether a call of a function value suspends here.
func (w susp) indirect(e *CallIndirect) bool {
	if ref, ok := FuncValueOf(e.Fn).(*VarRef); ok && w.plainVar(ref.Var) {
		return false
	}
	return funcTypeSuspends(e.Fn.Type())
}

// CallSuspendsIn and the functions below are what codegen asks while it
// emits an instance: plain lists the suspend-parameters that instance
// knows do not suspend (nil for a coroutine instance).
func CallSuspendsIn(e *Call, plain map[*Var]bool) bool { return susp{plain}.call(e) }

func IndirectSuspendsIn(e *CallIndirect, plain map[*Var]bool) bool {
	return susp{plain}.indirect(e)
}

// ClosureSuspendsIn reports whether a lambda passed for a suspend-parameter
// suspends, which decides the instance the call takes and the lambda's own.
func ClosureSuspendsIn(cl *Closure, plain map[*Var]bool) bool {
	return susp{plain}.argSuspends(cl)
}

// PlainEnv is a conditional function's plain instance's view: its own
// suspend-parameters do not suspend.
func PlainEnv(fn *Func) map[*Var]bool {
	env := map[*Var]bool{}
	for _, p := range fn.Params {
		if isSuspendParam(p) {
			env[p] = true
		}
	}
	return env
}

// IsSuspendParam reports whether a parameter is one a call can bind to a
// function value that does not suspend (D116): its type is a first-order
// suspending function type, or that made nullable.
func IsSuspendParam(p *Var) bool { return isSuspendParam(p) }

func isSuspendParam(p *Var) bool {
	ft := funcTypeOf(p.Type)
	if ft == nil || ft.C || !ft.Effects.Suspends {
		return false
	}
	for _, q := range ft.Params {
		if funcTypeSuspends(q.Type) {
			return false
		}
	}
	return true
}

func funcTypeOf(t types.Type) *types.Func {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	ft, _ := t.(*types.Func)
	return ft
}

func funcTypeSuspends(t types.Type) bool {
	ft := funcTypeOf(t)
	return ft != nil && ft.Effects.Suspends
}

// rootVar is the variable v stands for: itself, or through a closure's
// captures the enclosing function's.
func rootOf(v *Var) *Var {
	for v.Outer != nil {
		v = v.Outer
	}
	return v
}

// escapedSuspendParams lists the suspend-parameters whose value is used other
// than by calling it, passing it on to a conditional function's
// suspend-parameter, testing it for null, or capturing it. In a plain
// instance such a parameter holds a plain function, which a later call by
// its type would call as a coroutine: its function stays unconditional.
func escapedSuspendParams(prog *Program, candidates map[*Var]bool) map[*Var]bool {
	escaped := map[*Var]bool{}
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		allowed := map[*VarRef]bool{}
		allow := func(x Expr) {
			if ref, ok := FuncValueOf(x).(*VarRef); ok {
				allowed[ref] = true
			}
		}
		walkBlock(fn.Body, func(n any) {
			switch e := n.(type) {
			case *CallIndirect:
				allow(e.Fn)
			case *IsNull:
				allow(e.X)
			case *Call:
				if !e.Fn.Conditional {
					return
				}
				off := 0
				if e.Fn.Receiver != nil {
					off = 1
				}
				for i, p := range e.Fn.Params {
					if i+off < len(e.Args) && isSuspendParam(p) {
						allow(e.Args[i+off])
					}
				}
			}
		})
		walkBlock(fn.Body, func(n any) {
			if ref, ok := n.(*VarRef); ok && !allowed[ref] {
				if root := rootOf(ref.Var); candidates[root] {
					escaped[root] = true
				}
			}
		})
	}
	return escaped
}

func (c *Checker) inferSuspension(prog *Program) {
	// Each function's view in its coroutine instance (all) and, when it has
	// suspend-parameters, in its plain one (none). Both only grow as callees
	// are found to suspend, so the fixpoint is reached; a function whose
	// suspend-parameter escapes then loses its plain view, and the fixpoint
	// runs again.
	plain := map[*Func]map[*Var]bool{}
	candidates := map[*Var]bool{}
	for _, fn := range prog.Funcs {
		if !fn.IsClosure {
			if env := PlainEnv(fn); len(env) > 0 {
				plain[fn] = env
				for p := range env {
					candidates[p] = true
				}
			}
		}
	}
	for {
		c.suspensionFixpoint(prog, plain)
		escaped := escapedSuspendParams(prog, candidates)
		dropped := false
		for fn, env := range plain {
			for p := range env {
				if escaped[p] {
					delete(plain, fn)
					fn.Conditional = false
					dropped = true
					break
				}
			}
		}
		if !dropped {
			break
		}
	}
	c.checkDeclaredSuspension(prog)
}

// suspensionFixpoint computes Suspends and Conditional for every function.
func (c *Checker) suspensionFixpoint(prog *Program, plain map[*Func]map[*Var]bool) {
	changed := true
	for changed {
		changed = false
		for _, fn := range prog.Funcs {
			if fn.Body == nil {
				continue
			}
			all := fn.suspends || susp{}.block(fn.Body)
			cond := false
			if env := plain[fn]; all && env != nil {
				cond = !(fn.suspends || susp{env}.block(fn.Body))
			}
			if all != fn.Suspends || cond != fn.Conditional {
				fn.Suspends, fn.Conditional = all, cond
				changed = true
			}
		}
	}
}

// checkDeclaredSuspension holds declared effects on function types to what
// inference found, and refuses suspension where it cannot be.
func (c *Checker) checkDeclaredSuspension(prog *Program) {
	for _, fn := range prog.Funcs {
		if fn.IsClosure {
			if fn.Sig.Effects.Suspends {
				fn.Suspends = true
			} else if fn.Suspends {
				c.errorf(fn.Span, "this lambda suspends, so its type must be a suspending function type: 'fun(...): T suspends' (D40)")
			}
			fn.Conditional = false
			continue
		}
		if fn.tmpl == nil {
			continue
		}
		t := fn.tmpl
		if fn.Suspends && fn.CallerLoc {
			c.errorf(fn.Span, "a function marked '@caller_location' cannot suspend (D88)")
		}
		if fn.Suspends {
			if t.Impl != nil && t.Impl.Trait != nil {
				if sig := t.Impl.Trait.Methods[t.Name]; sig != nil && !sig.Effects.Suspends {
					c.errorf(fn.Span, "method '%s' suspends but trait '%s' declares it non-suspending; declare 'suspends' on the trait method (D40)", t.Name, t.Impl.Trait.Name)
				}
			}
			if t.Extern {
				c.errorf(fn.Span, "extern functions cannot suspend")
			}
			if fn.ExportC != "" {
				c.errorf(fn.Span, "an 'extern \"C\" fun' cannot suspend: C calls it and expects the answer before it returns (D69)")
			}
		}
		if t.Decl.Effects.Suspends {
			fn.Suspends, fn.Conditional = true, false // declared: always
		}
		if fn.Suspends && !fn.Conditional && t.Name == "$init" {
			c.errorf(fn.Span, "an 'init' block cannot suspend: construction is a plain expression (D28); do the waiting in a static function that builds the value")
		}
		fn.Sig.Effects.Suspends = fn.Suspends
	}
	for _, g := range prog.Globals {
		if g.Init != nil && exprSuspends(g.Init) {
			c.errorf(g.Span, "a global initializer cannot suspend; compute the value in 'main' and pass it down")
		}
	}
}

// exprSuspends is the coroutine instance's view.
func exprSuspends(e Expr) bool { return susp{}.expr(e) }

func (w susp) block(b *Block) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if w.stmt(s) {
			return true
		}
	}
	return b.Value != nil && w.expr(b.Value)
}

func (w susp) stmt(s Stmt) bool {
	switch s := s.(type) {
	case *Block:
		return w.block(s)
	case *VarDecl:
		return s.Init != nil && w.expr(s.Init)
	case *Assign:
		return w.expr(s.Target) || w.expr(s.Value)
	case *ExprStmt:
		return w.expr(s.X)
	case *Return:
		return s.Value != nil && w.expr(s.Value)
	case *Loop:
		if s.Cond != nil && w.expr(s.Cond) {
			return true
		}
		for _, p := range s.Post {
			if w.stmt(p) {
				return true
			}
		}
		return w.block(s.Body)
	case *With:
		return w.expr(s.Init) || w.expr(s.Close) || w.block(s.Body)
	case *ScopeBlock:
		return true
	}
	return false
}

func (w susp) expr(e Expr) bool {
	switch e := e.(type) {
	case nil:
		return false
	case *Call:
		return w.call(e) || w.any(e.Args)
	case *CallIndirect:
		return w.indirect(e) || w.expr(e.Fn) || w.any(e.Args)
	case *CallVirtual:
		if e.Sig.Effects.Suspends {
			return true
		}
		return w.expr(e.Obj) || w.any(e.Args)
	case *Builtin:
		switch e.Op {
		case "chan.send", "chan.recv", "task.sleep":
			return true
		}
		return w.any(e.Args)
	case *AwaitTask, *ScopeBlock, *Race, *Launch:
		return true
	case *Binary:
		return w.expr(e.L) || w.expr(e.R)
	case *Unary:
		return w.expr(e.X)
	case *Cast:
		return w.expr(e.X)
	case *ToString:
		return w.expr(e.X)
	case *StringConcat:
		return w.any(e.Parts)
	case *FieldGet:
		return w.expr(e.X)
	case *TupleGet:
		return w.expr(e.X)
	case *StructLit:
		return w.any(e.Fields)
	case *TupleLit:
		return w.any(e.Elems)
	case *AddrOf:
		return w.expr(e.X)
	case *Deref:
		return w.expr(e.X)
	case *SomeWrap:
		return w.expr(e.X)
	case *IsNull:
		return w.expr(e.X)
	case *Unwrap:
		return w.expr(e.X)
	case *MakeVariant:
		return w.expr(e.Value)
	case *VariantTest:
		return w.expr(e.X)
	case *VariantCast:
		return w.expr(e.X)
	case *TypeTest:
		return w.expr(e.X)
	case *Downcast:
		return w.expr(e.X)
	case *TraitTest:
		return w.expr(e.X)
	case *TraitCast:
		return w.expr(e.X)
	case *UnionTest:
		return w.expr(e.X)
	case *UnionCast:
		return w.expr(e.X)
	case *ErrorConvert:
		return w.expr(e.X)
	case *If:
		return w.expr(e.Cond) || w.block(e.Then) || w.block(e.Else)
	case *BlockExpr:
		return w.block(e.Block)
	case *Match:
		if e.Init != nil && w.expr(e.Init) {
			return true
		}
		for _, arm := range e.Arms {
			if w.expr(arm.Test) || w.expr(arm.Guard) || w.block(arm.Body) {
				return true
			}
			for _, b := range arm.Binds {
				if w.stmt(b) {
					return true
				}
			}
		}
	case *Try:
		return w.expr(e.X)
	case *Throw:
		return w.expr(e.Value)
	case *Elvis:
		return w.expr(e.L) || w.expr(e.R)
	case *Let:
		return w.expr(e.Init) || w.expr(e.Body)
	case *ListLit:
		return w.any(e.Elems)
	case *MapLit:
		for _, en := range e.Entries {
			if w.expr(en[0]) || w.expr(en[1]) {
				return true
			}
		}
	case *RangeLit:
		return w.expr(e.Lo) || w.expr(e.Hi)
	case *Box:
		return w.expr(e.X)
	}
	return false
}

func (w susp) any(es []Expr) bool {
	for _, e := range es {
		if w.expr(e) {
			return true
		}
	}
	return false
}
