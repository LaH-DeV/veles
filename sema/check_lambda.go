package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// Lambdas (D32) become separate functions taking an environment pointer.
// Variables of enclosing functions that a lambda uses are captured by
// reference: the outer variable is heap-promoted (AddrTaken) and the
// closure environment holds the cell's address, so both sides share it.

// owns reports whether v was declared in this function context.
func (f *fnCtx) owns(v *Var) bool {
	return f.vars[v]
}

// localVar resolves a variable found by name lookup to the variable this
// function should use, capturing across lambda boundaries as needed.
func (f *fnCtx) localVar(v *Var) *Var {
	if v.IsGlobal || f.owns(v) || f.parent == nil {
		return v
	}
	if inner, ok := f.captures[v]; ok {
		return inner
	}
	outer := f.parent.localVar(v)
	outer.AddrTaken = true
	inner := f.newVar(outer.Name, outer.Type, outer.Mutable, outer.Span)
	inner.Captured = true
	inner.ErrPoly = outer.ErrPoly
	inner.CapIndex = len(f.fn.CapVars)
	inner.Outer = outer
	f.fn.CapVars = append(f.fn.CapVars, inner)
	f.captureList = append(f.captureList, outer)
	f.captures[v] = inner
	// Smart casts on immutable outer bindings still hold inside (D5).
	if !outer.Mutable {
		for k, t := range f.parent.narrow {
			if k.v == outer && !isBoundsPath(k.path) {
				f.narrow[place{v: inner, path: k.path}] = t
			}
		}
	}
	return inner
}

// selfRef returns the receiver variable (a pointer to the receiver's
// place, D22), capturing it inside lambdas.
func (f *fnCtx) selfRef() *Var {
	if f.selfVar != nil {
		return f.selfVar
	}
	if f.parent == nil {
		return nil
	}
	outer := f.parent.selfRef()
	if outer == nil {
		return nil
	}
	inner := f.localVar(outer)
	inner.IsSelf = true
	f.selfVar = inner
	return inner
}

