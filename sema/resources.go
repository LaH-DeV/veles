package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
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
		if e.Async {
			// a task of the scope or gather around it (D141)
			if n := len(f.taskMarks); n > 0 {
				return f.taskMarks[n-1], false
			}
			return nil, false
		}
		// a struct built around it: `Session(conn: c)`
		if n, ok := e.Fun.(*ast.NameExpr); ok {
			if sym := f.scope.Lookup(n.Name); sym != nil && sym.Kind == SymType {
				for _, a := range e.Args {
					if c, isCall := a.Value.(*ast.CallExpr); isCall && c.Async {
						continue // a task the value holds goes where the value goes (D111)
					}
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
	if kind := f.c.taskBlocks[r]; kind != "" {
		const why = "a task belongs to the '%s' that started it, which has finished or cancelled it when the block ends; await it inside the block and keep its value (D141)"
		switch n, isName := e.(*ast.NameExpr); {
		case isName:
			f.errorf(e.Span(), "'%s' cannot %s: "+why, n.Name, doing, kind)
		case viaLambda:
			f.errorf(e.Span(), "this lambda captures a task, so it cannot %s: "+why, doing, kind)
		default:
			f.errorf(e.Span(), "this task cannot %s: "+why, doing, kind)
		}
		return true
	}
	if n, ok := e.(*ast.NameExpr); ok && n.Name != r.Name {
		f.errorf(e.Span(), "'%s' cannot %s: it holds '%s', which %s when its 'with' block ends (D100)", n.Name, doing, r.Name, f.closedWord(r))
	} else if viaLambda {
		f.errorf(e.Span(), "this lambda captures '%s', which %s when its 'with' block ends, so the lambda cannot %s; pass it as an argument instead, or keep it inside the block (D100)", r.Name, f.closedWord(r), doing)
	} else {
		f.errorf(e.Span(), "'%s' cannot %s: it %s when its 'with' block ends (D100)", r.Name, doing, f.closedWord(r))
	}
	return true
}

func (f *fnCtx) closedWord(r *Var) string {
	if f.c.lockVars[r] {
		return "points into a Mutex that is unlocked" // D107
	}
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
// on a mutable collection, a deque or a channel. Only the storing methods:
// `xs.withRaw(p => r.use(p))` or `xs.sortWith(…)` hands the lambda on for
// the call, which D100 allows.
func (f *fnCtx) checkStoreEscape(recv types.Type, e *ast.CallExpr) {
	if len(f.c.resources) == 0 || !storesArguments(recv) {
		return
	}
	m, ok := e.Fun.(*ast.MemberExpr)
	if !ok || !storingMethods[m.Name.Name] {
		return
	}
	for _, a := range e.Args {
		r, _ := f.heldResource(a.Value)
		if r == nil {
			continue
		}
		if f.c.taskBlocks[r] != "" {
			// a collection of the block's own may hold its tasks (D141)
			if f.outlives(m.X, r) && f.refuseEscape(a.Value, "be stored in '"+srcText(m.X)+"', which outlives the block") {
				return
			}
			continue
		}
		if f.refuseEscape(a.Value, "be stored in a collection or sent on a channel") {
			return
		}
	}
}

// openTaskBlock starts the marker of a scope or gather (D141): it counts as
// declared where the block opens, so whatever was declared before outlives
// the block's tasks (outlives), and each `async` in the block is its alias.
func (f *fnCtx) openTaskBlock(kind string) {
	f.c.nextVar++
	m := &Var{Name: kind, Type: &types.Task{Result: types.TUnit}, ID: f.c.nextVar}
	f.vars[m] = true
	if f.c.taskBlocks == nil {
		f.c.taskBlocks = map[*Var]string{}
	}
	f.c.taskBlocks[m] = kind
	f.markResource(m, m)
	f.taskMarks = append(f.taskMarks, m)
}

func (f *fnCtx) closeTaskBlock() {
	f.taskMarks = f.taskMarks[:len(f.taskMarks)-1]
}

// storingMethods are the methods of the types storesArguments names that
// keep an argument: the element, key, value or the function making one.
var storingMethods = map[string]bool{
	"push": true, "set": true, "setUnchecked": true, "insert": true, "addAll": true, "fill": true,
	"add": true, "getOrPut": true, "send": true, "trySend": true, "addFirst": true, "addLast": true,
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

// refuseHandClose: `r.close()` on a `with` value closes it twice — the
// block's end closes it again (D136). The fix removes the call; closing
// earlier is the block form's job.
func (f *fnCtx) refuseHandClose(callee *ast.MemberExpr, e *ast.CallExpr) {
	if callee.Name.Name != "close" || len(e.Args) != 0 || callee.Safe {
		return
	}
	n, ok := callee.X.(*ast.NameExpr)
	if !ok {
		return
	}
	r := f.resourceNamed(n.Name)
	if r == nil || f.c.lockVars[r] { // `n.close()` on a locked value closes the value, not the lock
		return
	}
	if _, isTask := r.Type.(*types.Task); isTask {
		return
	}
	f.c.errorFix(e.Pos, fixReplace("Remove the call", wholeLine(e.Pos), ""),
		"'%s' is closed when its 'with' block ends; closing it here would close it twice — remove the call, or give it a block of its own, 'with (%s = …) { … }', to close it earlier (D136)", n.Name, r.Name)
}

// wholeLine widens span to its whole line when nothing else is on it, so
// removing it leaves no blank line behind.
func wholeLine(span source.Span) source.Span {
	src := span.File.Content
	start, end := span.Start, span.End
	for start > 0 && (src[start-1] == ' ' || src[start-1] == '\t') {
		start--
	}
	for end < len(src) && (src[end] == ' ' || src[end] == '\t' || src[end] == '\r') {
		end++
	}
	if (start == 0 || src[start-1] == '\n') && (end == len(src) || src[end] == '\n') {
		if end < len(src) {
			end++
		}
		return source.Span{File: span.File, Start: start, End: end}
	}
	return span
}
