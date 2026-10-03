package llvm

import (
	"fmt"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Inline arrays (D121): `Array<T, N>` is LLVM's `[N x T]`, a value that
// lives in a local, a field or another aggregate. An element is reached
// through the array's address, so a dynamic index never needs the whole
// array as one SSA value.

func arrayLen(t types.Type) int64 {
	n, _ := t.(*types.Array).N()
	return n
}

// arrayElemPtr is the address of element i of the array arr denotes,
// bounds-checked inline: an index outside 0..<N (a negative one included,
// compared unsigned) goes to the runtime's panic with the location `where`.
// unchecked (D114) checks in a debug build only.
func (g *gen) arrayElemPtr(arr sema.Expr, i string, unchecked bool, span source.Span) string {
	n := arrayLen(arr.Type())
	base := g.place(arr)
	if unchecked {
		g.uncheckedIndexCheck(i, fmt.Sprint(n), "array", span)
	} else {
		ok := g.newTmp()
		g.emit("%s = icmp ult i64 %s, %d", ok, i, n)
		inL, outL := g.newLabel("idx.ok"), g.newLabel("idx.bad")
		g.emitTerm("br i1 %s, label %%%s, label %%%s, !prof !{!\"branch_weights\", i32 2000, i32 1}", ok, inL, outL)
		g.placeLabel(outL)
		wp, wl := g.strPtrLen(g.stringConst(g.where(span)))
		g.emit("call void @veles_array_index_panic(i64 %s, i64 %d, ptr %s, i64 %s)", i, n, wp, wl)
		g.emitTerm("unreachable")
		g.placeLabel(inL)
	}
	p := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i64 0, i64 %s", p, g.llType(arr.Type()), base, i)
	return p
}

// arrayLit builds an array from its elements: stored one by one into a
// temporary, which is then the value.
func (g *gen) arrayLit(e *sema.ListLit) string {
	t := e.Type()
	llt := g.llType(t)
	if len(e.Elems) == 0 {
		return "zeroinitializer"
	}
	tmp := g.alloca(llt)
	elem := t.(*types.Array).Elem
	for i, el := range e.Elems {
		v := g.expr(el)
		p := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i64 0, i64 %d", p, llt, tmp, i)
		g.storeVal(elem, v, p)
	}
	return g.loadVal(t, tmp)
}

// arrayBuiltin is the array operations the list ones do not cover:
// `array.make` (every element a copy of one value), `array.toList`,
// `list.toArray`.
func (g *gen) arrayBuiltin(e *sema.Builtin) (string, bool) {
	switch e.Op {
	case "array.make":
		return g.arrayMake(e.Type().(*types.Array), g.expr(e.Args[0])), true
	case "array.rawData":
		return g.place(e.Args[0]), true
	case "array.toList":
		return g.arrayToList(e), true
	case "list.toArray":
		return g.listToArray(e), true
	}
	return "", false
}

// arrayMake fills an array with copies of v: a store of zero for a zero
// constant, otherwise a loop.
func (g *gen) arrayMake(at *types.Array, v string) string {
	n := arrayLen(at)
	llt := g.llType(at)
	if n == 0 {
		return "zeroinitializer"
	}
	if v == "zeroinitializer" || v == "0" || v == "false" || v == "null" || v == "0.0" || v == "0.000000e+00" {
		if g.isMem(at) {
			return g.zeroMem(at)
		}
		return "zeroinitializer"
	}
	tmp := g.alloca(llt)
	idx := g.alloca("i64")
	g.emit("store i64 0, ptr %s", idx)
	headL, bodyL, doneL := g.newLabel("fill.head"), g.newLabel("fill.body"), g.newLabel("fill.done")
	g.emitTerm("br label %%%s", headL)
	g.placeLabel(headL)
	i := g.newTmp()
	g.emit("%s = load i64, ptr %s", i, idx)
	more := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %d", more, i, n)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, doneL)
	g.placeLabel(bodyL)
	p := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i64 0, i64 %s", p, llt, tmp, i)
	g.storeVal(at.Elem, v, p)
	next := g.newTmp()
	g.emit("%s = add nuw nsw i64 %s, 1", next, i)
	g.emit("store i64 %s, ptr %s", next, idx)
	g.emitTerm("br label %%%s", headL)
	g.placeLabel(doneL)
	return g.loadVal(at, tmp)
}

// arrayToList copies an array into a new immutable List.
func (g *gen) arrayToList(e *sema.Builtin) string {
	at := e.Args[0].Type().(*types.Array)
	n := arrayLen(at)
	src := g.place(e.Args[0])
	list := g.newTmp()
	g.emit("%s = call ptr @veles_list_new(ptr %s, i64 %d)", list, g.arrayDescOf(at.Elem), n)
	if n > 0 {
		size, _ := g.layout(at.Elem)
		data := g.newTmp()
		g.emit("%s = load ptr, ptr %s", data, list)
		g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", data, src, int64(size)*n)
		lenP := g.newTmp()
		g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 1", lenP, listHeader, list)
		g.emit("store i64 %d, ptr %s", n, lenP)
	}
	return list
}

// listToArray is `xs.toArray<N>()`: the elements of a list of exactly N
// as an array, null otherwise.
func (g *gen) listToArray(e *sema.Builtin) string {
	nt := e.Type().(*types.Nullable)
	at := nt.Elem.(*types.Array)
	n := arrayLen(at)
	list := g.expr(e.Args[0])
	have := g.listLen(list)
	fits := g.newTmp()
	g.emit("%s = icmp eq i64 %s, %d", fits, have, n)
	res := g.alloca(g.llType(nt))
	okL, badL, doneL := g.newLabel("toarr.ok"), g.newLabel("toarr.bad"), g.newLabel("toarr.done")
	g.emitTerm("br i1 %s, label %%%s, label %%%s", fits, okL, badL)
	g.placeLabel(okL)
	llt := g.llType(at)
	if g.isMem(nt) {
		// built where it ends up: the flag, then the elements copied in
		g.emit("store i1 true, ptr %s", g.fieldPtr(g.llType(nt), res, 0))
		if n > 0 {
			size, _ := g.layout(at.Elem)
			data := g.newTmp()
			g.emit("%s = load ptr, ptr %s", data, list)
			g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", g.fieldPtr(g.llType(nt), res, 1), data, int64(size)*n)
		}
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(badL)
		g.emit("store i1 false, ptr %s", g.fieldPtr(g.llType(nt), res, 0))
		g.emitTerm("br label %%%s", doneL)
		g.placeLabel(doneL)
		return res
	}
	tmp := g.alloca(llt)
	if n > 0 {
		size, _ := g.layout(at.Elem)
		data := g.newTmp()
		g.emit("%s = load ptr, ptr %s", data, list)
		g.emit("call void @llvm.memcpy.p0.p0.i64(ptr %s, ptr %s, i64 %d, i1 false)", tmp, data, int64(size)*n)
	}
	arr := g.newTmp()
	g.emit("%s = load %s, ptr %s", arr, llt, tmp)
	some := g.makeNullableT(nt, "true", arr)
	g.emit("store %s %s, ptr %s", g.llType(nt), some, res)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(badL)
	none := g.makeNullableT(nt, "false", "zeroinitializer")
	g.emit("store %s %s, ptr %s", g.llType(nt), none, res)
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(doneL)
	r := g.newTmp()
	g.emit("%s = load %s, ptr %s", r, g.llType(nt), res)
	return r
}

// arrayBase is the address of an array value held in an SSA register: the
// value is stored to a temporary, so that its elements can be indexed.
func (g *gen) arrayBase(t types.Type, v string) string {
	return g.spill(t, v)
}

// arrayLoop emits `for i in 0..<N { body(i) }`; the code after it follows
// the loop. body receives the index register.
func (g *gen) arrayLoop(n int64, body func(i string)) {
	idx := g.alloca("i64")
	g.emit("store i64 0, ptr %s", idx)
	condL, bodyL, endL := g.newLabel("arr.cond"), g.newLabel("arr.body"), g.newLabel("arr.end")
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(condL)
	iv := g.newTmp()
	g.emit("%s = load i64, ptr %s", iv, idx)
	more := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %d", more, iv, n)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, endL)
	g.placeLabel(bodyL)
	body(iv)
	next := g.newTmp()
	g.emit("%s = add nuw nsw i64 %s, 1", next, iv)
	g.emit("store i64 %s, ptr %s", next, idx)
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(endL)
}

func (g *gen) arrayElem(at *types.Array, base, i string) string {
	p := g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i64 0, i64 %s", p, g.llType(at), base, i)
	return g.loadVal(at.Elem, p)
}

// showArray is "[" + the elements joined by ", " + "]", as a list's.
func (g *gen) showArray(at *types.Array, v string) string {
	n := arrayLen(at)
	if n == 0 {
		return g.stringConst("[]")
	}
	base := g.arrayBase(at, v)
	res := g.alloca(strType)
	g.emit("store %s %s, ptr %s", strType, g.stringConst("["), res)
	g.arrayLoop(n, func(i string) {
		cur := g.newTmp()
		g.emit("%s = load %s, ptr %s", cur, strType, res)
		first := g.newTmp()
		g.emit("%s = icmp eq i64 %s, 0", first, i)
		sep := g.newTmp()
		g.emit("%s = select i1 %s, %s %s, %s %s", sep, first, strType, g.stringConst(""), strType, g.stringConst(", "))
		cur = g.concat(cur, sep)
		cur = g.concat(cur, g.show(at.Elem, g.arrayElem(at, base, i)))
		g.emit("store %s %s, ptr %s", strType, cur, res)
	})
	last := g.newTmp()
	g.emit("%s = load %s, ptr %s", last, strType, res)
	return g.concat(last, g.stringConst("]"))
}

// eqArray compares two arrays element by element, stopping at the first
// pair that differs.
func (g *gen) eqArray(at *types.Array, a, b string) string {
	n := arrayLen(at)
	if n == 0 {
		return "true"
	}
	pa, pb := g.arrayBase(at, a), g.arrayBase(at, b)
	idx := g.alloca("i64")
	g.emit("store i64 0, ptr %s", idx)
	condL, bodyL, nextL, endL := g.newLabel("aeq.cond"), g.newLabel("aeq.body"), g.newLabel("aeq.next"), g.newLabel("aeq.end")
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(condL)
	iv := g.newTmp()
	g.emit("%s = load i64, ptr %s", iv, idx)
	more := g.newTmp()
	g.emit("%s = icmp slt i64 %s, %d", more, iv, n)
	g.emitTerm("br i1 %s, label %%%s, label %%%s", more, bodyL, endL)
	g.placeLabel(bodyL)
	same := g.equal(at.Elem, g.arrayElem(at, pa, iv), g.arrayElem(at, pb, iv))
	g.emitTerm("br i1 %s, label %%%s, label %%%s", same, nextL, endL)
	g.placeLabel(nextL)
	next := g.newTmp()
	g.emit("%s = add nuw nsw i64 %s, 1", next, iv)
	g.emit("store i64 %s, ptr %s", next, idx)
	g.emitTerm("br label %%%s", condL)
	g.placeLabel(endL)
	// reached from the exhausted loop (all equal) or from a mismatch
	ok := g.newTmp()
	g.emit("%s = load i64, ptr %s", ok, idx)
	r := g.newTmp()
	g.emit("%s = icmp eq i64 %s, %d", r, ok, n)
	return r
}

// hashArray folds the element hashes in order, seeded with the length.
func (g *gen) hashArray(at *types.Array, v string) string {
	n := arrayLen(at)
	if n == 0 {
		return g.mix("17", "0")
	}
	base := g.arrayBase(at, v)
	acc := g.alloca("i64")
	g.emit("store i64 %s, ptr %s", g.mix("17", fmt.Sprint(n)), acc)
	g.arrayLoop(n, func(i string) {
		cur := g.newTmp()
		g.emit("%s = load i64, ptr %s", cur, acc)
		g.emit("store i64 %s, ptr %s", g.mix(cur, g.hash(at.Elem, g.arrayElem(at, base, i))), acc)
	})
	out := g.newTmp()
	g.emit("%s = load i64, ptr %s", out, acc)
	return out
}
