package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// `cede` (D197) is a statement with nothing after it: no operand, no
// arguments, no value. The forms a reader brings from other languages —
// `cede()`, `await cede`, `val x = cede`, `fun f() => cede` — are each
// reported with what to write instead; where the statement is plainly what
// was meant (`=> cede` as a body) it is read as the block it should be, so
// nothing cascades. selfhost/parser/cede.vs says the same, word for word.

// parseCede reads the statement at the cursor.
func (p *Parser) parseCede() ast.Stmt {
	t := p.next()
	p.cedeTrailer()
	return &ast.CedeStmt{Pos: t.Span}
}

// cedeTrailer reports what follows `cede` on its line and skips it: the
// statement has ended. `else` may follow, in `if (c) cede else …`.
func (p *Parser) cedeTrailer() {
	if p.at(lexer.Semi, lexer.RBrace, lexer.EOF, lexer.KwElse) {
		return
	}
	t := p.cur()
	switch {
	case t.Kind == lexer.LParen && p.peek(1).Kind == lexer.RParen:
		fix := &source.Fix{Title: "Remove the parentheses", Edits: []source.TextEdit{{Span: t.Span.To(p.peek(1).Span)}}}
		p.errorFix(t.Span, fix, "'cede' is a statement, not a function: write 'cede' without parentheses (D197)")
	case t.Kind == lexer.LParen:
		// no fix: deleting the arguments would delete what they compute
		p.errorf(t.Span, "'cede' is a statement, not a function, and takes no arguments: write 'cede' alone (D197)")
	case cedeValueUse(t.Kind):
		p.errorf(t.Span, "'cede' is a statement and has no value: %s cannot follow it (D197)", t.Describe())
	default:
		p.errorf(t.Span, "'cede' takes no operand, label, arguments or type arguments: write 'cede' alone (D197)")
	}
	p.skipToTerminator()
}

// cedeValueUse reports a token that would use what is before it as a
// value: a member access, an operator, an assignment.
func cedeValueUse(k lexer.TokenKind) bool {
	switch k {
	case lexer.Dot, lexer.SafeDot, lexer.Elvis, lexer.OrFail, lexer.Coalesce, lexer.Question,
		lexer.Range, lexer.RangeLt, lexer.Plus, lexer.Minus, lexer.Star, lexer.Slash, lexer.Percent,
		lexer.WrapPlus, lexer.WrapMinus, lexer.WrapStar, lexer.Eq, lexer.NotEq, lexer.LtEq, lexer.Gt,
		lexer.GtEq, lexer.AndAnd, lexer.OrOr, lexer.Caret, lexer.Shl, lexer.Shr, lexer.Pipe, lexer.Amp,
		lexer.KwAs, lexer.KwIs, lexer.KwIn, lexer.KwCatch:
		return true
	}
	return k.IsAssign()
}

// skipToTerminator skips the rest of a statement, up to the `;`, newline
// or `}` that ends it, over nesting; the caller's loop takes the end.
func (p *Parser) skipToTerminator() {
	depth := 0
	for !p.at(lexer.EOF) {
		switch p.cur().Kind {
		case lexer.LBrace, lexer.LParen, lexer.LBracket:
			depth++
		case lexer.RBrace, lexer.RParen, lexer.RBracket:
			if depth == 0 {
				return
			}
			depth--
		case lexer.Semi:
			if depth == 0 {
				return
			}
		}
		p.next()
	}
}

// cedeExpr reports `cede` where a value goes — `val x = cede`,
// `f(cede)`, `return cede` — and skips it.
func (p *Parser) cedeExpr() ast.Expr {
	t := p.next()
	p.errorf(t.Span, "'cede' is a statement and has no value: write it where a statement goes, on a line of its own (D197)")
	return &ast.BadExpr{Pos: t.Span}
}

// cedePrefixed reports `await cede`, `try cede` and `async cede`, the
// prefix (the token before the cursor's `cede`) at start.
func (p *Parser) cedePrefixed(start source.Span, prefix lexer.TokenKind) ast.Expr {
	t := p.next()
	upTo := source.Span{File: start.File, Start: start.Start, End: t.Span.Start}
	switch prefix {
	case lexer.KwAwait:
		fix := &source.Fix{Title: "Remove 'await'", Edits: []source.TextEdit{{Span: upTo}}}
		p.errorFix(start, fix, "'cede' is a statement that already marks where the task may wait: remove 'await' (D197)")
	case lexer.KwTry:
		fix := &source.Fix{Title: "Remove 'try'", Edits: []source.TextEdit{{Span: upTo}}}
		p.errorFix(start, fix, "'cede' is a statement and cannot fail: remove 'try' (D197)")
	default:
		fix := &source.Fix{Title: "Write 'async { cede }'", Edits: []source.TextEdit{{Span: start.To(t.Span), NewText: "async { cede }"}}}
		p.errorFix(start, fix, "'async' starts a task from a call or a block, and 'cede' is neither: write 'async { cede }' (D197)")
	}
	return &ast.BadExpr{Pos: start.To(t.Span)}
}

// cedeFunBody reads `fun name(…) => cede`: reported, and read as the block
// body `{ cede }` it should be.
func (p *Parser) cedeFunBody(arrow lexer.Token, name string) *ast.Block {
	t := p.next()
	fix := &source.Fix{Title: "Write the body as a block", Edits: []source.TextEdit{{Span: arrow.Span.To(t.Span), NewText: "{ cede }"}}}
	p.errorFix(t.Span, fix, "'cede' is a statement and has no value, so it cannot be an expression body: write 'fun %s(…) { cede }' (D197)", name)
	return &ast.Block{Stmts: []ast.Stmt{&ast.CedeStmt{Pos: t.Span}}, Pos: t.Span}
}

// cedeArmBody reads `=> cede` after a lambda's parameters or a `when` or
// `race` arm: reported, and read as the block `=> { cede }` it should be.
func (p *Parser) cedeArmBody() ast.Expr {
	t := p.next()
	fix := &source.Fix{Title: "Write a block", Edits: []source.TextEdit{{Span: t.Span, NewText: "{ cede }"}}}
	p.errorFix(t.Span, fix, "'cede' is a statement and has no value: write a block, '=> { cede }' (D197)")
	return &ast.BlockExpr{Block: &ast.Block{Stmts: []ast.Stmt{&ast.CedeStmt{Pos: t.Span}}, Pos: t.Span}}
}

// errorFix is errorf with a fix, attached when the error is reported (one
// error per token position).
func (p *Parser) errorFix(sp source.Span, fix *source.Fix, format string, args ...any) {
	n := len(p.diags.Items)
	p.errorf(sp, format, args...)
	if fix != nil && len(p.diags.Items) > n {
		p.diags.Items[len(p.diags.Items)-1].Fix = fix
	}
}
