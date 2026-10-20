package llvm

import (
	"github.com/LaH-DeV/veles/types"
)

// D188: freeze() and the two facts the collections keep for it, in a flags
// word at the end of every list and map header — a pointer into the storage
// was made (exposed), the collection was frozen. These are the places the
// compiler emits the checks: a write to an element, and the making of a
// pointer; the runtime's own changes (push, pop, clear, a map's set) check
// inside.

const (
	flagExposed = 1
	flagFrozen  = 2
	flagsField  = 6 // the index of the flags word in listHeader
)

// listGuard emits what a use of a mutable list's element place needs. Reads
// and immutable lists need nothing.
func (g *gen) listGuard(list string, listType types.Type, mode placeMode) {
	lt, ok := listType.(*types.List)
	if !ok || !lt.Mutable || mode == placeRead {
		return
	}
	fp, f, fast := g.newTmp(), g.newTmp(), g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, listHeader, list, flagsField)
	g.emit("%s = load i64, ptr %s", f, fp)
	slowL, doneL := g.newLabel("flags.slow"), g.newLabel("flags.done")
	if mode == placeWrite {
		// not frozen: the frozen bit is the higher of the two
		g.emit("%s = icmp ult i64 %s, %d", fast, f, flagFrozen)
	} else {
		// exposed already and not frozen: nothing to record
		g.emit("%s = icmp eq i64 %s, %d", fast, f, flagExposed)
	}
	g.emitTerm("br i1 %s, label %%%s, label %%%s, !prof !{!\"branch_weights\", i32 2000, i32 1}", fast, doneL, slowL)
	g.placeLabel(slowL)
	if mode == placeWrite {
		g.emit("call void @veles_list_write_check(ptr %s)", list)
	} else {
		g.emit("call void @veles_list_expose(ptr %s)", list)
	}
	g.emitTerm("br label %%%s", doneL)
	g.placeLabel(doneL)
}

// isFrozen is the frozen bit of the flags word at index field of header: one
// load and a test (D188).
func (g *gen) isFrozen(coll, header string, field int) string {
	fp, f, bit, r := g.newTmp(), g.newTmp(), g.newTmp(), g.newTmp()
	g.emit("%s = getelementptr inbounds %s, ptr %s, i32 0, i32 %d", fp, header, coll, field)
	g.emit("%s = load i64, ptr %s", f, fp)
	g.emit("%s = and i64 %s, %d", bit, f, flagFrozen)
	g.emit("%s = icmp ne i64 %s, 0", r, bit)
	return r
}
