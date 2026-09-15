package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// parsePattern parses one `when` pattern (D13). With binding=false a bare
// identifier is a value to compare against (Kotlin's rule for top-level
// arms); with binding=true, inside a destructuring, it binds a new name.
func (p *Parser) parsePattern(binding bool) ast.Pattern {
	start := p.span()
	switch p.cur().Kind {
	case lexer.Under:
		p.next()
		return &ast.WildcardPat{Pos: start}
	case lexer.KwIs:
		p.next()
		return p.parseTypePatternRest(start)
	case lexer.KwIn:
		p.next()
		r := p.parseBinary(bpElvis) // a range expression
		re, ok := r.(*ast.RangeExpr)
		if !ok {
			p.errorf(r.Span(), "expected a range after 'in'")
			return &ast.LiteralPat{Value: r}
		}
		return &ast.RangePat{Range: re, Pos: p.spanFrom(start)}
	case lexer.LParen:
		p.next()
		tp := &ast.TuplePat{}
		for !p.at(lexer.RParen, lexer.EOF) {
			tp.Elems = append(tp.Elems, p.parsePattern(true))
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
		tp.Pos = p.spanFrom(start)
		return tp
	case lexer.KwNull, lexer.KwTrue, lexer.KwFalse, lexer.Int, lexer.Float, lexer.String, lexer.Char:
		return &ast.LiteralPat{Value: p.parsePrimary()}
	case lexer.Minus:
		p.next()
		x := p.parsePrimary()
		return &ast.LiteralPat{Value: &ast.UnaryExpr{Op: lexer.Minus, X: x, Pos: p.spanFrom(start)}}
	case lexer.Ident:
		if binding {
			// `Some(x)` inside a destructuring is a nested variant pattern;
			// a plain name binds.
			if p.peek(1).Kind == lexer.LParen || (p.peek(1).Kind == lexer.Dot && p.peek(2).Kind == lexer.Ident) {
				return p.parseTypePatternRest(start)
			}
			t := p.next()
			return &ast.BindPat{Name: ast.Ident{Name: t.Text, Pos: t.Span}}
		}
		// `Variant(...)` or `Sealed.Variant(...)` destructures; a bare name is a
		// constant (or a field-less variant, decided by the checker).
		i := 0
		for p.peek(i).Kind == lexer.Ident && p.peek(i+1).Kind == lexer.Dot {
			i += 2
		}
		if p.peek(i).Kind == lexer.Ident && p.peek(i+1).Kind == lexer.LParen {
			return p.parseTypePatternRest(start)
		}
		x := p.parsePostfix()
		return &ast.LiteralPat{Value: x}
	case lexer.Dot:
		// leading-dot variant: `.none`, `.some(x)`
		p.next()
		name, _ := p.expectIdent()
		t := &ast.NamedType{Path: []ast.Ident{name}, Pos: p.spanFrom(start)}
		return p.parseTypePatternFields(t, start)
	}
	p.errorf(p.span(), "expected a pattern, found %s", p.cur().Describe())
	return &ast.LiteralPat{Value: &ast.BadExpr{Pos: p.span()}}
}

// parseTypePatternRest parses what follows `is`: a type and an optional
// destructuring list.
func (p *Parser) parseTypePatternRest(start source.Span) *ast.TypePat {
	t := p.parseType()
	return p.parseTypePatternFields(t, start)
}

func (p *Parser) parseTypePatternFields(t ast.Type, start source.Span) *ast.TypePat {
	tp := &ast.TypePat{Type: t}
	if p.at(lexer.LParen) {
		tp.HasArg = true
		p.next()
		for !p.at(lexer.RParen, lexer.EOF) {
			var fp ast.FieldPat
			if p.at(lexer.Ident) && p.peek(1).Kind == lexer.Colon {
				id := p.next()
				p.next()
				fp.Name = ast.Ident{Name: id.Text, Pos: id.Span}
				fp.Pat = p.parsePattern(true)
			} else if p.at(lexer.Ident) && (p.peek(1).Kind == lexer.Comma || p.peek(1).Kind == lexer.RParen) {
				id := p.next()
				fp.Name = ast.Ident{Name: id.Text, Pos: id.Span}
			} else {
				// positional sub-pattern; the checker matches it to the
				// variant's single field (e.g. `Some(Some(x))`)
				fp.Pat = p.parsePattern(true)
			}
			tp.Fields = append(tp.Fields, fp)
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
	}
	tp.Pos = p.spanFrom(start)
	return tp
}
