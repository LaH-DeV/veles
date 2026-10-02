package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D63: a mutable collection is accepted where its immutable form is
// expected only when it is moved — a fresh local of this function that has
// not escaped, at its last use — so no one else can see the storage that
// is from now on read-only. Anything else copies, explicitly.

// moveMethods are the methods that neither keep a reference to their
// receiver nor hand out a pointer into it. `ref` and `iter`
// are missing on purpose: a pointer or an iterator would outlive the move.
var moveMethods = map[string]bool{
	"push": true, "reserve": true, "pop": true, "set": true, "at": true, "atOrDefault": true,
	"len": true, "isEmpty": true, "clear": true, "insert": true, "removeAt": true, "addAll": true,
	"sort": true, "sortWith": true, "sortBy": true, "sortDescending": true, "sortByDescending": true,
	"reverse": true, "swap": true, "contains": true, "indexOf": true, "first": true, "last": true,
	"toList": true, "toMutable": true, "toMap": true, "toSet": true, "map": true, "filter": true,
	"fold": true, "forEach": true, "joinToString": true, "slice": true, "take": true, "drop": true,
	"sum": true, "min": true, "max": true, "any": true, "all": true, "count": true, "find": true,
	"sorted": true, "sortedWith": true, "sortedBy": true, "sortedDescending": true, "reversed": true,
	"distinct": true, "indices": true, "get": true, "remove": true, "add": true, "keys": true,
	"values": true, "entries": true, "containsKey": true, "getOrDefault": true,
	"byteAt": true, "byteAtUnchecked": true, "atUnchecked": true, "setUnchecked": true, "decodeUtf8": true, "isNotEmpty": true, "toString": true,
}

// copyMethod is the method that copies a mutable collection into its
// immutable form.
func copyMethod(t types.Type) string {
	switch t.(type) {
	case *types.Map:
		return "toMap"
	case *types.Set:
		return "toSet"
	}
	return "toList"
}

// convertMutable handles a mutable collection where its immutable form is
// wanted: a move when movable, otherwise an error with the copying fix.
func (f *fnCtx) convertMutable(x Expr, want types.Type, span source.Span) Expr {
	cast := &Cast{exprBase{want}, x}
	if span.File == nil {
		// compiler-written code (element-wise equality, derived bodies)
		// only reads what it is handed
		return cast
	}
	if f.movable(x, span) {
		return cast
	}
	name := copyMethod(want)
	text := spanText(span)
	var fix *source.Fix
	if text != "" {
		if !isSimpleOperand(text) {
			text = "(" + text + ")"
		}
		fix = fixReplace("Copy with '."+name+"()'", span, text+"."+name+"()")
	}
	f.c.errorFix(span, fix, "'%s' is a %s, and a %s never changes: copy it with '.%s()' (D63). Only a local built here that nothing else can see is passed without a copy, at its last use", srcTextSpan(span, "this value"), x.Type(), want, name)
	return cast
}

func srcTextSpan(span source.Span, fallback string) string {
	if t := spanText(span); t != "" {
		return t
	}
	return fallback
}

// isSimpleOperand reports whether text can take `.m()` without parentheses.
func isSimpleOperand(text string) bool {
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '(', r == ')', r == '"':
		default:
			return false
		}
	}
	return true
}

