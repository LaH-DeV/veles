package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D87: `protected` on a field whose type is a mutable collection also makes
// the collection's contents the owning type's. Outside the type's own
// declarations the field may only be looked at — the receiver of an
// immutable-form method, a loop head, an interpolated value — never taken
// (bound, passed, returned) and never changed through a mutating method.

// lookedField is a protected collection field read in a place D87 allows.
type lookedField struct {
	st   *types.Struct
	fld  *types.Field
	span source.Span
}

// isMutableCollection reports whether t is a MutableList, MutableMap or
// MutableSet, or a nullable one.
func isMutableCollection(t types.Type) bool {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	switch t := t.(type) {
	case *types.List:
		return t.Mutable
	case *types.Map:
		return t.Mutable
	case *types.Set:
		return t.Mutable
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
// type, a protected mutable collection read anywhere but a look-only place
// is refused; in one it is remembered, for the method that follows.
func (f *fnCtx) guardProtectedContents(st *types.Struct, fld *types.Field, span source.Span) {
	if !fld.Protected || !isMutableCollection(fld.Type) || f.insideType(st) {
		return
	}
	if f.lookAt == span {
		f.looked = &lookedField{st, fld, span}
		return
	}
	f.errorf(span, "cannot take '%s.%s' out of '%s': the field is 'protected var', so only '%s' changes or hands out its contents (D87); read it with an immutable method, loop over it or interpolate it, or copy it with '.toList()'", st.Name, fld.Name, st.Name, st.Name)
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

// mutatesCollection reports whether a built-in method changes its receiver:
// the catalogue lists the mutable family's own methods apart from the
// immutable form's.
func mutatesCollection(rt types.Type, name string) bool {
	family := builtinFamily(rt)
	d := lookupBuiltinDoc(family, name)
	return d != nil && d.Recv == family
}
