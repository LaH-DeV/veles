package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D231: lists, maps and sets are values. A change is made where the value
// lives — a variable, a field, a dereference, an element reached with
// `ref` — so that codegen can copy a shared header first and write the
// fresh one back to that place. A change to anything else changes a
// temporary copy.

// removedMut reports the `mut` prefix of a literal: every collection can
// change now, so the plain literal is the same value.
func (f *fnCtx) removedMut(span source.Span) {
	f.c.errorFix(span, fixDropMut(span), "'mut' before a literal was removed (D231): a collection can be changed in place, so the plain literal is enough")
}

// removedRename reports a collection method renamed by D231's follow-ups
// (`ref`/`refAt` → `at`, `getOrPut` → `atOrPut`), with a fix to the new name.
func (f *fnCtx) removedRename(e *ast.CallExpr, old, now, why string) {
	m, ok := e.Fun.(*ast.MemberExpr)
	if !ok {
		f.errorf(e.Pos, "'%s' was renamed '%s' (D231): %s", old, now, why)
		return
	}
	f.c.errorFix(m.Name.Pos, fixReplace("Replace with '"+now+"'", m.Name.Pos, now), "'%s' was renamed '%s' (D231): %s", old, now, why)
}

// accessName maps a removed accessor spelling of a list or an array onto
// the current one, reporting it with a fix: `at` is the pointer to change
// the element, `get` the copy (D231 addendum 2).
func (f *fnCtx) accessName(e *ast.CallExpr, name string) string {
	switch name {
	case "atUnchecked":
		f.removedRename(e, "atUnchecked", "getUnchecked", "it reads a copy, as 'get' does; 'at' gives a pointer")
		return "getUnchecked"
	case "atOrDefault":
		f.removedRename(e, "atOrDefault", "getOrDefault", "it reads a copy, as 'get' does; 'at' gives a pointer")
		return "getOrDefault"
	}
	return name
}

// removedSelfCopy reports `xs.toList()` on a list (and `toMap` on a map,
// `toSet` on a set): binding the value is the copy (D231 addendum). The fix
// drops the call.
func (f *fnCtx) removedSelfCopy(e *ast.CallExpr, recvExpr Expr, name, typeName string) {
	m, ok := e.Fun.(*ast.MemberExpr)
	if !ok {
		f.errorf(e.Pos, "'%s()' on a %s was removed (D231): the value is the copy", name, typeName)
		return
	}
	recv := srcText(m.X)
	if d, isDeref := recvExpr.(*Deref); isDeref {
		if r, ok := d.X.(*VarRef); ok && !r.Var.IsSelf {
			recv = "*" + recv // called through a pointer: the copy is what it points at
		}
	}
	f.c.errorFix(e.Pos, fixReplace("Replace with '"+recv+"'", e.Pos, recv),
		"'%s()' on a %s was removed (D231): '%s' is a %s already, and binding it, passing it or returning it copies it", name, typeName, recv, typeName)
}

// writeReceiver checks the receiver of a built-in change to a collection
// (writePlace).
func (f *fnCtx) writeReceiver(recv Expr, e *ast.CallExpr, lost bool) {
	name, recvAst := "it", ast.Expr(nil)
	if m, ok := e.Fun.(*ast.MemberExpr); ok {
		name, recvAst = "'"+m.Name.Name+"'", m.X
	}
	f.writePlace(recv, recvAst, e.Pos, name, lost)
}

// writePlace checks what a change to a collection is made to. A place is
// the collection itself; its root variable's facts (the bounds a loop
// proved, D62) no longer hold after the change. A temporary is a copy the
// change is lost on: an error when the operation gives nothing back
// (lost), as for a struct method that changes its receiver (receivers.go).
// what names the change in the message.
func (f *fnCtx) writePlace(recv Expr, recvAst ast.Expr, span source.Span, what string, lost bool) {
	if st := constOwner(recv); st != nil {
		f.errorf(span, "%s would change a collection inside '%s', a const struct, which is immutable all the way down (D212, D231)", what, st.Name)
		return
	}
	if isPlaceExpr(recv) && !inPacked(recv) {
		root := rootVar(recv)
		if root == nil {
			return // behind a pointer, or inside a collection's storage
		}
		if root.IsGlobal && !root.Mutable {
			f.errorf(span, "'%s' is a global 'val', a constant every task shares; changing it is not allowed — copy it first: 'var copy = %s' (D231)", root.Name, root.Name)
			return
		}
		if !strings.HasPrefix(root.Name, "$tmp") { // not a temporary the lowering made (newTemp)
			f.invalidateVarPaths(root)
			return
		}
	}
	if !lost {
		return
	}
	recvText := "the value"
	if recvAst != nil {
		recvText = srcText(recvAst)
		if repl, ok := elemReadCall(recvAst); ok {
			f.c.errorFix(recvAst.Span(), fixReplace("Replace with '"+repl+"'", recvAst.Span(), repl),
				"%s changes a copy of the element '%s', so the change would be lost; reach the element itself with '%s' (D231)", what, recvText, repl)
			return
		}
	}
	f.errorf(span, "%s changes a temporary copy of '%s' that is then discarded; bind it with 'var' to change a copy, or reach the place with 'at' or a pointer (D231)", what, recvText)
}

