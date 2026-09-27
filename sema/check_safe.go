package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// Assignment through `?.` (D5, v0.24): `x?.f = v` and `x?.f op= v` write
// only when `x` is present, and do nothing otherwise — Kotlin's and Swift's
// rule. The receiver `x` is lowered to a guarded place: a nullable variable
// or field after its null test (the payload in place), a nullable pointer
// (through it), or a collection element (`xs.at(i)?.n = 1`,
// `m.ref(k)?.n += 1`, a nullable pointer to the element, v0.27) — so the
// write lands where the value lives, never on a copy.

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
	var recv Expr
	if isPlaceSyntax(x) {
		// a nullable variable or field: test it, then write its payload
		lv, root := f.checkLValue(x, false)
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
		if _, isPtr := nt.Elem.(*types.Pointer); !isPtr {
			// a `val` struct stays immutable (D11); through a pointer the
			// binding does not matter (`val p = xs.ref(i); p?.n = 1`)
			if lv, _ = f.checkLValue(x, true); lv == nil {
				return nil, nil, nil, false
			}
			lv = f.narrowLValue(lv, x)
		}
		cond = &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, lv}, x.Span()}
		return nil, cond, &Unwrap{exprBase{nt.Elem}, lv}, true
	}
	recv = f.flattenNullable(f.checkExpr(x, nil))
	if types.IsInvalid(recv.Type()) {
		return nil, nil, nil, false
	}
	nt, isN := recv.Type().(*types.Nullable)
	if !isN {
		f.errorf(x.Span(), "'?.' on a non-nullable value of type '%s'; use '.'", recv.Type())
		return nil, nil, nil, false
	}
	// a temporary: writing through it reaches something only when it is a
	// reference (a pointer or a collection handle); an element read such
	// as `xs.at(i)` is a copy, and `xs.ref(i)` the pointer (D25, v0.27)
	if !isReferenceType(nt.Elem) {
		if _, isPtr := nt.Elem.(*types.Pointer); !isPtr {
			f.copyMutationHint(x, x.Span(), fmt.Sprintf("assigning through '?.' into a temporary value of type '%s' has no effect", nt.Elem))
			return nil, nil, nil, false
		}
	}
	tmp := f.newTemp(nt)
	pre = []Stmt{&VarDecl{Var: tmp, Init: recv}}
	cond = &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, ref(tmp)}, x.Span()}
	return pre, cond, &Unwrap{exprBase{nt.Elem}, ref(tmp)}, true
}

// Reading through `?.` (D70): a null after `?.` skips the rest of the
// postfix chain, as in Swift — `a?.b.c()` is `c`'s result made nullable,
// where Kotlin would want `a?.b?.c()`. Parentheses end a chain
// (`(a?.b).c` reads `.c` on the nullable), and so does an argument list:
// only the receiver spine is followed.

// safeBelow finds the `?.` member nearest the top of e's receiver spine,
// strictly below e itself, or nil when e is not such a chain. A receiver
// already bound (the chain being checked) ends the walk.
func (f *fnCtx) safeBelow(e ast.Expr) *ast.MemberExpr {
	var recv ast.Expr
	switch e := e.(type) {
	case *ast.MemberExpr:
		if e.Safe {
			return nil // `x?.name` itself: fieldAccess/methodCallOn handle it
		}
		recv = e.X
	case *ast.CallExpr:
		m, ok := e.Fun.(*ast.MemberExpr)
		if !ok || m.Safe {
			return nil
		}
		recv = m.X
	default:
		return nil
	}
	for {
		if _, bound := f.boundPlace[recv]; bound {
			return nil
		}
		switch r := recv.(type) {
		case *ast.MemberExpr:
			if r.Grouped {
				return nil
			}
			if r.Safe {
				return r
			}
			recv = r.X
		case *ast.CallExpr:
			m, ok := r.Fun.(*ast.MemberExpr)
			if r.Grouped || !ok {
				return nil
			}
			if m.Safe {
				return m
			}
			recv = m.X
		default:
			return nil
		}
	}
}

// safeChain checks `top`, a chain with the `?.` member s below its top:
// s's receiver is evaluated once; when it is null the chain is null (or
// nothing, for a unit call), otherwise the rest runs on its payload. A
// nullable variable or field is used in place, so a method that changes
// its receiver changes the stored value (as methodCallOn does for one step).
func (f *fnCtx) safeChain(top ast.Expr, s *ast.MemberExpr) Expr {
	recv := f.checkExpr(s.X, nil)
	if types.IsInvalid(recv.Type()) {
		return recv
	}
	recv = f.flattenNullable(recv)
	nt, ok := recv.Type().(*types.Nullable)
	if !ok {
		f.errorf(s.Pos, "'?.' on a non-nullable value of type '%s'; use '.'", recv.Type())
		return bad()
	}
	var tmp *Var
	payload := Expr(&Unwrap{exprBase{nt.Elem}, recv})
	test := recv
	if !isPlaceExpr(recv) {
		tmp = f.newTemp(nt)
		payload = &Unwrap{exprBase{nt.Elem}, &VarRef{exprBase{nt}, tmp}}
		test = &VarRef{exprBase{nt}, tmp}
	}
	if f.boundPlace == nil {
		f.boundPlace = map[ast.Expr]Expr{}
	}
	f.boundPlace[s.X] = payload
	s.Safe = false // checked as `.` for the duration; restored below
	inner := f.checkExpr(top, nil)
	s.Safe = true
	delete(f.boundPlace, s.X)
	if types.IsInvalid(inner.Type()) {
		return inner
	}
	notNull := &Unary{exprBase{types.TBool}, OpNot, &IsNull{exprBase{types.TBool}, test}, s.Pos}
	body := f.safeCallBranch(inner, notNull)
	if tmp == nil {
		return body
	}
	return &Let{exprBase{body.Type()}, tmp, recv, body}
}
