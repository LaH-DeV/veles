package sema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D213: where a program allocates on the GC heap, for the editor's opt-in
// `heap` hints and `veles explain --alloc`. Each site is a decision the
// checker made and the code generator follows: a collection literal or a
// collection operation's result, string building, a closure with captures,
// boxing into a trait object, a local in a heap cell, a boxed variant, a
// task, a channel. A call of a function that allocates inside is not a
// site; that function's own sites are. Structs, tuples, arrays, enums and
// numbers live on the stack or inline, and constants are static data, so
// they are never sites. allocsite_test.go holds the list to the allocation
// calls of the generated code.

// AllocSite is one place that allocates.
type AllocSite struct {
	Span source.Span // the allocating expression or declaration, as written
	// Stmt is the statement it belongs to, where the generated code's line
	// table places it
	Stmt  source.Span
	Fun   string // the function it is in, as diagnostics name it
	What  string // what is allocated, and why
	Avoid string // how to avoid it, when there is a way; "" when not
	// Later: a mutable collection whose storage comes with its first write
	// (D187), not where it is written
	Later bool
	// Global: a module value's initializer, run once at start-up
	Global bool
}

// noteAlloc records where x, an allocating node with no span of its own,
// was written; AllocSites reads it.
func (c *Checker) noteAlloc(x Expr, sp source.Span) Expr {
	if c.prog != nil && c.prog.allocAt != nil && sp.IsValid() {
		c.prog.allocAt[x] = sp
	}
	return x
}

