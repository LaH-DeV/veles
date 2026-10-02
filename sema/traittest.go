package sema

import (
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Run-time `is` on a trait object (D117, D135). A method table's slot 0
// names the concrete type it was built for, so `x is Circle` compares that
// with Circle's (codegen). `x is Flusher` needs more: a table for Flusher's
// methods over x's type, which the box was not built with. The program is
// linked whole, so for every trait some `is` tests for, the checker builds
// that table for every type that becomes a trait object anywhere and
// implements the trait; codegen lays the tables out by type identity.

// traitTests is the checker's record of both sides: the traits tested for
// and the types boxed, each in first-seen order, and the pairs done.
type traitTests struct {
	tested     []*types.Trait
	testedSeen map[*types.Trait]bool
	boxed      []types.Type
	boxedSeen  map[string]bool
	tables     map[*types.Trait]*TraitTable
	done       map[*types.Trait]int // how many of boxed each table has considered
}

// noteBoxed records a type that becomes a trait object.
func (c *Checker) noteBoxed(t types.Type) {
	k := types.Key(t)
	if c.tt.boxedSeen == nil {
		c.tt.boxedSeen = map[string]bool{}
	}
	if !c.tt.boxedSeen[k] {
		c.tt.boxedSeen[k] = true
		c.tt.boxed = append(c.tt.boxed, t)
	}
}

// noteTraitTest records a trait some `is` tests for at run time.
func (c *Checker) noteTraitTest(t *types.Trait) {
	if c.tt.testedSeen == nil {
		c.tt.testedSeen = map[*types.Trait]bool{}
	}
	if !c.tt.testedSeen[t] {
		c.tt.testedSeen[t] = true
		c.tt.tested = append(c.tt.tested, t)
	}
}

// finishInstances settles what the checked bodies left to instantiate: the
// `is` tables and the custom equality/hash/order/text of every type. Each
// can instantiate bodies that need the other, so it runs until neither adds
// a function.
func (c *Checker) finishInstances() {
	for {
		before := len(c.funcs)
		c.fillTraitTables()
		c.resolveAllCustom()
		if len(c.funcs) == before {
			return
		}
	}
}

// fillTraitTables builds the table of every tested trait over every boxed
// type that implements it. Instantiating a table's methods can check new
// bodies, which can box new types or test new traits, so it runs to a
// fixpoint with the instantiation queue.
func (c *Checker) fillTraitTables() {
	if c.tt.tables == nil {
		c.tt.tables = map[*types.Trait]*TraitTable{}
		c.tt.done = map[*types.Trait]int{}
	}
	f := &fnCtx{c: c}
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(c.tt.tested); i++ {
			tr := c.tt.tested[i]
			tab := c.tt.tables[tr]
			if tab == nil {
				tab = &TraitTable{Trait: tr}
				c.tt.tables[tr] = tab
				c.prog.TraitTables = append(c.prog.TraitTables, tab)
			}
			for c.tt.done[tr] < len(c.tt.boxed) {
				t := c.tt.boxed[c.tt.done[tr]]
				c.tt.done[tr]++
				changed = true
				if !f.implements(t, tr) {
					continue
				}
				methods, ok := f.objectMethods(t, tr, source.Span{})
				if ok {
					tab.Entries = append(tab.Entries, TraitEntry{t, methods})
				}
			}
		}
		c.drainQueue()
	}
}
