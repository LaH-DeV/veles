package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// parseType parses a type. Postfix `?` binds tighter than prefix `*` (D5),
// so `*T?` is a pointer to a nullable T and `(*T)?` a nullable pointer.
func (p *Parser) parseType() ast.Type {
	start := p.span()
	var t ast.Type
	switch p.cur().Kind {
	case lexer.Star:
		p.next()
		raw := p.accept(lexer.KwRaw)
		elem := p.parseType()
		return &ast.PointerType{Elem: elem, Raw: raw, Pos: p.spanFrom(start)}
	case lexer.LParen:
		p.next()
		var elems []ast.Type
		trailingComma := false
		for !p.at(lexer.RParen, lexer.EOF) {
			elems = append(elems, p.parseType())
			trailingComma = false
			if !p.accept(lexer.Comma) {
				break
			}
			trailingComma = true
		}
		p.expect(lexer.RParen)
		if len(elems) == 1 && !trailingComma {
			t = elems[0] // parenthesised type
		} else {
			t = &ast.TupleType{Elems: elems, Pos: p.spanFrom(start)}
		}
	case lexer.KwFun:
		p.next()
		ft := &ast.FunType{}
		p.parseFunType(ft, start)
		t = ft
	case lexer.Ident:
		if p.cur().Text == "sendable" && p.peek(1).Kind == lexer.KwFun {
			// `sendable fun(A): R`: a function value that may cross a task
			// boundary (D35); contextual keyword, like `extend`
			p.next()
			p.next()
			ft := &ast.FunType{Sendable: true}
			p.parseFunType(ft, start)
			t = ft
			break
		}
		nt := &ast.NamedType{}
		for {
			id, _ := p.expectIdent()
			nt.Path = append(nt.Path, id)
			if p.at(lexer.Dot) && p.peek(1).Kind == lexer.Ident {
				p.next()
				continue
			}
			break
		}
		if p.at(lexer.Lt) {
			nt.Args = p.parseTypeArgs()
		}
		nt.Pos = p.spanFrom(start)
		t = nt
	case lexer.KwSelfTy:
		p.next()
		t = &ast.SelfType{Pos: start}
		// `Self.Item`: an associated type of the implementing type
		for p.at(lexer.Dot) && p.peek(1).Kind == lexer.Ident {
			p.next()
			name, _ := p.expectIdent()
			t = &ast.AssocType{Base: t, Name: name, Pos: p.spanFrom(start)}
		}
	default:
		p.errorExpected("a type")
		return &ast.NamedType{Path: []ast.Ident{{Name: "<error>", Pos: start}}, Pos: start}
	}

	// postfix: `?`, and the removed `::Assoc` spelling (reported, then read as `.`)
	for {
		switch {
		case p.at(lexer.DblColon):
			p.dblColon()
			name, _ := p.expectIdent()
			t = &ast.AssocType{Base: t, Name: name, Pos: p.spanFrom(start)}
		case p.at(lexer.Question):
			p.next()
			t = &ast.NullableType{Elem: t, Pos: p.spanFrom(start)}
		default:
			return t
		}
	}
}

// parseTypeArgs parses `<T, U>` in type context.
func (p *Parser) parseTypeArgs() []ast.Type {
	lt, _ := p.expect(lexer.Lt)
	var args []ast.Type
	if p.atTypeClose() {
		p.errorf(lt.Span, "empty type argument list; drop the '<>'")
	}
	for !p.atTypeClose() {
		args = append(args, p.parseType())
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expectTypeClose()
	return args
}

// atTypeClose reports the end of a type-argument list: `>`, or the first
// half of a `>>` closing two lists at once (`List<List<i64>>`).
func (p *Parser) atTypeClose() bool {
	return p.at(lexer.Gt, lexer.Shr, lexer.EOF)
}

// expectTypeClose consumes a `>`; a `>>` is split, leaving one `>` for the
// enclosing list.
func (p *Parser) expectTypeClose() {
	if p.at(lexer.Shr) {
		first := p.toks[p.pos]
		first.Kind, first.Text = lexer.Gt, ">"
		first.Span.End--
		second := first
		second.Span.Start++
		second.Span.End++
		p.toks = append(p.toks[:p.pos+1], p.toks[p.pos:]...)
		p.toks[p.pos], p.toks[p.pos+1] = first, second
	}
	p.expect(lexer.Gt)
}

// parseErrorType parses the type after `throws`: a type or a union `A | B`.
func (p *Parser) parseErrorType() ast.Type {
	start := p.span()
	first := p.parseType()
	if !p.at(lexer.Pipe) {
		return first
	}
	u := &ast.ErrorUnionType{Members: []ast.Type{first}}
	for p.accept(lexer.Pipe) {
		u.Members = append(u.Members, p.parseType())
	}
	u.Pos = p.spanFrom(start)
	return u
}

// dblColon consumes a `::` and reports it: associated types are written
// with a dot (`Self.Item`, `I.Item`, v0.24). The fix rewrites it.
func (p *Parser) dblColon() {
	tok := p.cur()
	p.next()
	if tok.Span.Start == p.lastErrPos {
		return
	}
	p.lastErrPos = tok.Span.Start
	p.diags.Items = append(p.diags.Items, source.Diagnostic{
		Severity: source.Error,
		Span:     tok.Span,
		Message:  "'::' is not Veles; an associated type is written with a dot: 'Self.Item', 'I.Item'",
		Fix:      &source.Fix{Title: "Replace '::' with '.'", Edits: []source.TextEdit{{Span: tok.Span, NewText: "."}}},
	})
}

// parseFunType parses the rest of a function type after `fun`:
// `(A, B): R suspends throws E`.
func (p *Parser) parseFunType(ft *ast.FunType, start source.Span) {
	if _, ok := p.expect(lexer.LParen); ok {
		for !p.at(lexer.RParen, lexer.EOF) {
			ft.Params = append(ft.Params, p.parseType())
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
	}
	if p.accept(lexer.Colon) {
		ft.Ret = p.parseType()
	}
	ft.Effects = p.parseEffects()
	ft.Pos = p.spanFrom(start)
}
