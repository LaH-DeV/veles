package sema

import (
	"fmt"

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

// blockSnapshot holds D237 for an `async { … }` block: the block is made
// as its task starts, so a `val` holding a collection that it captures is
// copied into a snapshot `val` of the maker at that moment and captured
// instead (Rust's `async move`); the maker's own variable is not captured,
// so it keeps its place in the frame. A block that changes what it
// captures is refused already (D35, D231), so the copy is only read. The
// receiver is copied too when it holds a collection: the block reads
// `this` through a pointer to the copy. It returns what the block
// captures: the snapshot, or outer itself.
func (f *fnCtx) blockSnapshot(outer *Var) *Var {
	if outer.IsGlobal || outer.CapCopy || outer.SelfCopy {
		return outer
	}
	parent := f.parent
	if outer.IsSelf { // the receiver: a pointer variable, whatever its binding
		pt, ok := outer.Type.(*types.Pointer)
		if !ok || !holdsCollection(pt.Elem, map[types.Type]bool{}) {
			return outer
		}
		val := parent.newVar("this", pt.Elem, false, source.Span{})
		val.Span = outer.Span
		val.CapCopy = true
		val.heapPromote() // its address is captured
		ptr := parent.newVar("this", pt, false, source.Span{})
		ptr.Span = outer.Span
		ptr.SelfCopy = true
		*f.blockCopies = append(*f.blockCopies,
			&VarDecl{Var: val, Init: &Deref{exprBase{pt.Elem}, &VarRef{exprBase{pt}, outer}}},
			&VarDecl{Var: ptr, Init: &AddrOf{exprBase{pt}, &VarRef{exprBase{pt.Elem}, val}}})
		markUsed(outer)
		return ptr
	}
	if outer.Mutable || !holdsCollection(outer.Type, map[types.Type]bool{}) {
		return outer // a `var` is refused in a task (D35)
	}
	snap := parent.newVar(outer.Name, outer.Type, false, source.Span{})
	snap.Span = outer.Span
	snap.CapCopy = true
	*f.blockCopies = append(*f.blockCopies, &VarDecl{Var: snap, Init: &VarRef{exprBase{outer.Type}, outer}})
	markUsed(outer)
	return snap
}

// launchedValue is `async job()` on a function value (D103), checked once
// the receiver pass knows which methods change their receiver.
type launchedValue struct {
	maker *Func
	value Expr  // the function value, as the launch reads it
	call  *Call // the trampoline call the launch makes
	name  string
	// outlives: the task is handed back to the caller in a value (D111),
	// so it runs on after the function returns
	outlives bool
}

// checkLaunchedValues holds D237's other half: a function value started
// with `async` may not capture a collection the function that made it
// changes while the task may run — after the `async`, or anywhere in a
// loop around it. The value is followed to the closure it holds when it is
// bound in the launching function. When it is a parameter instead, the
// task normally ends before the function returns (its block waits for it),
// so the caller cannot run beside it — unless the task is handed back in a
// value (D111): then the parameter is a launched one, through any number
// of functions passing it on, and each call site that passes a closure is
// checked with the call as the launch.
func (c *Checker) checkLaunchedValues() {
	type heldParam struct {
		fn  *Func
		idx int
	}
	launchedParams := map[heldParam]bool{}
	paramIndex := func(fn *Func, v *Var) int {
		for i, p := range fn.Params {
			if p == v {
				return i
			}
		}
		return -1
	}
	for _, lv := range c.launchedValues {
		if clo := closureOf(lv.maker, lv.value); clo != nil {
			c.refuseChangedCaptures(lv.maker, clo, lv.call, lv.call.Span, func(v *Var, line int) string {
				return fmt.Sprintf("'%s' captures '%s', which %s changes at line %d while the task may run; copy it in with a capture list: '[%s] (…) => …' (D237)", lv.name, v.Name, lv.maker.Display, line, v.Name)
			})
			continue
		}
		if r, ok := lv.value.(*VarRef); ok && lv.outlives && r.Var.IsParam {
			if i := paramIndex(lv.maker, r.Var); i >= 0 {
				launchedParams[heldParam{lv.maker, i}] = true
			}
		}
	}
	c.launchedValues = nil
	if len(launchedParams) == 0 {
		return
	}
	// argIndex: where parameter i of fn sits among a call's arguments (a
	// method's receiver comes first)
	argIndex := func(call *Call, i int) int {
		if call.Fn.Receiver != nil {
			return i + 1
		}
		return i
	}
	funcs := c.funcs
	// a parameter handed on, unchanged, to a launched parameter is one too
	for changed := true; changed; {
		changed = false
		for _, fn := range funcs {
			if fn.Body == nil {
				continue
			}
			walkBlock(fn.Body, func(n any) {
				call, ok := n.(*Call)
				if !ok || call.Fn == nil {
					return
				}
				for i := range call.Fn.Params {
					if !launchedParams[heldParam{call.Fn, i}] {
						continue
					}
					j := argIndex(call, i)
					if j >= len(call.Args) {
						continue
					}
					if r, ok := call.Args[j].(*VarRef); ok && r.Var.IsParam {
						if k := paramIndex(fn, r.Var); k >= 0 && !launchedParams[heldParam{fn, k}] {
							launchedParams[heldParam{fn, k}] = true
							changed = true
						}
					}
				}
			})
		}
	}
	for _, fn := range funcs {
		if fn.Body == nil {
			continue
		}
		walkBlock(fn.Body, func(n any) {
			call, ok := n.(*Call)
			if !ok || call.Fn == nil {
				return
			}
			for i, p := range call.Fn.Params {
				if !launchedParams[heldParam{call.Fn, i}] {
					continue
				}
				j := argIndex(call, i)
				if j >= len(call.Args) {
					continue
				}
				clo := closureOf(fn, call.Args[j])
				if clo == nil {
					continue
				}
				callee, param := call.Fn.Display, p.Name
				c.refuseChangedCaptures(fn, clo, call, call.Span, func(v *Var, line int) string {
					return fmt.Sprintf("'%s' starts a task from its parameter '%s' and hands it back, and the lambda passed here captures '%s', which %s changes at line %d while that task may run; copy it in with a capture list: '[%s] (…) => …' (D237)", callee, param, v.Name, fn.Display, line, v.Name)
				})
			}
		})
	}
}

// refuseChangedCaptures reports, at at, the first collection clo captured
// that maker changes after launch (an expression in maker's body) or in a
// loop around it — while the task launch starts may run.
func (c *Checker) refuseChangedCaptures(maker *Func, clo *Closure, launch Expr, at source.Span, message func(v *Var, line int) string) {
	if maker.Body == nil {
		return
	}
	// the loops the launch sits in: a change anywhere in one of them may
	// run while an earlier round's task does
	var around []*Loop
	walkBlock(maker.Body, func(n any) {
		if l, ok := n.(*Loop); ok && containsNode(l.Body, launch) {
			around = append(around, l)
		}
	})
	inLoop := func(sp source.Span) bool {
		for _, l := range around {
			found := false
			walkBlock(l.Body, func(n any) {
				if s, ok := nodeSpan(n); ok && s == sp {
					found = true
				}
			})
			if found {
				return true
			}
		}
		return false
	}
	for _, v := range clo.Captures {
		if v.Mutable || v.IsSelf || v.IsGlobal || v.CapCopy || !holdsCollection(v.Type, map[types.Type]bool{}) {
			continue
		}
		var hit source.Span
		changes := func(e Expr, sp source.Span) {
			if hit.IsValid() || !sp.IsValid() || collRoot(e) != v {
				return
			}
			if (sp.File == at.File && sp.Start > at.Start) || inLoop(sp) {
				hit = sp
			}
		}
		walkBlock(maker.Body, func(n any) {
			switch n := n.(type) {
			case *Builtin:
				if collectionWrites[n.Op] && len(n.Args) > 0 {
					changes(n.Args[0], n.Span)
				}
			case *Assign:
				changes(n.Target, n.Pos)
			case *Call:
				if n.Fn != nil && n.Fn.Receiver != nil && n.Fn.WritesSelf && len(n.Args) > 0 {
					changes(n.Args[0], n.Span)
				}
				for _, a := range n.Args {
					if ad, ok := a.(*AddrOf); ok {
						changes(ad.X, n.Span)
					}
				}
			}
		})
		if hit.IsValid() {
			line, _ := hit.File.Position(hit.Start)
			c.errorf(at, "%s", message(v, line))
		}
	}
}

// closureOf is the closure a launched function value holds: the value
// itself, the closure a local of fn was bound to (through the block a
// capture list lowers to), or the one a field of a local struct was built
// with (`val h = Handler(run: () => …)`, then `async h.run()`).
func closureOf(fn *Func, value Expr) *Closure {
	unwrap := func(e Expr) *Closure {
		if b, ok := e.(*BlockExpr); ok && b.Block != nil {
			e = b.Block.Value
		}
		cl, _ := e.(*Closure)
		return cl
	}
	if cl := unwrap(value); cl != nil {
		return cl
	}
	if fg, ok := value.(*FieldGet); ok && fn.Body != nil {
		r, ok := fg.X.(*VarRef)
		if !ok || r.Var.Mutable {
			return nil // a `var` may have been given another value since
		}
		var found *Closure
		walkBlock(fn.Body, func(n any) {
			if d, ok := n.(*VarDecl); ok && d.Var == r.Var && d.Init != nil {
				if lit, ok := d.Init.(*StructLit); ok && fg.Index < len(lit.Fields) {
					found = unwrap(lit.Fields[fg.Index])
				}
			}
		})
		return found
	}
	r, ok := value.(*VarRef)
	if !ok || fn.Body == nil {
		return nil
	}
	var found *Closure
	walkBlock(fn.Body, func(n any) {
		if d, ok := n.(*VarDecl); ok && d.Var == r.Var && d.Init != nil {
			found = unwrap(d.Init)
		}
	})
	return found
}

// containsNode: target is somewhere in b.
func containsNode(b *Block, target Expr) bool {
	found := false
	walkBlock(b, func(n any) {
		if e, ok := n.(Expr); ok && e == target {
			found = true
		}
	})
	return found
}

// nodeSpan is the span a change node records.
func nodeSpan(n any) (source.Span, bool) {
	switch n := n.(type) {
	case *Builtin:
		return n.Span, true
	case *Assign:
		return n.Pos, true
	case *Call:
		return n.Span, true
	}
	return source.Span{}, false
}
