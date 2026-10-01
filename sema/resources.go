package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// A resource must not outlive its block (D100 part 3). A `with` binding is
// closed — a task cancelled — when its block ends, so the value may not
// leave the block: returned, made the block's value, assigned to anything
// declared outside it (an outer `var`, a field, a global), stored in a
// collection or sent on a channel — nor may a lambda that captures it.
// Passing it, or such a lambda, as an argument is allowed:
// `withTimeout(d, () => try read(conn))` must compile. A callee that keeps
// what it is passed is not caught; the docs say so (chapter 7).
//
// The checks are local: they look at the expression written at each of
// those places — the name, a tuple, list, map or struct literal around it,
// or a lambda that names it — and follow `val y = x` aliases.

// markResource records v as a resource, or as an alias of the resource
// root.
func (f *fnCtx) markResource(v, root *Var) {
	if f.c.resources == nil {
		f.c.resources = map[*Var]*Var{}
	}
	f.c.resources[v] = root
}

// resourceNamed is the resource a name in scope here refers to, if any.
func (f *fnCtx) resourceNamed(name string) *Var {
	if sym := f.scope.Lookup(name); sym != nil && sym.Kind == SymLocal {
		return f.c.resources[sym.Var]
	}
	return nil
}

// heldResource is the resource e's value holds, and whether it holds it
// through a lambda that captures it.
func (f *fnCtx) heldResource(e ast.Expr) (*Var, bool) {
	if len(f.c.resources) == 0 || e == nil {
		return nil, false
	}
	switch e := e.(type) {
	case *ast.NameExpr:
		return f.resourceNamed(e.Name), false
	case *ast.TupleExpr:
		return f.firstHeld(e.Elems...)
	case *ast.ListLit:
		return f.firstHeld(e.Elems...)
	case *ast.MapLit:
		for _, en := range e.Entries {
			if r, l := f.firstHeld(en.Key, en.Value); r != nil {
				return r, l
			}
		}
	case *ast.CallExpr:
		// a struct built around it: `Session(conn: c)`
		if n, ok := e.Fun.(*ast.NameExpr); ok {
			if sym := f.scope.Lookup(n.Name); sym != nil && sym.Kind == SymType {
				for _, a := range e.Args {
					if r, l := f.heldResource(a.Value); r != nil {
						return r, l
					}
				}
			}
		}
	case *ast.LambdaExpr:
		return f.capturedResource(e), true
	}
	return nil, false
}

func (f *fnCtx) firstHeld(es ...ast.Expr) (*Var, bool) {
	for _, x := range es {
		if r, l := f.heldResource(x); r != nil {
			return r, l
		}
	}
	return nil, false
}

// capturedResource is a resource a lambda names in its body (not one of
// its own parameters).
func (f *fnCtx) capturedResource(l *ast.LambdaExpr) *Var {
	own := map[string]bool{}
	for _, p := range l.Params {
		own[p.Name.Name] = true
	}
	var found *Var
	walkAST(l.Body, func(n any) bool {
		if found != nil {
			return false
		}
		if ne, ok := n.(*ast.NameExpr); ok && !own[ne.Name] {
			found = f.resourceNamed(ne.Name)
		}
		return true
	})
	return found
}

// refuseEscape reports e when its value holds a resource; doing says what
// the program does with it ("be returned").
func (f *fnCtx) refuseEscape(e ast.Expr, doing string) bool {
	r, viaLambda := f.heldResource(e)
	if r == nil {
		return false
	}
	if n, ok := e.(*ast.NameExpr); ok && n.Name != r.Name {
		f.errorf(e.Span(), "'%s' cannot %s: it holds '%s', which %s when its 'with' block ends (D100)", n.Name, doing, r.Name, closedWord(r))
	} else if viaLambda {
		f.errorf(e.Span(), "this lambda captures '%s', which %s when its 'with' block ends, so the lambda cannot %s; pass it as an argument instead, or keep it inside the block (D100)", r.Name, closedWord(r), doing)
	} else {
		f.errorf(e.Span(), "'%s' cannot %s: it %s when its 'with' block ends (D100)", r.Name, doing, closedWord(r))
	}
	return true
}

func closedWord(r *Var) string {
	if _, isTask := r.Type.(*types.Task); isTask {
		return "is cancelled"
	}
	return "is closed"
}

// checkReturnEscape: `return conn` inside the block that opened conn. A
// resource of an enclosing function, captured by a lambda, may be
// returned from the lambda to whoever called it with the resource open.
func (f *fnCtx) checkReturnEscape(value ast.Expr) {
	if r, _ := f.heldResource(value); r != nil && f.owns(r) {
		f.refuseEscape(value, "be returned")
	}
}

// checkAssignEscape: `outer = conn`, `this.conn = conn`, `cache.c = conn`
// where the target was declared before the resource (so it outlives it).
func (f *fnCtx) checkAssignEscape(s *ast.AssignStmt) {
	r, _ := f.heldResource(s.Value)
	if r == nil || !f.outlives(s.Target, r) {
		return
	}
	f.refuseEscape(s.Value, "be stored in '"+srcText(s.Target)+"', which outlives the block")
}

// outlives reports whether the place target names lives on after the block
// of resource r: a global, a parameter or `this`, a local declared before r,
// or anything reached through a pointer.
func (f *fnCtx) outlives(target ast.Expr, r *Var) bool {
	viaPointer := false
	for {
		switch t := target.(type) {
		case *ast.MemberExpr:
			target = t.X
			continue
		case *ast.IndexExpr:
			target = t.X
			continue
		case *ast.NameExpr:
			sym := f.scope.Lookup(t.Name)
			if sym == nil {
				return false
			}
			if sym.Kind != SymLocal {
				return true
			}
			v := sym.Var
			if _, isPtr := v.Type.(*types.Pointer); isPtr && target != ast.Expr(t) {
				viaPointer = true
			}
			return viaPointer || v.IsSelf || v.IsParam || v.IsGlobal || v.ID < r.ID
		case *ast.UnaryExpr:
			// `*p = conn`
			target = t.X
			viaPointer = true
			continue
		}
		return true
	}
}

// checkStoreEscape: an argument of a method that stores what it is given —
// on a mutable collection, a deque or a channel.
func (f *fnCtx) checkStoreEscape(recv types.Type, e *ast.CallExpr) {
	if len(f.c.resources) == 0 || !storesArguments(recv) {
		return
	}
	for _, a := range e.Args {
		if f.refuseEscape(a.Value, "be stored in a collection or sent on a channel") {
			return
		}
	}
}

func storesArguments(t types.Type) bool {
	switch t := t.(type) {
	case *types.Pointer:
		return storesArguments(t.Elem)
	case *types.List:
		return t.Mutable
	case *types.Map:
		return t.Mutable
	case *types.Set:
		return t.Mutable
	case *types.Channel:
		return true
	case *types.Struct:
		return t.Module == "std.prelude" && (t.Name == "Deque" || t.Name == "PriorityQueue")
	}
	return false
}
