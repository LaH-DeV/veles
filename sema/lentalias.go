package sema

import "github.com/LaH-DeV/veles/types"

// Lent aliases (D231). `val insts = prog.insts` binds a copy of a collection
// held in a place: the share pass marks it shared, so that a change to
// either holder copies it. When neither can change while the binding
// lives — the binding is only read, and nothing in the function writes the
// place it was read from, takes its address or hands it to a call that
// could — the binding is the place's own value under another name: it is
// lent, as the compiler's own temporaries are (newLentTemp), and nothing is
// marked. Found in the regex VM, where a search copied its program (six
// lists and a map) four times and marked every list each time.
//
// The place's root is a parameter the function only reads (ReadOnly), the
// receiver `this`, or a local; the path below it is a chain of fields. A
// write, an address taken or a write-through-pointer to an overlapping path
// (one a prefix of the other) rules it out, as does any change to the
// binding itself, the receiver handed to a call, or a capture. A root
// reached through a pointer (`this`, a pointer local) may be reached by a
// callee through a global, a closure or a struct it holds, so there any
// call or suspension rules it out too (found 2026-11-03).
func (s *sharer) lentAlias(v *Var, init Expr) bool {
	if s.fn == nil || s.fn.Body == nil || v.Mutable || v.AddrTaken || v.Captured || v.IsParam || v.IsGlobal || v.Lent {
		return false
	}
	if !holdsCollection(v.Type, map[types.Type]bool{}) {
		return false
	}
	root, path, ok := placePath(init)
	if !ok || root.IsGlobal || root.Captured || root.AddrTaken {
		return false
	}
	switch {
	case root.IsSelf:
	case root.IsParam:
		if !root.ReadOnly {
			return false
		}
	case root.Mutable:
		return false
	}
	reached := root.IsSelf || isPointer(root.Type)
	conflict := false
	// touches reports whether a written place reaches the binding or the
	// part of root it was read from
	touches := func(e Expr) bool {
		if collRoot(e) == v || derefRoot(e) == v {
			return true
		}
		if root.IsSelf {
			// another pointer to the receiver's type may point at it
			if p := derefRoot(e); p != nil && sameStruct(p.Type, root.Type) {
				return true
			}
		}
		r, p, ok := placePath(writtenPlace(e))
		if !ok {
			return collRoot(e) == root
		}
		if r != root && root.IsSelf && sameStruct(r.Type, root.Type) {
			return true
		}
		return r == root && overlaps(p, path)
	}
	passesRoot := func(args []Expr) bool {
		for _, a := range args {
			if r, ok := a.(*VarRef); ok && r.Var == root && root.IsSelf {
				return true // the receiver itself: the callee may change any of it
			}
		}
		return false
	}
	walkBlock(s.fn.Body, func(n any) {
		if conflict {
			return
		}
		switch n := n.(type) {
		case *Assign:
			conflict = touches(n.Target)
		case *AddrOf:
			conflict = touches(n.X)
		case *Builtin:
			conflict = collectionWrites[n.Op] && len(n.Args) > 0 && touches(n.Args[0])
		case *Call:
			conflict = reached || passesRoot(n.Args)
		case *CallIndirect:
			conflict = reached || passesRoot(n.Args)
		case *CallVirtual:
			conflict = reached || passesRoot(n.Args)
		case *Launch, *AwaitTask:
			conflict = reached
		}
	})
	return !conflict
}

// placePath is e as a variable and a chain of field indexes below it: `v`,
// `v.f.g`, `this.f` (the receiver read through its pointer).
func placePath(e Expr) (*Var, []int, bool) {
	var path []int
	for {
		switch x := e.(type) {
		case *FieldGet:
			path = append([]int{x.Index}, path...)
			e = x.X
		case *Deref:
			r, ok := x.X.(*VarRef)
			if !ok || !r.Var.IsSelf {
				return nil, nil, false
			}
			return r.Var, path, true
		case *VarRef:
			return x.Var, path, true
		default:
			return nil, nil, false
		}
	}
}

// writtenPlace is the collection place a write reaches: an element's list,
// a tuple's or a variant's whole.
func writtenPlace(e Expr) Expr {
	for {
		switch x := e.(type) {
		case *Builtin:
			if x.Op != "list.ref" && x.Op != "list.refUnchecked" && x.Op != "map.ref" {
				return e
			}
			e = x.Args[0]
		case *TupleGet:
			e = x.X
		case *Unwrap:
			e = x.X
		case *VariantCast:
			e = x.X
		default:
			return e
		}
	}
}

// overlaps: one field path is a prefix of the other.
func overlaps(a, b []int) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sameStruct: a and b are, or point at, the same type — two names that may
// reach one object.
func sameStruct(a, b types.Type) bool {
	if p, ok := a.(*types.Pointer); ok {
		a = p.Elem
	}
	if p, ok := b.(*types.Pointer); ok {
		b = p.Elem
	}
	return types.Identical(a, b)
}

