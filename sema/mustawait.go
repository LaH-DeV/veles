package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// A handle the block must await (D156 part 1). Outside a scope,
// `val h = async f()` starts f as a fail-fast child of the rest of its
// block — the with-task path of D100 part 2, so a failure, a panic or a
// cancellation cancels and joins it on the way out — and every normal way
// out of the block must have awaited it: falling off the end, `return`, a
// `break` or `continue` that leaves the block. The check is definite
// assignment's: "h was awaited" is a flow fact in the narrowing state, set
// by `await h`, `awaitAll(…, h, …)` or a race arm `await h`, so it joins
// and dies with control flow like any smart cast. h has no other use: a
// handle that is passed, stored, captured or cancelled could be awaited
// where the checker cannot see it, or never.

// awaitedHandle is one such handle while its block is being checked.
type awaitedHandle struct {
	v     *Var
	at    source.Span // the binding
	loops int         // how many loops enclose it: a break or continue to an outer one leaves its block
}

func awaitedFact(v *Var) place { return place{v: v, path: "$awaited"} }

// awaitedVal reports whether a `val` statement starts a task the block
// must await: one name, the whole value an `async` call, outside the
// scopes and gathers whose children need no await.
func (f *fnCtx) awaitedVal(s *ast.ValStmt) bool {
	call, ok := s.Value.(*ast.CallExpr)
	if !ok || !call.Async || s.Else != nil || s.Pattern != nil || len(f.scopes) > 0 || f.isGlobal {
		return false
	}
	if s.Binding.Name == nil || s.Binding.Tuple != nil || s.Kind == ast.BindConst {
		return false
	}
	if s.Kind == ast.BindVar {
		// checked as the `val` it should be, so nothing more is said
		f.errorf(s.Pos, "a task is bound with 'val' (it must then be awaited) or 'with' (it is cancelled when the block ends), not 'var': a handle is not reassigned (D156)")
	}
	return true
}

// valTaskRest is `val h = async f()` and the statements after it as the
// block form of `with h = async f()`, marked as one whose task must be
// awaited (withTask reads the mark).
func (f *fnCtx) valTaskRest(vs *ast.ValStmt, b *ast.Block, i int) ast.Stmt {
	rest := b.Stmts[i+1:]
	f.lintImmediateAwait(vs.Binding.Name.Name, vs.Value.(*ast.CallExpr), rest)
	end := source.Span{File: b.Pos.File, Start: b.Pos.End, End: b.Pos.End}
	body := &ast.Block{Stmts: rest, Pos: vs.Pos.To(end)}
	w := &ast.WithExpr{Bindings: []ast.WithBinding{{Name: *vs.Binding.Name, Value: vs.Value}}, Body: body, Pos: vs.Pos.To(end)}
	if f.awaitWiths == nil {
		f.awaitWiths = map[*ast.WithExpr]*ast.ValStmt{}
	}
	f.awaitWiths[w] = vs
	return &ast.ExprStmt{X: w}
}

// beginAwaited registers v, bound at `at`, as a handle its block must
// await; endAwaited checks the end of the block, which the body reached
// with type bodyT.
func (f *fnCtx) beginAwaited(v *Var, at source.Span) {
	h := &awaitedHandle{v: v, at: at, loops: len(f.loops)}
	f.awaited = append(f.awaited, h)
	if f.c.awaitedVars == nil {
		f.c.awaitedVars = map[*Var]*awaitedHandle{}
	}
	f.c.awaitedVars[v] = h
	delete(f.narrow, awaitedFact(v))
}

func (f *fnCtx) endAwaited(v *Var, bodyT types.Type) {
	h := f.awaited[len(f.awaited)-1]
	f.awaited = f.awaited[:len(f.awaited)-1]
	if _, done := f.narrow[awaitedFact(v)]; !done && !types.IsNever(bodyT) && !types.IsInvalid(bodyT) {
		f.errorf(h.at, "'%s' is not awaited on every path to the end of its block: a task started with 'val' is awaited before the block ends; await it, or start it with 'with %s = async …' to have it cancelled when the block ends (D156)", v.Name, v.Name)
	}
	delete(f.narrow, awaitedFact(v))
}

// refAwaited records the hover of the `val` keyword: what the block must
// do with the handle.
func (f *fnCtx) refAwaited(vs *ast.ValStmt, v *Var, w *ast.WithExpr) {
	if f.c.index == nil || !vs.Pos.IsValid() {
		return
	}
	kw := source.Span{File: vs.Pos.File, Start: vs.Pos.Start, End: vs.Pos.Start + len("val")}
	line, _ := vs.Pos.File.Position(w.Body.Pos.End - 1)
	doc := fmt.Sprintf("`%s` runs alongside the rest of this block, which must await it on every way out before line %d; a failure on the way out cancels it and waits for it to stop (D156).", v.Name, line)
	f.c.index.Refs = append(f.c.index.Refs, Ref{Span: kw, Kind: "keyword", Name: "val", Type: v.Type, Detail: "val " + v.Name + typeSuffix(v.Type), Doc: doc})
}