// AllocSites lists the allocation sites in the program's own functions (not
// the standard library's), in source order. A generic body's site is
// listed once, whatever number of instances share it.
func AllocSites(prog *Program) []AllocSite {
	lay := &types.Layout{}
	seen := map[string]bool{}
	var out []AllocSite
	add := func(s AllocSite) {
		key := s.Span.String() + s.What
		if !seen[key] {
			seen[key] = true
			out = append(out, s)
		}
	}
	var tests []source.Span
	for _, t := range prog.Tests {
		if t.tmpl != nil && t.tmpl.Decl != nil {
			tests = append(tests, t.tmpl.Decl.Pos) // the whole test, its lambdas too
		}
	}
	inTest := func(sp source.Span) bool {
		for _, t := range tests {
			if within(sp, t) {
				return true
			}
		}
		return false
	}
	for _, fn := range prog.Funcs {
		// the program's own code: not the standard library's, not what the
		// compiler writes (derived methods, an enum's tables) and not tests,
		// whose sites are mostly the messages of a failure
		if fn.Body == nil || fromStd(fn.Span) || fn.tmpl == nil && !fn.IsClosure || inTest(fn.Span) {
			continue
		}
		for _, p := range fn.Params {
			if p.AddrTaken && p.Span.IsValid() {
				add(AllocSite{Span: p.Span, Stmt: fn.Span, Fun: fn.Display, What: cellWhat(p), Avoid: cellAvoid})
			}
		}
		body := fn.Span // the declaration: a lambda's span is all of it, a function's its name
		if fn.tmpl != nil && fn.tmpl.Decl != nil {
			body = fn.tmpl.Decl.Pos
		}
		var stmt source.Span
		cells := map[*Var]bool{}
		// cell notes a local kept in a heap cell, wherever it is declared (a
		// binding, a pattern, a loop variable, the compiler's own temporary):
		// met at its declaration, at a reference, or where a lambda captures it
		cell := func(v *Var) {
			if v == nil || !v.AddrTaken || v.Captured || v.IsGlobal || v.IsParam || cells[v] {
				return
			}
			cells[v] = true
			switch {
			case v.Span.IsValid() && !fromStd(v.Span):
				add(AllocSite{Span: v.Span, Stmt: stmt, Fun: fn.Display, What: cellWhat(v), Avoid: cellAvoid})
			case stmt.IsValid():
				// a value a method keeps a pointer to, or one bound by a lowering ('try')
				add(AllocSite{Span: stmt, Stmt: stmt, Fun: fn.Display,
					What: fmt.Sprintf("a heap copy of a temporary '%s': a method called on it keeps a pointer to it", v.Type)})
			}
		}
		walkBlock(fn.Body, func(n any) {
			if v, ok := n.(blockValue); ok && v.block.ValuePos.IsValid() {
				stmt = v.block.ValuePos
				return
			}
			if s, ok := n.(Stmt); ok {
				if p := hirStmtPos(s); p.IsValid() {
					stmt = p
				}
				if d, ok := s.(*VarDecl); ok {
					cell(d.Var)
				}
				return
			}
			x, ok := n.(Expr)
			if !ok {
				return
			}
			switch e := x.(type) {
			case *VarRef:
				cell(e.Var)
			case *Closure:
				for _, v := range e.Captures {
					cell(v)
				}
			}
			if s, ok := allocSite(prog, lay, x); ok {
				if !s.Span.IsValid() {
					if !stmt.IsValid() {
						return
					}
					s.Span = stmt // made by the compiler for this statement
				}
				s.Stmt = stmt
				switch {
				case stmt.IsValid() && (fromStd(s.Span) || !within(s.Span, body)):
					// written elsewhere, made here: a field default of a type built
					// in this statement, or code inlined into it
					why := "a field default"
					if fromStd(s.Span) {
						why = "the standard library's code"
					}
					s.Span, s.Avoid = stmt, ""
					s.What += ", made here by " + why
				case fromStd(s.Span):
					return
				}
				s.Fun = fn.Display
				if !s.Stmt.IsValid() {
					s.Stmt = s.Span
				}
				add(s)
			}
		})
	}
	// a module value's initializer runs once, at start-up; a `const` is
	// static data and allocates nothing
	for _, g := range prog.Globals {
		if g.Init == nil || g.Const != nil || fromStd(g.Span) {
			continue
		}
		walkExpr(g.Init, func(n any) {
			if x, ok := n.(Expr); ok {
				if s, ok := allocSite(prog, lay, x); ok && (!s.Span.IsValid() || !fromStd(s.Span)) {
					if !s.Span.IsValid() {
						s.Span = g.Span
					}
					s.Stmt, s.Fun, s.Global = g.Span, g.Display, true
					add(s)
				}
			}
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Span, out[j].Span
		if a.File != b.File {
			return a.File.Path < b.File.Path
		}
		return a.Start < b.Start
	})
	return out
}

// within reports whether inner lies inside outer.
func within(inner, outer source.Span) bool {
	return inner.File == outer.File && inner.Start >= outer.Start && inner.End <= outer.End
}

// hirStmtPos is where a statement was written, as the line table has it.
func hirStmtPos(s Stmt) source.Span {
	switch s := s.(type) {
	case *VarDecl:
		return s.Pos
	case *Assign:
		return s.Pos
	case *ExprStmt:
		return s.Pos
	case *Return:
		return s.Pos
	case *Loop:
		return s.Pos
	case *With:
		return s.Pos
	}
	return source.Span{}
}

// cellAvoid: a cell made for '&x' goes when the function takes the value
// and returns the new one; one made for a capture, when the closure gets
// the value as a parameter
const cellAvoid = "pass the value instead of '&' or a capture, and return the new one if it changes"

func cellWhat(v *Var) string {
	return fmt.Sprintf("'%s' lives in a heap cell, shared with a closure that captures it or a pointer to it", v.Name)
}

// allocSite says whether x allocates, and what.
func allocSite(prog *Program, lay *types.Layout, x Expr) (AllocSite, bool) {
	at := prog.allocAt[x]
	switch e := x.(type) {
	case *StringConcat:
		if !concatAllocates(e) || !at.IsValid() {
			return AllocSite{}, false
		}
		return AllocSite{Span: at, What: "a new string: the pieces are copied into one", Avoid: "in a loop, collect the pieces in a StringBuilder"}, true
	case *ListLit:
		lt, ok := e.Type().(*types.List)
		if !ok {
			return AllocSite{}, false // an array literal: inline (D121)
		}
		if len(e.Elems) == 0 && !lt.Mutable {
			return AllocSite{}, false // an empty immutable list is one shared constant (D187)
		}
		if len(e.Elems) == 0 {
			return AllocSite{Span: at, What: fmt.Sprintf("a '%s': its storage comes with the first write (D187)", lt), Avoid: "'reserve(n)' sizes it once when the count is known", Later: true}, true
		}
		return AllocSite{Span: at, What: fmt.Sprintf("a new '%s' of %d %s", lt, len(e.Elems), plural(len(e.Elems), "element")), Avoid: constAvoid(e.Elems)}, true
	case *MapLit:
		mt := e.Type().(*types.Map)
		if len(e.Entries) == 0 && !mt.Mutable {
			return AllocSite{}, false
		}
		if len(e.Entries) == 0 {
			return AllocSite{Span: at, What: fmt.Sprintf("a '%s': its storage comes with the first write (D187)", mt), Avoid: "'reserve(n)' sizes it once when the count is known", Later: true}, true
		}
		return AllocSite{Span: at, What: fmt.Sprintf("a new '%s' with %d %s", mt, len(e.Entries), plural(len(e.Entries), "key"))}, true
	case *Closure:
		if len(e.Captures) == 0 {
			return AllocSite{}, false
		}
		names := make([]string, len(e.Captures))
		for i, v := range e.Captures {
			names[i] = "'" + v.Name + "'"
		}
		return AllocSite{Span: at, What: "a closure capturing " + strings.Join(names, ", "), Avoid: "pass " + strings.Join(names, ", ") + " as a parameter, or use a named function"}, true
	case *Box:
		return AllocSite{Span: at, What: fmt.Sprintf("the '%s' copied to the heap as a '%s' object (D9)", e.X.Type(), e.Trait.Name), Avoid: fmt.Sprintf("a type parameter '<T: %s>' keeps the value where it is", e.Trait.Name)}, true
	case *MakeVariant:
		if !lay.VariantBoxed(e.Variant) {
			return AllocSite{}, false
		}
		why := "an 'indirect' variant"
		if !templateOf(e.Variant).Indirect {
			why = "a variant of a tree, larger than a word"
		}
		return AllocSite{Span: at, What: fmt.Sprintf("'%s' in a box: %s (D198)", e.Variant.Name, why)}, true
	case *AddrOf:
		if hirPlace(e.X) {
			return AllocSite{}, false // '&' of a variable: the variable's cell is the site
		}
		if !at.IsValid() {
			// the compiler's own '&': a method that works on its receiver in
			// place, called on a temporary; placed at its statement
			return AllocSite{What: fmt.Sprintf("a heap copy of a temporary '%s', for a method to work on in place", e.X.Type()),
				Avoid: "bind the value to a 'var' first if the method is called on it again"}, true
		}
		return AllocSite{Span: at, What: fmt.Sprintf("a heap copy of the '%s', for '&' to point at (D10)", e.X.Type())}, true
	case *Launch:
		return AllocSite{Span: at, What: "a task: its frame and its handle"}, true
	case *Builtin:
		what, ok := builtinAlloc(e)
		if !ok || !e.Span.IsValid() {
			return AllocSite{}, false
		}
		s := AllocSite{Span: e.Span, What: what}
		if strings.HasSuffix(e.Op, ".new") && e.Op != "chan.new" {
			s.Later = true
			s.Avoid = "'reserve(n)' sizes it once when the count is known"
		}
		return s, true
	case *Call:
		if e.Fn == nil || !fromStd(e.Fn.Span) || !e.Span.IsValid() || !isCollection(e.Type()) || len(e.Args) == 0 {
			return AllocSite{}, false
		}
		recv := e.Args[0].Type() // a method's receiver, passed by address
		if p, ok := recv.(*types.Pointer); ok {
			recv = p.Elem
		}
		if !isCollection(recv) && !types.IsString(recv) {
			return AllocSite{}, false
		}
		name := e.Fn.Display
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		return AllocSite{Span: e.Span, What: fmt.Sprintf("a new '%s' from '%s'", e.Type(), name)}, true
	}
	return AllocSite{}, false
}

// concatAllocates mirrors the code generator: a literal "" adds nothing,
// and a single piece left is the result itself unless it is a number
// formatted into a temporary buffer.
func concatAllocates(e *StringConcat) bool {
	var parts []Expr
	for _, p := range e.Parts {
		if c, ok := p.(*StringConst); ok && c.Value == "" {
			continue
		}
		parts = append(parts, p)
	}
	switch len(parts) {
	case 0:
		return false
	case 1:
		ts, ok := parts[0].(*ToString)
		return ok && types.IsInteger(ts.X.Type())
	}
	return true
}

// builtinAlloc names what a runtime operation allocates: a new collection
// (or the header of a mutable one), the elements of a string or array, a
// channel.
func builtinAlloc(e *Builtin) (string, bool) {
	switch e.Op {
	case "list.copy", "list.slice", "list.sortedNative", "map.copy", "map.keys", "map.values", "map.entries",
		"array.toList", "string.bytes", "string.chars", "string.split":
		return fmt.Sprintf("a new '%s' from '%s'", e.Type(), e.Op[strings.Index(e.Op, ".")+1:]), true
	case "set.new":
		if st, ok := e.Type().(*types.Set); ok && st.Mutable {
			return fmt.Sprintf("a '%s': its storage comes with the first write (D187)", st), true
		}
	case "chan.new":
		return fmt.Sprintf("a '%s': its buffer and wait queues", e.Type()), true
	}
	return "", false
}

// hirPlace mirrors the code generator's isPlace: what '&' points at
// where it is, rather than at a heap copy.
func hirPlace(e Expr) bool {
	switch e := e.(type) {
	case *VarRef, *Deref, *Downcast:
		return true
	case *FieldGet:
		return hirPlace(e.X)
	case *TupleGet:
		return hirPlace(e.X)
	case *Unwrap:
		return hirPlace(e.X)
	case *VariantCast:
		return hirPlace(e.X)
	case *ConstTable:
		_, isArray := e.Value.(*CArray)
		return isArray
	case *Builtin:
		return e.Op == "list.ref" || e.Op == "list.refUnchecked"
	}
	return false
}

func isCollection(t types.Type) bool {
	switch t.(type) {
	case *types.List, *types.Map, *types.Set:
		return true
	}
	return false
}

// constAvoid suggests a constant for a literal of constants.
func constAvoid(elems []Expr) string {
	for _, e := range elems {
		switch e.(type) {
		case *IntConst, *FloatConst, *StringConst, *BoolConst:
		default:
			return ""
		}
	}
	return "a list of constants can be a module 'const', made once as static data"
}
