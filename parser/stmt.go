package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

func (p *Parser) parseBlock() *ast.Block {
	start := p.span()
	if _, ok := p.expect(lexer.LBrace); !ok {
		return &ast.Block{Pos: start}
	}
	return p.parseBlockRest(start)
}

// parseBlockRest reads the statements of a block whose `{` (and, in a
// handler, `name =>`) is already consumed, through the closing `}`.
func (p *Parser) parseBlockRest(start source.Span) *ast.Block {
	b := &ast.Block{}
	for {
		p.skipSemis()
		if p.at(lexer.RBrace, lexer.EOF) {
			break
		}
		before := p.pos
		b.Stmts = append(b.Stmts, p.parseStmt())
		if p.pos == before {
			p.errorf(p.span(), "unexpected %s", p.cur().Describe())
			p.next()
			p.syncStmt()
			continue
		}
		p.expectTerminator()
	}
	p.expect(lexer.RBrace)
	b.Pos = p.spanFrom(start)
	return b
}

// parseBodyOrStmt parses either a `{ block }` or a single statement, which
// is wrapped in a block so that later phases see one shape.
func (p *Parser) parseBodyOrStmt() *ast.Block {
	if p.at(lexer.LBrace) {
		return p.parseBlock()
	}
	// the body may start on the next line (`if (c)` newline `a`), as in
	// Kotlin; the line break is not the end of the statement
	if p.at(lexer.Semi) && p.cur().AutoSemi && p.peek(1).Kind != lexer.RBrace && p.peek(1).Kind != lexer.EOF {
		p.next()
		if p.at(lexer.LBrace) {
			return p.parseBlock()
		}
	}
	start := p.span()
	s := p.parseStmt()
	return &ast.Block{Stmts: []ast.Stmt{s}, Pos: p.spanFrom(start)}
}

