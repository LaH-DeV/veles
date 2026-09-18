package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

func (p *Parser) parseBlock() *ast.Block {
	start := p.span()
	b := &ast.Block{}
	if _, ok := p.expect(lexer.LBrace); !ok {
		b.Pos = start
		return b
	}
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
	name, _ := p.expectIdent()
	b := ast.Binding{Name: &name}
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
		s.Binding = p.parseBinding()
		if p.accept(lexer.Assign) {
			s.Value = p.parseExpr()
		} else if s.Kind != ast.BindVar || s.Binding.Type == nil {
			p.errorf(p.span(), "'%s' binding needs an initializer", s.Kind)
		}
		s.Pos = p.spanFrom(start)
		return s

	case lexer.KwReturn:
		p.next()
		s := &ast.ReturnStmt{}
		if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
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
		return p.parseWith()

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
		case lexer.Ident, lexer.Comma, lexer.Under, lexer.Colon:
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

func (p *Parser) parseWith() ast.Stmt {
	start := p.span()
	p.next() // with
	s := &ast.WithStmt{}
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
