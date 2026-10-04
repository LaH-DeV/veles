package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// Supertraits (D58): `trait Codable : Encodable, Decodable { }` requires
// its supers. A bound `T: Codable` carries the supers' bounds, so `T` has
// their methods; an impl of `Codable` needs impls of the supers — and an
// empty impl derives the ones the type lacks (derive.go). A trait whose
// only content is its supers is implemented by whatever implements all of
// them. A trait with supertraits IS a trait object: its table composes the
// supers' (objectSlots below), so an object answers to every inherited
// method; what a trait object still cannot do is become another trait's
// object, which would need the concrete type back.

// resolveSupers resolves a trait's supertraits. It runs for every trait
// before any bound is read, since expansion is transitive.
func (c *Checker) resolveSupers(t *types.Trait) {
	ctx := c.traitDecl[t]
	d := ctx.decl.(*ast.TraitDecl)
	if len(d.Supers) == 0 {
		return
	}
	if d.Sealed {
		c.errorf(d.Supers[0].Span(), "a sealed trait has no supertraits: its variants implement traits one by one (D12)")
		return
	}
	env := c.envFor(ctx, selfParamOf(t))
	for _, s := range d.Supers {
		st := c.resolveType(env, s)
		tr, ok := st.(*types.Trait)
		if !ok {
			if !types.IsInvalid(st) {
				c.errorf(s.Span(), "supertrait '%s' is not a trait", st)
			}
			continue
		}
		if tr == t {
			c.errorf(s.Span(), "trait '%s' cannot require itself", t.Name)
			continue
		}
		if containsTrait(t.Supers, tr) {
			c.errorf(s.Span(), "supertrait '%s' is listed twice", tr.Name)
			continue
		}
		t.Supers = append(t.Supers, tr)
	}
}

// checkSuperCycles reports a trait that reaches itself through its supers.
func (c *Checker) checkSuperCycles() {
	for _, t := range c.traits {
		if containsTrait(allSupers(t), t) {
			d := c.traitDecl[t].decl.(*ast.TraitDecl)
			c.errorf(d.Name.Pos, "trait '%s' requires itself through its supertraits", t.Name)
			t.Supers = nil // so nothing loops on it later
		}
	}
}

// allSupers is the transitive closure of a trait's supertraits, not
// including the trait itself unless a cycle brings it back.
func allSupers(t *types.Trait) []*types.Trait {
	var out []*types.Trait
	seen := map[*types.Trait]bool{}
	var walk func(x *types.Trait)
	walk = func(x *types.Trait) {
		for _, s := range x.Supers {
			if seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
			walk(s)
		}
	}
	walk(t)
	return out
}

// withSupers appends a bound and, transitively, its supertraits.
func (c *Checker) withSupers(bounds []*types.Trait, t *types.Trait) []*types.Trait {
	if !containsTrait(bounds, t) {
		bounds = append(bounds, t)
	}
	for _, s := range allSupers(t) {
		if !containsTrait(bounds, s) {
			bounds = append(bounds, s)
		}
	}
	return bounds
}

// isCombination reports whether a trait is nothing but its supertraits —
// no methods, no associated types — so that implementing every super is
// implementing it.
func isCombination(t *types.Trait) bool {
	return len(t.Supers) > 0 && len(t.MethodList) == 0 && len(t.AssocTypes) == 0
}

// superOwning returns the trait among t and its supers that declares the
// method, or nil.
func superOwning(t *types.Trait, method string) *types.Trait {
	if _, ok := t.Methods[method]; ok {
		return t
	}
	for _, s := range allSupers(t) {
		if _, ok := s.Methods[method]; ok {
			return s
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// trait objects of a trait with supertraits (D58/D9)

// objSlot is one entry in a trait object's method table: the method and the
// trait that declares it — the object's own trait, or one of its supers.
type objSlot struct {
	Owner *types.Trait
	Name  string
	Sig   *types.Func
}

// objectOrder is the trait itself and its supers, each once, supers first
// (depth first). It is the order the vtable is laid out in, so a subtrait's
// table starts with its first super's — not that anything depends on it
// yet: a trait object is never converted to another trait's object, which
// would need the concrete type back (checklist §9 Q5, `is Trait`).
func objectOrder(t *types.Trait) []*types.Trait {
	var out []*types.Trait
	seen := map[*types.Trait]bool{}
	var walk func(x *types.Trait)
	walk = func(x *types.Trait) {
		if seen[x] {
			return
		}
		seen[x] = true
		for _, s := range x.Supers {
			walk(s)
		}
		out = append(out, x)
	}
	walk(t)
	return out
}

// objectSlots is the vtable layout of a trait used as an object: every
// method the object answers to. A name declared both by the trait and by a
// super is one slot, the most derived declaration; two supers that neither
// requires the other declaring the same name is an ambiguity, returned as
// the second result (the trait is then not object safe).
func objectSlots(t *types.Trait) ([]objSlot, string) {
	order := objectOrder(t)
	owner := map[string]*types.Trait{}
	for _, tr := range order {
		for _, name := range tr.MethodList {
			prev, seen := owner[name]
			if seen && !containsTrait(allSupers(tr), prev) {
				return nil, "supertraits '" + prev.Name + "' and '" + tr.Name +
					"' both declare '" + name + "'"
			}
			owner[name] = tr
		}
	}
	var out []objSlot
	for _, tr := range order {
		for _, name := range tr.MethodList {
			if owner[name] != tr {
				continue // overridden by a more derived declaration
			}
			out = append(out, objSlot{Owner: tr, Name: name, Sig: tr.Methods[name]})
		}
	}
	return out, ""
}

// findSlot is the object's slot for a method name, and its index.
func findSlot(slots []objSlot, name string) (objSlot, int) {
	for i, s := range slots {
		if s.Name == name {
			return s, i
		}
	}
	return objSlot{}, -1
}

// ObjectSigs is the declaration of each slot of t's object table, in slot
// order — what codegen needs to know of a slot besides the impl's function:
// a slot that suspends is called, and filled, as a coroutine (D40).
func ObjectSigs(t *types.Trait) []*types.Func {
	slots, _ := objectSlots(t)
	out := make([]*types.Func, len(slots))
	for i, s := range slots {
		out[i] = s.Sig
	}
	return out
}

// sendableTraitOf is the prelude's `Sendable` among t's supers, or nil.
func sendableTraitOf(t *types.Trait) *types.Trait {
	for _, s := range allSupers(t) {
		if isSendableTrait(s) {
			return s
		}
	}
	return nil
}

// objectIs reports whether t is a trait object whose trait is, or requires,
// trait: `with s = <io.Stream>` closes through the Closeable it requires.
func objectIs(t types.Type, trait *types.Trait) bool {
	o, ok := t.(*types.Trait)
	return ok && (o == trait || containsTrait(allSupers(o), trait))
}