// ownedOnceParams are the parameters a function keeps that need no mark
// where it keeps them. A parameter the function keeps is not ReadOnly, so
// its caller marked the value shared or moved it in: it is the function's
// own. Used whole exactly once — outside any loop, never changed, its
// parts only read — that one keep is a move (`Machine(prog, …)` in
// `Machine.of(prog)`), as returning a local at its end is.
func ownedOnceParams(fn *Func) map[*Var]bool {
	if fn == nil || fn.Body == nil {
		return nil
	}
	cands := map[*Var]bool{}
	for _, p := range fn.Params {
		if p != nil && !p.ReadOnly && !p.Captured && !p.AddrTaken && !p.IsSelf && !p.Lent &&
			holdsCollection(p.Type, map[types.Type]bool{}) {
			cands[p] = true
		}
	}
	if len(cands) == 0 {
		return nil
	}
	whole := map[*Var]int{}
	inLoop := map[*VarRef]bool{}
	reads := map[string]bool{"list.len": true, "list.get": true, "map.get": true, "map.len": true, "map.contains": true}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Loop:
			walkStmt(n, func(m any) {
				if r, ok := m.(*VarRef); ok {
					inLoop[r] = true
				}
			})
		case *VarRef:
			whole[n.Var]++
		case *FieldGet:
			if r, ok := n.X.(*VarRef); ok {
				whole[r.Var]-- // a part read: counted above as a VarRef
			}
		case *Builtin:
			if len(n.Args) > 0 {
				if r, ok := n.Args[0].(*VarRef); ok && cands[r.Var] {
					if reads[n.Op] {
						whole[r.Var]--
					} else if collectionWrites[n.Op] {
						delete(cands, r.Var)
					}
				}
			}
		case *Assign:
			if v := collRoot(n.Target); v != nil {
				delete(cands, v)
			}
		case *AddrOf:
			if v := collRoot(n.X); v != nil {
				delete(cands, v)
			}
		}
	})
	out := map[*Var]bool{}
	for p := range cands {
		if whole[p] == 1 {
			out[p] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	// the one whole use must not repeat
	walkBlock(fn.Body, func(n any) {
		if r, ok := n.(*VarRef); ok && out[r.Var] && inLoop[r] {
			delete(out, r.Var)
		}
	})
	return out
}

// partsMoved are the tuple-holding locals read only as their parts, each
// part once, by the bindings that follow their declaration in the same
// block — what `val (d, n) = draft()` lowers to (a temporary, then one
// `val` per part). Nothing reads the tuple after its parts are taken, so
// taking a part is a move: without it the first change to `d.headers`
// copied the list for nothing (found by the copy hint, 2026-11-02).
func partsMoved(fn *Func) map[*Var]bool {
	if fn == nil || fn.Body == nil {
		return nil
	}
	refs := map[*Var]int{}
	walkBlock(fn.Body, func(n any) {
		if r, ok := n.(*VarRef); ok {
			refs[r.Var]++
		}
	})
	out := map[*Var]bool{}
	var visit func(b *Block)
	visit = func(b *Block) {
		for i, st := range b.Stmts {
			d, ok := st.(*VarDecl)
			if !ok || d.Init == nil || held(d.Init) {
				continue
			}
			v := d.Var
			if _, isTuple := types.Underlying(v.Type).(*types.Tuple); !isTuple || v.IsParam || v.Captured || v.AddrTaken || v.IsGlobal || v.Lent {
				continue
			}
			taken := map[int]bool{}
			for _, next := range b.Stmts[i+1:] {
				nd, ok := next.(*VarDecl)
				if !ok {
					break
				}
				tg, ok := nd.Init.(*TupleGet)
				if !ok {
					break
				}
				r, ok := tg.X.(*VarRef)
				if !ok || r.Var != v || taken[tg.Index] {
					break
				}
				taken[tg.Index] = true
			}
			if len(taken) > 0 && refs[v] == len(taken) {
				out[v] = true
			}
		}
	}
	blocks := []*Block{fn.Body}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Block:
			blocks = append(blocks, n)
		case *Loop:
			blocks = append(blocks, n.Body)
		case *If:
			blocks = append(blocks, n.Then, n.Else)
		case *BlockExpr:
			blocks = append(blocks, n.Block)
		case *Match:
			for _, arm := range n.Arms {
				blocks = append(blocks, arm.Body)
			}
		case *With:
			blocks = append(blocks, n.Body)
		case *ScopeBlock:
			blocks = append(blocks, n.Body)
		case *Race:
			for _, arm := range n.Arms {
				blocks = append(blocks, arm.Body)
			}
		}
	})
	for _, b := range blocks {
		if b != nil {
			visit(b)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
