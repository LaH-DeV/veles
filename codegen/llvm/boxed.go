package llvm

import "github.com/LaH-DeV/veles/types"

// Boxed variants (D198): in a recursive sealed type (a tree), and for a
// variant marked `indirect`, the payload after the tag is one pointer to a
// heap copy of the variant, made where the value is constructed. Reading the
// variant loads through it; the box is shared by every copy of the sealed
// value, so a write through a narrowed place copies it first (a variant
// with no `var` field cannot be written, and is never copied).

// boxedVariant reports whether payloadT is a variant stored behind a pointer
// in the sealed type t.
func (g *gen) boxedVariant(t, payloadT types.Type) (*types.Struct, bool) {
	if _, ok := t.(*types.Sealed); !ok {
		return nil, false
	}
	v, ok := payloadT.(*types.Struct)
	if !ok || !g.layouter().VariantBoxed(v) {
		return nil, false
	}
	return v, true
}

// makeBoxed builds the sealed value of type t holding a boxed variant.
func (g *gen) makeBoxed(t types.Type, tag int, v *types.Struct, payload string) string {
	box := g.gcAlloc(v)
	g.storeVal(v, payload, box)
	if !g.isMem(t) {
		return g.makeTagged(g.llType(t), tag, "ptr", box)
	}
	tmp := g.newMem(t)
	g.emit("store i32 %d, ptr %s", tag, g.fieldPtr(g.llType(t), tmp, 0))
	g.emit("store ptr %s, ptr %s", box, g.fieldPtr(g.llType(t), tmp, 1))
	return tmp
}

// extractBoxed reads the boxed variant v out of the sealed value x of type t.
func (g *gen) extractBoxed(t types.Type, x string, v *types.Struct) string {
	var box string
	if !g.isMem(t) {
		box = g.extractTagged(g.llType(t), x, "ptr")
	} else {
		box = g.newTmp()
		g.emit("%s = load ptr, ptr %s", box, g.fieldPtr(g.llType(t), x, 1))
	}
	return g.loadVal(v, box)
}

// boxedPlace is the variant's storage behind the payload slot at slot; for
// a write or a pointer that may outlive the access, a copy of the box is
// made first and put in the slot, so no other copy of the value sees it.
func (g *gen) boxedPlace(slot string, v *types.Struct, mode placeMode) string {
	box := g.newTmp()
	g.emit("%s = load ptr, ptr %s", box, slot)
	if mode == placeRead || !hasVarField(v) {
		return box
	}
	fresh := g.gcAlloc(v)
	size, _ := g.layout(v)
	g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", fresh, box, size)
	g.emit("store ptr %s, ptr %s", fresh, slot)
	return fresh
}

func hasVarField(v *types.Struct) bool {
	for _, f := range v.Fields {
		if f.Var || f.Protected {
			return true
		}
	}
	return false
}
