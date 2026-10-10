package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// The waiting operations of the D146 types — `Event.wait`,
// `Watch.changed`, `Subscription.recv` — always suspend, so they are
// awaited as `ch.recv()` is (D16), and each can be a `race` arm. The
// prelude writes each such method `m` with two non-public halves (see
// std/prelude/notify.vs): `mSignal()`, a channel that is ready when there is
// something to take, and `mTake()`, which takes it. A race arm on `x.m()`
// waits on `x.mSignal()` like a channel arm and, when it wins, runs
// `x.mTake()` for the arm's value — so a losing arm has taken nothing.

// awaitedHalves gives the two halves of t when t is such a method.
func (f *fnCtx) awaitedHalves(t *FuncTemplate) (signal, take *FuncTemplate) {
	if t.Owner == nil || t.Module == nil || t.Module.Path != "std/prelude" || !t.Pub {
		return nil, nil
	}
	ms := f.c.methods[t.Owner]
	signal, take = ms[t.Name+"Signal"], ms[t.Name+"Take"]
	if signal == nil || take == nil {
		return nil, nil
	}
	return signal, take
}

// awaitedCall checks the call e of the awaited method t: written with
// `await`, or the source of a race arm, which it lowers. It returns the
// arm's source, or nil when the call is checked as an ordinary one.
func (f *fnCtx) awaitedCall(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, recv Expr, viaPointer bool, callee *ast.MemberExpr, e *ast.CallExpr) Expr {
	signal, take := f.awaitedHalves(t)
	if signal == nil {
		return nil
	}
	awaited := f.awaitNext
	f.awaitNext = false
	if f.raceCall != e {
		if !awaited {
			f.errorf(e.Pos, "'%s()' always suspends and must be awaited: 'await %s.%s()' (D16, D146)", t.Name, srcText(callee.X), t.Name)
		}
		return nil
	}
	if len(e.Args) > 0 {
		f.arityError(e.Pos, recv.Type(), t.Name, 0)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	// the receiver's place, which both halves read (the take may record
	// what this holder has seen, as Watch does)
	var recvArg Expr
	switch {
	case viaPointer:
		recvArg = recv.(*Deref).X
	case isPlaceExpr(recv) && !isReferenceType(recv.Type()):
		recvArg = &AddrOf{exprBase{&types.Pointer{Elem: recv.Type()}}, recv}
	default:
		f.errorf(callee.X.Span(), "a race arm on '%s()' needs its receiver in a variable: 'val x = …' before the race (D146)", t.Name)
		return bad()
	}
	f.raceTake = f.callTemplateRecv(take, ownerSubst, nil, recvArg, nil, e.Pos, nil)
	return f.callTemplateRecv(signal, ownerSubst, nil, recvArg, nil, e.Pos, nil)
}
