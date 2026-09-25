package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// Lint: `loop (true) { ... }` is `loop { ... }`, the form the language has
// for a loop that only ends by `break` or `return` (notes T4). The fix
// deletes the condition with its parentheses; it is offered only when the
// body is braced, since `loop stmt` without a condition does not read as a
// loop at a glance.
func (f *fnCtx) lintLoopTrue(s *ast.LoopStmt) {
	lit, ok := s.Cond.(*ast.BoolLit)
	if !ok || !lit.Value {
		return
	}
	f.warnFix(lit.Pos, fixLoopTrue(s, lit.Pos), "'loop (true)' is written 'loop { ... }': a loop with no condition runs until 'break' or 'return'")
}

// fixLoopTrue removes `(true)` and the space before it, keeping everything
// else: `loop :outer (true) {` becomes `loop :outer {`.
func fixLoopTrue(s *ast.LoopStmt, cond source.Span) *source.Fix {
	if cond.File == nil || s.Body == nil || s.Body.Pos.File != cond.File || cond.File.Content[s.Body.Pos.Start] != '{' {
		return nil
	}
	src := cond.File.Content
	open := cond.Start
	for open > 0 && (src[open-1] == ' ' || src[open-1] == '\t') {
		open--
	}
	if open == 0 || src[open-1] != '(' {
		return nil
	}
	open--
	close := cond.End
	for close < len(src) && (src[close] == ' ' || src[close] == '\t') {
		close++
	}
	if close == len(src) || src[close] != ')' {
		return nil
	}
	close++
	for open > 0 && (src[open-1] == ' ' || src[open-1] == '\t') {
		open--
	}
	return fixReplace("Write 'loop { ... }'", source.Span{File: cond.File, Start: open, End: close}, "")
}