func (p *Parser) parseBinding() ast.Binding {
	start := p.span()
	if p.at(lexer.LParen) {
		p.next()
		b := ast.Binding{}
		for !p.at(lexer.RParen, lexer.EOF) {
			b.Tuple = append(b.Tuple, p.parseBinding())
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
		if len(b.Tuple) < 2 {
			p.errorf(p.spanFrom(start), "tuple destructuring needs at least two names")
		}
		b.Pos = p.spanFrom(start)
		return b
	}
	if p.at(lexer.Under) {
		t := p.next()
		return ast.Binding{Name: &ast.Ident{Name: "_", Pos: t.Span}, Pos: t.Span}
	}
	// `&x`: the loop variable refers to the element in place (D42)
	isRef := p.accept(lexer.Amp)
	name, _ := p.expectIdent()
	b := ast.Binding{Name: &name, Ref: isRef}
	if p.accept(lexer.Colon) {
		b.Type = p.parseType()
	}
	b.Pos = p.spanFrom(start)
	return b
}

func (p *Parser) parseStmt() ast.Stmt {
	start := p.span()
	switch p.cur().Kind {
	case lexer.KwVal, lexer.KwVar, lexer.KwConst:
		s := &ast.ValStmt{}
		switch p.next().Kind {
		case lexer.KwVar:
			s.Kind = ast.BindVar
		case lexer.KwConst:
			s.Kind = ast.BindConst
		}
		// `val JObj(fields) = doc else { ... }`: a name followed by `(` is a
		// variant pattern, which only a let-else can bind; so is a list
		// pattern `val [a, b] = xs else { ... }` (D62)
		if p.at(lexer.LBracket) || p.at(lexer.LParen) && p.tupleHoldsPattern() || p.at(lexer.Ident) && (p.peek(1).Kind == lexer.LParen || (p.peek(1).Kind == lexer.Dot && p.peek(2).Kind == lexer.Ident && p.peek(3).Kind == lexer.LParen)) {
			s.Pattern = p.parsePattern(true)
		} else {
			s.Binding = p.parseBinding()
		}
		if p.accept(lexer.Assign) {
			s.Value = p.parseExpr()
		} else if s.Kind != ast.BindVar || s.Binding.Type == nil || s.Pattern != nil {
			p.errorf(p.span(), "'%s' binding needs an initializer", s.Kind)
		}
		// let-else: the `else` may start the next line
		if p.at(lexer.Semi) && p.cur().AutoSemi && p.peek(1).Kind == lexer.KwElse {
			p.next()
		}
		if p.accept(lexer.KwElse) {
			if p.at(lexer.LBrace) {
				s.Else = p.parseHandler()
			} else {
				// `else return`: one statement, like the body of `if (c) stmt`
				hs := p.span()
				body := p.parseBodyOrStmt()
				s.Else = &ast.Handler{Body: body, Pos: p.spanFrom(hs)}
			}
		} else if s.Pattern != nil {
			example := "Variant(field)"
			if _, isList := s.Pattern.(*ast.ListPat); isList {
				example = "[a, b]"
			}
			p.errorf(p.span(), "a pattern in a 'val' can fail to match, so it needs 'else { ... }' to say what happens then: 'val %s = x else { return }'", example)
		}
		s.Pos = p.spanFrom(start)
		return s

	case lexer.KwReturn:
		p.next()
		s := &ast.ReturnStmt{}
		// a bare `return` also ends at a closing bracket or a comma, where
		// it stands as an expression: `!(return)`, `f(x ?: return)`
		if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF, lexer.RParen, lexer.RBracket, lexer.Comma) {
			s.Value = p.parseExpr()
		}
		s.Pos = p.spanFrom(start)
		return s

	case lexer.KwThrow:
		p.next()
		s := &ast.ThrowStmt{}
		if p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
			p.errorf(p.span(), "'throw' needs an error value, e.g. 'throw ParseError(text: s)'")
		} else {
			s.Value = p.parseExpr()
		}
		s.Pos = p.spanFrom(start)
		return s

	case lexer.KwBreak, lexer.KwContinue:
		isBreak := p.next().Kind == lexer.KwBreak
		var label *ast.Ident
		if p.at(lexer.Ident) {
			t := p.next()
			label = &ast.Ident{Name: t.Text, Pos: t.Span}
		}
		if isBreak {
			return &ast.BreakStmt{Label: label, Pos: p.spanFrom(start)}
		}
		return &ast.ContinueStmt{Label: label, Pos: p.spanFrom(start)}

	case lexer.KwLoop:
		return p.parseLoop(start)

	case lexer.KwFor:
		p.errorf(p.span(), "there is no 'for'; every loop is spelled 'loop': 'loop (x in xs)', 'loop (cond)' or 'loop { }'")
		return p.parseLoop(start)

	case lexer.KwWith:
		w := p.parseWith()
		return &ast.ExprStmt{X: w}

	case lexer.KwScope:
		p.next()
		body := p.parseBlock()
		return &ast.ScopeStmt{Body: body, Pos: p.spanFrom(start)}

	case lexer.KwFun:
		fn := p.parseFun(nil, funContextFree)
		return &ast.FunStmt{Fun: fn}

	case lexer.KwPub, lexer.KwStruct, lexer.KwTrait, lexer.KwImpl, lexer.KwUse, lexer.KwSealed:
		p.errorf(p.span(), "%s is only allowed at module level", p.cur().Describe())
		p.syncStmt()
		return &ast.BadStmt{Pos: start}
	}
	if p.atErrorDecl() || p.atExtendDecl() {
		p.errorf(p.span(), "'%s' declarations are only allowed at module level", p.cur().Text)
		p.syncStmt()
		return &ast.BadStmt{Pos: start}
	}

	// expression or assignment
	x := p.parseExpr()
	if _, bad := x.(*ast.BadExpr); bad {
		p.syncStmt()
		return &ast.BadStmt{Pos: start}
	}
	switch p.cur().Kind {
	case lexer.Assign, lexer.PlusEq, lexer.MinusEq, lexer.StarEq, lexer.SlashEq, lexer.PercentEq:
		op := p.next().Kind
		val := p.parseExpr()
		return &ast.AssignStmt{Target: x, Op: op, Value: val, Pos: p.spanFrom(start)}
	}
	return &ast.ExprStmt{X: x}
}

