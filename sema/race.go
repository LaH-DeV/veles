package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `race { body }` (D169). The body is statements, as a gather's is; every
// task it starts into the race — `async f()` or `async { }` written in it,
// in a loop or a branch, not inside a nested scope, gather, race or `with`
// — is a contestant. Lowered as
//
//	var won: T
//	scope(cancel) {
//	  val tasks: MutableList<Task<T>> = []
//	  body, each contestant's handle pushed onto tasks as it starts
//	  if (tasks.len() == 0) panic(…)        // only when a loop could start none
//	  won = await tasks.get(raceFirst(tasks))
//	}
//	won
//
// The first task to finish wins: its value is won, and the scope's end
// cancels and joins the others. A task that fails first fails the scope —
// the scope's fail-fast cancels the rest and its error leaves the function,
// which is D168's "first to finish, an error included".

// raceCtx is a race body being checked.
type raceCtx struct {
	group  *ScopeBlock
	tasks  *Var        // the contestants' handles, made at the first one
	elem   *types.Task // their handle type
	n      int         // contestants written
	looped bool        // one is written in a loop of the body
	loops  int         // len(f.loops) when the body began
}

// raceDispatch checks a `race`: the block of D169, or the removed race with
// arms — now `select` — reported with the fix and checked as one.
func (f *fnCtx) raceDispatch(e *ast.RaceExpr, want types.Type) Expr {
	if e.Body != nil {
		return f.raceBlock(e, want)
	}
	kw := source.Span{File: e.Pos.File, Start: e.Pos.Start, End: e.Pos.Start + len("race")}
	f.c.errorFix(kw, fixReplace("Write 'select'", kw, "select"),
		"a 'race' with arms is a 'select' now; 'race { }' runs tasks and the first to finish wins (D169)")
	return f.selectExpr(e.Arms, e.Pos, want)
}

func (f *fnCtx) raceBlock(e *ast.RaceExpr, want types.Type) Expr {
	if f.isGlobal {
		f.errorf(e.Pos, "'race' cannot appear in a global initializer; compute the value in 'main' and pass it down")
		return bad()
	}
	group := &ScopeBlock{Span: e.Pos, Cancel: true}
	group.T = types.TUnit
	rc := &raceCtx{group: group, loops: len(f.loops)}
	f.races = append(f.races, rc)
	f.scopes = append(f.scopes, group)
	f.openTaskBlock("race")
	body := f.checkBlock(e.Body, nil, false)
	f.closeTaskBlock()
	f.scopes = f.scopes[:len(f.scopes)-1]
	f.races = f.races[:len(f.races)-1]
	if rc.tasks == nil {
		if rc.n == 0 {
			f.errorf(e.Pos, "this 'race' starts no task: write what races as 'async f()' or 'async { … }' in its body (D169)")
		}
		return bad()
	}
	if rc.n == 1 && !rc.looped {
		f.warnf(e.Pos, "a 'race' of one task is the call itself: call it directly (D169)")
	}
	t := rc.elem.Result
	won := f.newTemp(t)
	list := rc.tasks.Type
	stmts := []Stmt{&VarDecl{Var: rc.tasks, Init: &ListLit{exprBase{list}, nil}}}
	stmts = append(stmts, body.Stmts...)
	if body.Value != nil {
		stmts = append(stmts, &ExprStmt{X: body.Value})
	}
	if rc.looped {
		// a loop may start none: the race would wait forever
		n := &Builtin{exprBase{types.TI64}, "list.len", []Expr{ref(rc.tasks)}, e.Pos}
		none := &Binary{exprBase{types.TBool}, OpEq, n, &IntConst{exprBase{types.TI64}, 0, false}, e.Pos}
		msg := &StringConst{exprBase{types.TString}, "race: its body started no task (D169)"}
		fail := &Block{Stmts: []Stmt{&ExprStmt{X: &Builtin{exprBase{types.TNever}, "panic", []Expr{msg}, e.Pos}}}, Type: types.TNever}
		stmts = append(stmts, &ExprStmt{X: &If{exprBase{types.TUnit}, none, fail, nil}})
	}
	first := &RaceFirst{exprBase{types.TI64}, ref(rc.tasks)}
	handle := &Builtin{exprBase{rc.elem}, "list.get", []Expr{ref(rc.tasks), first}, e.Pos}
	stmts = append(stmts, &Assign{Target: ref(won), Value: f.suspending(&AwaitTask{exprBase{t}, handle}, e.Pos, "race")})
	group.Body = &Block{Stmts: stmts, Type: types.TUnit}
	group.ErrTo = f.currentErrType()
	if f.errType == nil && f.throws {
		group.ErrTo = nil // inferred: filled in at codegen time, as for `scope`
	}
	f.suspending(group, e.Pos, "race")
	return &BlockExpr{exprBase{t}, &Block{Stmts: []Stmt{&VarDecl{Var: won}, group}, Value: ref(won), Type: t}}
}

// raceContestant registers a task just started into the race being
// checked — its handle goes onto the race's list — and leaves any other
// launch as it is.
func (f *fnCtx) raceContestant(x Expr, at source.Span) Expr {
	if len(f.races) == 0 || len(f.scopes) == 0 {
		return x
	}
	rc := f.races[len(f.races)-1]
	if f.scopes[len(f.scopes)-1] != rc.group {
		return x
	}
	tt, ok := x.Type().(*types.Task)
	if !ok {
		return x
	}
	rc.n++
	if len(f.loops) > rc.loops {
		rc.looped = true
	}
	switch {
	case rc.tasks == nil:
		rc.elem = tt
		rc.tasks = f.newTemp(&types.List{Elem: tt, Mutable: true})
	case !types.Identical(tt.Result, rc.elem.Result):
		f.errorf(at, "the tasks of a 'race' give one type: this one gives '%s', the first gives '%s' (D169)", tt.Result, rc.elem.Result)
		return x
	}
	h := f.newTemp(tt)
	push := &Builtin{exprBase{types.TUnit}, "list.push", []Expr{ref(rc.tasks), ref(h)}, at}
	return &BlockExpr{exprBase{tt}, &Block{Stmts: []Stmt{&VarDecl{Var: h, Init: x}, &ExprStmt{X: push}}, Value: ref(h), Type: tt}}
}
