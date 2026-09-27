package sema

import (
	"github.com/LaH-DeV/veles/types"
)

// Suspension inference (D2, amended by D40): a function suspends when it
// contains a suspension point or directly calls a function that suspends.
// The set is computed to a fixpoint over the call graph; dynamic dispatch
// (function values, trait objects) uses the effect declared on the type.

func (c *Checker) inferSuspension(prog *Program) {
	changed := true
	for changed {
		changed = false
		for _, fn := range prog.Funcs {
			if fn.Suspends || fn.Body == nil {
				continue
			}
			if fn.suspends || blockSuspends(fn.Body) {
				fn.Suspends = true
				changed = true
			}
		}
	}
	// declared effects on function types must agree with inference
	for _, fn := range prog.Funcs {
		if fn.IsClosure {
			if fn.Sig.Effects.Suspends {
				fn.Suspends = true
			} else if fn.Suspends {
				c.errorf(fn.Span, "this lambda suspends, so its type must be a suspending function type: 'fun(...): T suspends' (D40)")
			}
			continue
		}
		if fn.tmpl == nil {
			continue
		}
		t := fn.tmpl
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
			fn.Suspends = true
		}
		if fn.Suspends && t.Name == "$init" {
			c.errorf(fn.Span, "an 'init' block cannot suspend: construction is a plain expression (D28); do the waiting in a static function that builds the value")
		}
		fn.Sig.Effects.Suspends = fn.Suspends
	}
	for _, g := range prog.Globals {
		if g.Init != nil && exprSuspends(g.Init) {
			c.errorf(g.Span, "a global initializer cannot suspend")
		}
	}
}

func blockSuspends(b *Block) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if stmtSuspends(s) {
			return true
		}
	}
	return b.Value != nil && exprSuspends(b.Value)
}

func stmtSuspends(s Stmt) bool {
	switch s := s.(type) {
	case *Block:
		return blockSuspends(s)
	case *VarDecl:
		return s.Init != nil && exprSuspends(s.Init)
	case *Assign:
		return exprSuspends(s.Target) || exprSuspends(s.Value)
	case *ExprStmt:
		return exprSuspends(s.X)
	case *Return:
		return s.Value != nil && exprSuspends(s.Value)
	case *Loop:
		if s.Cond != nil && exprSuspends(s.Cond) {
			return true
		}
		for _, p := range s.Post {
			if stmtSuspends(p) {
				return true
			}
		}
		return blockSuspends(s.Body)
	case *With:
		return exprSuspends(s.Init) || exprSuspends(s.Close) || blockSuspends(s.Body)
	case *ScopeBlock:
		return true
	}
	return false
}

func exprSuspends(e Expr) bool {
	switch e := e.(type) {
	case nil:
		return false
	case *Call:
		if e.Fn.Suspends {
			return true
		}
		return anySuspends(e.Args)
	case *CallIndirect:
		if ft, ok := e.Fn.Type().(*types.Func); ok && ft.Effects.Suspends {
			return true
		}
		return exprSuspends(e.Fn) || anySuspends(e.Args)
	case *CallVirtual:
		if e.Sig.Effects.Suspends {
			return true
		}
		return exprSuspends(e.Obj) || anySuspends(e.Args)
	case *Builtin:
		switch e.Op {
		case "chan.send", "chan.recv", "task.sleep":
			return true
		}
		return anySuspends(e.Args)
	case *AwaitTask, *ScopeBlock, *Race, *Launch:
		return true
	case *Binary:
		return exprSuspends(e.L) || exprSuspends(e.R)
	case *Unary:
		return exprSuspends(e.X)
	case *Cast:
		return exprSuspends(e.X)
	case *ToString:
		return exprSuspends(e.X)
	case *StringConcat:
		return anySuspends(e.Parts)
	case *FieldGet:
		return exprSuspends(e.X)
	case *TupleGet:
		return exprSuspends(e.X)
	case *StructLit:
		return anySuspends(e.Fields)
	case *TupleLit:
		return anySuspends(e.Elems)
	case *AddrOf:
		return exprSuspends(e.X)
	case *Deref:
		return exprSuspends(e.X)
	case *SomeWrap:
		return exprSuspends(e.X)
	case *IsNull:
		return exprSuspends(e.X)
	case *Unwrap:
		return exprSuspends(e.X)
	case *MakeVariant:
		return exprSuspends(e.Value)
	case *VariantTest:
		return exprSuspends(e.X)
	case *VariantCast:
		return exprSuspends(e.X)
	case *UnionTest:
		return exprSuspends(e.X)
	case *UnionCast:
		return exprSuspends(e.X)
	case *ErrorConvert:
		return exprSuspends(e.X)
	case *If:
		return exprSuspends(e.Cond) || blockSuspends(e.Then) || blockSuspends(e.Else)
	case *BlockExpr:
		return blockSuspends(e.Block)
	case *Match:
		if e.Init != nil && exprSuspends(e.Init) {
			return true
		}
		for _, arm := range e.Arms {
			if exprSuspends(arm.Test) || exprSuspends(arm.Guard) || blockSuspends(arm.Body) {
				return true
			}
			for _, b := range arm.Binds {
				if stmtSuspends(b) {
					return true
				}
			}
		}
	case *Try:
		return exprSuspends(e.X)
	case *Throw:
		return exprSuspends(e.Value)
	case *Elvis:
		return exprSuspends(e.L) || exprSuspends(e.R)
	case *Let:
		return exprSuspends(e.Init) || exprSuspends(e.Body)
	case *ListLit:
		return anySuspends(e.Elems)
	case *MapLit:
		for _, en := range e.Entries {
			if exprSuspends(en[0]) || exprSuspends(en[1]) {
				return true
			}
		}
	case *RangeLit:
		return exprSuspends(e.Lo) || exprSuspends(e.Hi)
	case *Box:
		return exprSuspends(e.X)
	}
	return false
}

func anySuspends(es []Expr) bool {
	for _, e := range es {
		if exprSuspends(e) {
			return true
		}
	}
	return false
}
