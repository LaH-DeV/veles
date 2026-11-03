package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/types"
)

// Lint: a function that changes a by-value struct parameter (R20 follow-up
// 1, decided 2026-09-25 as a warning). A struct is a value (D7), so a
// parameter is the caller's value copied; assigning one of its `var`
// fields, or calling a method that writes `this` on it, changes the copy
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
// Writing through a `*State` field is seen by the caller, and rootVar stops
// at a pointer, so those writes are not counted; pushing into a collection
// field changes the copy like any other field since D231.
func (c *Checker) lintParamCopies(prog *Program) {
	for _, fn := range prog.Funcs {
		if fn.Body == nil || len(fn.Params) == 0 {
			continue
		}
		for _, p := range fn.Params {
			st, ok := types.Underlying(p.Type).(*types.Struct)
			if !ok || !p.Span.IsValid() {
				continue
			}
			if sync, _ := syncWrapper(st); sync {
				// a Watch's copy keeps what its holder has seen, on purpose
				// (D146); the shared state behind every wrapper is the caller's
				continue
			}
			if fn.Sig != nil && types.Identical(fn.Sig.Ret, p.Type) {
				continue
			}
			if what := paramWrite(fn, p, holdsCollection(p.Type, map[types.Type]bool{})); what != "" {
				c.warnLint("param_copy", p.Span, nil, "'%s' is a copy of the caller's '%s': this function %s, and the caller never sees the change; take '%s: *%s' (and pass '&x'), or return the changed value",
					p.Name, p.Type, what, p.Name, p.Type)
			}
		}
	}
}

// paramWrite says how fn changes its by-value parameter p — "assigns
// 'p.f'" or "calls 'm', which changes it" — or "" when it does not, or when
// the copy is plainly intended (p used whole, or its address taken). With
// strict (p holds a collection) a use of p whole does not excuse the change:
// passing such a struct on was how its collections were shared before D231
// (a StringBuilder handed down a chain of helpers that append to it).
func paramWrite(fn *Func, p *Var, strict bool) string {
	what := ""
	escaped := false
	placed := map[*VarRef]bool{} // VarRefs of p that only root a place
	recvs := map[*AddrOf]bool{}
	mark := func(e Expr) {
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
		case *Builtin:
			// a change to a collection held in it: a value too since D231
			if collectionWrites[n.Op] && collRoot(n.Args[0]) == p {
				mark(n.Args[0])
				if what == "" {
					what = fmt.Sprintf("changes '%s'", placeText(n.Args[0]))
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
	if whole && !strict {
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
	case *Deref:
		// `this` reads its place through the receiver pointer; a field of a
		// pointer reads through it the same way (`p.items`)
		if r, ok := x.X.(*VarRef); ok {
			if r.Var.IsSelf {
				return "this"
			}
			return r.Var.Name
		}
	}
	return "it"
}
