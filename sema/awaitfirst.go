package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// awaitFirst (D168) was removed by D169: `race { async a(); async b() }` says
// the same. It still parses as a call; the error's fix writes the race, and
// the call is checked as that race so nothing more is said.
func (f *fnCtx) awaitFirstCall(e *ast.CallExpr, want types.Type) Expr {
	var parts []string
	race := &ast.RaceExpr{Body: &ast.Block{Pos: e.Pos}, Pos: e.Pos}
	for _, a := range e.Args {
		parts = append(parts, srcText(a.Value))
		race.Body.Stmts = append(race.Body.Stmts, &ast.ExprStmt{X: a.Value})
	}
	text := "race { " + strings.Join(parts, "; ") + " }"
	f.c.errorFix(e.Pos, fixReplace("Write a 'race'", e.Pos, text),
		"'awaitFirst' is now a 'race': '%s' runs the tasks, the first to finish wins and the others are cancelled (D169)", text)
	if f.isGlobal || len(e.Args) == 0 {
		return bad()
	}
	return f.raceBlock(race, want)
}
