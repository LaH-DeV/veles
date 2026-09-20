package sema

import (
	"sort"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The prelude's Equatable, Hashable, Comparable and Display traits let a
// struct or sealed type replace the structural behaviour of `==`, map
// hashing, the ordering operators and interpolation. The checker resolves
// the impl methods of every concrete struct and sealed type in the program
// (resolveAllCustom) and hands them to the backend in Program.Custom, so
// the structural helpers find them however a value is reached — as a
// field, an element, a variant or a map key.

// customOps returns the custom operations of a concrete struct or sealed
// type, instantiating the impl methods on first use; nil when the type has
// none (or is not a named type: the built-in types keep their behaviour).
func (c *Checker) customOps(t types.Type) *CustomOps {
	var span source.Span
	switch t := t.(type) {
	case *types.Struct:
		span = c.declSpan(t.Decl)
	case *types.Sealed:
		span = c.declSpan(t.Decl)
	case *types.Tuple:
		// lexicographic order over ordered elements (tuple_order.go);
		// equality, hashing and printing stay structural
		if fn := c.tupleCompare(t); fn != nil {
			return &CustomOps{Compare: fn}
		}
		return nil
	default:
		return nil
	}
	if types.ContainsTypeParam(t) {
		return nil
	}
	key := types.Key(t)
	if ops, done := c.prog.Custom[key]; done {
		return ops
	}
	if c.prog.Custom == nil {
		c.prog.Custom = map[string]*CustomOps{}
	}
	c.prog.Custom[key] = nil // guard against recursion through field types
	ops := &CustomOps{
		Equals:   c.customMethod(t, "Equatable", "equals", span),
		Hash:     c.customMethod(t, "Hashable", "hash", span),
		Compare:  c.customMethod(t, "Comparable", "compareTo", span),
		ToString: c.customMethod(t, "Display", "toString", span),
	}
	if ops.Equals == nil && ops.Hash == nil && ops.Compare == nil && ops.ToString == nil {
		return nil
	}
	c.prog.Custom[key] = ops
	return ops
}

// customMethod instantiates method `name` of t's impl of the prelude trait,
// or returns nil when t does not implement it.
func (c *Checker) customMethod(t types.Type, traitName, name string, span source.Span) *Func {
	trait := c.traitNamed(traitName)
	if trait == nil {
		return nil
	}
	impl := c.findImplFor(t, trait)
	if impl == nil {
		return nil
	}
	tmpl, ok := impl.Methods[name]
	if !ok {
		return nil
	}
	subst := map[*types.TypeParam]types.Type{}
	unify(impl.Target, t, subst)
	return c.instantiate(tmpl, subst, nil, span)
}

// implementsPrelude reports whether t has an impl of the named prelude
// trait (a type parameter counts when one of its bounds is that trait).
func (c *Checker) implementsPrelude(t types.Type, traitName string) bool {
	trait := c.traitNamed(traitName)
	if trait == nil {
		return false
	}
	if tp, ok := t.(*types.TypeParam); ok {
		for _, b := range tp.Bounds {
			if b == trait {
				return true
			}
		}
		return false
	}
	if tt, ok := t.(*types.Tuple); ok && traitName == "Comparable" {
		return c.tupleCompare(tt) != nil
	}
	return c.findImplFor(t, trait) != nil
}

// declSpan is the position of a struct or sealed declaration, for the
// diagnostics of an instantiation made on its behalf.
func (c *Checker) declSpan(decl any) source.Span {
	switch d := decl.(type) {
	case interface{ Span() source.Span }:
		return d.Span()
	}
	return source.Span{}
}

// resolveAllCustom instantiates the custom operations of every concrete
// struct and sealed type in the program, so that the backend's helpers
// find them however a value reaches `==`, a map or an interpolation (a
// field of a generic instance, a key type used only through a method).
// Their bodies may instantiate further types, so it repeats until stable.
func (c *Checker) resolveAllCustom() {
	for {
		before := len(c.funcs)
		for _, s := range c.structs {
			if len(s.TypeParams) == 0 {
				c.customOps(s)
			}
			for _, inst := range sortedStructInstances(s) {
				c.customOps(inst)
			}
		}
		for _, s := range c.sealeds {
			if len(s.TypeParams) == 0 {
				c.customOps(s)
			}
			for _, k := range sortedKeys(s.Instances) {
				c.customOps(s.Instances[k])
			}
		}
		c.drainQueue()
		if len(c.funcs) == before {
			return
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// recvArg passes x as the receiver of fn: a pointer to its place, or to a
// copy when x is a temporary (D22 v0.30). A synthesized comparison that is
// not a method (tuple_order.go) takes the value itself.
func recvArg(fn *Func, x Expr) Expr {
	if !fn.isMethod() {
		return x
	}
	return &AddrOf{exprBase{&types.Pointer{Elem: x.Type()}}, x}
}

// isMethod reports whether fn takes a receiver — known from its template
// before the body is checked (which is what sets fn.Receiver).
func (fn *Func) isMethod() bool {
	if fn.Receiver != nil {
		return true
	}
	t := fn.tmpl
	return t != nil && !t.Decl.Static && (t.Owner != nil || t.Impl != nil || t.Trait != nil)
}
