package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `do { body } catch (e) { handler }` (D98): a block whose failed `try`s and
// `throw`s go to a handler in the same function, instead of leaving it.
//
// There is no new HIR and no new code generation. The block becomes a
// one-shot loop, the technique the eager collection operations use for a
// throwing function argument (lower_try.go): a failing `try` stores its error
// in a slot and breaks out of the loop, and the code after the loop tests the
// slot,
//
//	var err: E? = null
//	var value: T? = null
//	loop {
//	  ...                      // a failing `try`:  err = Some(e); break
//	  value = Some(<last expression>)
//	  break
//	}
//	if (err == null) value! else handler
//
// The loop is only in the lowering: the checker's loop stack does not see it,
// so a `break` or `continue` written in the block still names the loop the
// program put around the `do`, and `return` leaves the function.

// catchFrame is the `do` block being checked: where a failure goes, and the
// error types that can reach the handler.
type catchFrame struct {
	loop    *Loop
	errVal  *Var         // the slot holding the failure; its type is known when the block is done
	members []types.Type // the error types seen, in order of first appearance
	fixes   []func(u types.Type)
	pos     source.Span
	// `try chain catch (e) { }`: a `try` whose chain goes on past the Result is
	// what was written, so it needs no parentheses to say so
	quiet bool
	// an error was already reported for something inside the block that would
	// have reached the handler, so "nothing can fail" would be a second, wrong one
	poisoned bool
}

// add notes an error type (or the members of a union) a failure can carry.
func (fr *catchFrame) add(t types.Type) {
	for _, m := range types.UnionMembers(t) {
		known := false
		for _, k := range fr.members {
			if types.Identical(k, m) {
				known = true
				break
			}
		}
		if !known {
			fr.members = append(fr.members, m)
		}
	}
}

// fail is the failure branch of a `try` or `throw`: store the error, leave
// the loop. The nodes it builds cannot know the union's final type yet, so
// they are patched when the block is done.
func (fr *catchFrame) fail(val Expr, from types.Type) *Block {
	conv := &ErrorConvert{exprBase{}, val, from}
	wrap := &SomeWrap{exprBase{}, conv}
	slot := &VarRef{exprBase{}, fr.errVal}
	fr.fixes = append(fr.fixes, func(u types.Type) {
		conv.T = u
		wrap.T = &types.Nullable{Elem: u}
		slot.T = wrap.T
	})
	return &Block{Stmts: []Stmt{&Assign{Target: slot, Value: wrap}, &Break{Loop: fr.loop}}, Type: types.TNever}
}