func (p *Parser) parseLoop(start source.Span) ast.Stmt {
	if !p.accept(lexer.KwFor) { // already diagnosed by the caller
		p.expect(lexer.KwLoop)
	}
	s := &ast.LoopStmt{}
	if p.accept(lexer.Colon) {
		// `loop :outer (x in xs) { ... break outer }` (section 4b)
		id, _ := p.expectIdent()
		s.Label = &id
	}
	if p.at(lexer.LParen) {
		// `loop (x in c)` or `loop (cond)`. A binding followed by `in` is the
		// iteration form; `(a, b) in` destructures tuples.
		if p.looksLikeForIn() {
			p.next()
			b := p.parseBinding()
			s.Var = &b
			p.expect(lexer.KwIn)
			s.Iter = p.parseExpr()
			p.expect(lexer.RParen)
		} else {
			p.next()
			s.Cond = p.parseExpr()
			p.closeCondition()
		}
	}
	s.Body = p.parseBodyOrStmt()
	s.Pos = p.spanFrom(start)
	return s
}

// looksLikeForIn scans `( binding in` without consuming.
func (p *Parser) looksLikeForIn() bool {
	i := 1
	depth := 0
	for {
		t := p.peek(i)
		switch t.Kind {
		case lexer.LParen:
			depth++
		case lexer.RParen:
			if depth == 0 {
				return false
			}
			depth--
		case lexer.KwIn:
			return depth == 0
		case lexer.Ident, lexer.Comma, lexer.Under, lexer.Colon, lexer.Amp:
			// part of a binding
		case lexer.EOF:
			return false
		default:
			if depth == 0 {
				return false
			}
		}
		i++
		if i > 64 {
			return false
		}
	}
}

