package llvm

import (
	"fmt"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// D231: lists, maps and sets are values copied on their first change. A
// collection is still one word, a pointer to its header; the header's flags
// word says whether more than one value holds it (shared) and whether a
// pointer into its storage was made (exposed, D188). Three things are
// emitted here:
//
//   - a share (sema.Share, and the elements of a fresh copy): the copy's
//     handle is the same header marked shared — or, for an exposed one, a
//     copy of its own;
//   - before a change made through a place, the header is made unique: a
//     shared one is copied, and the copy stored back in that place;
//   - per type, a function that marks every collection held by value in a
//     value laid out in memory, which the runtime calls on each element of
//     a fresh copy.

const (
	flagExposed   = 1
	flagShared    = 4
	flagLent      = 8  // a value loop walks the header: writers copy (D102 addendum)
	flagsField    = 6  // the index of the flags word in listHeader
	mapFlagsField = 13 // and in mapHeader
)

// needsShare reports whether a value of t holds a list, map or set by value:
// directly, in a field, an element of a tuple or an array, a nullable's
// payload, or an unboxed variant. A boxed variant (D198) is copied on its
// first change itself (boxedPlace), which shares what it holds then.
func (g *gen) needsShare(t types.Type) bool {
	key := types.Key(t)
	if v, ok := g.shareNeed[key]; ok {
		return v
	}
	g.shareNeed[key] = false // a type reaching itself holds it through a pointer
	v := g.holdsCollection(t)
	g.shareNeed[key] = v
	return v
}

func (g *gen) holdsCollection(t types.Type) bool {
	switch t := types.Underlying(t).(type) {
	case *types.List, *types.Map, *types.Set:
		return true
	case *types.Nullable:
		return g.needsShare(t.Elem)
	case *types.Tuple:
		for _, e := range t.Elems {
			if g.needsShare(e) {
				return true
			}
		}
	case *types.Array:
		return g.needsShare(t.Elem)
	case *types.Struct:
		if t.Union {
			return false
		}
		for _, f := range t.Fields {
			if g.needsShare(f.Type) {
				return true
			}
		}
	case *types.Sealed:
		for _, v := range t.Variants {
			if _, boxed := g.boxedVariant(t, v); !boxed && g.needsShare(v) {
				return true
			}
		}
	}
	return false
}

// shareFn is the function that marks the collections of a t in memory
// (`void (ptr)`), or "null" when a t holds none.
func (g *gen) shareFn(t types.Type) string {
	if !g.needsShare(t) {
		return "null"
	}
	key := types.Key(t)
	if name, ok := g.shareFns[key]; ok {
		return "@" + name
	}
	name := "share." + mangleType(t)
	g.shareFns[key] = name
	g.pending = append(g.pending, func() {
		g.defineHelper(name, "void", []string{"ptr %p"}, func() {
			g.shareAt(t, "%p")
			g.emitTerm("ret void")
		})
	})
	return "@" + name
}

// shareAt marks the collections of the t stored at p, in place.
func (g *gen) shareAt(t types.Type, p string) {
	switch t := types.Underlying(t).(type) {
	case *types.List, *types.Map, *types.Set:
		h := g.newTmp()
		g.emit("%s = load ptr, ptr %s", h, p)
		g.emit("store ptr %s, ptr %s", g.shareHandle(t, h), p)
	case *types.Nullable:
		// an absent one is all zeros: null handles, which a share leaves
		g.shareAt(t.Elem, g.nullPayloadPtr(t, p))
	case *types.Tuple:
		for i, e := range t.Elems {
			if g.needsShare(e) {
				g.shareAt(e, g.fieldPtr(g.llType(t), p, g.fidx(t, i)))
			}
		}
	case *types.Struct:
		for _, f := range t.Fields {
			if g.needsShare(f.Type) {
				g.shareAt(f.Type, g.fieldPtr(g.llType(t), p, g.fidx(t, f.Index)))
			}
		}
	case *types.Array:
		n := t.Len.(*types.Const).V
		et := g.llType(t.Elem)
		for i := int64(0); i < n; i++ {
			ep := g.newTmp()
			g.emit("%s = getelementptr inbounds %s, ptr %s, i64 %d", ep, et, p, i)
			g.shareAt(t.Elem, ep)
		}
	case *types.Sealed:
		tag := g.newTmp()
		g.emit("%s = load i32, ptr %s", tag, g.fieldPtr(g.llType(t), p, 0))
		payload := g.fieldPtr(g.llType(t), p, 1)
		doneL := g.newLabel("share.done")
		for _, v := range t.Variants {
			if _, boxed := g.boxedVariant(t, v); boxed || !g.needsShare(v) {
				continue
			}
			is := g.newTmp()
			g.emit("%s = icmp eq i32 %s, %d", is, tag, v.Tag)
			hitL, nextL := g.newLabel("share.variant"), g.newLabel("share.next")
			g.emitTerm("br i1 %s, label %%%s, label %%%s", is, hitL, nextL)
			g.placeLabel(hitL)
			g.shareAt(v, payload)
			g.emitTerm("br label %%%s", doneL)
			g.placeLabel(nextL)
		}
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(doneL)
	default:
		panic(fmt.Sprintf("shareAt: %s holds no collection", t))
	}
}

// shareHandle is the handle a copy of the collection h holds.
func (g *gen) shareHandle(t types.Type, h string) string {
	// a header shared already (the common case for a value copied again
	// and again, a constant) needs nothing: tested here, not in a call
	header, field := listHeader, flagsField
	if _, isList := types.Underlying(t).(*types.List); !isList {
		header, field = mapHeader, mapFlagsField
	}
	slot := g.alloca("ptr")
	g.emit("store ptr %s, ptr %s", h, slot)
	isNull, fp, f, bit, done := g.newTmp(), g.newTmp(), g.newTmp(), g.newTmp(), g.newTmp()
	testL, callL, doneL := g.newLabel("share.test"), g.newLabel("share.call"), g.newLabel("share.done")
	g.emit("%s = icmp eq ptr %s, null", isNull, h)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", isNull, doneL, testL)
	g.placeLabel(testL)
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, header, h, field)
	g.emit("%s = load i64, ptr %s", f, fp)
	g.emit("%s = and i64 %s, %d", bit, f, flagShared)
	g.emit("%s = icmp ne i64 %s, 0", done, bit)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", done, doneL, callL)
	g.placeLabel(callL)
	r := g.newTmp()
	switch t := types.Underlying(t).(type) {
	case *types.List:
		g.emit("%s = call ptr @veles_list_share(ptr %s, ptr %s)", r, h, g.shareFn(t.Elem))
	case *types.Map:
		g.emit("%s = call ptr @veles_map_share(ptr %s, ptr %s, ptr %s)", r, h, g.shareFn(t.Key), g.shareFn(t.Value))
	case *types.Set:
		g.emit("%s = call ptr @veles_map_share(ptr %s, ptr %s, ptr null)", r, h, g.shareFn(t.Elem))
	}
	g.emit("store ptr %s, ptr %s", r, slot)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(doneL)
	out := g.newTmp()
	g.emit("%s = load ptr, ptr %s", out, slot)
	return out
}

