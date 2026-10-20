package sema

import "github.com/LaH-DeV/veles/types"

// D187/D188: collections that never leave their function. The header of a
// MutableList / MutableMap / MutableSet is a heap object only because it has
// an identity others may share; one that is created in a function and only
// ever used there — read, pushed to, looped over — can live in the frame, and
// costs no allocation at all. Codegen asks StackCollections for the
// initializers it may build in a stack slot.
//
// What may be done with it: the collection's own operations (the Builtin
// ops below) with it as the receiver, and the plain aliases lowering makes
// of it (`loop (x in xs)` copies it into a temporary). Anything else — an
// argument, a return value, a stored field, a closure capture, an assignment,
// `&` — and it is a heap object as before. The storage the header points at
// stays on the heap: only the header moves.

// stackOps are the Builtin ops that use a collection without keeping a
// reference to its header: result values are new objects, or pointers into
// the element storage. For each, which arguments are such uses.
var stackOps = map[string][]int{
	"list.len": {0}, "list.get": {0}, "list.ref": {0}, "list.refUnchecked": {0},
	"list.push": {0}, "list.pop": {0}, "list.clear": {0}, "list.reserve": {0},
	"list.copy": {0}, "list.slice": {0}, "list.touch": {0}, "list.mods": {0}, "list.checkMods": {0},
	"list.joinText": {0}, "list.appendText": {0}, "list.sortedNative": {0}, "list.toArray": {0}, "list.rawData": {0},
	"list.addAll":   {0, 1},
	"list.isFrozen": {0}, "map.isFrozen": {0},
	"map.len": {0}, "map.get": {0}, "map.ref": {0}, "map.refOrPanic": {0}, "map.contains": {0},
	"map.set": {0}, "map.remove": {0}, "map.clear": {0}, "map.reserve": {0}, "map.copy": {0},
	"map.keys": {0}, "map.values": {0}, "map.entries": {0}, "map.mods": {0}, "map.checkMods": {0},
	"set.add": {0},
}

// stackCandidate reports whether e builds an empty-or-literal mutable
// collection a fresh header of which nobody else holds.
func stackCandidate(e Expr) bool {
	switch e := e.(type) {
	case *ListLit:
		lt, ok := e.Type().(*types.List)
		return ok && lt.Mutable
	case *MapLit:
		mt, ok := e.Type().(*types.Map)
		return ok && mt.Mutable
	case *Builtin:
		if e.Op == "set.new" {
			st, ok := e.Type().(*types.Set)
			return ok && st.Mutable
		}
	}
	return false
}

// stackVarOK: a variable whose value is only ever the one it was declared
// with and that nothing else can reach.
func stackVarOK(v *Var) bool {
	return v != nil && !v.IsParam && !v.IsGlobal && !v.Captured && !v.AddrTaken && !v.IsSelf
}

type stackCand struct {
	init    Expr
	escaped bool
}

// StackCollections returns the collection-building expressions of fn whose
// header may live in fn's frame. A function that can suspend is left alone:
// its frame is a coroutine's.
func StackCollections(fn *Func) map[Expr]bool {
	if fn == nil || fn.Body == nil || fn.Extern || fn.Suspends {
		return nil
	}
	cands := map[Expr]*stackCand{}
	// a variable's own collection, and its plain aliases
	group := map[*Var]*stackCand{}
	// a struct variable built here, with the candidate behind each field
	structs := map[*Var]map[int]*stackCand{}

	walkBlock(fn.Body, func(n any) {
		d, ok := n.(*VarDecl)
		if !ok || !stackVarOK(d.Var) {
			return
		}
		switch init := d.Init.(type) {
		case *StructLit:
			for i, fe := range init.Fields {
				if fe != nil && stackCandidate(fe) {
					c := &stackCand{init: fe}
					cands[fe] = c
					if structs[d.Var] == nil {
						structs[d.Var] = map[int]*stackCand{}
					}
					structs[d.Var][i] = c
				}
			}
		default:
			if stackCandidate(d.Init) {
				c := &stackCand{init: d.Init}
				cands[d.Init] = c
				group[d.Var] = c
			}
		}
	})
	if len(cands) == 0 {
		return nil
	}
	// aliases: `val t = xs` (a lowering temporary or a plain copy of the name)
	for changed := true; changed; {
		changed = false
		walkBlock(fn.Body, func(n any) {
			d, ok := n.(*VarDecl)
			if !ok || !stackVarOK(d.Var) || group[d.Var] != nil {
				return
			}
			if r, ok := d.Init.(*VarRef); ok && group[r.Var] != nil {
				group[d.Var] = group[r.Var]
				changed = true
			}
		})
	}

	okRef := map[*VarRef]bool{}     // a mention that is a permitted use
	okField := map[*FieldGet]bool{} // a field read of a struct variable that is a permitted use
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *VarDecl:
			if r, ok := n.Init.(*VarRef); ok && group[n.Var] != nil && group[r.Var] == group[n.Var] {
				okRef[r] = true // the alias being made
			}
		case *Builtin:
			for _, i := range stackOps[n.Op] {
				if i >= len(n.Args) {
					continue
				}
				switch a := n.Args[i].(type) {
				case *VarRef:
					okRef[a] = true
				case *FieldGet:
					okField[a] = true
				}
			}
		case *FieldGet:
			if r, ok := n.X.(*VarRef); ok && structs[r.Var] != nil {
				okRef[r] = true // reading a field of the struct is not the struct escaping
				if c := structs[r.Var][n.Index]; c != nil && !okField[n] {
					c.escaped = true // the field itself is handed somewhere
				}
			}
		case *VarRef:
			if c := group[n.Var]; c != nil && !okRef[n] {
				c.escaped = true
			}
			if fs := structs[n.Var]; fs != nil && !okRef[n] {
				for _, c := range fs {
					c.escaped = true // the struct as a whole is used
				}
			}
		case *Closure:
			for _, v := range n.Captures {
				if c := group[v]; c != nil {
					c.escaped = true
				}
				for _, c := range structs[v] {
					c.escaped = true
				}
			}
		}
	})
	out := map[Expr]bool{}
	for e, c := range cands {
		if !c.escaped {
			out[e] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