func (p *Parser) parseWith() *ast.WithExpr {
	start := p.span()
	p.next() // with
	s := &ast.WithExpr{}
	if _, ok := p.expect(lexer.LParen); ok {
		for !p.at(lexer.RParen, lexer.EOF) {
			name, ok := p.expectIdent()
			if !ok {
				p.syncParen()
				break
			}
			b := ast.WithBinding{Name: name}
			if _, ok := p.expect(lexer.Assign); ok {
				b.Value = p.parseExpr()
			}
			s.Bindings = append(s.Bindings, b)
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
	}
	s.Body = p.parseBlock()
	s.Pos = p.spanFrom(start)
	return s
}

// atHandler: a `{` after `??` always opens a handler — no other expression
// begins with a brace.
func (p *Parser) atHandler() bool { return p.at(lexer.LBrace) }

// parseHandler is `{ stmts }` or `{ e => stmts }`: the failure branch of a
// let-else and of `r ?? { ... }`, with a Result's error bound to `e`.
// parseDoCatch reads `do { body } catch (e) { handler }` (D98), with the
// cursor on `do`. The `catch` may start the next line, as an `else` may.
func (p *Parser) parseDoCatch() ast.Expr {
	start := p.span()
	p.next() // do
	body := p.parseBlock()
	if p.at(lexer.Semi) && p.cur().AutoSemi && p.peek(1).Kind == lexer.KwCatch {
		p.next()
	}
	if !p.at(lexer.KwCatch) {
		if p.at(lexer.KwReserved) && p.cur().Text == "while" {
			p.errorf(p.span(), "there is no do-while: write 'loop { ...; if (!cond) break }', or 'loop (cond) { ... }' to test first; 'do' opens 'do { ... } catch (e) { ... }' (D98)")
		} else {
			p.errorf(p.span(), "a 'do' block needs a 'catch (e) { ... }' after it: what to do when a 'try' or 'throw' inside fails (D98)")
		}
		// a well-formed placeholder, not a BadExpr: the statement parser would
		// resynchronise on a BadExpr and swallow the terminator, and the next
		// statement would be reported too
		return &ast.CatchExpr{Body: body, Handler: &ast.Handler{Body: &ast.Block{Pos: p.spanFrom(start)}, Pos: p.spanFrom(start)}, Pos: p.spanFrom(start)}
	}
	p.next() // catch
	h := p.parseCatchHandler()
	return &ast.CatchExpr{Body: body, Handler: h, Pos: p.spanFrom(start)}
}

// atCatch reports a `catch` at the cursor, or on the next line.
func (p *Parser) atCatch() bool {
	if p.at(lexer.KwCatch) {
		return true
	}
	return p.at(lexer.Semi) && p.cur().AutoSemi && p.peek(1).Kind == lexer.KwCatch
}

// parseCatchPostfix reads `catch (e) { handler }` after an expression (D98),
// with the cursor on `catch` or on the line break before it.
func (p *Parser) parseCatchPostfix(x ast.Expr, start source.Span) ast.Expr {
	if p.at(lexer.Semi) {
		p.next()
	}
	p.next() // catch
	h := p.parseCatchHandler()
	return &ast.CatchExpr{X: x, Handler: h, Pos: p.spanFrom(start)}
}

// parseCatchHandler is what follows the word `catch` (D98): `(e) { ... }`,
// which binds the error, or `{ ... }`, which does not. The binding is in a
// head like `when (v)` and `loop (x in xs)`; the block is an ordinary one.
func (p *Parser) parseCatchHandler() *ast.Handler {
	start := p.span()
	h := &ast.Handler{}
	switch {
	case p.at(lexer.LParen):
		p.next()
		if p.at(lexer.Ident) || p.at(lexer.Under) {
			t := p.next()
			h.Err = &ast.Ident{Name: t.Text, Pos: t.Span}
		} else {
			p.errorExpected("the name for the error")
		}
		p.expect(lexer.RParen)
		h.Body = p.parseBlock()
	case p.at(lexer.LBrace) && (p.peek(1).Kind == lexer.Ident || p.peek(1).Kind == lexer.Under) && p.peek(2).Kind == lexer.FatArrow:
		p.errorf(p.span(), "the error is named in a head: 'catch (%s) { ... }', not 'catch { %s => ... }' (D98)", p.peek(1).Text, p.peek(1).Text)
		return p.parseHandler()
	default:
		h.Body = p.parseBlock()
	}
	h.Pos = p.spanFrom(start)
	return h
}

func (p *Parser) parseHandler() *ast.Handler {
	start := p.span()
	h := &ast.Handler{}
	if (p.peek(1).Kind == lexer.Ident || p.peek(1).Kind == lexer.Under) && p.peek(2).Kind == lexer.FatArrow {
		p.errorf(p.span(), "a handler that sees the error is 'catch (%s) { ... }' now: 'r catch (%s) { ... }' (D98)", p.peek(1).Text, p.peek(1).Text)
		p.next() // {
		t := p.next()
		h.Err = &ast.Ident{Name: t.Text, Pos: t.Span}
		p.next() // =>
		h.Body = p.parseBlockRest(start)
	} else {
		h.Body = p.parseBlock()
	}
	h.Pos = p.spanFrom(start)
	return h
}

// tupleHoldsPattern reports whether the parenthesised binding at the cursor
// nests a pattern that can fail — a list pattern or a variant — as in
// `val (label, [a, b]) = pair else ...`. Such a `val` is parsed as a
// pattern (a let-else); a plain `(a, b: i64)` stays a tuple binding.
func (p *Parser) tupleHoldsPattern() bool {
	depth := 0
	for i := 0; ; i++ {
		switch t := p.peek(i); t.Kind {
		case lexer.LParen:
			depth++
		case lexer.RParen:
			depth--
			if depth == 0 {
				return false
			}
		case lexer.LBracket:
			return true
		case lexer.Ident:
			if p.peek(i+1).Kind == lexer.LParen && depth > 0 {
				return true
			}
		case lexer.EOF, lexer.Assign, lexer.LBrace, lexer.Semi:
			return false
		}
	}
}