// catchExpr checks `do { ... } catch (e) { ... }`, `r catch (e) { ... }` and
// `try chain catch (e) { ... }`.
func (f *fnCtx) catchExpr(e *ast.CatchExpr, want types.Type) Expr {
	quiet := false
	if e.X != nil {
		if _, isTry := e.X.(*ast.TryExpr); !isTry {
			// a Result-valued expression: `r catch (e) { h }` is what `??` with a
			// handler was
			return f.coalesceOf(e.X, nil, e.Handler, e.Pos, want, "catch")
		}
		// `try chain catch (e) { h }` is `do { try chain } catch (e) { h }`
		e = &ast.CatchExpr{Body: &ast.Block{Stmts: []ast.Stmt{&ast.ExprStmt{X: e.X}}, Pos: e.X.Span()}, Handler: e.Handler, Pos: e.Pos}
		quiet = true
	}
	f.c.nextLoop++
	lp := &Loop{ID: f.c.nextLoop, hasBreak: true}
	fr := &catchFrame{loop: lp, errVal: f.newTemp(types.TInvalid), pos: e.Pos, quiet: quiet}

	// the handler may run from anywhere in the block, so what the block
	// assigned or shrank is not known to it, and nothing the block proved
	// holds after it
	f.invalidateAssigned(e.Body)
	if astMayShrink(e.Body) {
		f.killMutableBounds()
	}
	saved := f.saveNarrow()
	prev := f.catching
	f.catching = fr
	body := f.checkBlock(e.Body, want, true)
	f.catching = prev
	f.restoreNarrow(saved)

	var u types.Type
	switch {
	case len(fr.members) > 0:
		u = types.MakeErrorUnion(fr.members...)
	case fr.poisoned:
		return &BlockExpr{exprBase{body.Type}, body}
	case f.neverInstance():
		// generic code whose failures come through a `throws E` it was
		// handed (`retry`'s `f`): in this instance E is Never, so nothing
		// reaches the handler, but the template's shape — a handler that
		// may fall through to the code after it — is kept, so that code is
		// not unreachable here
		u = types.TNever
	default:
		f.errorf(e.Pos, "nothing in this 'do' block can fail: no 'try' or 'throw' in it reaches the 'catch'; drop the 'do' and the 'catch' (D98)")
		return &BlockExpr{exprBase{body.Type}, body}
	}
	slotType := &types.Nullable{Elem: u}
	fr.errVal.Type = slotType
	for _, fix := range fr.fixes {
		fix(u)
	}

	// the body, ending in what carries its value out of the loop
	tb := body.Type
	stmts := append([]Stmt{}, body.Stmts...)
	var res *Var
	if body.Value != nil {
		if types.IsUnit(tb) {
			stmts = append(stmts, &ExprStmt{X: body.Value})
		} else if !types.IsNever(tb) {
			res = f.newTemp(&types.Nullable{Elem: tb})
			stmts = append(stmts, &Assign{Target: ref(res), Value: &SomeWrap{exprBase{res.Type}, body.Value}})
		}
	}
	if !types.IsNever(tb) {
		stmts = append(stmts, &Break{Loop: lp})
	}
	lp.Body = &Block{Stmts: stmts, Type: types.TUnit}

	// the handler sees the error as the union of everything that could fail
	errValue := &Unwrap{exprBase{u}, ref(fr.errVal)}
	handlerWant := tb
	if types.IsNever(tb) {
		handlerWant = want
	}
	hb := f.handlerBlock(e.Handler, errValue, handlerWant, false)
	f.restoreNarrow(saved)

	out := &Block{Stmts: []Stmt{&VarDecl{Var: fr.errVal, Init: &NullConst{exprBase{slotType}}}}}
	if res != nil {
		out.Stmts = append(out.Stmts, &VarDecl{Var: res, Init: &NullConst{exprBase{res.Type}}})
	}
	out.Stmts = append(out.Stmts, lp)
	result := tb
	if types.IsNever(tb) {
		// only a failure gets out of the loop: the handler is what follows
		result = hb.Type
		out.Stmts = append(out.Stmts, hb.Stmts...)
		out.Value = hb.Value
		out.Type = result
		return &BlockExpr{exprBase{result}, out}
	}
	var then *Block
	if res != nil {
		then = &Block{Value: &Unwrap{exprBase{tb}, ref(res)}, Type: tb}
	} else {
		then = &Block{Type: types.TUnit}
	}
	pick := &If{exprBase{result}, &IsNull{exprBase{types.TBool}, ref(fr.errVal)}, then, hb}
	if types.IsUnit(result) {
		out.Stmts = append(out.Stmts, &ExprStmt{X: pick})
	} else {
		out.Value = pick
	}
	out.Type = result
	return &BlockExpr{exprBase{result}, out}
}

// catchTry is `try x` inside a `do` block: on `Err` the error goes to the
// handler, on `Ok` the payload is the value.
func (f *fnCtx) catchTry(x Expr, rs *types.Sealed) Expr {
	fr := f.catching
	okT, errT := rs.TypeArgs[0], rs.TypeArgs[1]
	fr.add(errT)
	r := f.newTemp(rs)
	okV, errV := rs.Variants[0], rs.Variants[1]
	payload := &FieldGet{exprBase{okT}, &VariantCast{exprBase{okV}, ref(r), okV}, 0, okV.Fields[0].Name}
	errPayload := &FieldGet{exprBase{errV.Fields[0].Type}, &VariantCast{exprBase{errV}, ref(r), errV}, 0, errV.Fields[0].Name}
	fail := fr.fail(errPayload, errT)
	body := &If{exprBase{okT}, &VariantTest{exprBase{types.TBool}, ref(r), errV}, fail, &Block{Value: payload, Type: okT}}
	return &Let{exprBase{okT}, r, x, body}
}

// catchThrow is `throw e` inside a `do` block: the same road a failed `try`
// takes. It is typed Never, like every throw.
func (f *fnCtx) catchThrow(errv Expr) Expr {
	fr := f.catching
	et := errv.Type()
	fr.add(et)
	return &BlockExpr{exprBase{types.TNever}, fr.fail(errv, et)}
}
