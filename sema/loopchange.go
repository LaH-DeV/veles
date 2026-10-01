package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// D102: a collection cannot change while a loop walks it. What the
// compiler sees — a call on the loop's own collection that changes its
// length or order — is refused (refuseLoopChanges); a change through
// another path is caught when the loop next steps, by the modification
// count the runtime keeps on lists and maps (modsGuard).

// modsGuard is the count read before the loop and the check made at the
// start of every step: a panic at the loop's line when they differ.
func (f *fnCtx) modsGuard(coll *Var, isMap bool, s *ast.LoopStmt) (decl, check Stmt) {
	op := "list."
	if isMap {
		op = "map."
	}
	saved := f.newTemp(types.TI64)
	collRef := &VarRef{exprBase{coll.Type}, coll}
	decl = &VarDecl{Var: saved, Init: &Builtin{exprBase{types.TI64}, op + "mods", []Expr{collRef}, s.Pos}}
	msg := &StringConst{exprBase{types.TString}, "'" + srcText(s.Iter) + "' changed while a loop walked it"}
	check = &ExprStmt{X: &Builtin{exprBase{types.TUnit}, op + "checkMods", []Expr{collRef, &VarRef{exprBase{types.TI64}, saved}, msg}, s.Pos}}
	return decl, check
}

// mutableColl: a collection D102 guards — one whose length or order a
// call can change.
func mutableColl(t types.Type) bool {
	switch t := t.(type) {
	case *types.List:
		return t.Mutable
	case *types.Map:
		return t.Mutable
	case *types.Set:
		return t.Mutable
	case *types.Struct:
		return t.Module == "std.prelude" && t.Name == "Deque"
	}
	return false
}

// structuralCalls are, per kind of collection, the built-in and prelude
// methods that change its length or order.
var structuralCalls = map[string]map[string]bool{
	"list":  {"push": true, "pop": true, "clear": true, "insert": true, "removeAt": true, "addAll": true, "sort": true, "sortWith": true, "swap": true},
	"map":   {"remove": true, "clear": true, "getOrPut": true, "set": true},
	"set":   {"add": true, "remove": true, "clear": true},
	"deque": {"addFirst": true, "addLast": true, "removeFirst": true, "removeLast": true, "clear": true},
}

// refuseLoopChanges reports a call in the body of `loop (… in c)` that
// changes c's length or order, c being named by a local, a parameter or a
// `this.f` path. A lambda in the body counts when it is called there — as
// an argument it is.
func (f *fnCtx) refuseLoopChanges(s *ast.LoopStmt, t types.Type) {
	if !mutableColl(t) || !isPlacePath(s.Iter) {
		return
	}
	kind, copyCall := "list", ".toList()"
	switch t.(type) {
	case *types.Map:
		kind, copyCall = "map", ".toMap()"
	case *types.Set:
		kind, copyCall = "set", ".toSet()"
	case *types.Struct:
		kind = "deque"
	}
	coll := srcText(s.Iter)
	loopKey := ""
	if kind == "map" && s.Var != nil && len(s.Var.Tuple) == 2 && s.Var.Tuple[0].Name != nil {
		loopKey = s.Var.Tuple[0].Name.Name
	}
	called := map[*ast.LambdaExpr]bool{}
	walkAST(s.Body, func(n any) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if l, isL := c.Fun.(*ast.LambdaExpr); isL {
				called[l] = true
			}
			for _, a := range c.Args {
				if l, isL := a.Value.(*ast.LambdaExpr); isL {
					called[l] = true
				}
			}
		}
		return true
	})
	line, _ := s.Pos.File.Position(s.Pos.Start)
	walkAST(s.Body, func(n any) bool {
		switch n := n.(type) {
		case *ast.LambdaExpr:
			return called[n] // stored, not run here
		case *ast.CallExpr:
			m, ok := n.Fun.(*ast.MemberExpr)
			if !ok || m.Safe || srcText(m.X) != coll {
				return true
			}
			name := m.Name.Name
			if !structuralCalls[kind][name] && !(kind != "deque" && f.extendsMutable(t, name)) {
				return true
			}
			if kind == "map" && name == "set" && len(n.Args) > 0 {
				if k, isName := n.Args[0].Value.(*ast.NameExpr); isName && loopKey != "" && k.Name == loopKey {
					return true // replaces the value of the key being visited
				}
			}
			fix := fixReplace("Loop over a copy", s.Iter.Span(), coll+copyCall)
			f.c.errorFix(n.Pos, fix, "'%s.%s' changes '%s' while the loop at line %d walks it; loop over a copy ('%s%s'), or collect the changes and apply them after the loop (D102)", coll, name, coll, line, coll, copyCall)
		}
		return true
	})
}

// isPlacePath: a local, a parameter, `this`, or a field path from one.
func isPlacePath(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.NameExpr:
		return true
	case *ast.SelfExpr:
		return true
	case *ast.MemberExpr:
		return !e.Safe && isPlacePath(e.X)
	}
	return false
}

// extendsMutable: name is a method of an `extend` block on the mutable
// collection type itself — it may change the collection in any way.
func (f *fnCtx) extendsMutable(t types.Type, name string) bool {
	for _, ext := range f.c.extends {
		if _, ok := ext.Methods[name]; !ok {
			continue
		}
		switch et := ext.Target.(type) {
		case *types.List:
			if et.Mutable && unify(ext.Target, t, map[*types.TypeParam]types.Type{}) {
				return true
			}
		case *types.Map:
			if et.Mutable && unify(ext.Target, t, map[*types.TypeParam]types.Type{}) {
				return true
			}
		case *types.Set:
			if et.Mutable && unify(ext.Target, t, map[*types.TypeParam]types.Type{}) {
				return true
			}
		}
	}
	return false
}
