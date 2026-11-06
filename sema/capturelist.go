package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D235: a capture list `[a, b = expr]` before a lambda's parameters copies
// the listed values when the lambda is made; what is not listed is shared
// with the maker, as every capture is. Each entry is a snapshot — a fresh
// `val` of the maker that nothing else can reach — and each call of the
// lambda starts from its own copy of that snapshot, a `val` of the call the
// body may change in place as it may a parameter (D231): a change lasts
// for the call, and the next call starts from the snapshot again (D235
// addendum). So the lambda expression lowers to
// `{ val a = a; val b = expr; <closure> }`, and the closure's body opens
// with `val a = <captured a>; val b = <captured b>`.

// captureSnapshots checks e's capture list in the maker f: the
// declarations of the snapshots, the snapshots, and where each is listed.
func (f *fnCtx) captureSnapshots(e *ast.LambdaExpr) ([]Stmt, []*Var, []source.Span) {
	var decls []Stmt
	var snaps []*Var
	var spans []source.Span
	seen := map[string]bool{}
	for _, c := range e.Captures {
		name := c.Name.Name
		var init Expr
		if c.Value == nil {
			sym := f.scope.Lookup(name)
			if sym == nil || sym.Var == nil {
				if sym == nil {
					f.errorf(c.Name.Pos, "'%s' in the capture list is not a variable in scope (D235)", name)
				} else {
					f.errorf(c.Name.Pos, "'%s' in the capture list is not a variable; write '%s = …' to copy a value into the lambda (D235)", name, name)
				}
				continue
			}
			init = f.checkExpr(&ast.NameExpr{Name: name, Pos: c.Name.Pos}, nil)
		} else {
			init = f.checkExpr(c.Value, nil)
		}
		if seen[name] {
			f.errorf(c.Name.Pos, "'%s' is listed twice in the capture list (D235)", name)
			continue
		}
		seen[name] = true
		if types.IsInvalid(init.Type()) {
			continue
		}
		// made with no span, so the index records nothing for it: the
		// editor sees the call's copy (callCopies)
		v := f.newVar(name, init.Type(), false, source.Span{})
		v.CapCopy = true
		v.Span = c.Name.Pos
		if c.Value == nil {
			// the copy of `[routes]` is the variable routes to the editor:
			// its uses go to routes' declaration, so a rename renames them too
			if r, ok := init.(*VarRef); ok {
				root := r.Var
				for root.Outer != nil {
					root = root.Outer
				}
				v.Span = root.Span
			}
		} else {
			v.InitText = srcText(c.Value) // for the hover
		}
		// at the entry, so the copy hint of a later change of the original
		// names the capture list as what shared it
		decls = append(decls, &VarDecl{Var: v, Init: init, Pos: c.Pos})
		snaps = append(snaps, v)
		spans = append(spans, c.Name.Pos)
	}
	return decls, snaps, spans
}

// callCopies declares, in the lambda l, each call's own copy of the
// snapshots under the listed names, and returns the copies and the
// statements that make them at the start of the body.
func (l *fnCtx) callCopies(snaps []*Var, spans []source.Span) ([]*Var, []Stmt) {
	var copies []*Var
	var init []Stmt
	for i, v := range snaps {
		captured := l.localVar(v)
		w := l.newVar(v.Name, v.Type, false, source.Span{})
		w.CapCopy = true
		w.Span = v.Span
		w.InitText = v.InitText
		if v.InitText != "" {
			l.c.refVar(spans[i], w) // `name = expr`: the name is declared here
		}
		if l.scope.LookupLocal(v.Name) == nil { // a parameter of that name hides it
			l.scope.Insert(&Symbol{Name: v.Name, Kind: SymLocal, Var: w, Span: spans[i]})
		}
		init = append(init, &VarDecl{Var: w, Init: &VarRef{exprBase{v.Type}, captured}})
		copies = append(copies, w)
	}
	return copies, init
}

// withSnapshots wraps the closure clo in the declarations of its
// snapshots, and warns about a listed name the body never reads.
func (f *fnCtx) withSnapshots(clo Expr, decls []Stmt, copies []*Var, spans []source.Span) Expr {
	for i, w := range copies {
		if !w.used {
			f.warnf(spans[i], "'%s' is copied into the lambda but the lambda never uses it; remove it from the capture list (D235)", w.Name)
		}
	}
	if len(decls) == 0 {
		return clo
	}
	return &BlockExpr{exprBase{clo.Type()}, &Block{Stmts: decls, Value: clo, Type: clo.Type()}}
}
