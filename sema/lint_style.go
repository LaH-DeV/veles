package sema

import (
	"os"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Style lints found by reading the repository's own code (2026-10-20,
// veles-syntax-ideas.md): each one points at a longer way of writing
// something the language already says shortly, and offers the short form.

// ---------------------------------------------------------------------------
// redundant_discard

// lintRedundantDiscard: `val _ = seen.add(x)` discards a value a statement may
// simply ignore. Only for values of plain types (numbers, text, booleans, a
// nullable of those, a collection): a `Result` must be named, a task handle or
// a closeable resource is a different rule.
func (f *fnCtx) lintRedundantDiscard(s *ast.ValStmt, init Expr) {
	if s.Kind != ast.BindVal || s.Binding.Name == nil || s.Binding.Name.Name != "_" || s.Binding.Type != nil ||
		s.Value == nil || s.Else != nil || init == nil || !plainDiscard(init.Type()) {
		return
	}
	// an `if` or `when` used as a value joins its branches' types; as a statement it
	// does not, so the binding is not always redundant
	switch s.Value.(type) {
	case *ast.IfExpr, *ast.WhenExpr:
		return
	}
	span := source.Span{File: s.Pos.File, Start: s.Pos.Start, End: s.Value.Span().Start}
	if span.File == nil || span.End <= span.Start || f.skipStd(s.Pos) {
		return
	}
	f.warnLint("redundant_discard", span, fixReplace("Drop 'val _ ='", span, ""),
		"'val _ =' discards a value of type '%s' that a statement may simply ignore; write the expression alone", init.Type())
}

func plainDiscard(t types.Type) bool {
	switch t := types.Underlying(t).(type) {
	case *types.Basic:
		return !types.IsUnit(t) && !types.IsNever(t) && !types.IsInvalid(t)
	case *types.Nullable:
		return plainDiscard(t.Elem)
	case *types.List, *types.Map, *types.Set:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// else_if_ladder

// lintElseIfLadder: three or more branches of `if … else if … else if` are a
// `when { … }` with the conditions as arms. Skipped when a condition binds
// (`val x = e &&`, D95), which a `when` arm cannot do.
func (f *fnCtx) lintElseIfLadder(e *ast.IfExpr) {
	if f.c.ladderSeen[e] {
		return // a link of a chain whose head was looked at
	}
	var conds []ast.Expr
	var blocks []*ast.Block
	var last *ast.Block
	cur := e
	for {
		conds = append(conds, cur.Cond)
		blocks = append(blocks, cur.Then)
		last = cur.Then
		if cur.Else == nil {
			break
		}
		next := elseIfOf(cur.Else)
		if next == nil {
			blocks = append(blocks, cur.Else) // the final `else { }`
			last = cur.Else
			break
		}
		if f.c.ladderSeen == nil {
			f.c.ladderSeen = map[*ast.IfExpr]bool{}
		}
		f.c.ladderSeen[next] = true
		cur = next
	}
	if len(conds) < 3 {
		return
	}
	for _, c := range conds {
		binds := false
		walkAST(c, func(n any) bool {
			if _, ok := n.(*ast.LetCond); ok {
				binds = true
			}
			return !binds
		})
		if binds {
			return
		}
	}
	file := e.Pos.File
	if file == nil || last == nil || f.skipStd(e.Pos) || last.Pos.End > len(file.Content) {
		return
	}
	span := source.Span{File: file, Start: e.Pos.Start, End: last.Pos.End}
	indent := lineIndent(file.Content, span.Start)
	var sb strings.Builder
	sb.WriteString("when {\n")
	for i, c := range conds {
		sb.WriteString(indent + "  " + spanText(c.Span()) + " => " + spanText(blocks[i].Pos) + "\n")
	}
	if len(blocks) > len(conds) {
		sb.WriteString(indent + "  else => " + spanText(blocks[len(blocks)-1].Pos) + "\n")
	}
	sb.WriteString(indent + "}")
	f.warnLint("else_if_ladder", e.Pos, fixReplace("Write it as a 'when { }'", span, sb.String()),
		"%d conditions in a row: a 'when { … }' with one arm per condition reads straighter than an else-if ladder", len(conds))
}

// elseIfOf is the `if` of `else if`, nil for a real `else { … }` block (an
// `else { if … }` written with braces is not a ladder link).
func elseIfOf(b *ast.Block) *ast.IfExpr {
	if b == nil || len(b.Stmts) != 1 || b.Pos.File == nil || b.Pos.Start >= len(b.Pos.File.Content) || b.Pos.File.Content[b.Pos.Start] == '{' {
		return nil
	}
	if es, ok := b.Stmts[0].(*ast.ExprStmt); ok {
		if ie, ok := es.X.(*ast.IfExpr); ok {
			return ie
		}
	}
	return nil
}

// lineIndent is the leading whitespace of the line that holds offset at.
func lineIndent(text string, at int) string {
	start := strings.LastIndexByte(text[:at], '\n') + 1
	end := start
	for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

// ---------------------------------------------------------------------------
// continue_skips_step

// lintContinueSkipsStep: in `loop (i < n) { …; i += 1 }` a `continue` before
// the step jumps over it, and the loop never advances.
func (f *fnCtx) lintContinueSkipsStep(loop *ast.LoopStmt) {
	if loop.Cond == nil || loop.Var != nil || loop.Body == nil || f.skipStd(loop.Pos) {
		return
	}
	for k, s := range loop.Body.Stmts {
		as, ok := s.(*ast.AssignStmt)
		if !ok || (as.Op != lexer.PlusEq && as.Op != lexer.MinusEq) {
			continue
		}
		target, ok := as.Target.(*ast.NameExpr)
		if !ok || !mentionsName(loop.Cond, target.Name) {
			continue
		}
		// a counter steps by a constant; `offset += sent` is progress made, and a
		// `continue` that retries without it (after a wait) is the point
		if _, constant := as.Value.(*ast.IntLit); !constant {
			continue
		}
		var at []source.Span
		scanContinues(loop.Body.Stmts[:k], target.Name, false, &at)
		for _, sp := range at {
			f.warnLint("continue_skips_step", sp, nil,
				"this 'continue' skips '%s %s ...' below: the loop does not advance and may never end; step first, or use 'loop (%s in …)'",
				target.Name, opText(as.Op), target.Name)
		}
		return
	}
}

func opText(op lexer.TokenKind) string {
	if op == lexer.MinusEq {
		return "-="
	}
	return "+="
}

// scanContinues finds the `continue`s of this loop in stmts that are not
// preceded, on their path, by an assignment to v.
func scanContinues(stmts []ast.Stmt, v string, advanced bool, out *[]source.Span) {
	for _, s := range stmts {
		switch s := s.(type) {
		case *ast.ContinueStmt:
			if s.Label == nil && !advanced {
				*out = append(*out, s.Pos)
			}
			return
		case *ast.AssignStmt:
			if t, ok := s.Target.(*ast.NameExpr); ok && t.Name == v {
				advanced = true
			}
		case *ast.ExprStmt:
			scanContinuesIn(s.X, v, advanced, out)
		case *ast.Block:
			scanContinues(s.Stmts, v, advanced, out)
		}
	}
}

func scanContinuesIn(e ast.Expr, v string, advanced bool, out *[]source.Span) {
	switch e := e.(type) {
	case *ast.IfExpr:
		if e.Then != nil {
			scanContinues(e.Then.Stmts, v, advanced, out)
		}
		if e.Else != nil {
			scanContinues(e.Else.Stmts, v, advanced, out)
		}
	case *ast.BlockExpr:
		if e.Block != nil {
			scanContinues(e.Block.Stmts, v, advanced, out)
		}
	case *ast.WhenExpr:
		for _, arm := range e.Arms {
			scanContinuesIn(arm.Body, v, advanced, out)
		}
	}
}

// ---------------------------------------------------------------------------
// loop_is_range

// lintCounterLoop: `var i = a` followed by `loop (i < b) { …; i += 1 }`, with
// `i` touched nowhere else and unused afterwards and `b` unable to change, is
// `loop (i in a..<b)`: the same values, no manual step to forget (or to jump
// over with `continue`).
func (f *fnCtx) lintCounterLoop(stmts []ast.Stmt, i int) {
	decl, ok := stmts[i].(*ast.ValStmt)
	if !ok || decl.Kind != ast.BindVar || decl.Binding.Name == nil || decl.Binding.Type != nil || decl.Value == nil || i+1 >= len(stmts) {
		return
	}
	name := decl.Binding.Name.Name
	loop, ok := stmts[i+1].(*ast.LoopStmt)
	if !ok || loop.Var != nil || loop.Label != nil || loop.Body == nil || loop.Cond == nil || len(loop.Body.Stmts) == 0 {
		return
	}
	cmp, ok := loop.Cond.(*ast.BinaryExpr)
	if !ok || cmp.Op != lexer.Lt {
		return
	}
	if x, ok := cmp.L.(*ast.NameExpr); !ok || x.Name != name {
		return
	}
	body := loop.Body.Stmts
	step, ok := body[len(body)-1].(*ast.AssignStmt)
	if !ok || step.Op != lexer.PlusEq {
		return
	}
	if t, ok := step.Target.(*ast.NameExpr); !ok || t.Name != name {
		return
	}
	if one, ok := step.Value.(*ast.IntLit); !ok || one.Text != "1" {
		return
	}
	// the counter is assigned only by that step, never taken by address, and
	// nothing jumps over the step with a `continue`
	assigns, hasContinue := 0, false
	for _, s := range body {
		if assignsName(s, name) {
			walkAST(s, func(n any) bool {
				if as, ok := n.(*ast.AssignStmt); ok {
					if t, ok := as.Target.(*ast.NameExpr); ok && t.Name == name {
						assigns++
					}
				}
				return true
			})
		}
		walkAST(s, func(n any) bool {
			switch n.(type) {
			case *ast.ContinueStmt:
				hasContinue = true
			case *ast.LambdaExpr:
				return false
			}
			return true
		})
	}
	if assigns != 1 || hasContinue || mentionsName(cmp.R, name) {
		return
	}
	for _, s := range stmts[i+2:] {
		if mentionsName(s, name) {
			return
		}
	}
	if !f.loopInvariant(cmp.R) {
		return
	}
	file := decl.Pos.File
	if file == nil || f.skipStd(decl.Pos) {
		return
	}
	from, bound := spanText(decl.Value.Span()), spanText(cmp.R.Span())
	if from == "" || bound == "" {
		return
	}
	declLines, ok := wholeLines(file, decl.Pos)
	if !ok {
		return
	}
	stepLines, ok := wholeLines(file, step.Pos)
	if !ok {
		return
	}
	fix := &source.Fix{Title: "Write it as 'loop (" + name + " in …)'", Edits: []source.TextEdit{
		{Span: declLines, NewText: ""},
		{Span: loop.Cond.Span(), NewText: name + " in " + rangeOperand(from) + "..<" + rangeOperand(bound)},
		{Span: stepLines, NewText: ""},
	}}
	f.warnLint("loop_is_range", loop.Pos, fix,
		"'%s' only counts up from %s to %s: write 'loop (%s in %s..<%s)' and drop the step", name, from, bound, name, rangeOperand(from), rangeOperand(bound))
}

// rangeOperand parenthesises text unless it is a plain name, number or chain
// of members and calls.
func rangeOperand(text string) string {
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '(', r == ')':
		default:
			return "(" + text + ")"
		}
	}
	return text
}

// wholeLines widens a statement's span to the lines it alone occupies, so that
// removing it leaves no blank line; false when other code shares the line.
func wholeLines(file *source.File, sp source.Span) (source.Span, bool) {
	text := file.Content
	if sp.Start < 0 || sp.End > len(text) {
		return sp, false
	}
	start := strings.LastIndexByte(text[:sp.Start], '\n') + 1
	if strings.TrimSpace(text[start:sp.Start]) != "" {
		return sp, false
	}
	end := sp.End
	for end < len(text) && (text[end] == ' ' || text[end] == '\t' || text[end] == '\r') {
		end++
	}
	if end < len(text) && text[end] == '\n' {
		end++
	} else if end < len(text) && text[end] != '}' {
		// a comment or more code on the same line: leave it alone
		if !strings.HasPrefix(strings.TrimSpace(text[end:]), "//") {
			return sp, false
		}
		nl := strings.IndexByte(text[end:], '\n')
		if nl < 0 {
			end = len(text)
		} else {
			end += nl + 1
		}
	}
	return source.Span{File: file, Start: start, End: end}, true
}

// loopInvariant reports whether the value of e cannot change while a loop
// runs: a number, an immutable binding or field, the length of one, and
// arithmetic on those.
func (f *fnCtx) loopInvariant(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.IntLit:
		return true
	case *ast.BinaryExpr:
		switch e.Op {
		case lexer.Plus, lexer.Minus, lexer.Star:
			return f.loopInvariant(e.L) && f.loopInvariant(e.R)
		}
	case *ast.CallExpr:
		m, ok := e.Fun.(*ast.MemberExpr)
		if ok && !m.Safe && m.Name.Name == "len" && len(e.Args) == 0 {
			t, immutable := f.pathType(m.X)
			if !immutable {
				return false
			}
			switch types.Underlying(t).(type) {
			case *types.List, *types.Array:
				return !isMutableType(t)
			case *types.Basic:
				return types.IsString(t)
			}
		}
	case *ast.NameExpr, *ast.MemberExpr, *ast.SelfExpr:
		t, immutable := f.pathType(e)
		return immutable && types.IsInteger(t)
	}
	return false
}

func isMutableType(t types.Type) bool {
	switch t := types.Underlying(t).(type) {
	case *types.List:
		return t.Mutable
	case *types.Map:
		return t.Mutable
	case *types.Set:
		return t.Mutable
	}
	return false
}

// pathType is the type of a name or a chain of fields (`this.src`), and
// whether nothing along the way can be assigned: a `val`, a parameter, a
// field not declared `var`.
func (f *fnCtx) pathType(e ast.Expr) (types.Type, bool) {
	switch e := e.(type) {
	case *ast.SelfExpr:
		if f.selfVar == nil {
			return nil, false
		}
		return f.selfVar.Type, true
	case *ast.NameExpr:
		sym := f.lookup(e.Name)
		if sym == nil || sym.Kind != SymLocal || sym.Var == nil {
			return nil, false
		}
		return sym.Var.Type, !sym.Var.Mutable && !sym.Var.AddrTaken
	case *ast.MemberExpr:
		if e.Safe {
			return nil, false
		}
		t, immutable := f.pathType(e.X)
		if !immutable || t == nil {
			return nil, false
		}
		if p, ok := types.Underlying(t).(*types.Pointer); ok {
			t = p.Elem
		}
		st, ok := types.Underlying(t).(*types.Struct)
		if !ok {
			return nil, false
		}
		for _, fld := range st.Fields {
			if fld.Name == e.Name.Name {
				return fld.Type, !fld.Var && !fld.Protected
			}
		}
	}
	return nil, false
}

// lintStd lifts the guard below, so a sweep of the repository (VELES_LINT_STD=1
// with `veles check --fix`) reaches the standard library as well.
var lintStd = os.Getenv("VELES_LINT_STD") != ""

// skipStd: a style lint does not speak about code the compiler wrote (derived
// implements carry the span of the `implement` that asked for them) or about the
// standard library's own files, which a user cannot change.
func (f *fnCtx) skipStd(span source.Span) bool {
	return f.c.syntheticSpans[span] || (!lintStd && f.c.stdSpan(span))
}
