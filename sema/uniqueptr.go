package sema

import "github.com/LaH-DeV/veles/types"

// UniquePointers finds the local pointers to a list, map or set
// (`val stack = &scratch.stack`) whose collection, once made unique where
// the pointer is bound, stays unique for the rest of the function (D231): no
// copy of it is taken (a Share reaching the same place), no header is
// assigned over it, no loop lends it, and no call is handed a pointer that
// could reach it. Codegen makes the collection unique once, at the binding,
// and writes through the pointer without the copy-on-write test after
// that — the test that kept the regex backtracker 7% behind (every push and
// set reloaded the header's flags).
func UniquePointers(fn *Func) map[*Var]bool {
	if fn == nil || fn.Body == nil {
		return nil
	}
	type cand struct {
		root *Var
		path []int
		elem types.Type
		init *AddrOf
		// reached: the collection lives behind a pointer (`this`, a pointer
		// parameter or local, a dereference on the way) or in a local whose
		// address is taken elsewhere — a callee may reach it through a global,
		// a closure or a struct it holds, copy it and leave the pointer
		// writing into the copy's storage (found 2026-11-03), so any call or
		// suspension rules it out
		reached bool
	}
	cands := map[*Var]cand{}
	walkBlock(fn.Body, func(n any) {
		d, ok := n.(*VarDecl)
		if !ok || d.Var.Mutable || d.Var.Captured || d.Var.AddrTaken || d.Var.IsGlobal {
			return
		}
		a, ok := d.Init.(*AddrOf)
		if !ok {
			return
		}
		switch a.X.Type().(type) {
		case *types.List, *types.Map, *types.Set:
		default:
			return
		}
		root, path, ok := refPath(a.X)
		if !ok || root.IsGlobal || root.Captured {
			return
		}
		cands[d.Var] = cand{root, path, a.X.Type(), a, root.IsSelf || isPointer(root.Type) || throughDeref(a.X)}
	})
	if len(cands) == 0 {
		return nil
	}
	walkBlock(fn.Body, func(n any) {
		if a, ok := n.(*AddrOf); ok {
			if r, _, ok := refPath(a.X); ok {
				for v, c := range cands {
					if r == c.root && a != c.init {
						c.reached = true
						cands[v] = c
					}
				}
			}
		}
	})
	// reaches reports whether a place may be, or contain, candidate c's
	// collection: through the pointer itself, the same path, or another
	// name of the root's type
	reaches := func(v *Var, c cand, e Expr) bool {
		if derefRoot(e) == v || collRoot(e) == v {
			return true
		}
		r, p, ok := refPath(e)
		if !ok {
			return true // a place not followed: assume it may
		}
		if r == c.root {
			return overlaps(p, c.path)
		}
		return sameStruct(r.Type, c.root.Type) || pointsTo(r.Type, c.elem)
	}
	drop := func(match func(v *Var, c cand) bool) {
		for v, c := range cands {
			if match(v, c) {
				delete(cands, v)
			}
		}
	}
	callArgs := func(args []Expr) {
		for _, a := range args {
			t := a.Type()
			if p, ok := t.(*types.Pointer); ok && holdsCollection(p.Elem, map[types.Type]bool{}) {
				drop(func(v *Var, c cand) bool { return true }) // a callee could copy or replace it
				return
			}
			if r, ok := a.(*VarRef); ok {
				drop(func(v *Var, c cand) bool { return r.Var == v || r.Var == c.root && c.root.IsSelf })
			}
			// a value lent to the call: the callee marks it shared if it keeps it
			if holdsCollection(t, map[types.Type]bool{}) {
				drop(func(v *Var, c cand) bool { return reaches(v, c, a) })
			}
		}
	}
	exits := exitCalls(fn.Body)
	// onCall drops what a call may undo, but a pointer the function never
	// uses again once the call returns, on its way out (exitCalls)
	onCall := func(call Expr, args []Expr) {
		used, exit := exits[call]
		keep := map[*Var]cand{}
		if exit {
			for v, c := range cands {
				if !used[v] {
					keep[v] = c
				}
			}
		}
		if args != nil {
			callArgs(args)
		}
		drop(func(v *Var, c cand) bool { return c.reached })
		for v, c := range keep {
			cands[v] = c
		}
	}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Share:
			drop(func(v *Var, c cand) bool { return reaches(v, c, n.X) })
		case *Assign:
			// a header assigned over the place (an element of another type
			// is written in it, not over it)
			if holdsCollection(n.Target.Type(), map[types.Type]bool{}) {
				drop(func(v *Var, c cand) bool { return reaches(v, c, n.Target) })
			}
		case *Builtin:
			if (n.Op == "list.lend" || n.Op == "list.slice" || n.Op == "list.takeText") && len(n.Args) > 0 {
				drop(func(v *Var, c cand) bool { return reaches(v, c, n.Args[0]) })
			}
		case *Call:
			onCall(n, n.Args)
		case *CallIndirect:
			onCall(n, n.Args)
		case *CallVirtual:
			onCall(n, n.Args)
		case *Launch, *AwaitTask:
			drop(func(v *Var, c cand) bool { return c.reached })
		}
	})
	if len(cands) == 0 {
		return nil
	}
	// only a pointer written through gains anything from the one test
	written := map[*Var]bool{}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Builtin:
			if collectionWrites[n.Op] && len(n.Args) > 0 {
				if v := derefRoot(n.Args[0]); v != nil {
					written[v] = true
				}
			}
		case *Assign:
			if v := derefRoot(writtenPlace(n.Target)); v != nil {
				written[v] = true
			}
		}
	})
	out := map[*Var]bool{}
	for v := range cands {
		if written[v] {
			out[v] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// exitCalls finds the calls after which the function only leaves: a call
// in a `throw` or `return` statement, or in a statement followed in its
// block only by statements that end in one. For each it gives the
// variables still read on the way out. A collection a callee copies there
// is never written again through a pointer the function holds, so such a
// call does not undo a pointer made unique once (inflate's `throw
// corrupt(…)` and `this.finishBlock(); return CONTINUE`). Nil when the
// function holds a resource (`with`, a scope): its close runs on the way
// out and may reach anything.
func exitCalls(body *Block) map[Expr]map[*Var]bool {
	hasClose := false
	walkBlock(body, func(n any) {
		switch n.(type) {
		case *With, *ScopeBlock:
			hasClose = true
		}
	})
	if hasClose {
		return nil
	}
	out := map[Expr]map[*Var]bool{}
	var scan func(b *Block)
	scan = func(b *Block) {
		if b == nil {
			return
		}
		for i, s := range b.Stmts {
			c := &exitCollect{scan: scan}
			c.stmt(s)
			rest, value := b.Stmts[i+1:], b.Value
			if !exitsHere(s) && !endsInExit(rest, value) || jumps(append([]Stmt{s}, rest...)) {
				continue
			}
			// what the statement itself reads counts too: its branches run
			// after a call in its condition
			used := readVars(append([]Stmt{s}, rest...), value)
			for _, call := range c.calls {
				out[call] = used
			}
		}
		if b.Value != nil {
			c := &exitCollect{scan: scan}
			c.expr(b.Value)
			if _, ok := b.Value.(*Throw); ok {
				used := readVars(nil, b.Value)
				for _, call := range c.calls {
					out[call] = used
				}
			}
		}
	}
	scan(body)
	return out
}

// readVars: every variable the statements and the value refer to.
func readVars(stmts []Stmt, value Expr) map[*Var]bool {
	used := map[*Var]bool{}
	note := func(n any) {
		if v, ok := n.(*VarRef); ok {
			used[v.Var] = true
		}
	}
	for _, s := range stmts {
		walkStmt(s, note)
	}
	walkExpr(value, note)
	return used
}

// jumps: a `break` or `continue` among the statements, which goes back
// into a loop rather than out of the function.
func jumps(stmts []Stmt) bool {
	found := false
	for _, s := range stmts {
		walkStmt(s, func(n any) {
			switch n.(type) {
			case *Break, *Continue:
				found = true
			}
		})
	}
	return found
}

// exitsHere: s returns or throws, whatever it computes first.
func exitsHere(s Stmt) bool {
	switch s := s.(type) {
	case *Return:
		return true
	case *ExprStmt:
		_, ok := s.X.(*Throw)
		return ok
	}
	return false
}

// endsInExit: the statements (and value) after a statement end the
// function on every way through them — they hold no loop, and the last
// returns or throws.
func endsInExit(rest []Stmt, value Expr) bool {
	if value != nil {
		if _, ok := value.(*Throw); !ok {
			return false
		}
	} else if len(rest) == 0 || !exitsHere(rest[len(rest)-1]) {
		return false
	}
	for _, s := range rest {
		if _, ok := s.(*Loop); ok {
			return false
		}
	}
	return true
}

// exitCollect gathers the calls of one statement outside its nested
// blocks, which exitCalls scans on their own.
type exitCollect struct {
	calls []Expr
	scan  func(*Block)
}

func (c *exitCollect) expr(e Expr) {
	if e == nil {
		return
	}
	switch e.(type) {
	case *Call, *CallIndirect, *CallVirtual:
		c.calls = append(c.calls, e)
	}
	eachChild(e, c)
}

func (c *exitCollect) block(b *Block) { c.scan(b) }

func (c *exitCollect) stmt(s Stmt) {
	switch s := s.(type) {
	case *Block:
		c.scan(s)
	case *VarDecl:
		c.expr(s.Init)
	case *Assign:
		c.expr(s.Target)
		c.expr(s.Value)
	case *ExprStmt:
		c.expr(s.X)
	case *Return:
		c.expr(s.Value)
	case *Loop:
		// a call in a loop's condition may be followed by another round;
		// its body is a block of its own
		c.scan(s.Body)
		c.calls = nil
	default:
		c.calls = nil
	}
}

// refPath is e as a root variable and a chain of fields, reading through
// pointers (`scratch.stack` with scratch a `*Scratch`).
func refPath(e Expr) (*Var, []int, bool) {
	var path []int
	for {
		switch x := e.(type) {
		case *FieldGet:
			path = append([]int{x.Index}, path...)
			e = x.X
		case *Deref:
			e = x.X
		case *VarRef:
			return x.Var, path, true
		default:
			return nil, nil, false
		}
	}
}

// throughDeref: the place e names is reached through a dereference.
func throughDeref(e Expr) bool {
	for {
		switch x := e.(type) {
		case *FieldGet:
			e = x.X
		case *Deref:
			return true
		default:
			return false
		}
	}
}

// pointsTo: t is a pointer to elem's type.
func pointsTo(t, elem types.Type) bool {
	p, ok := t.(*types.Pointer)
	return ok && types.Identical(p.Elem, elem)
}