// collectionWrites are the built-ins that change the collection in their
// first argument, which is its place (D231).
var collectionWrites = map[string]bool{
	"list.push": true, "list.pop": true, "list.clear": true, "list.reserve": true, "list.addAll": true,
	"list.move": true, "list.touch": true, "list.appendText": true, "list.trimText": true, "list.asciiCase": true,
	"list.replaceText": true, "list.insertText": true, "list.rawData": true, "list.own": true,
	"map.set": true, "map.remove": true, "map.clear": true, "map.reserve": true, "map.ref": true,
	"map.refOrPanic": true, "set.add": true, "map.own": true,
}

// collRoot is rootVar through the elements of lists too: an element lies in
// the list's value, so a change to it is a change to the variable holding
// the list (D231), though a pointer to it does not point into that variable.
func collRoot(e Expr) *Var {
	for {
		if b, ok := e.(*Builtin); ok && (b.Op == "list.ref" || b.Op == "list.refUnchecked") {
			e = b.Args[0]
			continue
		}
		switch x := e.(type) {
		case *FieldGet:
			e = x.X
			continue
		case *TupleGet:
			e = x.X
			continue
		case *Unwrap:
			e = x.X
			continue
		case *VariantCast:
			e = x.X
			continue
		}
		return rootVar(e)
	}
}

// lintCollectionParamCopies is D7's lost-change warning (lint_paramcopy.go)
// for a list, map or set parameter (D231): a function that changes its
// parameter changes its own copy, which the caller never sees. Silent where
// the copy is the point: the function returns the parameter's type, or
// takes its address. Unlike a struct's, a parameter kept somewhere is still
// reported — a lowered pattern keeps it before any change (std's SHA-1
// state was lost that way, silently); `var mine = xs` says the copy is meant.
func (c *Checker) lintCollectionParamCopies(prog *Program) {
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		for _, p := range fn.Params {
			if !p.Span.IsValid() || p.AddrTaken || !isMutableCollection(p.Type) {
				continue
			}
			if fn.Sig != nil && types.Identical(fn.Sig.Ret, p.Type) {
				continue
			}
			what := ""
			walkBlock(fn.Body, func(n any) {
				switch n := n.(type) {
				case *Builtin:
					if what == "" && collectionWrites[n.Op] && collRoot(n.Args[0]) == p {
						what = "changes it"
						if m, ok := strings.CutPrefix(n.Op, "list."); ok {
							what = "calls '" + m + "' on it"
						} else if m, ok := strings.CutPrefix(n.Op, "map."); ok {
							what = "calls '" + m + "' on it"
						}
					}
				case *Assign:
					if what == "" && collRoot(n.Target) == p {
						what = "sets an element of it"
					}
				case *Call:
					if what == "" && n.Fn.Receiver != nil && n.Fn.WritesSelf && len(n.Args) > 0 {
						if a, ok := n.Args[0].(*AddrOf); ok && collRoot(a.X) == p {
							what = "calls '" + n.Fn.Display + "', which changes it"
						}
					}
				}
			})
			if what == "" {
				continue
			}
			c.warnLint("param_copy", p.Span, nil, "'%s' is a copy of the caller's '%s' (D231): this function %s, and the caller never sees the change; take '%s: *%s' (and pass '&x'), or return the changed value",
				p.Name, p.Type, what, p.Name, p.Type)
		}
	}
}

