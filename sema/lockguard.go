package sema

import (
	"regexp"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `with n = notes.lock()` (D107): a Mutex held to the end of a block. The
// prelude's `lock()` returns the taken lock and a pointer to the value; the
// `with` holds the pair under a hidden name, binds `n` to the pointer, and
// its close is the unlock. The held region — the with's body — may not
// suspend: a task that waits while holding a lock can deadlock the threads
// the executor runs on. `withLock` gets the same rule from its lambda type
// (`fun(*T): R` cannot suspend); the guard gets it from the region.
//
// Suspension points the checker sees (await, race, scope, gather, async,
// send, recv, sleep, a call of a suspending function value) are refused as
// they are checked; a call of a named function is refused after suspension
// inference, which is when the compiler knows whether it suspends.

// HeldLock is the region of a `with … = m.lock()`, for the messages.
type HeldLock struct {
	Mutex string // `notes`, as written
	Name  string // `n`; "" for `with notes.lock()` (D109)
	Stmt  bool   // the statement form: the region is the rest of the block
}

// message is the error for a suspension inside the region; what suspends
// is named by the caller ("'await'", "'fetch'").
func (h *HeldLock) message(what string) string {
	if h.Stmt {
		item := h.Mutex + ".lock()"
		if h.Name != "" {
			item = h.Name + " = " + item
		}
		return what + " suspends, and the lock on '" + h.Mutex + "' is held until the end of this block; release it first with the block form 'with (" + item + ") { … }' (D107)"
	}
	return what + " suspends, and the lock on '" + h.Mutex + "' is held until the end of its 'with' block; do the waiting after that block (D107)"
}

// isMutexLock: `m.lock()` on a prelude Mutex.
func isMutexLock(rt types.Type, name string) bool {
	st, ok := rt.(*types.Struct)
	return ok && name == "lock" && st.Module == "std.prelude" && st.Name == "Mutex"
}

// lockUse is a `m.lock()` call: allowed only as the value of a `with`
// item, which withBindings marks in f.lockOK before checking it.
func (f *fnCtx) lockUse(callee *ast.MemberExpr, e *ast.CallExpr) {
	if f.lockOK == e {
		f.lockSeen = true
		return
	}
	m := srcText(callee.X)
	var fix *source.Fix
	if at, ok := valBefore(e.Pos); ok {
		fix = fixReplace("Hold it with 'with'", at, "with")
	}
	f.c.errorFix(e.Pos, fix, "'lock()' holds the lock to the end of a 'with' block, so it is usable only as a 'with' value: 'with n = %s.lock()'; for one expression, '%s.withLock(n => …)' (D107)", m, m)
}

var valHead = regexp.MustCompile(`(?:^|\n)[ \t]*(val|var)[ \t]+[A-Za-z_][A-Za-z0-9_]*[ \t]*=[ \t]*$`)

// valBefore finds the `val` of `val n = ` written right before span on its
// line, so the fix can turn it into `with`.
func valBefore(span source.Span) (source.Span, bool) {
	src := span.File.Content
	lineStart := span.Start
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	m := valHead.FindStringSubmatchIndex(src[lineStart:span.Start])
	if m == nil {
		return source.Span{}, false
	}
	return source.Span{File: span.File, Start: lineStart + m[2], End: lineStart + m[3]}, true
}

// withLocked is item i of s when it is `n = m.lock()`: the prelude's
// `Locked<T>` pair is held under a hidden name and closed (unlocked) like
// any resource; `n` is its pointer, a resource too, so it cannot leave the
// region (D100 part 3); the items after it, and the body, are the region.
func (f *fnCtx) withLocked(s *ast.WithExpr, i int, call *ast.CallExpr, init Expr, closeable *types.Trait, want types.Type, asValue bool, result **Var) ([]Stmt, types.Type) {
	b := s.Bindings[i]
	lt, ok := init.Type().(*types.Struct)
	if !ok || len(lt.Fields) != 2 {
		panic("sema: the prelude's Mutex.lock must return Locked<T>, the lock and value pair (D107)")
	}
	pair := f.withVar(ast.WithBinding{Value: b.Value}, lt)
	at := pair.Span
	closeCall := f.checkExpr(&ast.CallExpr{Fun: &ast.MemberExpr{X: nameOf(pair, at), Name: ast.Ident{Name: "close", Pos: at}, Pos: at}, Pos: at}, types.TUnit)
	h := &HeldLock{Mutex: srcText(call.Fun.(*ast.MemberExpr).X), Name: b.Name.Name, Stmt: f.stmtWiths[s]}
	if h.Name == "_" {
		h.Name = ""
	}
	var bind []Stmt
	if h.Name != "" {
		fld := lt.Fields[1] // value: *T
		v := f.newVar(h.Name, fld.Type, false, b.Name.Pos)
		f.declareLocal(h.Name, v, b.Name.Pos)
		f.markResource(v, v)
		if f.c.lockVars == nil {
			f.c.lockVars = map[*Var]bool{}
		}
		f.c.lockVars[v] = true
		bind = []Stmt{&VarDecl{Var: v, Init: &FieldGet{exprBase{fld.Type}, ref(pair), 1, fld.Name}}}
		f.refLock(s, i, h, fld.Type)
	} else {
		f.refLock(s, i, h, lt)
	}
	f.held = append(f.held, h)
	inner, bodyT := f.withBindings(s, i+1, closeable, want, asValue, result)
	f.held = f.held[:len(f.held)-1]
	body := &Block{Stmts: append(bind, inner...), Type: types.TUnit}
	if types.IsNever(bodyT) {
		body.Type = types.TNever
	}
	return []Stmt{&With{Var: pair, Init: init, Close: closeCall, Body: body, Lock: h}}, bodyT
}

// refLock records the hover of the `with` keyword of a lock (D107).
func (f *fnCtx) refLock(s *ast.WithExpr, i int, h *HeldLock, t types.Type) {
	if f.c.index == nil || i > 0 || !s.Pos.IsValid() {
		return
	}
	kw := source.Span{File: s.Pos.File, Start: s.Pos.Start, End: s.Pos.Start + len("with")}
	line, _ := s.Pos.File.Position(s.Body.Pos.End - 1)
	doc := "The lock on `" + h.Mutex + "` is held to the end of this block, line " + itoa(line) + ", and released on every way out of it before then; nothing in between may suspend (D107)."
	detail := "with " + h.Mutex + ".lock()"
	if h.Name != "" {
		detail = "with " + h.Name + typeSuffix(t)
	}
	f.c.index.Refs = append(f.c.index.Refs, Ref{Span: kw, Kind: "keyword", Name: "with", Type: t, Detail: detail, Doc: doc})
}

// refuseHeldSuspension: what (at span) suspends inside a held region.
func (f *fnCtx) refuseHeldSuspension(span source.Span, what string) {
	if len(f.held) == 0 {
		return
	}
	f.errorf(span, "%s", f.held[len(f.held)-1].message(what))
}

// checkHeldRegions refuses calls of suspending functions inside held
// regions, once suspension inference has run.
func (c *Checker) checkHeldRegions(prog *Program) {
	reported := map[source.Span]bool{} // a call in two nested regions: once
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		walkBlock(fn.Body, func(n any) {
			w, ok := n.(*With)
			if !ok || w.Lock == nil {
				return
			}
			walkBlock(w.Body, func(n any) {
				if call, ok := n.(*Call); ok && (susp{}).call(call) && call.Span.IsValid() && !reported[call.Span] {
					reported[call.Span] = true
					c.errorf(call.Span, "%s", w.Lock.message("'"+call.Fn.Display+"'"))
				}
			})
		})
	}
}
