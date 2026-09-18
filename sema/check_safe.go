package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// Assignment through `?.` (D5, v0.24): `x?.f = v` and `x?.f op= v` write
// only when `x` is present, and do nothing otherwise — Kotlin's and Swift's
// rule. The receiver `x` is lowered to a guarded place: a nullable variable
// or field after its null test (the payload in place), a nullable pointer
// (through it), or a collection element (`xs.at(i)?.n = 1`,
// `m.get(k)?.n += 1`, via elemSafePlace) — so the write lands where the
// value lives, never on a copy.

// safeMemberOf finds the `?.` member in an assignment target's field chain
// (`x?.a.b = v` → the `x?.a` node), or nil when there is none.
func safeMemberOf(target ast.Expr) *ast.MemberExpr {
	for {
		m, ok := target.(*ast.MemberExpr)
		if !ok {
			return nil
		}
		if m.Safe {
			return m
		}
		target = m.X
	}
}

// safeAssign lowers `x?.rest = v`. The receiver of the safe member is
// bound to its place (f.boundPlace) and the member is checked as an
// ordinary one, so field lookup, privacy and mutability follow the usual
// path; the assignment is then guarded by the presence test.
func (f *fnCtx) safeAssign(s *ast.AssignStmt, safe *ast.MemberExpr) []Stmt {
	pre, cond, place, ok := f.safeReceiver(safe.X)
	if !ok {
		f.checkExpr(s.Value, nil)
		return nil
	}
	if f.boundPlace == nil {
		f.boundPlace = map[ast.Expr]Expr{}
	}
	f.boundPlace[safe.X] = place
	safe.Safe = false // checked as `.` for the duration; the AST is restored below
	defer func() {
		safe.Safe = true
		delete(f.boundPlace, safe.X)
	}()
	target, root := f.checkLValue(s.Target, true)
	if target == nil {
		f.checkExpr(s.Value, nil)
		return nil
	}
	var value Expr
	if s.Op == lexer.Assign {
		value = f.coerce(f.checkExpr(s.Value, target.Type()), target.Type(), s.Value.Span())
	} else {
		value = f.makeBinary(BinOpFromToken(s.Op), target, f.checkExprTo(s.Value, target.Type()), s.Pos)
	}
	if root != nil {
		f.invalidatePaths(root)
	} else if p, ok := f.placeOf(safe.X); ok {
		f.invalidatePlace(p)
	}
	body := &Block{Stmts: []Stmt{&Assign{Target: target, Value: value}}, Type: types.TUnit}
	stmts := append(pre, &ExprStmt{X: &If{exprBase{types.TUnit}, cond, body, nil}})
	return stmts
}

// safeReceiver lowers the left side of a `?.` write to a guarded place:
// pre runs first, cond says whether the value is present, place is it.
func (f *fnCtx) safeReceiver(x ast.Expr) (pre []Stmt, cond Expr, place Expr, ok bool) {
	// a collection element: `xs.at(i)`, `xs.first()`, `m.get(k)`, ...
	sp, recv := f.elemSafePlace(x)
	if sp != nil {
		if !sp.mutable {
			f.errorf(x.Span(), "cannot assign into an element of an immutable collection through '?.'; use MutableList / MutableMap (D25)")
			return nil, nil, nil, false
		}
		return sp.pre, sp.cond, sp.place, true
	}
	if recv == nil {
		if isPlaceSyntax(x) {
			// a nullable variable or field: test it, then write its payload
			lv, root := f.checkLValue(x, true) // a `val` struct stays immutable (D11)
			if lv == nil {
				return nil, nil, nil, false
			}
			markUsed(root) // the presence test reads it
			lv = f.narrowLValue(lv, x)
			nt, isN := lv.Type().(*types.Nullable)
			if !isN {
				f.errorf(x.Span(), "'?.' on a non-nullable value of type '%s'; use '.'", lv.Type())
				return nil, nil, nil, false
			}
			cond = &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, lv}, x.Span()}
			return nil, cond, &Unwrap{exprBase{nt.Elem}, lv}, true
		}
		recv = f.checkExpr(x, nil)
	}
	if types.IsInvalid(recv.Type()) {
		return nil, nil, nil, false
	}
	nt, isN := recv.Type().(*types.Nullable)
	if !isN {
		f.errorf(x.Span(), "'?.' on a non-nullable value of type '%s'; use '.'", recv.Type())
		return nil, nil, nil, false
	}
	// a temporary: writing through it reaches something only when it is a
	// reference (a pointer or a collection handle)
	if !isReferenceType(nt.Elem) {
		if _, isPtr := nt.Elem.(*types.Pointer); !isPtr {
			f.errorf(x.Span(), "assigning through '?.' into a temporary value of type '%s' has no effect; bind it to a variable first, or make the receiver a place (a variable, a field, 'xs.at(i)', 'm.get(k)')", nt.Elem)
			return nil, nil, nil, false
		}
	}
	tmp := f.newTemp(nt)
	pre = []Stmt{&VarDecl{Var: tmp, Init: recv}}
	cond = &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, ref(tmp)}, x.Span()}
	return pre, cond, &Unwrap{exprBase{nt.Elem}, ref(tmp)}, true
}
