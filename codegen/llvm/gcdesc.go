package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// Type descriptors for the collector (I3): each heap object's header points
// at a descriptor listing the byte offsets of the words that may hold
// pointers. The collector validates candidates against its page table, so
// payloads that are pointers only for some variants are listed anyway.

const gcDecls = `declare ptr @veles_gc_alloc(ptr, i64)
declare ptr @veles_alloc_words(i64)
declare void @veles_gc_root(ptr, ptr)
declare void @veles_gc_collect()
declare i64 @veles_gc_collections()
declare i64 @veles_gc_live_bytes()
`

// fieldOffsets returns the byte offset of each field under natural layout.
func (g *gen) fieldOffsets(fields []types.Type) []int {
	offs := make([]int, len(fields))
	size := 0
	for i, f := range fields {
		fs, fa := g.layout(f)
		size = (size + fa - 1) / fa * fa
		offs[i] = size
		size += fs
	}
	return offs
}

// pointerOffsets lists the candidate pointer words inside a value of type t.
func (g *gen) pointerOffsets(t types.Type) []int {
	switch tt := t.(type) {
	case *types.Basic:
		if tt.Kind == types.String {
			return []int{0} // data pointer (heap or constant; the collector filters)
		}
		return nil
	case *types.Pointer, *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return []int{0}
	case *types.Func:
		return []int{8} // environment
	case *types.Trait:
		return []int{0} // boxed data
	case *types.Nullable:
		if isPtrLike(tt.Elem) {
			return []int{0}
		}
		_, ea := g.layout(tt.Elem)
		base := (1 + ea - 1) / ea * ea
		return shift(g.pointerOffsets(tt.Elem), base)
	case *types.Tuple:
		return g.compositeOffsets(tt.Elems)
	case *types.Range:
		return nil
	case *types.Struct:
		var fs []types.Type
		for _, f := range tt.Fields {
			fs = append(fs, f.Type)
		}
		return g.compositeOffsets(fs)
	case *types.Sealed, *types.ErrorUnion:
		words := g.payloadWords(t)
		var offs []int
		for i := 0; i < words; i++ {
			offs = append(offs, 8+8*i)
		}
		return offs
	}
	return nil
}

func shift(offs []int, by int) []int {
	out := make([]int, len(offs))
	for i, o := range offs {
		out[i] = o + by
	}
	return out
}

func (g *gen) compositeOffsets(fields []types.Type) []int {
	var out []int
	offs := g.fieldOffsets(fields)
	for i, f := range fields {
		out = append(out, shift(g.pointerOffsets(f), offs[i])...)
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
	size, _ := g.layout(t)
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