// movable reports whether x, converted at span, is a move (D63).
func (f *fnCtx) movable(x Expr, span source.Span) bool {
	ref, ok := x.(*VarRef)
	if !ok {
		return false
	}
	v := ref.Var
	// AddrTaken is not consulted: a method call on a `var` sets it, and an
	// explicit `&c` is a mention the walk below does not allow
	if v.IsParam || v.IsGlobal || v.Captured || v.IsSelf || !f.vars[v] || f.bodyAST == nil {
		return false
	}
	var decl *ast.ValStmt
	var declLoops []*ast.LoopStmt
	var conv *ast.NameExpr
	var convLoops []*ast.LoopStmt
	allowed := map[*ast.NameExpr]bool{}
	var uses []*ast.NameExpr
	escaped := false
	var loops []*ast.LoopStmt

	var visit func(n any) bool
	visit = func(n any) bool {
		if escaped {
			return false
		}
		switch n := n.(type) {
		case *ast.LambdaExpr:
			// any mention inside a lambda may run after the move
			walkAST(n.Body, func(m any) bool {
				if ne, ok := m.(*ast.NameExpr); ok && ne.Name == v.Name {
					escaped = true
				}
				return !escaped
			})
			return false
		case *ast.ValStmt:
			if n.Binding.Name != nil && n.Binding.Name.Pos == v.Span {
				decl = n
				declLoops = append([]*ast.LoopStmt(nil), loops...)
			}
		case *ast.LoopStmt:
			if ne, ok := n.Iter.(*ast.NameExpr); ok && ne.Name == v.Name {
				if hasRefBinding(n.Var) {
					escaped = true // pointers into it
					return false
				}
				allowed[ne] = true
			}
			walkAST(n.Iter, visit)
			if n.Cond != nil {
				walkAST(n.Cond, visit)
			}
			loops = append(loops, n)
			walkAST(n.Body, visit)
			loops = loops[:len(loops)-1]
			return false
		case *ast.CallExpr:
			if m, ok := n.Fun.(*ast.MemberExpr); ok && !m.Safe {
				if ne, ok := m.X.(*ast.NameExpr); ok && ne.Name == v.Name && moveMethods[m.Name.Name] {
					allowed[ne] = true
				}
			}
		case *ast.AssignStmt:
			if ne, ok := n.Target.(*ast.NameExpr); ok && ne.Name == v.Name {
				escaped = true // reassigned: which list is it now?
				return false
			}
		case *ast.NameExpr:
			if n.Name != v.Name || n.Pos.Start < v.Span.End {
				return true
			}
			if n.Pos == span {
				conv = n
				convLoops = append([]*ast.LoopStmt(nil), loops...)
				return true
			}
			uses = append(uses, n)
		}
		return true
	}
	walkAST(f.bodyAST, visit)
	if escaped || decl == nil || conv == nil {
		return false
	}
	// fresh: built here, not a handle to someone else's collection
	switch init := decl.Value.(type) {
	case *ast.ListLit, *ast.MapLit:
	case *ast.CallExpr:
		m, ok := init.Fun.(*ast.MemberExpr)
		if !ok || m.Name.Name != "toMutable" {
			return false
		}
	default:
		return false
	}
	for _, u := range uses {
		// a mention in return position is a move on a path of its own
		if !allowed[u] && !f.inReturnPosition(u) {
			return false // passed, stored or aliased somewhere
		}
	}
	if f.inReturnPosition(conv) {
		return true
	}
	// the last mention, and not inside a loop that repeats it
	for _, u := range uses {
		if u.Pos.Start > conv.Pos.Start {
			return false
		}
	}
	return len(convLoops) == len(declLoops)
}

// inReturnPosition reports whether the name is returned: `return c`, or
// the last expression of the function's body.
func (f *fnCtx) inReturnPosition(n *ast.NameExpr) bool {
	found := false
	walkAST(f.bodyAST, func(m any) bool {
		if found {
			return false
		}
		switch m := m.(type) {
		case *ast.LambdaExpr:
			return false
		case *ast.ReturnStmt:
			if m.Value == ast.Expr(n) {
				found = true
			}
		}
		return true
	})
	if found {
		return true
	}
	body := f.bodyAST
	if be, ok := body.(*ast.BlockExpr); ok {
		body = be.Block
	}
	switch b := body.(type) {
	case *ast.Block:
		if len(b.Stmts) > 0 {
			if es, ok := b.Stmts[len(b.Stmts)-1].(*ast.ExprStmt); ok && es.X == ast.Expr(n) {
				return true
			}
		}
	case ast.Expr:
		return b == ast.Expr(n)
	}
	return false
}