// leaveAwaited checks a `return` (loops < 0) or a `break`/`continue` to
// the loop at index loops: every handle declared inside what it leaves
// must have been awaited on the way.
func (f *fnCtx) leaveAwaited(at source.Span, loops int) {
	for _, h := range f.awaited {
		if loops >= 0 && h.loops <= loops {
			continue // declared outside the loop it leaves: still in its block
		}
		if _, done := f.narrow[awaitedFact(h.v)]; done {
			continue
		}
		line, _ := h.at.File.Position(h.at.Start)
		f.errorf(at, "'%s' is not awaited on this path (it was started at line %d); await it before leaving, or start it with 'with %s = async …' to have it cancelled when the block ends (D156)", h.v.Name, line, h.v.Name)
	}
}

// loopIndex is the index in f.loops of the loop lp.
func (f *fnCtx) loopIndex(lp *Loop) int {
	for i := len(f.loops) - 1; i >= 0; i-- {
		if f.loops[i].hir == lp {
			return i
		}
	}
	return len(f.loops)
}

// awaitedUse is a name naming v: allowed only as what `await` or
// `awaitAll` waits for, in the function that declared it.
func (f *fnCtx) awaitedUse(v *Var, e *ast.NameExpr) {
	if f.c.awaitedVars[v] == nil {
		return
	}
	if f.vars[v] && f.awaitOperand == ast.Expr(e) {
		f.narrow[awaitedFact(v)] = types.TUnit
		return
	}
	f.errorf(e.Pos, "'%s' must be awaited by this block, so it cannot be passed on, stored, captured or cancelled: await it here and use its value, start it with 'with %s = async …' to be able to cancel it, or in a 'scope' to hand it around (D156)", v.Name, v.Name)
}

// lintImmediateAwait (part 1b): the first thing the rest of the block
// evaluates is `await name`, so nothing runs alongside the task; calling
// the function is the same, without a task. Not in a `scope(on: e)` body,
// where the launch is what puts the call on e.
func (f *fnCtx) lintImmediateAwait(name string, call *ast.CallExpr, rest []ast.Stmt) {
	if len(rest) == 0 {
		return
	}
	if n := len(f.scopes); n > 0 && f.scopes[n-1].On != nil {
		return
	}
	aw := f.firstEvaluated(rest[0])
	if aw == nil {
		return
	}
	n, ok := aw.X.(*ast.NameExpr)
	if !ok || n.Name != name {
		return
	}
	if call.Block {
		f.warnf(aw.Pos, "this 'async { }' runs with nothing alongside it; write its statements here directly: a task is for running something while this block does something else (D156, D169)")
		return
	}
	plain := *call
	plain.Async = false
	text := srcText(&plain)
	if t := srcText(call); len(t) > len("async ") && t[:len("async ")] == "async " {
		text = t[len("async "):]
	}
	f.warnf(aw.Pos, "'%s' runs with nothing alongside it; call '%s' directly: a task is for running something while this block does something else (D156)", srcText(call.Fun), text)
}

// firstEvaluated is the `await` a statement evaluates before anything
// else, if it does: `val v = await h`, `return await h`, `x = await h`,
// `f(await h, …)`, `await h + 1`, `"${await h}"`.
func (f *fnCtx) firstEvaluated(s ast.Stmt) *ast.AwaitExpr {
	var x ast.Expr
	switch s := s.(type) {
	case *ast.ValStmt:
		x = s.Value
	case *ast.ExprStmt:
		x = s.X
	case *ast.ReturnStmt:
		x = s.Value
	case *ast.AssignStmt:
		x = s.Value
	}
	for x != nil {
		switch e := x.(type) {
		case *ast.AwaitExpr:
			return e
		case *ast.CallExpr:
			if e.Async || len(e.Args) == 0 {
				return nil
			}
			if _, plainName := e.Fun.(*ast.NameExpr); !plainName {
				m, isMember := e.Fun.(*ast.MemberExpr)
				if !isMember {
					return nil
				}
				// `io.println(…)`: a module is evaluated to nothing; any
				// other receiver is evaluated first
				n, isName := m.X.(*ast.NameExpr)
				if !isName {
					return nil
				}
				if sym := f.lookup(n.Name); sym == nil || sym.Kind != SymModule {
					return nil
				}
			}
			x = e.Args[0].Value
		case *ast.BinaryExpr:
			x = e.L
		case *ast.StringLit:
			// the constant text before the first value runs nothing
			x = nil
			for _, p := range e.Parts {
				if p.Expr != nil {
					x = p.Expr
					break
				}
			}
		default:
			return nil
		}
	}
	return nil
}