// lintChangedCopies reports a local that copies a collection out of a field
// of `this` or from behind a pointer and then changes it, when the changed
// copy is never used as a whole afterwards (stored, passed on, returned,
// pointed at): the change was meant for the field, as it reached it when a
// collection was a shared reference (D231's migration hazard — std's
// inflate lost its output to `val out = this.out; out.push(b)`).
func (c *Checker) lintChangedCopies(prog *Program) {
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		from := map[*Var]Expr{} // a local → the place it copies
		changed := map[*Var]source.Span{}
		whole := map[*Var]bool{}
		walkBlock(fn.Body, func(n any) {
			switch n := n.(type) {
			case *VarDecl:
				if p := copiedPlace(n.Init); p != nil && isMutableCollection(n.Var.Type) && n.Var.Span.IsValid() {
					from[n.Var] = p
				}
			case *Builtin:
				if !collectionWrites[n.Op] {
					break
				}
				if v := collRoot(n.Args[0]); v != nil {
					if _, ok := changed[v]; !ok {
						changed[v] = n.Span
					}
				}
			case *Assign:
				// an element set (`r.set(i, v)`) is a change to r
				if b, isElem := n.Target.(*Builtin); isElem {
					if v := collRoot(n.Target); v != nil {
						if _, ok := changed[v]; !ok {
							changed[v] = b.Span
						}
					}
				}
			case *Share:
				if v := heldIn(n.X); v != nil {
					whole[v] = true
				}
			case *Return:
				if r, ok := n.Value.(*VarRef); ok {
					whole[r.Var] = true
				}
			case *AddrOf:
				if v := rootVar(n.X); v != nil {
					whole[v] = true
				}
			case *Call:
				args := n.Args
				if n.Fn.Receiver != nil && len(args) > 0 {
					if a, ok := args[0].(*AddrOf); ok && n.Fn.WritesSelf {
						if v := collRoot(a.X); v != nil {
							if _, ok := changed[v]; !ok {
								changed[v] = n.RecvSpan
							}
						}
					}
					args = args[1:]
				}
				for _, a := range args { // passed on, if only lent
					if r, ok := a.(*VarRef); ok {
						whole[r.Var] = true
					}
				}
			}
		})
		if fn.Body.Value != nil {
			if r, ok := fn.Body.Value.(*VarRef); ok {
				whole[r.Var] = true
			}
		}
		for v, place := range from {
			at, ok := changed[v]
			if !ok || whole[v] {
				continue
			}
			if b, isElem := unwrapped(place).(*Builtin); isElem {
				// an element read: `at` reaches the element itself (D231)
				text := spanText(b.Span)
				if text == "" {
					text = "the element"
				}
				c.warnLint("copy_changed", at, nil, "'%s' is a copy of the element '%s' (D231): this change does not reach the collection; reach the element with '%s' and change it through that",
					v.Name, text, strings.Replace(text, ".get(", ".at(", 1))
				continue
			}
			text := placeText(place)
			c.warnLint("copy_changed", at, nil, "'%s' is a copy of '%s' (D231): this change does not reach '%s'; change '%s' itself, or bind 'val %s = &%s' to change it through '%s'",
				v.Name, text, text, text, v.Name, text, v.Name)
		}
	}
}

// copiedPlace is the place a binding's value is a copy of — a field, an
// element, a value behind a pointer (`this.out`, `xs.at(i)`, `*p`), not a
// plain variable — looking through `?:`, `if`, blocks and the wrappers the
// value passes; nil when it is not such a copy.
func copiedPlace(e Expr) Expr {
	switch x := e.(type) {
	case *Share:
		if partOf(x.X) {
			return x.X
		}
	case *Elvis:
		return copiedPlace(x.L)
	case *SomeWrap:
		return copiedPlace(x.X)
	case *Unwrap:
		return copiedPlace(x.X)
	case *If:
		if x.Then != nil && x.Then.Value != nil {
			if p := copiedPlace(x.Then.Value); p != nil {
				return p
			}
		}
		if x.Else != nil && x.Else.Value != nil {
			return copiedPlace(x.Else.Value)
		}
	case *BlockExpr:
		if x.Block.Value != nil {
			return copiedPlace(x.Block.Value)
		}
	case *Let:
		return copiedPlace(x.Body)
	}
	return nil
}

// unwrapped is e without the Unwraps a narrowed read adds.
func unwrapped(e Expr) Expr {
	for {
		u, ok := e.(*Unwrap)
		if !ok {
			return e
		}
		e = u.X
	}
}

// partOf: e is a part of something rather than a whole variable — a field,
// an element, or what a pointer points at.
func partOf(e Expr) bool {
	switch x := e.(type) {
	case *FieldGet, *TupleGet, *Deref:
		return true
	case *Unwrap:
		return partOf(x.X)
	case *Builtin:
		switch x.Op {
		case "list.get", "list.ref", "list.refUnchecked", "map.get":
			return true
		}
	}
	return false
}

// constOwner is the `const struct` a place lies in through its fields, or
// nil: a change to a collection there would break D212's promise.
func constOwner(e Expr) *types.Struct {
	for {
		switch x := e.(type) {
		case *FieldGet:
			if st, ok := types.Underlying(x.X.Type()).(*types.Struct); ok && st.Const {
				return st
			}
			e = x.X
		case *TupleGet:
			e = x.X
		case *Unwrap:
			e = x.X
		case *Builtin:
			if x.Op != "list.ref" && x.Op != "list.refUnchecked" {
				return nil
			}
			e = x.Args[0]
		default:
			return nil
		}
	}
}

// ownForWrite is the statement before a loop that changes a collection's
// elements in place (`loop (&x in xs)`, `loop ((k, &v) in m)`): the
// collection is made unique where it lives and marked exposed, so the
// pointers the loop hands out reach it and a copy taken meanwhile does not
// share them (D231, D188's exposed mark).
func (f *fnCtx) ownForWrite(coll Expr, collAst ast.Expr, span source.Span) Stmt {
	f.writePlace(coll, collAst, span, "'&' in the loop", true)
	op := "list.own"
	if _, ok := coll.Type().(*types.List); !ok {
		op = "map.own"
	}
	return &ExprStmt{X: &Builtin{exprBase{types.TUnit}, op, []Expr{coll}, span}}
}
