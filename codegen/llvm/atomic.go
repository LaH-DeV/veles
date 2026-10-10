package llvm

import (
	"fmt"
	"strconv"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

// atomic lowers the prelude's Atomic builtins (D66, D144): a value that
// fits a machine word is read, written, swapped, compared-and-swapped and
// (an integer) added to or masked with one instruction on its integer bits
// — a bool as a byte, a float as the integer of its width, so a compare is
// bitwise; a pointer as itself. Each takes its memory orders last.
func (g *gen) atomic(e *sema.Builtin) string {
	elem := e.Args[0].Type().(*types.Pointer).Elem
	if e.Op == "atomicLockFree" {
		if sema.AtomicWord(elem) {
			return "true"
		}
		return "false"
	}
	if !sema.AtomicWord(elem) {
		// behind `if (atomicLockFree(cell))`: never reached
		for _, a := range e.Args {
			g.expr(a)
		}
		g.emitTerm("unreachable")
		return "zeroinitializer"
	}
	vt := g.llType(elem)
	it, size := atomicInt(elem)
	cell := g.expr(e.Args[0])
	bits := func(v string) string { // the value as the integer stored
		switch {
		case vt == it:
			return v
		case vt == "i1":
			t := g.newTmp()
			g.emit("%s = zext i1 %s to i8", t, v)
			return t
		default:
			t := g.newTmp()
			g.emit("%s = bitcast %s %s to %s", t, vt, v, it)
			return t
		}
	}
	value := func(b string) string { // the integer loaded as the value
		switch {
		case vt == it:
			return b
		case vt == "i1":
			t := g.newTmp()
			g.emit("%s = trunc i8 %s to i1", t, b)
			return t
		default:
			t := g.newTmp()
			g.emit("%s = bitcast %s %s to %s", t, it, b, vt)
			return t
		}
	}
	align := fmt.Sprintf("align %d", size)
	order := func(i int) (string, string) { // an order argument: its value and type
		return g.expr(e.Args[i]), g.llType(e.Args[i].Type())
	}
	switch e.Op {
	case "atomicLoad":
		o, ot := order(1)
		return g.byOrder(o, ot, loadOrders, it, func(ord string) string {
			b := g.newTmp()
			g.emit("%s = load atomic %s, ptr %s %s, %s", b, it, cell, ord, align)
			return b
		}, value)
	case "atomicStore":
		v := bits(g.expr(e.Args[1]))
		o, ot := order(2)
		g.byOrder(o, ot, storeOrders, "", func(ord string) string {
			g.emit("store atomic %s %s, ptr %s %s, %s", it, v, cell, ord, align)
			return ""
		}, nil)
		return "zeroinitializer"
	case "atomicCompareAndSwap", "atomicCompareExchange":
		want := bits(g.expr(e.Args[1]))
		next := bits(g.expr(e.Args[2]))
		so, st := order(3)
		fo, ft := order(4)
		field, rt := 1, "i1"
		if e.Op == "atomicCompareExchange" {
			field, rt = 0, it
		}
		r := g.byOrder(so, st, allOrders, rt, func(success string) string {
			return g.byOrder(fo, ft, loadOrders, rt, func(failure string) string {
				pair := g.newTmp()
				g.emit("%s = cmpxchg ptr %s, %s %s, %s %s %s %s, %s", pair, cell, it, want, it, next, success, failure, align)
				x := g.newTmp()
				g.emit("%s = extractvalue { %s, i1 } %s, %d", x, it, pair, field)
				return x
			}, nil)
		}, nil)
		if e.Op == "atomicCompareExchange" {
			return value(r)
		}
		return r
	default: // the read-modify-writes, each returning the old value
		v := bits(g.expr(e.Args[1]))
		o, ot := order(2)
		op := map[string]string{"atomicSwap": "xchg", "atomicAdd": "add", "atomicSub": "sub",
			"atomicAnd": "and", "atomicOr": "or", "atomicXor": "xor"}[e.Op]
		return g.byOrder(o, ot, allOrders, it, func(ord string) string {
			b := g.newTmp()
			g.emit("%s = atomicrmw %s ptr %s, %s %s %s, %s", b, op, cell, it, v, ord, align)
			return b
		}, value)
	}
}

// The LLVM ordering of each MemoryOrder case (Relaxed, Acquire, Release,
// AcqRel, SeqCst) for an operation; one it cannot take acts as seq_cst (an
// order written as a case is refused at compile time instead, D144).
var (
	allOrders   = []string{"monotonic", "acquire", "release", "acq_rel", "seq_cst"}
	loadOrders  = []string{"monotonic", "acquire", "seq_cst", "seq_cst", "seq_cst"}
	storeOrders = []string{"monotonic", "seq_cst", "release", "seq_cst", "seq_cst"}
)

// byOrder emits `emit` for the ordering a MemoryOrder value selects: once
// when the value is a constant, otherwise behind a switch with one block per
// distinct ordering (which an inlined call with a constant order folds
// away). rt is the LLVM type of what emit returns ("" for nothing); conv
// turns the result into the caller's value.
func (g *gen) byOrder(o, ot string, orders []string, rt string, emit func(string) string, conv func(string) string) string {
	if conv == nil {
		conv = func(v string) string { return v }
	}
	if n, err := strconv.Atoi(o); err == nil && n >= 0 && n < len(orders) {
		return conv(emit(orders[n]))
	}
	var slot string
	if rt != "" {
		slot = g.alloca(rt)
	}
	end := g.newLabel("order.end")
	labels := map[string]string{}
	var distinct []string
	for _, ord := range orders {
		if _, ok := labels[ord]; !ok {
			labels[ord] = g.newLabel("order." + ord)
			distinct = append(distinct, ord)
		}
	}
	cases := ""
	for i, ord := range orders {
		if ord != "seq_cst" {
			cases += fmt.Sprintf(" %s %d, label %%%s", ot, i, labels[ord])
		}
	}
	g.emitTerm("switch %s %s, label %%%s [%s ]", ot, o, labels["seq_cst"], cases)
	for _, ord := range distinct {
		g.placeLabel(labels[ord])
		v := emit(ord)
		if rt != "" {
			g.emit("store %s %s, ptr %s", rt, v, slot)
		}
		g.emitTerm("br label %%%s", end)
	}
	g.placeLabel(end)
	if rt == "" {
		return ""
	}
	v := g.newTmp()
	g.emit("%s = load %s, ptr %s", v, rt, slot)
	return conv(v)
}

// atomicInt is the integer type an atomic value is stored as (a pointer is
// stored as itself), and its size.
func atomicInt(t types.Type) (string, int) {
	t = types.Underlying(t)
	switch t := t.(type) {
	case *types.Pointer, *types.Nullable:
		return "ptr", 8
	case *types.Basic:
		switch t.Kind {
		case types.Bool, types.I8, types.U8:
			return "i8", 1
		case types.I16, types.U16:
			return "i16", 2
		case types.I32, types.U32, types.F32:
			return "i32", 4
		}
	}
	return "i64", 8
}