// lambdaExpr checks a lambda against an optional expected function type.
func (f *fnCtx) lambdaExpr(e *ast.LambdaExpr, want types.Type) Expr {
	var expected *types.Func
	if ft, ok := numericHint(want).(*types.Func); ok {
		expected = ft
	}
	// D37: a lambda with N parameters adapts to an expected single tuple
	// parameter of arity N.
	tupleAdapt := false
	if expected != nil && len(expected.Params) == 1 && len(e.Params) > 1 {
		if tt, ok := expected.Params[0].Type.(*types.Tuple); ok && len(tt.Elems) == len(e.Params) {
			tupleAdapt = true
		}
	}
	if expected != nil && !tupleAdapt && len(expected.Params) != len(e.Params) {
		f.errorf(e.Pos, "lambda takes %d parameter(s) but a function of %d is expected here", len(e.Params), len(expected.Params))
		return bad()
	}

	f.c.nextLambda++
	name := "lambda"
	if f.fn != nil {
		name = f.fn.Name + ".lambda" + fmt.Sprint(f.c.nextLambda)
	} else {
		name = "v_global.lambda" + fmt.Sprint(f.c.nextLambda)
	}
	fn := &Func{Name: name, Display: "lambda", IsClosure: true, Span: e.Pos, Sig: &types.Func{}}
	l := f.c.newFnCtx(fn, f.module, f.file, f.env, f.subst)
	l.bodyAST = e.Body
	l.parent = f
	l.scope = NewScope(f.scope)
	l.unsafe = f.unsafe
	l.isLambda = true

	// parameters
	var params []*Var
	var tupleParam *Var
	if tupleAdapt {
		tt := expected.Params[0].Type.(*types.Tuple)
		tupleParam = l.newVar("$tuple", tt, false, e.Pos)
		fn.Params = []*Var{tupleParam}
		fn.Sig.Params = []types.Param{{Name: "tuple", Type: tt}}
		var patternInit []Stmt
		for i, p := range e.Params {
			pt := tt.Elems[i]
			if p.Pattern != nil {
				// `((size, hash), files) => ...` over a pair: the element is
				// itself destructured (D37)
				v, parts := l.bindPattern(p.Pattern, pt, false)
				params = append(params, v)
				patternInit = append(patternInit, parts...)
				continue
			}
			if p.Type != nil {
				declared := f.resolve(p.Type)
				if !types.Identical(declared, pt) {
					f.errorf(p.Pos, "parameter '%s' is declared '%s' but the tuple element is '%s'", p.Name.Name, declared, pt)
				}
			}
			v := l.newVar(p.Name.Name, pt, false, p.Name.Pos)
			v.IsParam = true
			l.declareLocal(p.Name.Name, v, p.Name.Pos)
			params = append(params, v)
		}
		l.paramInit = patternInit
	} else {
		var patternInit []Stmt
		for i, p := range e.Params {
			if p.Pattern != nil {
				// `((size, hash), files) => ...`: the parameter is the whole
				// tuple, bound to a hidden variable and destructured into the
				// body's first statements (D37)
				var pt types.Type = types.TInvalid
				if expected != nil && !types.ContainsTypeParam(expected.Params[i].Type) {
					pt = expected.Params[i].Type
				} else {
					f.errorf(p.Pos, "cannot infer the type of this tuple parameter; a destructuring parameter needs a known function type")
				}
				v, parts := l.bindPattern(p.Pattern, pt, false)
				params = append(params, v)
				fn.Params = append(fn.Params, v)
				fn.Sig.Params = append(fn.Sig.Params, types.Param{Name: "tuple", Type: pt})
				patternInit = append(patternInit, parts...)
				continue
			}
			var pt types.Type
			if p.Type != nil {
				pt = f.resolve(p.Type)
				if expected != nil && !types.ContainsTypeParam(expected.Params[i].Type) && !types.Identical(pt, expected.Params[i].Type) {
					f.errorf(p.Pos, "parameter '%s' is declared '%s' but '%s' is expected here", p.Name.Name, pt, expected.Params[i].Type)
				}
			} else if expected != nil && !types.ContainsTypeParam(expected.Params[i].Type) {
				pt = expected.Params[i].Type
			} else {
				f.errorf(p.Name.Pos, "cannot infer the type of lambda parameter '%s'; annotate it: '(%s: T) => ...'", p.Name.Name, p.Name.Name)
				pt = types.TInvalid
			}
			v := l.newVar(p.Name.Name, pt, false, p.Name.Pos)
			v.IsParam = true
			l.declareLocal(p.Name.Name, v, p.Name.Pos)
			params = append(params, v)
			fn.Params = append(fn.Params, v)
			fn.Sig.Params = append(fn.Sig.Params, types.Param{Name: p.Name.Name, Type: pt})
		}
		l.paramInit = patternInit
	}

	// return type and effects
	if e.Ret != nil {
		l.retType = f.resolve(e.Ret)
	} else if expected != nil && expected.Ret != nil && !types.ContainsTypeParam(expected.Ret) {
		l.retType = expected.Ret
	}
	if expected != nil && expected.Effects.Throws && !types.ContainsTypeParam(expected.Effects.Error) {
		l.throws = true
		l.errType = expected.Effects.Error
	} else {
		// no expected error, or `throws E` with E still to be inferred from
		// this very lambda (a generic higher-order function): infer it
		l.throws = true // inferred: any error raised in the body makes it throw
		l.inferThrows = true
	}
	if expected != nil && expected.Effects.Suspends {
		fn.Sig.Effects.Suspends = true
	}

	// body
	var body *Block
	if be, ok := e.Body.(*ast.BlockExpr); ok {
		body = l.checkBlock(be.Block, l.retType, true)
		if body.Value != nil {
			if l.retType == nil {
				l.retType = body.Value.Type()
			}
			body.Stmts = append(body.Stmts, &Return{Value: body.Value})
			body.Value = nil
		} else if l.retType == nil {
			if types.IsNever(body.Type) && l.inferredRet != nil {
				l.retType = l.inferredRet
			} else {
				l.retType = types.TUnit
			}
		} else if !types.IsUnit(l.retType) && !types.IsNever(body.Type) {
			f.errorf(e.Pos, "lambda must return a value of type '%s'", l.retType)
		}
	} else {
		var x Expr
		if l.retType != nil {
			x = l.checkExprTo(e.Body, l.retType)
		} else {
			x = l.checkExpr(e.Body, nil)
			l.retType = x.Type()
			if types.IsNever(l.retType) {
				l.retType = types.TUnit
			}
		}
		if types.IsUnit(l.retType) {
			body = &Block{Stmts: []Stmt{&ExprStmt{X: x}}, Type: types.TUnit}
		} else {
			body = &Block{Stmts: []Stmt{&Return{Value: x}}, Type: types.TNever}
		}
	}
	if tupleAdapt {
		var pre []Stmt
		tt := tupleParam.Type.(*types.Tuple)
		for i, v := range params {
			pre = append(pre, &VarDecl{Var: v, Init: &TupleGet{exprBase{tt.Elems[i]}, &VarRef{exprBase{tt}, tupleParam}, i}})
		}
		body.Stmts = append(append(pre, l.paramInit...), body.Stmts...)
	} else if len(l.paramInit) > 0 {
		body.Stmts = append(append([]Stmt{}, l.paramInit...), body.Stmts...)
	}
	l.reportUnused()
	fn.Body = body
	fn.Sig.Ret = l.retType
	suspends := fn.Sig.Effects.Suspends
	if l.inferThrows {
		errs := append([]types.Type{}, fn.inferredErrors...)
		if expected != nil && expected.Effects.Throws {
			// `throws E | Fail`: the lambda may throw Fail whatever else it
			// throws — its type carries the named members so that E binds to
			// the rest and the instance's signature matches exactly
			for _, m := range types.UnionMembers(expected.Effects.Error) {
				if !types.ContainsTypeParam(m) {
					errs = append(errs, m)
				}
			}
		}
		if len(errs) > 0 {
			fn.Sig.Effects = types.Effects{Throws: true, Error: types.MakeErrorUnion(errs...)}
		}
	} else if l.throws {
		fn.Sig.Effects = types.Effects{Throws: true, Error: l.errType}
	}
	fn.Sig.Effects.Suspends = suspends
	// D35: a closure may cross a task boundary when every capture is a
	// `val` of a Sendable type — nothing it reaches can change under
	// another task. That is a fact about this closure, so it is part of its
	// type; where a `sendable fun` is expected, the offending capture is named.
	fn.Sig.Sendable = true
	for _, v := range l.captureList {
		// `self` is captured as the pointer to the receiver's place (D22):
		// what matters is the receiver's type, and it is never a `var`
		ct := v.Type
		isVar := v.Mutable
		if v.IsSelf {
			ct = ct.(*types.Pointer).Elem
			isVar = false
		}
		if isVar || !sendable(ct) || hasVarFields(ct) {
			fn.Sig.Sendable = false
			if expected != nil && expected.Sendable {
				why := "a 'var'"
				switch {
				case isVar:
				case !sendable(ct):
					why = "of type '" + ct.String() + "', which is not Sendable"
				default:
					why = "of type '" + ct.String() + "', which has 'var' fields another task could see change; copy what the lambda needs into vals first"
				}
				f.errorf(e.Pos, "this lambda cannot cross a task boundary: it captures '%s', %s (D35: a sendable function may capture only vals of Sendable types without 'var' fields)", v.Name, why)
				fn.Sig.Sendable = true // reported once; no second mismatch error
			}
			break
		}
	}
	fn.checked = true
	f.c.funcs = append(f.c.funcs, fn)
	return &Closure{exprBase{fn.Sig}, fn, l.captureList}
}