// shareValue is sema.Share: the value v of type t, its collections marked
// shared. A value in memory is copied first: v may be the place it was read
// from, which keeps its own handles (an exposed one is not replaced there).
func (g *gen) shareValue(t types.Type, v string) string {
	switch types.Underlying(t).(type) {
	case *types.List, *types.Map, *types.Set:
		return g.shareHandle(t, v)
	}
	if !g.needsShare(t) {
		return v
	}
	if g.isMem(t) {
		tmp := g.newMem(t)
		g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", tmp, v, g.memSize(t))
		g.emit("call void %s(ptr %s)", g.shareFn(t), tmp)
		return tmp
	}
	llt := g.llType(t)
	slot := g.alloca(llt)
	g.emit("store %s %s, ptr %s", llt, v, slot)
	g.emit("call void %s(ptr %s)", g.shareFn(t), slot)
	r := g.newTmp()
	g.emit("%s = load %s, ptr %s", r, llt, slot)
	return r
}

// shareElems marks the elements of a fresh list whose element handles are
// also held elsewhere (a copy, a map's keys or values, a slice).
func (g *gen) shareElems(list string, elem types.Type) {
	if fn := g.shareFn(elem); fn != "null" {
		g.emit("call void @veles_list_share_elems(ptr %s, ptr %s)", list, fn)
	}
}

// uniqueAt makes the collection stored at p unique before a change: a shared
// header is copied and the copy stored back at p. It returns the header the
// change may write. The test is one load of the flags word.
func (g *gen) uniqueAt(t types.Type, p string) string {
	return g.uniqueAtFor(t, p, false)
}

