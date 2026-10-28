package llvm

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// Type descriptors for the collector (I3): each heap object's header points
// at a descriptor listing the byte offsets of the words that may hold
// pointers. The collector validates candidates against its page table, so
// a word that is a pointer in only some of a sealed value's variants is
// listed anyway.

const gcDecls = `declare ptr @veles_gc_alloc(ptr, i64)
declare ptr @veles_alloc_words(i64)
declare void @veles_gc_root(ptr, ptr)
declare void @veles_gc_collect()
declare i64 @veles_gc_collections()
declare i64 @veles_gc_live_bytes()
`

// pointerOffsets lists the candidate pointer words inside a value of type t.
func (g *gen) pointerOffsets(t types.Type) []int {
	switch tt := t.(type) {
	case *types.Enum:
		return nil
	case *types.Basic:
		if tt.Kind == types.String {
			return []int{0} // data pointer (heap or constant; the collector filters)
		}
		return nil
	case *types.Pointer, *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return []int{0}
	case *types.Func:
		if tt.C {
			return nil // a C function pointer: code, not the collector's
		}
		return []int{8} // environment
	case *types.Trait:
		return []int{0} // boxed data
	case *types.Nullable:
		if isPtrLike(tt.Elem) {
			return []int{0}
		}
		if _, ok := g.niche(tt); ok {
			return g.pointerOffsets(tt.Elem)
		}
		offs, _, _ := g.layouter().Placed([]types.Type{types.TBool, tt.Elem})
		return shift(g.pointerOffsets(tt.Elem), offs[1])
	case *types.Tuple:
		offs, _, _ := g.layouter().Placed(tt.Elems)
		var out []int
		for i, e := range tt.Elems {
			out = append(out, shift(g.pointerOffsets(e), offs[i])...)
		}
		return out
	case *types.Range:
		return nil
	case *types.Array:
		// the element's pointers, once per element (D121)
		per := g.pointerOffsets(tt.Elem)
		if len(per) == 0 {
			return nil
		}
		size, _ := g.layout(tt.Elem)
		var out []int
		for i := int64(0); i < arrayLen(tt); i++ {
			out = append(out, shift(per, int(i)*size)...)
		}
		return out
	case *types.Struct:
		if tt.Union {
			return nil // C data: numbers and raw pointers, nothing of the collector's (D120)
		}
		var out []int
		for i, off := range g.layouter().StructPlan(tt).Offsets {
			out = append(out, shift(g.pointerOffsets(tt.Fields[i].Type), off)...)
		}
		return out
	case *types.Sealed:
		// the union of what the variants hold: a word of a number is not
		// scanned, so it cannot keep garbage alive by looking like an address
		var offs []int
		for _, v := range tt.Variants {
			if g.layouter().VariantBoxed(v) {
				offs = append(offs, 0) // the box (D198)
			} else {
				offs = append(offs, g.pointerOffsets(v)...)
			}
		}
		return shift(distinctSorted(offs), g.layouter().PayloadOffset(t))
	case *types.ErrorUnion:
		var offs []int
		for _, m := range tt.Members {
			offs = append(offs, g.pointerOffsets(m)...)
		}
		return shift(distinctSorted(offs), g.layouter().PayloadOffset(t))
	}
	return nil
}

// distinctSorted sorts offs and drops repeats.
func distinctSorted(offs []int) []int {
	slices.Sort(offs)
	return slices.Compact(offs)
}

func shift(offs []int, by int) []int {
	out := make([]int, len(offs))
	for i, o := range offs {
		out[i] = o + by
	}
	return out
}

// descOf returns the descriptor constant for objects of type t.
func (g *gen) descOf(t types.Type) string {
	return g.descriptor(t, 0)
}

// arrayDescOf returns the descriptor for arrays of elements of type t.
func (g *gen) arrayDescOf(t types.Type) string {
	return g.descriptor(t, 1)
}

func (g *gen) descriptor(t types.Type, kind int) string {
	prefix := "@desc."
	if kind == 1 {
		prefix = "@adesc."
	}
	name := prefix + mangleType(t)
	key := fmt.Sprintf("%d:%s", kind, types.Key(t))
	if existing, ok := g.descs[key]; ok {
		return existing
	}
	// distinct types can share a mangled name; disambiguate
	for g.descNames[name] {
		name += "_"
	}
	g.descNames[name] = true
	g.descs[key] = name
	size, align := g.layout(t)
	if align > 8 {
		kind |= align << 8 // the collector hands out bodies this aligned (D120, veles_gc.c)
	}
	offs := g.pointerOffsets(t)
	parts := make([]string, len(offs))
	for i, o := range offs {
		parts[i] = fmt.Sprintf("i64 %d", o)
	}
	fmt.Fprintf(&g.descOut, "%s = internal constant { i64, i64, i64, [%d x i64] } { i64 %d, i64 %d, i64 %d, [%d x i64] [%s] }\n",
		name, len(offs), size, kind, len(offs), len(offs), strings.Join(parts, ", "))
	return name
}

// gcAlloc emits a collector allocation of one value of type t.
func (g *gen) gcAlloc(t types.Type) string {
	size, _ := g.layout(t)
	cell := g.newTmp()
	g.emit("%s = call ptr @veles_gc_alloc(ptr %s, i64 %d)", cell, g.descOf(t), size)
	return cell
}
