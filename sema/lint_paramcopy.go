package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/types"
)

// Lint: a function that changes a by-value struct parameter (R20 follow-up
// 1, decided 2026-09-25 as a warning). A struct is a value (D7), so a
// parameter is the caller's value copied; assigning one of its `var`
// fields, or calling a method that writes `self` on it, changes the copy
// and the caller never sees it:
//
//	fun step(f: Fuzzer) { f.rng.next() }   // the caller's generator never moves
//
// examples/fuzz fed the same input 2000 times through exactly this. The
// warning sits on the parameter, where the fix is written: take `*Fuzzer`
// (and pass `&x`), or return the changed value.
//
// Silent where the copy is the point: the function returns a value of the
// parameter's type (the copy-change-return idiom), the parameter is used
// whole as a value after all (copy, change, store it somewhere), or its
// address is taken (a pointer may be written through on purpose).
// Reference-typed fields are not involved at all: pushing into a
// MutableList field, or writing through a `*State` field, is seen by the
// caller, and rootVar stops at a pointer, so those writes are not counted.
func (c *Checker) lintParamCopies(prog *Program) {
	for _, fn := range prog.Funcs {
		if fn.Body == nil || len(fn.Params) == 0 {
			continue
		}
		for _, p := range fn.Params {
			if _, ok := types.Underlying(p.Type).(*types.Struct); !ok || !p.Span.IsValid() {
				continue
			}
			if fn.Sig != nil && types.Identical(fn.Sig.Ret, p.Type) {
				continue
			}
			if what := paramWrite(fn, p); what != "" {
				c.warnf(p.Span, "'%s' is a copy of the caller's '%s': this function %s, and the caller never sees the change; take '%s: *%s' (and pass '&x'), or return the changed value",
					p.Name, p.Type, what, p.Name, p.Type)
			}
		}
	}
}

// paramWrite says how fn changes its by-value parameter p — "assigns
// 'p.f'" or "calls 'm', which changes it" — or "" when it does not, or when
// the copy is plainly intended (p used whole, or its address taken).
func paramWrite(fn *Func, p *Var) string {
	what := ""
	escaped := false
	placed := map[*VarRef]bool{} // VarRefs of p that only root a place
	recvs := map[*AddrOf]bool{}
	var mark func(e Expr)
	mark = func(e Expr) {
		for {
			switch x := e.(type) {
			case *VarRef:
				placed[x] = true
				return
			case *FieldGet:
				e = x.X
			case *TupleGet:
				e = x.X
			default:
				return
			}
		}
	}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Assign:
			if rootVar(n.Target) == p {
				mark(n.Target)
				if what == "" {
					what = fmt.Sprintf("assigns '%s'", placeText(n.Target))
				}
			}
		case *Call:
			if n.Fn.Receiver != nil && len(n.Args) > 0 {
				if a, ok := n.Args[0].(*AddrOf); ok && rootVar(a.X) == p {
					recvs[a] = true
					mark(a.X)
					if n.Fn.WritesSelf && what == "" {
						what = fmt.Sprintf("calls '%s', which changes '%s'", n.Fn.Display, placeText(a.X))
					}
				}
			}
		case *AddrOf:
			if !recvs[n] && rootVar(n.X) == p {
				escaped = true
			}
		case *FieldGet:
			mark(n)
		case *TupleGet:
			mark(n)
		}
	})
	if what == "" || escaped {
		return ""
	}
	whole := false
	walkBlock(fn.Body, func(n any) {
		if r, ok := n.(*VarRef); ok && r.Var == p && !placed[r] {
			whole = true
		}
	})
	if whole {
		return ""
	}
	return what
}

// placeText spells a place rooted in a variable: `f.rng`, `p.pos.x`.
func placeText(e Expr) string {
	switch x := e.(type) {
	case *VarRef:
		return x.Var.Name
	case *FieldGet:
		return placeText(x.X) + "." + x.Name
	case *TupleGet:
		return fmt.Sprintf("%s.%d", placeText(x.X), x.Index)
	}
	return "it"
}
