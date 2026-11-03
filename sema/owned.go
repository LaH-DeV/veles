package sema

import "github.com/LaH-DeV/veles/types"

// OwnedCollections finds the local lists and maps a function never lets
// anything else hold once they are bound (D231): no Share of them (the
// share pass's mark of a second holder), no address taken (a pointer, a
// prelude method's `this`), no capture, no reassignment, and no slice or
// text taken over their storage (the two built-ins that mark the header
// shared themselves). Such a variable's header is made unique once, where
// it is bound — one built from an empty literal starts as a fresh header
// instead of the shared empty constant — and a change through it needs no
// copy-on-write test after that: the test is what kept the `map`/`filter`
// prelude loops and the regex backtracker's visited set behind the
// immutable lists they replaced. It returns the variables and the empty
// literals that initialise some of them.
func OwnedCollections(fn *Func) (map[*Var]bool, map[Expr]bool) {
	if fn == nil || fn.Body == nil || fn.Extern {
		return nil, nil
	}
	init := map[*Var]Expr{}
	walkBlock(fn.Body, func(n any) {
		d, ok := n.(*VarDecl)
		if !ok || !ownedVarOK(d.Var) {
			return
		}
		if d.Init != nil {
			init[d.Var] = d.Init
		}
	})
	if len(init) == 0 {
		return nil, nil
	}
	drop := func(e Expr) {
		walkExpr(e, func(n any) {
			if r, ok := n.(*VarRef); ok {
				delete(init, r.Var)
			}
		})
	}
	// a value lent to a call: the callee marks it shared if it keeps it
	dropArgs := func(args []Expr) {
		for _, a := range args {
			if r, ok := a.(*VarRef); ok {
				delete(init, r.Var)
			}
		}
	}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Share:
			drop(n.X)
		case *AddrOf:
			drop(n.X)
		case *Assign:
			if r, ok := n.Target.(*VarRef); ok {
				delete(init, r.Var)
			}
		case *Builtin:
			if (n.Op == "list.slice" || n.Op == "list.takeText") && len(n.Args) > 0 {
				drop(n.Args[0])
			}
		case *VarDecl:
			if n.Var.Lent && n.Var.LentFor != nil {
				drop(n.Init) // a loop's snapshot of a list its body changes
			}
		case *Call:
			dropArgs(n.Args)
		case *CallIndirect:
			dropArgs(n.Args)
		case *CallVirtual:
			dropArgs(n.Args)
		}
	})
	if len(init) == 0 {
		return nil, nil
	}
	// only a local that is written gains anything: one only read is never
	// made unique (that would copy a shared value for nothing)
	written := map[*Var]bool{}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Builtin:
			if collectionWrites[n.Op] && len(n.Args) > 0 {
				if v := collRoot(n.Args[0]); v != nil {
					written[v] = true
				}
			}
		case *Assign:
			if v := collRoot(n.Target); v != nil {
				written[v] = true
			}
		}
	})
	for v, e := range init {
		if !written[v] {
			if _, lit := e.(*ListLit); !lit {
				if _, mlit := e.(*MapLit); !mlit {
					delete(init, v)
				}
			}
		}
	}
	vars, lits := map[*Var]bool{}, map[Expr]bool{}
	for v, e := range init {
		vars[v] = true
		switch lit := e.(type) {
		case *ListLit:
			if len(lit.Elems) == 0 {
				lits[e] = true
			}
		case *MapLit:
			if len(lit.Entries) == 0 {
				lits[e] = true
			}
		}
	}
	return vars, lits
}

// ownedVarOK: a local of the function itself holding a list or a map.
func ownedVarOK(v *Var) bool {
	if v == nil || v.IsGlobal || v.IsParam || v.IsSelf || v.AddrTaken || v.Captured || v.Lent {
		return false
	}
	switch v.Type.(type) {
	case *types.List, *types.Map:
		return true
	}
	return false
}

// growOps are the changes that grow a list: a copy made for one is given room
// (codegen's veles_list_unique_grow, D231 addendum 3).
var growOps = map[string]bool{"list.push": true, "list.addAll": true, "list.appendText": true, "list.insertText": true}

// GrownCollections are the local lists (and the local pointers to lists)
// that fn grows: an owned local or a unique pointer made unique where it is
// bound is copied with room when it is one of them, as the change would
// otherwise copy the elements again at once to make that room.
func GrownCollections(fn *Func) map[*Var]bool {
	if fn == nil || fn.Body == nil {
		return nil
	}
	out := map[*Var]bool{}
	walkBlock(fn.Body, func(n any) {
		b, ok := n.(*Builtin)
		if !ok || !growOps[b.Op] || len(b.Args) == 0 {
			return
		}
		if v := collRoot(b.Args[0]); v != nil {
			out[v] = true
		}
		if v := derefRoot(b.Args[0]); v != nil {
			out[v] = true
		}
	})
	return out
}
