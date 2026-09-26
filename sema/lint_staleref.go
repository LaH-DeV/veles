package sema

import (
	"reflect"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// Lint: a reference into a collection (`xs.ref(i)`,
// `m.ref(k)`, `loop (&x in xs)`) is a pointer into the collection's
// storage (D25). It stays valid as memory — the collector keeps the old
// buffer alive — but once the collection is grown or rearranged it points
// at stale storage or at a different element, so a write through it is
// lost or lands elsewhere. This lint warns when the collection is changed
// that way while such a reference is still used:
//
//	val p = xs.ref(0) ?: panic("…")
//	xs.push(y)          // may move the elements
//	p.n = 1             // written into the old buffer
//
// and when the body of `loop (&x in xs)` changes `xs` itself. It is
// syntactic: the collection is matched by its source text, so it sees
// `xs` and `this.items` but not two names for one list.

// layoutChangers are the methods that may move or reorder elements.
var layoutChangers = map[string]bool{
	"push": true, "insert": true, "addAll": true, "removeAt": true, "pop": true,
	"clear": true, "sort": true, "sortWith": true, "swap": true,
	"set": true, "remove": true, // maps: an insert may grow or compact
}

type liveRef struct {
	name string      // the variable holding the reference
	coll string      // source text of the collection expression
	span source.Span // where the reference was taken
}

// lintStaleRefs runs over the statements of one block. References taken
// in this block are checked against every later statement, descending
// into nested blocks; nested blocks check their own references when they
// are checked in turn.
func (f *fnCtx) lintStaleRefs(stmts []ast.Stmt) {
	var live []liveRef
	for i, s := range stmts {
		if len(live) > 0 {
			f.checkLayoutChanges(s, stmts[i+1:], live)
		}
		if vs, ok := s.(*ast.ValStmt); ok && vs.Binding.Name != nil && vs.Value != nil {
			if coll, ok := refSource(vs.Value); ok {
				live = append(live, liveRef{vs.Binding.Name.Name, coll, vs.Value.Span()})
			}
		}
		if ls, ok := s.(*ast.LoopStmt); ok && ls.Var != nil && hasRefBinding(ls.Var) {
			coll := srcText(ls.Iter)
			refs := []liveRef{{refBindingName(ls.Var), coll, ls.Var.Pos}}
			forEachCall(ls.Body, func(c *ast.CallExpr) {
				f.reportLayoutChange(c, refs, true)
			})
		}
	}
}

// refSource recognises `<coll>.ref(...)`, also as `<coll>.ref(...) ?: panic(…)`
// (D62), and returns the collection's source text.
func refSource(e ast.Expr) (string, bool) {
	if el, isElvis := e.(*ast.ElvisExpr); isElvis {
		e = el.L
	}
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	m, ok := call.Fun.(*ast.MemberExpr)
	if !ok || m.Name.Name != "ref" {
		return "", false
	}
	coll := srcText(m.X)
	return coll, coll != ""
}

// refBindingName is the `&name` of a loop binding (the value part of a
// `(k, &v)` tuple).
func refBindingName(b *ast.Binding) string {
	if b.Ref && b.Name != nil {
		return b.Name.Name
	}
	for i := range b.Tuple {
		if n := refBindingName(&b.Tuple[i]); n != "" {
			return n
		}
	}
	return ""
}

// checkLayoutChanges reports a layout-changing call inside s on a
// collection that a live reference points into, when that reference is
// still mentioned in the statements after s.
func (f *fnCtx) checkLayoutChanges(s ast.Stmt, rest []ast.Stmt, live []liveRef) {
	var stillUsed []liveRef
	for _, r := range live {
		if mentionsName(rest, r.name) {
			stillUsed = append(stillUsed, r)
		}
	}
	if len(stillUsed) == 0 {
		return
	}
	forEachCall(s, func(c *ast.CallExpr) {
		f.reportLayoutChange(c, stillUsed, false)
	})
}

// reportLayoutChange warns when call is `<coll>.<changer>(...)` for the
// collection of one of the references.
func (f *fnCtx) reportLayoutChange(call *ast.CallExpr, refs []liveRef, inLoop bool) {
	m, ok := call.Fun.(*ast.MemberExpr)
	if !ok || !layoutChangers[m.Name.Name] {
		return
	}
	coll := srcText(m.X)
	for _, r := range refs {
		if r.coll != coll {
			continue
		}
		if inLoop {
			f.c.warnf(call.Pos, "'%s.%s' changes the collection while 'loop (%s in %s)' walks it by reference; '%s' may go stale and the loop may not end — collect the changes and apply them after the loop", coll, m.Name.Name, spanText(r.span), coll, r.name)
		} else {
			f.c.warnf(call.Pos, "'%s.%s' may move or reorder the elements while '%s' (a reference into '%s' taken at %s) is still used; a write through it would be lost — take the reference after this call", coll, m.Name.Name, r.name, coll, refLine(r.span))
		}
		return
	}
}

// ---------------------------------------------------------------------------
// reflective walks over the syntax tree

// forEachCall calls fn for every call expression under node.
func forEachCall(node any, fn func(*ast.CallExpr)) {
	walkAST(node, func(n any) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			fn(c)
		}
		return true
	})
}

// mentionsName reports whether any name expression under node is name.
func mentionsName(node any, name string) bool {
	found := false
	walkAST(node, func(n any) bool {
		if found {
			return false
		}
		if x, ok := n.(*ast.NameExpr); ok && x.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

// walkAST visits every syntax node reachable from node through struct
// fields, slices and pointers; visit returns false to stop descending.
func walkAST(node any, visit func(any) bool) {
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem())
			}
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			if v.Type().Elem().Kind() == reflect.Struct {
				if !visit(v.Interface()) {
					return
				}
			}
			walk(v.Elem())
		case reflect.Struct:
			if v.Type() == reflect.TypeOf(source.Span{}) {
				return
			}
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node))
}

// refLine is "line N" for a span, for the message.
func refLine(sp source.Span) string {
	if sp.File == nil {
		return "an earlier line"
	}
	line, _ := sp.File.Position(sp.Start)
	return "line " + itoa(line)
}

// spanText is the source under a span.
func spanText(sp source.Span) string {
	if sp.File == nil || sp.Start < 0 || sp.End > len(sp.File.Content) {
		return ""
	}
	return sp.File.Content[sp.Start:sp.End]
}
