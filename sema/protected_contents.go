package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D87: `protected` on a field whose type is a collection also makes the
// collection's contents the owning type's: outside the type's own
// declarations it is never changed through a method that changes it. Since
// D231 a collection is a value, so taking it (binding, passing, returning)
// is a copy and allowed; only the change in place is refused.

// lookedField is a protected collection field read in a place D87 allows.
type lookedField struct {
	st   *types.Struct
	fld  *types.Field
	span source.Span
}

// isMutableCollection reports whether t is a list, map or set, or a
// nullable one: a collection that can be changed in place (D231).
func isMutableCollection(t types.Type) bool {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	switch t.(type) {
	case *types.List, *types.Map, *types.Set:
		return true
	}
	return false
}

// lookOnly marks e — checked next — as a place where a protected collection
// may be looked at: a method receiver, a loop head, an interpolated value.
func (f *fnCtx) lookOnly(e ast.Expr) {
	if m, ok := e.(*ast.MemberExpr); ok {
		f.lookAt = m.Pos
	}
}

// guardProtectedContents is called for every read of a field. Outside the
// type, a protected collection read as a method's receiver is remembered,
// for the method that follows; any other read is a copy (D231).
func (f *fnCtx) guardProtectedContents(st *types.Struct, fld *types.Field, span source.Span) {
	if !fld.Protected || !isMutableCollection(fld.Type) || f.insideType(st) {
		return
	}
	if f.lookAt == span {
		f.looked = &lookedField{st, fld, span}
	}
}

// takeLooked returns the protected field the last look-only check saw, once.
func (f *fnCtx) takeLooked() *lookedField {
	l := f.looked
	f.looked = nil
	return l
}

// refuseContentsChange reports a mutating method called on a protected
// collection from outside its type.
func (f *fnCtx) refuseContentsChange(l *lookedField, method string, span source.Span) {
	f.errorf(span, "cannot call '%s' on '%s.%s': the field is 'protected var', so only '%s' changes its contents (D87); call a method of '%s' that does", method, l.st.Name, l.fld.Name, l.st.Name, l.st.Name)
}

// changingMethods are, per collection, the built-in and prelude methods
// that change it in place (D231).
var changingMethods = map[string]map[string]bool{
	"List": {"push": true, "addAll": true, "reserve": true, "set": true, "setUnchecked": true, "at": true, "pop": true,
		"clear": true, "insert": true, "removeAt": true, "sort": true, "sortWith": true, "swap": true, "fill": true,
		"updateAt": true, "incrementAt": true, "pushI": true, "pushU": true, "pushIsizeBe": true, "pushIsizeLe": true,
		"pushUsizeBe": true, "pushUsizeLe": true},
	"Map": {"set": true, "reserve": true, "at": true, "remove": true, "clear": true, "atOrPut": true},
	"Set": {"add": true, "reserve": true, "remove": true, "clear": true},
}

// mutatesCollection reports whether a built-in or prelude method changes
// its receiver.
func mutatesCollection(rt types.Type, name string) bool {
	return changingMethods[builtinFamily(rt)][name]
}