// uniqueAtFor is uniqueAt; grow: the change grows a list, whose copy is
// made with room (veles_list_unique_grow)
func (g *gen) uniqueAtFor(t types.Type, p string, grow bool) string {
	header, field, fn, args := listHeader, flagsField, "veles_list_unique", ""
	switch t := types.Underlying(t).(type) {
	case *types.List:
		if grow {
			fn = "veles_list_unique_grow"
		}
		args = g.shareFn(t.Elem)
	case *types.Map:
		header, field, fn, args = mapHeader, mapFlagsField, "veles_map_unique", g.shareFn(t.Key)+", ptr "+g.shareFn(t.Value)
	case *types.Set:
		header, field, fn, args = mapHeader, mapFlagsField, "veles_map_unique", g.shareFn(t.Elem)+", ptr null"
	default:
		panic(fmt.Sprintf("uniqueAt: %s", t))
	}
	h := g.newTmp()
	g.emit("%s = load ptr, ptr %s", h, p)
	fp, f, bit, shared := g.newTmp(), g.newTmp(), g.newTmp(), g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, header, h, field)
	g.emit("%s = load i64, ptr %s", f, fp)
	g.emit("%s = and i64 %s, %d", bit, f, flagShared|flagLent)
	g.emit("%s = icmp ne i64 %s, 0", shared, bit)
	copyL, doneL := g.newLabel("cow.copy"), g.newLabel("cow.done")
	g.emitTerm("br i1 %s, label %%%s, label %%%s, !prof !{!\"branch_weights\", i32 1, i32 2000}", shared, copyL, doneL)
	g.placeLabel(copyL)
	fresh := g.newTmp()
	g.emit("%s = call ptr @%s(ptr %s, ptr %s)", fresh, fn, h, args)
	g.emit("store ptr %s, ptr %s", fresh, p)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(doneL)
	r := g.newTmp() // p holds the header either way
	g.emit("%s = load ptr, ptr %s", r, p)
	return r
}

// knownUnique: coll's header is its place's own for the whole function, so a
// change writes it without the test (sema.UniquePointers, OwnedCollections)
func (g *gen) knownUnique(coll sema.Expr) bool {
	if d, ok := coll.(*sema.Deref); ok {
		if r, ok := d.X.(*sema.VarRef); ok && g.uniquePtr[r.Var] {
			return true
		}
	}
	r, ok := coll.(*sema.VarRef)
	return ok && g.ownedVar[r.Var]
}

// growableColl is writableColl for a change that grows the list coll (push,
// addAll, a text append): a shared header's copy gets room to grow, so the
// change does not copy the elements a second time (D231 addendum 3).
func (g *gen) growableColl(coll sema.Expr) string {
	if g.knownUnique(coll) {
		return g.writableColl(coll, placeWrite)
	}
	return g.uniqueAtFor(coll.Type(), g.placeAs(coll, placeWrite), true)
}

// clearColl empties the collection coll. A header of its own is emptied in
// place, keeping its storage for what comes next. A shared or lent one
// holds elements another holder still sees, so the place gets the type's
// empty constant instead: clearing copies nothing and allocates nothing
// (the next change copies the empty one, as after `xs = []`). The editor's
// copy hint says so ("copies xs as empty").
func (g *gen) clearColl(coll sema.Expr) {
	t := coll.Type()
	header, field, fn, empty := listHeader, flagsField, "veles_list_clear", ""
	switch u := types.Underlying(t).(type) {
	case *types.List:
		empty = g.emptyList(u.Elem)
	case *types.Map:
		header, field, fn, empty = mapHeader, mapFlagsField, "veles_map_clear", g.emptyMap(u.Key, u.Value)
	case *types.Set:
		header, field, fn, empty = mapHeader, mapFlagsField, "veles_map_clear", g.emptyMap(u.Elem, nil)
	default:
		panic(fmt.Sprintf("clearColl: %s", t))
	}
	if g.knownUnique(coll) {
		g.emit("call void @%s(ptr %s)", fn, g.writableColl(coll, placeWrite))
		return
	}
	p := g.placeAs(coll, placeWrite)
	h := g.newTmp()
	g.emit("%s = load ptr, ptr %s", h, p)
	fp, f, bit, shared := g.newTmp(), g.newTmp(), g.newTmp(), g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, header, h, field)
	g.emit("%s = load i64, ptr %s", f, fp)
	g.emit("%s = and i64 %s, %d", bit, f, flagShared|flagLent)
	g.emit("%s = icmp ne i64 %s, 0", shared, bit)
	emptyL, clearL, doneL := g.newLabel("clear.empty"), g.newLabel("clear.own"), g.newLabel("clear.done")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", shared, emptyL, clearL)
	g.placeLabel(emptyL)
	g.emit("store ptr %s, ptr %s", empty, p)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(clearL)
	g.emit("call void @%s(ptr %s)", fn, h)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(doneL)
}

// writableColl is the header a change to the collection coll may write: coll's
// own place made unique (a temporary when coll is not a place, whose change
// is lost). Expose marks it as having a pointer into its storage.
func (g *gen) writableColl(coll sema.Expr, mode placeMode) string {
	t := coll.Type()
	var h string
	if g.knownUnique(coll) {
		// made unique where it was bound: no test
		h = g.newTmp()
		g.emit("%s = load ptr, ptr %s", h, g.placeAs(coll, placeWrite))
	} else {
		h = g.uniqueAt(t, g.placeAs(coll, placeWrite))
	}
	if mode == placeExpose {
		if _, isList := types.Underlying(t).(*types.List); isList {
			g.emit("call void @veles_list_own(ptr %s)", h)
		} else {
			g.emit("call void @veles_map_own(ptr %s)", h)
		}
	}
	return h
}
